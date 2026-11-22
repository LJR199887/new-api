package openai

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestHandleFa2AsyncImageResponsePollsAndNormalizesResult(t *testing.T) {
	for _, modelName := range []string{"gpt-image-2", "gpt-image-2.5-flare", "gpt-image-2.5-sunburst"} {
		t.Run(modelName, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			var receivedAuthorization string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "/v1/images/generations/task-123", r.URL.Path)
				receivedAuthorization = r.Header.Get("Authorization")
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"task_id":"task-123","status":"succeeded","image_url":"https://cdn.example.com/result.webp"}`))
			}))
			defer server.Close()

			submitRequest := httptest.NewRequest(http.MethodPost, server.URL+"/v1/images/generations", bytes.NewReader(nil))
			submitRequest.Header.Set("Authorization", "Bearer upstream-key")
			submitResponse := &http.Response{
				StatusCode: http.StatusAccepted,
				Header:     make(http.Header),
				Body:       io.NopCloser(bytes.NewBufferString(`{"task_id":"task-123","poll_url":"/v1/images/generations/task-123","status":"queued"}`)),
				Request:    submitRequest,
			}
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", bytes.NewReader(nil))
			var recordedIDs []string
			common.SetContextKey(c, constant.ContextKeyRecordImageUpstreamTaskID, func(id string) {
				recordedIDs = append(recordedIDs, id)
			})
			info := &relaycommon.RelayInfo{
				OriginModelName: modelName,
				ChannelMeta: &relaycommon.ChannelMeta{
					ChannelBaseUrl:    server.URL,
					UpstreamModelName: modelName,
				},
			}

			require.True(t, isFa2ImageRelay(info))
			usage, apiErr := (&Adaptor{}).handleFa2AsyncImageResponse(c, info, submitResponse)
			require.Nil(t, apiErr)
			require.NotNil(t, usage)
			require.Equal(t, http.StatusOK, recorder.Code)
			require.Equal(t, "Bearer upstream-key", receivedAuthorization)
			var response dto.ImageResponse
			require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
			require.Len(t, response.Data, 1)
			require.Equal(t, "https://cdn.example.com/result.webp", response.Data[0].Url)
			require.Equal(t, []string{"task-123"}, recordedIDs)
			require.NotContains(t, recorder.Body.String(), "task-123")

		})
	}
}

func TestRecordFa2ImageUpstreamIDWhenPollingFails(t *testing.T) {
	for _, tc := range []struct {
		name       string
		submitBody string
		pollBody   string
		expected   []string
	}{
		{
			name:       "ID from submission",
			submitBody: `{"task_id":"provider-id"}`,
			pollBody:   `{"status":"failed","error":"generation failed"}`,
			expected:   []string{"provider-id"},
		},
		{
			name:       "ID from poll when submission only supplies URL",
			submitBody: `{"poll_url":"/v1/images/generations/provider-id"}`,
			pollBody:   `{"task_id":"provider-id","status":"failed","error":"generation failed"}`,
			expected:   []string{"provider-id"},
		},
		{
			name:       "no explicit upstream ID",
			submitBody: `{"poll_url":"/v1/images/generations/provider-id"}`,
			pollBody:   `{"status":"failed","error":"generation failed"}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.WriteString(w, tc.pollBody)
			}))
			defer server.Close()
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", nil)
			var recordedIDs []string
			common.SetContextKey(c, constant.ContextKeyRecordImageUpstreamTaskID, func(id string) {
				recordedIDs = append(recordedIDs, id)
			})
			response := &http.Response{
				Body:    io.NopCloser(bytes.NewBufferString(tc.submitBody)),
				Request: httptest.NewRequest(http.MethodPost, server.URL+"/v1/images/generations", nil),
			}
			_, apiErr := (&Adaptor{}).handleFa2AsyncImageResponse(c, nil, response)
			require.NotNil(t, apiErr)
			require.Contains(t, apiErr.Error(), "generation failed")
			require.Equal(t, tc.expected, recordedIDs)
		})
	}
}

func TestResolveFa2ImagePollURLRejectsDifferentOrigin(t *testing.T) {
	response := &http.Response{Request: httptest.NewRequest(http.MethodPost, "https://api.example.com/v1/images/generations", nil)}
	_, err := resolveFa2ImagePollURL(response, nil, "https://attacker.example/v1/images/generations/task", "")
	require.ErrorContains(t, err, "submission origin")
}
