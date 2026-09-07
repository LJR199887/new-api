package openai

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestHandleFa2AsyncImageResponsePollsAndNormalizesResult(t *testing.T) {
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
	info := &relaycommon.RelayInfo{
		OriginModelName: "gpt-image-2",
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelBaseUrl:    server.URL,
			UpstreamModelName: "gpt-image-2",
		},
	}

	usage, apiErr := (&Adaptor{}).handleFa2AsyncImageResponse(c, info, submitResponse)
	require.Nil(t, apiErr)
	require.NotNil(t, usage)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "Bearer upstream-key", receivedAuthorization)
	var response dto.ImageResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.Len(t, response.Data, 1)
	require.Equal(t, "https://cdn.example.com/result.webp", response.Data[0].Url)
}

func TestResolveFa2ImagePollURLRejectsDifferentOrigin(t *testing.T) {
	response := &http.Response{Request: httptest.NewRequest(http.MethodPost, "https://api.example.com/v1/images/generations", nil)}
	_, err := resolveFa2ImagePollURL(response, nil, "https://attacker.example/v1/images/generations/task", "")
	require.ErrorContains(t, err, "submission origin")
}
