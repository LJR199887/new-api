package sora

import (
	"bytes"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	projectcommon "github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

func TestModelListIncludesAdobeVideoVariants(t *testing.T) {
	models := make(map[string]bool, len(ModelList))
	for _, modelName := range ModelList {
		models[modelName] = true
	}

	for _, modelName := range []string{"veo31", "veo31-fast", "veo31-ref", "kling-v3"} {
		if !models[modelName] {
			t.Fatalf("expected ModelList to include %s", modelName)
		}
	}
}

func TestNormalizeGrokVideoRequestAddsResolutionAliases(t *testing.T) {
	body := map[string]interface{}{
		"model":   "grok-imagine-video",
		"quality": "high",
		"preset":  "fun",
	}

	normalizeGrokVideoRequest(body, "grok-imagine-video")

	if got := body["quality"]; got != "high" {
		t.Fatalf("expected quality to stay high, got %#v", got)
	}
	if got := body["resolution_name"]; got != "720p" {
		t.Fatalf("expected resolution_name 720p, got %#v", got)
	}

	videoConfig, ok := body["video_config"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected video_config map, got %#v", body["video_config"])
	}
	if got := videoConfig["resolution_name"]; got != "720p" {
		t.Fatalf("expected video_config.resolution_name 720p, got %#v", got)
	}
	if got := videoConfig["preset"]; got != "fun" {
		t.Fatalf("expected video_config.preset fun, got %#v", got)
	}
}

func TestNormalizeGrokVideoRequestBackfillsSecondsFromDuration(t *testing.T) {
	body := map[string]interface{}{
		"model":    "grok-imagine-video",
		"duration": float64(10),
	}

	normalizeGrokVideoRequest(body, "grok-imagine-video")

	if got := body["seconds"]; got != "10" {
		t.Fatalf("expected seconds to be backfilled from duration, got %#v", got)
	}
}

func TestNormalizeGrokVideoRequestBackfillsQualityFromResolutionName(t *testing.T) {
	body := map[string]interface{}{
		"model":           "grok-imagine-video",
		"resolution_name": "720p",
	}

	normalizeGrokVideoRequest(body, "grok-imagine-video")

	if got := body["quality"]; got != "high" {
		t.Fatalf("expected quality high, got %#v", got)
	}
	if got := body["resolution_name"]; got != "720p" {
		t.Fatalf("expected resolution_name 720p, got %#v", got)
	}
}

func TestNormalizeGrokVideoRequestClampsUnsupportedSeconds(t *testing.T) {
	body := map[string]interface{}{
		"model":    "grok-imagine-video",
		"duration": float64(10),
		"seconds":  "8",
	}

	normalizeGrokVideoRequest(body, "grok-imagine-video")

	if got := body["seconds"]; got != "10" {
		t.Fatalf("expected unsupported seconds to default to 10, got %#v", got)
	}
}

func TestNormalizeGrokVideoRequestPromotesImageReference(t *testing.T) {
	body := map[string]interface{}{
		"model":  "grok-imagine-video",
		"image":  "https://example.com/cover.png",
		"images": []interface{}{"https://example.com/frame-2.png"},
	}

	normalizeGrokVideoRequest(body, "grok-imagine-video")

	if _, exists := body["image"]; exists {
		t.Fatalf("expected legacy image field to be removed")
	}
	if _, exists := body["images"]; exists {
		t.Fatalf("expected legacy images field to be removed")
	}

	imageReference, ok := body["image_reference"].([]interface{})
	if !ok {
		t.Fatalf("expected image_reference array, got %#v", body["image_reference"])
	}
	if len(imageReference) != 2 {
		t.Fatalf("expected 2 image references, got %#v", imageReference)
	}
	if imageReference[0] != "https://example.com/cover.png" {
		t.Fatalf("unexpected first image reference %#v", imageReference[0])
	}
	if imageReference[1] != "https://example.com/frame-2.png" {
		t.Fatalf("unexpected second image reference %#v", imageReference[1])
	}
}

func TestNormalizeSoraVideoRequestBackfillsDurationAndAspectRatio(t *testing.T) {
	body := map[string]interface{}{
		"model":   "sora-2",
		"seconds": "10",
		"size":    "1280x720",
	}

	normalizeSoraVideoRequest(body, "sora-2")

	if got := body["duration"]; got != 10 {
		t.Fatalf("expected duration to be backfilled from seconds, got %#v", got)
	}
	if got := body["aspect_ratio"]; got != "16:9" {
		t.Fatalf("expected aspect_ratio to be backfilled from size, got %#v", got)
	}
	if _, exists := body["seconds"]; exists {
		t.Fatalf("expected seconds to be removed after normalization")
	}
	if _, exists := body["size"]; exists {
		t.Fatalf("expected size to be removed after normalization")
	}
}

func TestNormalizeSoraVideoRequestKeepsExplicitDurationAndAspectRatio(t *testing.T) {
	body := map[string]interface{}{
		"model":        "sora-2-pro",
		"duration":     float64(10),
		"seconds":      "8",
		"aspect_ratio": "9:16",
		"size":         "1024x1792",
	}

	normalizeSoraVideoRequest(body, "sora-2-pro")

	if got := body["duration"]; got != 10 {
		t.Fatalf("expected explicit duration to be preserved, got %#v", got)
	}
	if got := body["aspect_ratio"]; got != "9:16" {
		t.Fatalf("expected explicit aspect_ratio to be preserved, got %#v", got)
	}
}

func TestBuildRequestBodyConvertsSoraInputReferenceToImageURL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/v1/video/async-generations", strings.NewReader(`{
		"model": "sora2",
		"prompt": "make it cinematic",
		"duration": 10,
		"aspect_ratio": "16:9",
		"input_reference": "data:image/png;base64,aGVsbG8="
	}`))
	c.Request.Header.Set("Content-Type", "application/json")

	adaptor := &TaskAdaptor{}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "sora-2",
		},
	}

	bodyReader, err := adaptor.BuildRequestBody(c, info)
	if err != nil {
		t.Fatalf("BuildRequestBody returned error: %v", err)
	}

	if contentType := c.Request.Header.Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("expected application/json content type, got %s", contentType)
	}

	raw, err := io.ReadAll(bodyReader)
	if err != nil {
		t.Fatalf("read request body failed: %v", err)
	}

	var payload map[string]any
	if err := projectcommon.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("unmarshal request payload failed: %v", err)
	}
	if got := payload["duration"]; got != float64(10) {
		t.Fatalf("expected duration=10, got %#v", got)
	}
	if got := payload["aspect_ratio"]; got != "16:9" {
		t.Fatalf("expected aspect_ratio=16:9, got %#v", got)
	}
	if got, ok := payload["async"].(bool); !ok || !got {
		t.Fatalf("expected async=true, got %#v", payload["async"])
	}
	if _, exists := payload["input_reference"]; exists {
		t.Fatalf("expected input_reference to be removed from upstream payload")
	}
	if got := payload["image_url"]; got != "data:image/png;base64,aGVsbG8=" {
		t.Fatalf("expected image_url to be populated, got %#v", got)
	}
}

func TestNormalizeSoraVideoRequestAcceptsSora2Alias(t *testing.T) {
	body := map[string]interface{}{
		"model":        "sora2",
		"prompt":       "make an ad",
		"duration":     float64(4),
		"aspect_ratio": "16:9",
		"image":        "https://example.com/input.jpg",
	}

	normalizeSoraVideoRequest(body, "sora2")

	if got := body["image_url"]; got != "https://example.com/input.jpg" {
		t.Fatalf("expected image to be normalized to image_url, got %#v", got)
	}
	if _, exists := body["image"]; exists {
		t.Fatalf("expected image to be removed after normalization")
	}
	if got, ok := body["async"].(bool); !ok || !got {
		t.Fatalf("expected async=true, got %#v", body["async"])
	}
}

func TestNormalizeSoraVideoRequestAcceptsVeoImageURLFormat(t *testing.T) {
	for _, modelName := range []string{"veo31", "veo31-fast"} {
		t.Run(modelName, func(t *testing.T) {
			body := map[string]interface{}{
				"model":          modelName,
				"prompt":         "Create a smooth cinematic motion",
				"duration":       float64(4),
				"aspect_ratio":   "16:9",
				"resolution":     "720p",
				"image_url":      "https://example.com/a.png",
				"reference_mode": "",
			}

			normalizeSoraVideoRequest(body, modelName)

			if got := body["image_url"]; got != "https://example.com/a.png" {
				t.Fatalf("expected image_url to be preserved, got %#v", got)
			}
			if got := body["reference_mode"]; got != "frame" {
				t.Fatalf("expected %s reference_mode=frame, got %#v", modelName, got)
			}
			if got, ok := body["async"].(bool); !ok || !got {
				t.Fatalf("expected async=true, got %#v", body["async"])
			}
		})
	}
}

func TestNormalizeSoraVideoRequestDefaultsVeoRefToImageMode(t *testing.T) {
	body := map[string]interface{}{
		"model":   "veo31-ref",
		"prompt":  "Animate this still image",
		"image":   "https://example.com/source.png",
		"seconds": "4",
		"size":    "1280x720",
	}

	normalizeSoraVideoRequest(body, "veo31-ref")

	if got := body["image_url"]; got != "https://example.com/source.png" {
		t.Fatalf("expected image to be normalized to image_url, got %#v", got)
	}
	if got := body["reference_mode"]; got != "image" {
		t.Fatalf("expected veo31-ref reference_mode=image, got %#v", got)
	}
	if got := body["duration"]; got != 4 {
		t.Fatalf("expected seconds to backfill duration, got %#v", got)
	}
	if got := body["aspect_ratio"]; got != "16:9" {
		t.Fatalf("expected size to backfill aspect_ratio, got %#v", got)
	}
}

func TestBuildRequestBodyNormalizesVeoVideoGenerationPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/v1/video/async-generations", strings.NewReader(`{
		"model": "veo31-fast",
		"prompt": "Create a smooth cinematic motion",
		"duration": 4,
		"aspect_ratio": "16:9",
		"resolution": "720p",
		"image_url": "https://example.com/a.png"
	}`))
	c.Request.Header.Set("Content-Type", "application/json")

	adaptor := &TaskAdaptor{}
	info := &relaycommon.RelayInfo{
		RequestURLPath: "/v1/video/generations",
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "veo31-fast",
		},
	}

	bodyReader, err := adaptor.BuildRequestBody(c, info)
	if err != nil {
		t.Fatalf("BuildRequestBody returned error: %v", err)
	}

	raw, err := io.ReadAll(bodyReader)
	if err != nil {
		t.Fatalf("read request body failed: %v", err)
	}

	var payload map[string]any
	if err := projectcommon.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("unmarshal request payload failed: %v", err)
	}
	if got := payload["image_url"]; got != "https://example.com/a.png" {
		t.Fatalf("expected image_url to be forwarded, got %#v", got)
	}
	if got := payload["reference_mode"]; got != "frame" {
		t.Fatalf("expected reference_mode=frame, got %#v", got)
	}
	if got, ok := payload["async"].(bool); !ok || !got {
		t.Fatalf("expected async=true, got %#v", payload["async"])
	}
}

func TestBuildRequestBodyNormalizesKlingV3VideoGenerationPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/v1/video/async-generations", strings.NewReader(`{
		"model": "kling-v3",
		"prompt": "cinematic motion",
		"duration": 8,
		"aspect_ratio": "9:16",
		"resolution": "1080p",
		"reference_mode": "frame",
		"images": ["https://example.com/a.png", "https://example.com/b.png", "https://example.com/c.png"]
	}`))
	c.Request.Header.Set("Content-Type", "application/json")

	adaptor := &TaskAdaptor{}
	info := &relaycommon.RelayInfo{
		RequestURLPath: "/v1/video/generations",
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "kling-v3",
		},
	}

	bodyReader, err := adaptor.BuildRequestBody(c, info)
	if err != nil {
		t.Fatalf("BuildRequestBody returned error: %v", err)
	}

	raw, err := io.ReadAll(bodyReader)
	if err != nil {
		t.Fatalf("read request body failed: %v", err)
	}

	var payload map[string]any
	if err := projectcommon.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("unmarshal request payload failed: %v", err)
	}
	if got := payload["model"]; got != "kling-v3" {
		t.Fatalf("expected model=kling-v3, got %#v", got)
	}
	if got := payload["duration"]; got != float64(8) {
		t.Fatalf("expected duration=8, got %#v", got)
	}
	if got := payload["aspect_ratio"]; got != "9:16" {
		t.Fatalf("expected aspect_ratio=9:16, got %#v", got)
	}
	if got, ok := payload["async"].(bool); !ok || !got {
		t.Fatalf("expected async=true, got %#v", payload["async"])
	}
	if got, ok := payload["generate_audio"].(bool); !ok || !got {
		t.Fatalf("expected generate_audio=true, got %#v", payload["generate_audio"])
	}
	if got := payload["image_url"]; got != "https://example.com/a.png" {
		t.Fatalf("expected image_url to use first image, got %#v", got)
	}
	images, ok := payload["images"].([]any)
	if !ok || len(images) != 2 {
		t.Fatalf("expected two forwarded images, got %#v", payload["images"])
	}
	if _, exists := payload["resolution"]; exists {
		t.Fatalf("expected resolution to be removed for kling-v3")
	}
	if _, exists := payload["reference_mode"]; exists {
		t.Fatalf("expected reference_mode to be removed for kling-v3")
	}
}

func TestBuildRequestBodyRejectsInvalidKlingV3Duration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/v1/video/async-generations", strings.NewReader(`{
		"model": "kling-v3",
		"prompt": "cinematic motion",
		"duration": 16
	}`))
	c.Request.Header.Set("Content-Type", "application/json")

	adaptor := &TaskAdaptor{}
	info := &relaycommon.RelayInfo{
		RequestURLPath: "/v1/video/generations",
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "kling-v3",
		},
	}

	if _, err := adaptor.BuildRequestBody(c, info); err == nil {
		t.Fatalf("expected invalid kling-v3 duration to fail")
	}
}

func TestBuildRequestURLUsesVideoGenerationsPath(t *testing.T) {
	adaptor := &TaskAdaptor{baseURL: "https://upstream.example"}
	url, err := adaptor.BuildRequestURL(&relaycommon.RelayInfo{
		RequestURLPath:  "/v1/video/generations",
		OriginModelName: "veo31-fast",
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "veo31-fast",
		},
	})
	if err != nil {
		t.Fatalf("BuildRequestURL returned error: %v", err)
	}
	if url != "https://upstream.example/v1/video/generations" {
		t.Fatalf("expected video generations URL, got %s", url)
	}
}

func TestBuildRequestURLUsesVideoGenerationsPathForSora(t *testing.T) {
	adaptor := &TaskAdaptor{baseURL: "https://upstream.example"}
	url, err := adaptor.BuildRequestURL(&relaycommon.RelayInfo{
		RequestURLPath:  "/v1/video/generations",
		OriginModelName: "sora2",
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "sora-2",
		},
	})
	if err != nil {
		t.Fatalf("BuildRequestURL returned error: %v", err)
	}
	if url != "https://upstream.example/v1/video/generations" {
		t.Fatalf("expected video generations URL for sora, got %s", url)
	}
}

func TestBuildRequestURLKeepsGrokOnOpenAIVideosPath(t *testing.T) {
	adaptor := &TaskAdaptor{baseURL: "https://upstream.example"}
	url, err := adaptor.BuildRequestURL(&relaycommon.RelayInfo{
		RequestURLPath:  "/v1/video/generations",
		OriginModelName: "grok-imagine-video",
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "grok-imagine-video",
		},
	})
	if err != nil {
		t.Fatalf("BuildRequestURL returned error: %v", err)
	}
	if url != "https://upstream.example/v1/videos" {
		t.Fatalf("expected OpenAI videos URL for Grok, got %s", url)
	}
}

func TestBuildRequestBodyBuildsGrokVideoMultipartPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/v1/video/async-generations", strings.NewReader(`{
		"model": "grok-imagine-video",
		"prompt": "animate it",
		"duration": 8,
		"image": "data:image/png;base64,iVBORw0KGgo="
	}`))
	c.Request.Header.Set("Content-Type", "application/json")

	adaptor := &TaskAdaptor{}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "grok-imagine-video",
		},
	}

	bodyReader, err := adaptor.BuildRequestBody(c, info)
	if err != nil {
		t.Fatalf("BuildRequestBody returned error: %v", err)
	}

	raw, err := io.ReadAll(bodyReader)
	if err != nil {
		t.Fatalf("read request body failed: %v", err)
	}

	mediaType, params, err := mime.ParseMediaType(c.Request.Header.Get("Content-Type"))
	if err != nil {
		t.Fatalf("parse content type failed: %v", err)
	}
	if mediaType != "multipart/form-data" {
		t.Fatalf("expected multipart/form-data, got %s", mediaType)
	}
	form, err := multipart.NewReader(bytes.NewReader(raw), params["boundary"]).ReadForm(1024 * 1024)
	if err != nil {
		t.Fatalf("read multipart form failed: %v", err)
	}
	if got := form.Value["model"][0]; got != "grok-imagine-video" {
		t.Fatalf("expected upstream model to be grok-imagine-video, got %#v", got)
	}
	if got := form.Value["seconds"][0]; got != "10" {
		t.Fatalf("expected unsupported duration to default to 10 seconds, got %#v", got)
	}
	if len(form.File["input_reference[]"]) != 1 {
		t.Fatalf("expected JSON image reference to be forwarded as input_reference[], got %#v", form.File)
	}
}

func TestBuildRequestBodyMapsGrokJsonImageReferenceToFiles(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/v1/video/async-generations", strings.NewReader(`{
		"model": "grok-imagine-video",
		"prompt": "animate it",
		"seconds": 10,
		"image_reference": ["data:image/png;base64,iVBORw0KGgo="]
	}`))
	c.Request.Header.Set("Content-Type", "application/json")

	adaptor := &TaskAdaptor{}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "grok-imagine-video",
		},
	}

	bodyReader, err := adaptor.BuildRequestBody(c, info)
	if err != nil {
		t.Fatalf("BuildRequestBody returned error: %v", err)
	}
	raw, err := io.ReadAll(bodyReader)
	if err != nil {
		t.Fatalf("read request body failed: %v", err)
	}
	_, params, err := mime.ParseMediaType(c.Request.Header.Get("Content-Type"))
	if err != nil {
		t.Fatalf("parse content type failed: %v", err)
	}
	form, err := multipart.NewReader(bytes.NewReader(raw), params["boundary"]).ReadForm(1024 * 1024)
	if err != nil {
		t.Fatalf("read multipart form failed: %v", err)
	}
	files := form.File["input_reference[]"]
	if len(files) != 1 {
		t.Fatalf("expected JSON image_reference to be uploaded as input_reference[], got %#v", form.File)
	}
	if got := files[0].Header.Get("Content-Type"); got != "image/png" {
		t.Fatalf("expected image/png content type, got %q", got)
	}
}

func TestBuildRequestBodyMapsGrokMultipartImageField(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("model", "grok-imagine-video")
	_ = writer.WriteField("prompt", "animate it")
	_ = writer.WriteField("duration", "10")
	part, err := writer.CreateFormFile("image", "source.png")
	if err != nil {
		t.Fatalf("create form file failed: %v", err)
	}
	if _, err := part.Write([]byte("fake image")); err != nil {
		t.Fatalf("write form file failed: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer failed: %v", err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/v1/video/async-generations", bytes.NewReader(body.Bytes()))
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())

	adaptor := &TaskAdaptor{}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "grok-imagine-video",
		},
	}

	bodyReader, err := adaptor.BuildRequestBody(c, info)
	if err != nil {
		t.Fatalf("BuildRequestBody returned error: %v", err)
	}
	raw, err := io.ReadAll(bodyReader)
	if err != nil {
		t.Fatalf("read request body failed: %v", err)
	}
	_, params, err := mime.ParseMediaType(c.Request.Header.Get("Content-Type"))
	if err != nil {
		t.Fatalf("parse content type failed: %v", err)
	}
	form, err := multipart.NewReader(bytes.NewReader(raw), params["boundary"]).ReadForm(1024 * 1024)
	if err != nil {
		t.Fatalf("read multipart form failed: %v", err)
	}
	if got := form.Value["seconds"][0]; got != "10" {
		t.Fatalf("expected duration to be forwarded as seconds, got %q", got)
	}
	if len(form.File["input_reference[]"]) != 1 {
		t.Fatalf("expected image file to be forwarded as input_reference[], got %#v", form.File)
	}
}

func TestBuildRequestURLKeepsOpenAIVideosPath(t *testing.T) {
	adaptor := &TaskAdaptor{baseURL: "https://upstream.example"}
	url, err := adaptor.BuildRequestURL(&relaycommon.RelayInfo{
		RequestURLPath: "/v1/videos",
	})
	if err != nil {
		t.Fatalf("BuildRequestURL returned error: %v", err)
	}
	if url != "https://upstream.example/v1/videos" {
		t.Fatalf("expected OpenAI videos URL, got %s", url)
	}
}

func TestBuildTaskFetchURLUsesStoredVideoGenerationsPath(t *testing.T) {
	url, err := buildTaskFetchURL("https://upstream.example", map[string]any{
		"task_id":      "upstream-task",
		"model":        "veo31-fast",
		"request_path": "/v1/video/generations",
	})
	if err != nil {
		t.Fatalf("buildTaskFetchURL returned error: %v", err)
	}
	if url != "https://upstream.example/v1/video/generations/upstream-task" {
		t.Fatalf("expected video generations fetch URL, got %s", url)
	}
}

func TestBuildTaskFetchURLUsesStoredVideoGenerationsPathForSoraAlias(t *testing.T) {
	url, err := buildTaskFetchURL("https://upstream.example", map[string]any{
		"task_id":      "upstream-task",
		"model":        "sora2",
		"origin_model": "sora2",
		"request_path": "/v1/video/generations",
	})
	if err != nil {
		t.Fatalf("buildTaskFetchURL returned error: %v", err)
	}
	if url != "https://upstream.example/v1/video/generations/upstream-task" {
		t.Fatalf("expected video generations fetch URL for sora alias, got %s", url)
	}
}

func TestBuildTaskFetchURLKeepsGrokOnOpenAIVideosPath(t *testing.T) {
	url, err := buildTaskFetchURL("https://upstream.example", map[string]any{
		"task_id":      "upstream-task",
		"model":        "grok-imagine-video",
		"request_path": "/v1/video/generations",
	})
	if err != nil {
		t.Fatalf("buildTaskFetchURL returned error: %v", err)
	}
	if url != "https://upstream.example/v1/videos/upstream-task" {
		t.Fatalf("expected OpenAI videos fetch URL for Grok, got %s", url)
	}
}

func TestBuildTaskFetchURLDefaultsToOpenAIVideosPath(t *testing.T) {
	url, err := buildTaskFetchURL("https://upstream.example", map[string]any{
		"task_id": "upstream-task",
	})
	if err != nil {
		t.Fatalf("buildTaskFetchURL returned error: %v", err)
	}
	if url != "https://upstream.example/v1/videos/upstream-task" {
		t.Fatalf("expected OpenAI videos fetch URL, got %s", url)
	}
}

func TestParseTaskResultAcceptsFloatProgress(t *testing.T) {
	adaptor := &TaskAdaptor{}
	taskInfo, err := adaptor.ParseTaskResult([]byte(`{"status":"completed","progress":100.0,"video_url":"https://cdn.example/video.mp4"}`))
	if err != nil {
		t.Fatalf("ParseTaskResult returned error: %v", err)
	}
	if taskInfo.Status != "SUCCESS" {
		t.Fatalf("expected success status, got %s", taskInfo.Status)
	}
	if taskInfo.Url != "https://cdn.example/video.mp4" {
		t.Fatalf("expected video url, got %s", taskInfo.Url)
	}
}

func TestParseTaskResultMapsRunningToInProgress(t *testing.T) {
	adaptor := &TaskAdaptor{}
	taskInfo, err := adaptor.ParseTaskResult([]byte(`{"status":"running","progress":1.0,"created":1776350152}`))
	if err != nil {
		t.Fatalf("ParseTaskResult returned error: %v", err)
	}
	if taskInfo.Status != "IN_PROGRESS" {
		t.Fatalf("expected in-progress status, got %s", taskInfo.Status)
	}
	if taskInfo.CreatedAt != 1776350152 {
		t.Fatalf("expected created timestamp, got %d", taskInfo.CreatedAt)
	}
}

func TestParseTaskResultReadsVideoURLFromDataArray(t *testing.T) {
	adaptor := &TaskAdaptor{}
	taskInfo, err := adaptor.ParseTaskResult([]byte(`{"status":"completed","progress":100.0,"data":[{"url":"https://cdn.example/from-data.mp4"}]}`))
	if err != nil {
		t.Fatalf("ParseTaskResult returned error: %v", err)
	}
	if taskInfo.Status != "SUCCESS" {
		t.Fatalf("expected success status, got %s", taskInfo.Status)
	}
	if taskInfo.Url != "https://cdn.example/from-data.mp4" {
		t.Fatalf("expected data array video url, got %s", taskInfo.Url)
	}
}

func TestParseTaskResultFailsCompletedWithoutURL(t *testing.T) {
	adaptor := &TaskAdaptor{}
	taskInfo, err := adaptor.ParseTaskResult([]byte(`{"status":"completed","progress":100.0}`))
	if err != nil {
		t.Fatalf("ParseTaskResult returned error: %v", err)
	}
	if taskInfo.Status != "FAILURE" {
		t.Fatalf("expected failure status, got %s", taskInfo.Status)
	}
	if taskInfo.Reason == "" {
		t.Fatalf("expected failure reason")
	}
}

func TestParseTaskResultReadsStringError(t *testing.T) {
	adaptor := &TaskAdaptor{}
	taskInfo, err := adaptor.ParseTaskResult([]byte(`{
		"id": "vidgen-xxxxxxxxxxxxxxxxxxxxxxxx",
		"object": "video.generation",
		"created": 1776657884,
		"model": "sora2",
		"status": "failed",
		"task_id": "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
		"progress": 1.0,
		"error": "video poll failed: 451 {\"error_code\":\"video_unsafe\",\"message\":\"The generated video appears to be unsafe. Try modifying the prompts or the seeds.\"}"
	}`))
	if err != nil {
		t.Fatalf("ParseTaskResult returned error: %v", err)
	}
	if taskInfo.Status != "FAILURE" {
		t.Fatalf("expected failure status, got %s", taskInfo.Status)
	}
	if !strings.Contains(taskInfo.Reason, "video poll failed: 451") {
		t.Fatalf("expected string error reason, got %q", taskInfo.Reason)
	}
	if !strings.Contains(taskInfo.Reason, "video_unsafe") {
		t.Fatalf("expected upstream error code in reason, got %q", taskInfo.Reason)
	}
}

func TestDoResponsePrefersTaskIDForUpstreamPolling(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	resp := &http.Response{
		StatusCode: http.StatusAccepted,
		Body: io.NopCloser(strings.NewReader(`{
			"id":"vidgen-abc123",
			"object":"video.generation",
			"created":1776410000,
			"model":"sora2",
			"status":"queued",
			"task_id":"abc123def456",
			"progress":0
		}`)),
	}

	adaptor := &TaskAdaptor{}
	info := &relaycommon.RelayInfo{
		TaskRelayInfo: &relaycommon.TaskRelayInfo{
			PublicTaskID: "task_public_123",
		},
	}

	upstreamID, _, taskErr := adaptor.DoResponse(c, resp, info)
	if taskErr != nil {
		t.Fatalf("DoResponse returned error: %v", taskErr)
	}
	if upstreamID != "abc123def456" {
		t.Fatalf("expected upstream task_id to be preferred, got %s", upstreamID)
	}
}
