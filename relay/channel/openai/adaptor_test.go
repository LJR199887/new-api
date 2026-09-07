package openai

import (
	"bytes"
	"io"
	"mime"
	"mime/multipart"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"

	"github.com/gin-gonic/gin"
)

func TestConvertImageRequestAllowsJSONEditPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("POST", "/v1/images/edits", nil)

	adaptor := &Adaptor{}
	converted, err := adaptor.ConvertImageRequest(ctx, &relaycommon.RelayInfo{
		RelayMode: relayconstant.RelayModeImagesEdits,
	}, dto.ImageRequest{
		Model:          "grok-imagine-image-edit",
		Prompt:         "enhance this image",
		N:              lo.ToPtr(uint(1)),
		Size:           "1024x1024",
		ResponseFormat: "url",
		Image:          []byte(`{"url":"data:image/png;base64,aGVsbG8="}`),
	})
	if err != nil {
		t.Fatalf("ConvertImageRequest returned error: %v", err)
	}

	body, ok := converted.(*bytes.Buffer)
	if !ok {
		t.Fatalf("expected *bytes.Buffer, got %T", converted)
	}

	mediaType, params, err := mime.ParseMediaType(ctx.Request.Header.Get("Content-Type"))
	if err != nil {
		t.Fatalf("parse content type failed: %v", err)
	}
	if mediaType != "multipart/form-data" {
		t.Fatalf("unexpected media type: %s", mediaType)
	}

	form, err := multipart.NewReader(bytes.NewReader(body.Bytes()), params["boundary"]).ReadForm(1 << 20)
	if err != nil {
		t.Fatalf("read multipart form failed: %v", err)
	}
	if form.Value["model"][0] != "grok-imagine-image-edit" {
		t.Fatalf("unexpected model: %v", form.Value["model"])
	}
	if form.Value["prompt"][0] != "enhance this image" {
		t.Fatalf("unexpected prompt: %v", form.Value["prompt"])
	}

	files := form.File["image"]
	if len(files) != 1 {
		t.Fatalf("expected one image file, got %d", len(files))
	}
	file, err := files[0].Open()
	if err != nil {
		t.Fatalf("open image file failed: %v", err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("read image file failed: %v", err)
	}
	if string(content) != "hello" {
		t.Fatalf("unexpected image file content: %q", string(content))
	}
}

func TestConvertImageRequestSupportsMultipleJSONEditImages(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("POST", "/v1/images/edits", nil)

	adaptor := &Adaptor{}
	converted, err := adaptor.ConvertImageRequest(ctx, &relaycommon.RelayInfo{
		RelayMode: relayconstant.RelayModeImagesEdits,
	}, dto.ImageRequest{
		Model:  "gpt-image2",
		Prompt: "combine these references",
		Image: []byte(`[
			{"data":"aGVsbG8=","filename":"first.png","mime_type":"image/png"},
			{"data":"d29ybGQ=","filename":"second.png","mime_type":"image/png"}
		]`),
	})
	if err != nil {
		t.Fatalf("ConvertImageRequest returned error: %v", err)
	}

	body, ok := converted.(*bytes.Buffer)
	if !ok {
		t.Fatalf("expected *bytes.Buffer, got %T", converted)
	}

	mediaType, params, err := mime.ParseMediaType(ctx.Request.Header.Get("Content-Type"))
	if err != nil {
		t.Fatalf("parse content type failed: %v", err)
	}
	if mediaType != "multipart/form-data" {
		t.Fatalf("unexpected media type: %s", mediaType)
	}

	form, err := multipart.NewReader(bytes.NewReader(body.Bytes()), params["boundary"]).ReadForm(1 << 20)
	if err != nil {
		t.Fatalf("read multipart form failed: %v", err)
	}
	files := form.File["image[]"]
	if len(files) != 2 {
		t.Fatalf("expected two image[] files, got %d", len(files))
	}
}

func TestConvertImageRequestPreservesImageUrlsForGenerations(t *testing.T) {
	adaptor := &Adaptor{}

	converted, err := adaptor.ConvertImageRequest(nil, &relaycommon.RelayInfo{
		RelayMode: relayconstant.RelayModeImagesGenerations,
	}, dto.ImageRequest{
		Model:            "nano-banana",
		Prompt:           "put logo on toothpaste",
		ImageUrls:        []byte(`["https://example.com/1.png","https://example.com/2.png"]`),
		AspectRatio:      "16:9",
		OutputResolution: "2K",
		ExtraBody:        []byte(`{"google":{"image_config":{"aspect_ratio":"16:9","image_size":"2K"}}}`),
	})
	if err != nil {
		t.Fatalf("ConvertImageRequest returned error: %v", err)
	}

	encoded, err := common.Marshal(converted)
	if err != nil {
		t.Fatalf("marshal converted request: %v", err)
	}
	if gjson.GetBytes(encoded, "image_urls.0").String() != "https://example.com/1.png" {
		t.Fatalf("unexpected first image_urls item: %s", string(encoded))
	}
	if gjson.GetBytes(encoded, "image_urls.1").String() != "https://example.com/2.png" {
		t.Fatalf("unexpected second image_urls item: %s", string(encoded))
	}
	if gjson.GetBytes(encoded, "extra_body.google.image_config.aspect_ratio").String() != "16:9" {
		t.Fatalf("unexpected extra_body aspect ratio: %s", string(encoded))
	}
}

func TestConvertImageRequestStripsUnsupportedFa2Fields(t *testing.T) {
	n := uint(1)
	adaptor := &Adaptor{}
	converted, err := adaptor.ConvertImageRequest(nil, &relaycommon.RelayInfo{
		RelayMode:       relayconstant.RelayModeImagesGenerations,
		OriginModelName: "gpt-image-2",
	}, dto.ImageRequest{
		Model:            "gpt-image-2",
		Prompt:           "draw a lighthouse",
		N:                &n,
		ResponseFormat:   "url",
		AspectRatio:      "16:9",
		OutputResolution: "2K",
		ImageUrls:        []byte(`["https://example.com/reference.png"]`),
		ExtraBody:        []byte(`{"ignored":true}`),
	})
	require.NoError(t, err)

	encoded, err := common.Marshal(converted)
	require.NoError(t, err)
	require.Equal(t, "gpt-image-2", gjson.GetBytes(encoded, "model").String())
	require.Equal(t, "https://example.com/reference.png", gjson.GetBytes(encoded, "image_urls.0").String())
	require.False(t, gjson.GetBytes(encoded, "n").Exists())
	require.False(t, gjson.GetBytes(encoded, "response_format").Exists())
	require.False(t, gjson.GetBytes(encoded, "extra_body").Exists())
}
