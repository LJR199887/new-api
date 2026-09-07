package openai

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

const (
	fa2ImagePollInterval = 5 * time.Second
	fa2ImagePollTimeout  = 15 * time.Minute
)

type fa2ImageTaskResponse struct {
	TaskID   string `json:"task_id"`
	PollURL  string `json:"poll_url"`
	Status   string `json:"status"`
	URL      string `json:"url"`
	ImageURL string `json:"image_url"`
	Created  int64  `json:"created"`
	Error    any    `json:"error"`
}

func isFa2ImageRelay(info *relaycommon.RelayInfo) bool {
	return info != nil && (common.IsFa2ImageModel(info.OriginModelName) || common.IsFa2ImageModel(info.UpstreamModelName))
}

func resolveFa2ImagePollURL(resp *http.Response, info *relaycommon.RelayInfo, pollURL string, taskID string) (*url.URL, error) {
	var baseURL *url.URL
	if resp != nil && resp.Request != nil && resp.Request.URL != nil {
		copyURL := *resp.Request.URL
		baseURL = &copyURL
	} else if info != nil {
		parsed, err := url.Parse(strings.TrimRight(info.ChannelBaseUrl, "/") + "/")
		if err != nil {
			return nil, err
		}
		baseURL = parsed
	}
	if baseURL == nil || baseURL.Scheme == "" || baseURL.Host == "" {
		return nil, fmt.Errorf("invalid fa2api image endpoint")
	}

	pollURL = strings.TrimSpace(pollURL)
	if pollURL == "" && strings.TrimSpace(taskID) != "" {
		pollURL = "/v1/images/generations/" + url.PathEscape(strings.TrimSpace(taskID))
	}
	ref, err := url.Parse(pollURL)
	if err != nil || pollURL == "" {
		return nil, fmt.Errorf("invalid fa2api image poll_url")
	}
	target := baseURL.ResolveReference(ref)
	if !strings.EqualFold(target.Scheme, baseURL.Scheme) || !strings.EqualFold(target.Host, baseURL.Host) {
		return nil, fmt.Errorf("fa2api image poll_url must use the submission origin")
	}
	if !strings.HasPrefix(target.Path, "/v1/images/generations/") {
		return nil, fmt.Errorf("invalid fa2api image poll path")
	}
	return target, nil
}

func fa2ImageErrorMessage(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case map[string]any:
		if message := strings.TrimSpace(common.Interface2String(typed["message"])); message != "" {
			return message
		}
	}
	return strings.TrimSpace(common.Interface2String(value))
}

func fa2ImageNewAPIError(err error, status int) *types.NewAPIError {
	return types.NewOpenAIError(err, types.ErrorCodeBadResponse, status)
}

