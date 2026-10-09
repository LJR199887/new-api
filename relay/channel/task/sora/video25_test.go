package sora

import (
	"fmt"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
)

func TestVideo25DocumentedModes(t *testing.T) {
	tests := []struct {
		name  string
		model string
		mode  string
		input map[string]interface{}
	}{
		{"text", "video-2.5", "t2v", map[string]interface{}{}},
		{"text-480p", "video-2.5-480p", "t2v", map[string]interface{}{"resolution": "480p"}},
		{"frames", "video-2.5", "i2v_start_end", map[string]interface{}{"start_image_url": "https://example.com/start.png", "end_image_url": "https://example.com/end.png"}},
		{"image", "video-2.5-480p", "i2v_ref", map[string]interface{}{"image_urls": []any{"https://example.com/image.png"}}},
		{"video-audio", "video-2.5", "t2v_video_ref", map[string]interface{}{"video_urls": []any{"https://example.com/motion.mp4"}, "audio_urls": []any{"https://example.com/music.mp3"}}},
		{"multimodal", "video-2.5-480p", "i2v_ref", map[string]interface{}{"image_urls": []any{"data:image/png;base64,AAAA"}, "video_urls": []any{"https://example.com/motion.mp4"}, "audio_urls": []any{"https://example.com/music.mp3"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := tt.input
			body["prompt"] = "cinematic forest motion"
			body["model"] = tt.model
			body["generation_mode"] = tt.mode
			if err := normalizeSeedanceVideoRequest(body, tt.model); err != nil {
				t.Fatal(err)
			}
			if got := body["generation_mode"]; got != tt.mode {
				t.Fatalf("mode = %v, want %s", got, tt.mode)
			}
			wantResolution := "720p"
			if tt.model == "video-2.5-480p" {
				wantResolution = "480p"
			}
			if body["resolution"] != wantResolution || body["aspect_ratio"] != "9:16" {
				t.Fatalf("unexpected resolution or ratio: %#v", body)
			}
			wantDuration := 5
			if tt.mode == "i2v_ref" || tt.mode == "i2v_start_end" {
				wantDuration = 4
			}
			if body["duration"] != wantDuration {
				t.Fatalf("duration = %v, want %d", body["duration"], wantDuration)
			}
			for _, key := range []string{"image_urls", "video_urls", "audio_urls", "start_image_url", "end_image_url"} {
				if tt.input[key] != nil && body[key] == nil {
					t.Fatalf("lost %s: %#v", key, body)
				}
			}
			for _, legacy := range []string{"size", "start_frame", "video_reference", "audio_reference", "async"} {
				if _, ok := body[legacy]; ok {
					t.Fatalf("unexpected legacy field %s", legacy)
				}
			}
		})
	}
}

func TestVideo25LegacyCreativeCenterAliases(t *testing.T) {
	body := map[string]interface{}{
		"prompt": "forest motion", "image_url": "https://example.com/a.png",
		"video_reference": []any{map[string]any{"url": "https://example.com/v.mp4", "duration": 20}},
		"audio_reference": []any{map[string]any{"url": "https://example.com/a.mp3", "duration": 25}},
	}
	if err := normalizeVideo25Request(body, "video-2.5"); err != nil {
		t.Fatal(err)
	}
	if _, hasMode := body["generation_mode"]; hasMode {
		t.Fatalf("generation_mode should be inferred by the receiving service: %#v", body)
	}
	if len(body["video_urls"].([]string)) != 1 || len(body["audio_urls"].([]string)) != 1 {
		t.Fatalf("legacy references not normalized: %#v", body)
	}
	if _, ok := body["video_reference"]; ok {
		t.Fatalf("legacy reference object leaked: %#v", body)
	}
}

func TestVideo25LimitsAndValidation(t *testing.T) {
	makeURLs := func(n int, ext string) []any {
		out := make([]any, n)
		for i := range out {
			out[i] = fmt.Sprintf("https://example.com/%d.%s", i, ext)
		}
		return out
	}
	for _, tt := range []struct {
		name string
		body map[string]interface{}
	}{
		{"too many images", map[string]interface{}{"image_urls": makeURLs(31, "png")}},
		{"too many videos", map[string]interface{}{"video_urls": makeURLs(11, "mp4")}},
		{"too many audios", map[string]interface{}{"image_urls": makeURLs(1, "png"), "audio_urls": makeURLs(11, "mp3")}},
		{"audio only", map[string]interface{}{"audio_urls": makeURLs(1, "mp3")}},
		{"frame missing end", map[string]interface{}{"start_image_url": "https://example.com/a.png"}},
		{"frame mixed", map[string]interface{}{"start_image_url": "https://example.com/a.png", "end_image_url": "https://example.com/b.png", "video_urls": makeURLs(1, "mp4")}},
		{"wrong mode", map[string]interface{}{"generation_mode": "t2v", "image_urls": makeURLs(1, "png")}},
		{"wrong resolution", map[string]interface{}{"resolution": "1080p"}},
		{"wrong aspect", map[string]interface{}{"aspect_ratio": "2:1"}},
		{"short duration", map[string]interface{}{"duration": 3}},
		{"long duration", map[string]interface{}{"duration": 31}},
		{"fractional duration", map[string]interface{}{"duration": 4.5}},
		{"local path", map[string]interface{}{"image_urls": []any{"C:/image.png"}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			tt.body["prompt"] = "forest motion"
			if err := normalizeVideo25Request(tt.body, "video-2.5"); err == nil {
				t.Fatalf("expected validation error for %s", tt.name)
			}
		})
	}
	for _, duration := range []int{4, 15, 30} {
		body := map[string]interface{}{"prompt": "forest motion", "duration": duration}
		if err := normalizeVideo25Request(body, "video-2.5"); err != nil || body["duration"] != duration {
			t.Fatalf("duration %d not accepted: %v %#v", duration, err, body)
		}
	}
	body := map[string]interface{}{"prompt": "forest motion", "image_urls": makeURLs(30, "png"), "video_urls": makeURLs(10, "mp4"), "audio_urls": makeURLs(10, "mp3")}
	if err := normalizeVideo25Request(body, "video-2.5"); err != nil {
		t.Fatalf("documented limits should pass: %v", err)
	}
}

func TestVideo25RejectsShortPrompt(t *testing.T) {
	if err := normalizeVideo25Request(map[string]interface{}{"prompt": strings.Repeat("a", 2)}, "video-2.5"); err == nil {
		t.Fatal("expected short prompt to be rejected")
	}
}

func TestVideo25BuildRequestBodyForwardsCanonicalReferences(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, modelName := range []string{"video-2.5", "video-2.5-480p"} {
		t.Run(modelName, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("POST", "/v1/video/generations", strings.NewReader(fmt.Sprintf(`{"model":%q,"prompt":"forest motion","image_urls":["https://example.com/a.png"],"video_urls":["https://example.com/v.mp4"],"audio_urls":["https://example.com/a.mp3"]}`, modelName)))
			c.Request.Header.Set("Content-Type", "application/json")
			adaptor := &TaskAdaptor{}
			info := &relaycommon.RelayInfo{RequestURLPath: "/v1/video/generations", ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: modelName}}
			bodyReader, err := adaptor.BuildRequestBody(c, info)
			if err != nil {
				t.Fatal(err)
			}
			bodyBytes, err := io.ReadAll(bodyReader)
			if err != nil {
				t.Fatal(err)
			}
			var body map[string]any
			if err := common.Unmarshal(bodyBytes, &body); err != nil {
				t.Fatal(err)
			}
			if body["image_urls"] == nil || body["video_urls"] == nil || body["audio_urls"] == nil {
				t.Fatalf("incorrect upstream payload: %#v", body)
			}
			if _, hasMode := body["generation_mode"]; hasMode {
				t.Fatalf("generation_mode should remain absent: %#v", body)
			}
			if body["duration"] != float64(4) {
				t.Fatalf("image-reference default duration should be 4: %#v", body)
			}
		})
	}
}

