package helper

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"

	"github.com/gin-gonic/gin"
)

func TestGetAndValidOpenAIImageRequestAppliesFa2Defaults(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, test := range []struct {
		model      string
		resolution string
	}{
		{"gpt-image-2", "2K"},
		{"nano-banana-pro", "1K"},
		{"nano-banana2", "1K"},
		{"seedream-5-0", "2K"},
	} {
		t.Run(test.model, func(t *testing.T) {
			body := fmt.Sprintf(`{"model":%q,"prompt":"draw a lighthouse"}`, test.model)
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			ctx.Request = httptest.NewRequest("POST", "/v1/images/generations", strings.NewReader(body))
			ctx.Request.Header.Set("Content-Type", "application/json")

			req, err := GetAndValidOpenAIImageRequest(ctx, relayconstant.RelayModeImagesGenerations)
			require.NoError(t, err)
			require.Equal(t, test.resolution, req.OutputResolution)
			require.Equal(t, "1:1", req.AspectRatio)
			require.NotNil(t, req.N)
			require.Equal(t, uint(1), *req.N)
		})
	}
}

func TestGetAndValidOpenAIImageRequestRejectsInvalidFa2Parameters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name    string
		body    string
		message string
	}{
		{"short prompt", `{"model":"gpt-image-2","prompt":"ab"}`, "at least 3"},
		{"resolution", `{"model":"seedream-5-0","prompt":"draw it","output_resolution":"4K"}`, "output_resolution"},
		{"aspect", `{"model":"nano-banana2","prompt":"draw it","aspect_ratio":"3:1"}`, "aspect_ratio"},
		{"count", `{"model":"nano-banana-pro","prompt":"draw it","n":2}`, "n must be 1"},
		{"seed", `{"model":"gpt-image-2","prompt":"draw it","seed":0}`, "unsupported"},
		{"reference scheme", `{"model":"gpt-image-2","prompt":"draw it","image_urls":["file:///tmp/a.png"]}`, "HTTP(S)"},
		{"reference data type", `{"model":"gpt-image-2","prompt":"draw it","image_urls":["data:image/gif;base64,R0lGODlh"]}`, "PNG, JPEG, or WEBP"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			ctx.Request = httptest.NewRequest("POST", "/v1/images/generations", strings.NewReader(test.body))
			ctx.Request.Header.Set("Content-Type", "application/json")
			_, err := GetAndValidOpenAIImageRequest(ctx, relayconstant.RelayModeImagesGenerations)
			require.Error(t, err)
			require.Contains(t, err.Error(), test.message)
		})
	}
}

func TestGetAndValidOpenAIImageRequestEnforcesFa2ReferenceLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	urls := make([]string, 11)
	for index := range urls {
		urls[index] = fmt.Sprintf("https://example.com/%d.png", index)
	}
	encodedURLs, err := common.Marshal(urls)
	require.NoError(t, err)
	body := fmt.Sprintf(`{"model":"nano-banana-pro","prompt":"draw it","image_urls":%s}`, encodedURLs)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("POST", "/v1/images/generations", strings.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")

	_, err = GetAndValidOpenAIImageRequest(ctx, relayconstant.RelayModeImagesGenerations)
	require.Error(t, err)
	require.Contains(t, err.Error(), "at most 10")
}

func TestGetAndValidOpenAIImageRequestRejectsFa2EditsEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := `{"model":"gpt-image-2","prompt":"edit it","image_urls":["https://example.com/reference.png"]}`
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("POST", "/v1/images/edits", strings.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")

	_, err := GetAndValidOpenAIImageRequest(ctx, relayconstant.RelayModeImagesEdits)
	require.ErrorContains(t, err, "/v1/images/generations")
}