func waitForFa2ImagePoll(ctx context.Context) error {
	timer := time.NewTimer(fa2ImagePollInterval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (a *Adaptor) handleFa2AsyncImageResponse(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (*dto.Usage, *types.NewAPIError) {
	responseBody, err := io.ReadAll(resp.Body)
	service.CloseResponseBodyGracefully(resp)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError)
	}

	var submitted fa2ImageTaskResponse
	if err := common.Unmarshal(responseBody, &submitted); err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusBadGateway)
	}
	pollURL, err := resolveFa2ImagePollURL(resp, info, submitted.PollURL, submitted.TaskID)
	if err != nil {
		return nil, fa2ImageNewAPIError(err, http.StatusBadGateway)
	}

	ctx := context.Background()
	if c != nil && c.Request != nil {
		ctx = c.Request.Context()
	}
	ctx, cancel := context.WithTimeout(ctx, fa2ImagePollTimeout)
	defer cancel()

	pollHeaders := make(http.Header)
	if resp.Request != nil {
		pollHeaders = resp.Request.Header.Clone()
	} else if err := a.SetupRequestHeader(c, &pollHeaders, info); err != nil {
		return nil, fa2ImageNewAPIError(err, http.StatusInternalServerError)
	}
	for _, key := range []string{"Content-Type", "Content-Length", "Transfer-Encoding"} {
		pollHeaders.Del(key)
	}

	client := service.GetHttpClient()
	if client == nil {
		client = http.DefaultClient
	}
	if info != nil && strings.TrimSpace(info.ChannelSetting.Proxy) != "" {
		client, err = service.NewProxyHttpClient(info.ChannelSetting.Proxy)
		if err != nil {
			return nil, fa2ImageNewAPIError(err, http.StatusInternalServerError)
		}
	}
	clientCopy := *client
	clientCopy.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}
	client = &clientCopy

	for {
		pollRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, pollURL.String(), nil)
		if err != nil {
			return nil, fa2ImageNewAPIError(err, http.StatusInternalServerError)
		}
		pollRequest.Header = pollHeaders.Clone()
		pollResp, err := client.Do(pollRequest)
		if err != nil {
			if waitErr := waitForFa2ImagePoll(ctx); waitErr != nil {
				return nil, fa2ImageNewAPIError(fmt.Errorf("fa2api image polling timed out: %w", waitErr), http.StatusGatewayTimeout)
			}
			continue
		}
		pollBody, readErr := io.ReadAll(pollResp.Body)
		service.CloseResponseBodyGracefully(pollResp)
		if readErr != nil {
			return nil, types.NewOpenAIError(readErr, types.ErrorCodeReadResponseBodyFailed, http.StatusBadGateway)
		}
		if pollResp.StatusCode == http.StatusTooManyRequests || pollResp.StatusCode >= http.StatusInternalServerError {
			if waitErr := waitForFa2ImagePoll(ctx); waitErr != nil {
				return nil, fa2ImageNewAPIError(fmt.Errorf("fa2api image polling timed out: %w", waitErr), http.StatusGatewayTimeout)
			}
			continue
		}
		if pollResp.StatusCode < http.StatusOK || pollResp.StatusCode >= http.StatusMultipleChoices {
			return nil, fa2ImageNewAPIError(fmt.Errorf("fa2api image poll HTTP %d: %s", pollResp.StatusCode, strings.TrimSpace(string(pollBody))), http.StatusBadGateway)
		}

		var task fa2ImageTaskResponse
		if err := common.Unmarshal(pollBody, &task); err != nil {
			return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusBadGateway)
		}
		switch strings.ToLower(strings.TrimSpace(task.Status)) {
		case "succeeded", "completed", "success":
			resultURL := strings.TrimSpace(task.ImageURL)
			if resultURL == "" {
				resultURL = strings.TrimSpace(task.URL)
			}
			if resultURL == "" {
				return nil, fa2ImageNewAPIError(fmt.Errorf("fa2api image task completed without a result URL"), http.StatusBadGateway)
			}
			created := task.Created
			if created == 0 {
				created = time.Now().Unix()
			}
			normalized, err := common.Marshal(dto.ImageResponse{
				Created: created,
				Data:    []dto.ImageData{{Url: resultURL}},
			})
			if err != nil {
				return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
			}
			resp.StatusCode = http.StatusOK
			resp.Header.Set("Content-Type", "application/json")
			service.IOCopyBytesGracefully(c, resp, normalized)
			return &dto.Usage{}, nil
		case "failed", "failure", "error":
			message := fa2ImageErrorMessage(task.Error)
			if message == "" {
				message = "fa2api image generation failed"
			}
			return nil, fa2ImageNewAPIError(fmt.Errorf("%s", message), http.StatusBadGateway)
		case "queued", "running", "processing", "in_progress", "submitted", "":
			if waitErr := waitForFa2ImagePoll(ctx); waitErr != nil {
				return nil, fa2ImageNewAPIError(fmt.Errorf("fa2api image polling timed out: %w", waitErr), http.StatusGatewayTimeout)
			}
		default:
			return nil, fa2ImageNewAPIError(fmt.Errorf("unknown fa2api image task status: %s", task.Status), http.StatusBadGateway)
		}
	}
}