func TestVideo25MappedModelDropsCreativeCenterInternalFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/video/generations", strings.NewReader(`{
		"model":"video-2.5",
		"prompt":"forest motion",
		"duration":10,
		"image_urls":["https://example.com/a.png"],
		"group":"default",
		"request_id":"creative-request-123",
		"user":"creative-center-123",
		"metadata":{"creative_index":1}
	}`))
	c.Request.Header.Set("Content-Type", "application/json")
	info := &relaycommon.RelayInfo{
		OriginModelName: "video-2.5",
		RequestURLPath:  "/v1/video/generations",
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "seedance-2.5",
		},
	}
	reader, err := (&TaskAdaptor{}).BuildRequestBody(c, info)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	if err := common.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"group", "request_id", "user", "metadata"} {
		if _, exists := body[key]; exists {
			t.Fatalf("internal field %s leaked to mapped provider: %s", key, raw)
		}
	}
	if body["model"] != "seedance-2.5" || body["image_urls"] == nil || body["duration"] != float64(10) {
		t.Fatalf("unexpected mapped request: %s", raw)
	}
}

func TestVideo25BillingDefaultMatchesImageReference(t *testing.T) {
	gin.SetMode(gin.TestMode)
	adaptor := &TaskAdaptor{}
	for _, tt := range []struct {
		name string
		req  relaycommon.TaskSubmitReq
		want float64
	}{
		{"text", relaycommon.TaskSubmitReq{}, 5},
		{"image", relaycommon.TaskSubmitReq{ImageURLs: []string{"https://example.com/a.png"}}, 4},
		{"frames", relaycommon.TaskSubmitReq{StartImageURL: "https://example.com/a.png", EndImageURL: "https://example.com/b.png"}, 4},
	} {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Set("task_request", tt.req)
			info := &relaycommon.RelayInfo{OriginModelName: "video-2.5", ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "seedance-2.5"}, TaskRelayInfo: &relaycommon.TaskRelayInfo{}}
			if got := adaptor.EstimateBilling(c, info)["seconds"]; got != tt.want {
				t.Fatalf("seconds = %v, want %v", got, tt.want)
			}
		})
	}
}