func TestGetAndValidOpenAIImageRequestAllowsGPTImage2SixImages(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := `{
		"model":"gpt-image2",
		"prompt":"make a campaign image",
		"size":"3:2",
		"image_urls":[
			"https://example.com/1.png",
			"https://example.com/2.png",
			"https://example.com/3.png",
			"https://example.com/4.png",
			"https://example.com/5.png",
			"https://example.com/6.png"
		]
	}`
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("POST", "/v1/images/generations", strings.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")

	req, err := GetAndValidOpenAIImageRequest(ctx, relayconstant.RelayModeImagesGenerations)

	require.NoError(t, err)
	require.Equal(t, "gpt-image2", req.Model)
	require.Equal(t, "3:2", req.Size)
	require.Equal(t, "3:2", req.AspectRatio)
	require.Equal(t, "1K", req.OutputResolution)
	require.Empty(t, req.ImageUrls)
	require.Equal(t, "https://example.com/1.png", gjson.GetBytes(req.Messages, "0.content.1.image_url.url").String())
	require.Equal(t, "https://example.com/6.png", gjson.GetBytes(req.Messages, "0.content.6.image_url.url").String())
}

func TestGetAndValidOpenAIImageRequestRejectsGPTImage2InvalidSize(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := `{
		"model":"gpt-image2",
		"prompt":"make a campaign image",
		"size":"1024x1024"
	}`
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("POST", "/v1/images/generations", strings.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")

	_, err := GetAndValidOpenAIImageRequest(ctx, relayconstant.RelayModeImagesGenerations)

	require.Error(t, err)
	require.Contains(t, err.Error(), "size must be one of")
}

func TestGetAndValidOpenAIImageRequestRejectsGPTImage2InvalidOutputResolution(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := `{
		"model":"gpt-image2",
		"prompt":"make a campaign image",
		"aspect_ratio":"1:1",
		"output_resolution":"2K"
	}`
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("POST", "/v1/images/generations", strings.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")

	_, err := GetAndValidOpenAIImageRequest(ctx, relayconstant.RelayModeImagesGenerations)

	require.Error(t, err)
	require.Contains(t, err.Error(), "output_resolution must be 1K")
}

func TestGetAndValidOpenAIImageRequestConvertsGPTImage2ImageToMessages(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := `{
		"model":"gpt-image2",
		"prompt":"make a campaign image",
		"aspect_ratio":"1:1",
		"image":"https://example.com/source.png"
	}`
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("POST", "/v1/images/generations", strings.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")

	req, err := GetAndValidOpenAIImageRequest(ctx, relayconstant.RelayModeImagesGenerations)

	require.NoError(t, err)
	require.Empty(t, req.Image)
	require.Equal(t, "https://example.com/source.png", gjson.GetBytes(req.Messages, "0.content.1.image_url.url").String())
}

func TestGetAndValidOpenAIImageRequestRejectsGPTImage2TooManyJSONImages(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := `{
		"model":"gpt-image2",
		"prompt":"make a campaign image",
		"aspect_ratio":"16:9",
		"image_urls":[
			"https://example.com/1.png",
			"https://example.com/2.png",
			"https://example.com/3.png",
			"https://example.com/4.png",
			"https://example.com/5.png",
			"https://example.com/6.png",
			"https://example.com/7.png"
		]
	}`
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("POST", "/v1/images/generations", strings.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")

	_, err := GetAndValidOpenAIImageRequest(ctx, relayconstant.RelayModeImagesGenerations)

	require.Error(t, err)
	require.Contains(t, err.Error(), "at most 6 uploaded images")
}

func TestGetAndValidOpenAIImageRequestRejectsGPTImage2TooManyMultipartImages(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("model", "gpt-image2"))
	require.NoError(t, writer.WriteField("prompt", "make a campaign image"))
	require.NoError(t, writer.WriteField("size", "1:1"))
	for i := 0; i < 7; i++ {
		part, err := writer.CreateFormFile("image[]", fmt.Sprintf("image-%d.png", i))
		require.NoError(t, err)
		_, err = part.Write([]byte("fake image"))
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("POST", "/v1/images/edits", &body)
	ctx.Request.Header.Set("Content-Type", writer.FormDataContentType())

	_, err := GetAndValidOpenAIImageRequest(ctx, relayconstant.RelayModeImagesEdits)

	require.Error(t, err)
	require.Contains(t, err.Error(), "at most 6 uploaded images")
}
