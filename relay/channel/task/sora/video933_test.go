package sora

import (
	"bytes"
	"encoding/base64"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
)

var video933Models = []string{"933-video2.0", "933-video2.0-480p", "933-video2.0-mini", "933-video2.0-mini-480p"}

func Test933ModelRoutingAndBilling(t *testing.T) {
	for _, name := range video933Models {
		t.Run(name, func(t *testing.T) {
			if !common.IsDurationOnlyBillingModel(name) || !common.StringsContains(ModelList, name) || !isVideoGenerationsTaskModel(name) {
				t.Fatal("model registration missing")
			}
			a := &TaskAdaptor{baseURL: "https://upstream.example"}
			info := &relaycommon.RelayInfo{OriginModelName: name, TaskRelayInfo: &relaycommon.TaskRelayInfo{}, ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: name}}
			u, _ := a.BuildRequestURL(info)
			if u != a.baseURL+videoGenerationsTaskPath {
				t.Fatal(u)
			}
			u, _ = buildTaskFetchURL(a.baseURL, map[string]any{"task_id": "abc", "model": name})
			if u != a.baseURL+videoGenerationsTaskPath+"/abc" {
				t.Fatal(u)
			}
			for _, duration := range []int{0, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15} {
				c, _ := gin.CreateTestContext(httptest.NewRecorder())
				c.Set("task_request", relaycommon.TaskSubmitReq{Duration: duration})
				want := duration
				if want == 0 {
					want = 5
				}
				if got := a.EstimateBilling(c, info)["seconds"]; got != float64(want) {
					t.Fatalf("billing seconds = %v", got)
				}
			}
		})
	}
	if common.Is933VideoModel("933-video2.0-fast") {
		t.Fatal("unrequested model enabled")
	}
}

func Test933NormalizesWithoutAdobeFieldConversion(t *testing.T) {
	for _, name := range video933Models {
		for _, ratio := range []string{"16:9", "9:16", "4:3", "3:4", "1:1", "21:9"} {
			body := map[string]any{"seconds": "4", "metadata": map[string]any{"ratio": ratio}, "seed": 0, "watermark": false, "async": false,
				"image_url": "https://example.com/a.png", "video_reference": []any{map[string]any{"url": "https://example.com/v.mp4", "duration": 15}}, "audio_urls": []string{"https://example.com/a.mp3"}}
			if err := normalize933VideoRequest(body, name); err != nil {
				t.Fatal(err)
			}
			if body["aspect_ratio"] != ratio || body["duration"] != 4 || body["model"] != name {
				t.Fatalf("wrong payload: %#v", body)
			}
			if body["seed"] != 0 || body["watermark"] != false || body["async"] != false {
				t.Fatal("explicit zero/false dropped")
			}
			resolution := "720p"
			if strings.HasSuffix(name, "-480p") {
				resolution = "480p"
			}
			if body["resolution"] != resolution {
				t.Fatal("wrong resolution")
			}
			for _, key := range []string{"seconds", "metadata", "size", "image_url", "video_reference", "start_frame"} {
				if _, ok := body[key]; ok {
					t.Fatalf("leaked Adobe field %s", key)
				}
			}
			if len(body["image_urls"].([]string)) != 1 || len(body["video_urls"].([]string)) != 1 {
				t.Fatal("references lost")
			}
		}
	}
	body := map[string]any{"start_image_url": "https://example.com/start.png", "end_image_url": "https://example.com/end.png"}
	if err := normalize933VideoRequest(body, video933Models[0]); err != nil {
		t.Fatal(err)
	}
	if body["start_image_url"] != "https://example.com/start.png" || body["end_image_url"] != "https://example.com/end.png" {
		t.Fatal("frame order lost")
	}
}

func Test933RejectsInvalidReferencesAndParameters(t *testing.T) {
	many := func(n int) []string {
		refs := make([]string, n)
		for i := range refs {
			refs[i] = "https://example.com/a"
		}
		return refs
	}
	cases := []map[string]any{
		{"duration": 0}, {"duration": 3}, {"duration": 16}, {"duration": 4.5}, {"duration": 4, "seconds": "5"},
		{"resolution": "480p"}, {"metadata": map[string]any{"resolution": "1080p"}}, {"aspect_ratio": "2:1"},
		{"image_urls": many(10)}, {"video_urls": many(4)}, {"audio_urls": many(4)},
		{"image_url": "https://example.com/a", "image_urls": many(1)}, {"image_urls": []any{false}},
		{"start_image_url": "https://example.com/a"}, {"end_image_url": "https://example.com/a"},
		{"start_frame": many(2), "end_frame": many(2)}, {"reference_mode": "first_last"},
		{"start_image_url": "https://example.com/a", "end_image_url": "https://example.com/b", "audio_urls": many(1)},
		{"start_image_url": "https://example.com/a", "end_image_url": "https://example.com/b", "image_urls": many(1)},
		{"image_url": "file:///etc/passwd"}, {"image_url": "data:image/png;base64,?"},
		{"video_reference": []any{map[string]any{"url": "https://example.com/v", "duration": 1.99}}},
		{"audio_reference": []any{map[string]any{"url": "https://example.com/v", "duration": 15.01}}},
		{"audio_reference": []any{map[string]any{"url": "https://example.com/v", "duration": "NaN"}}},
		{"video_reference": []any{map[string]any{"url": "https://example.com/v", "duration": 8}, map[string]any{"url": "https://example.com/v2", "duration": 8}}},
		{"n": 2}, {"count": 0}, {"image_guidance": many(1)},
	}
	for i, body := range cases {
		if err := normalize933VideoRequest(body, video933Models[0]); err == nil {
			t.Errorf("case %d accepted: %#v", i, body)
		}
	}
	body := map[string]any{"image_urls": many(9), "video_urls": many(3), "audio_urls": many(3)}
	if err := normalize933VideoRequest(body, video933Models[0]); err != nil {
		t.Fatalf("limits should be inclusive: %v", err)
	}
}

func Test933ImageByteLimit(t *testing.T) {
	for _, size := range []int{0, video933ImageMaxBytes - 1, video933ImageMaxBytes, video933ImageMaxBytes + 1} {
		raw := bytes.Repeat([]byte{'x'}, size)
		err := check933ImageSize(bytes.NewReader(raw))
		if (err == nil) != (size > 0 && size <= video933ImageMaxBytes) {
			t.Fatalf("size=%d err=%v", size, err)
		}
		if size >= video933ImageMaxBytes {
			_, err = video933URL("data:image/png;base64,"+base64.StdEncoding.EncodeToString(raw), true)
			if (err == nil) != (size <= video933ImageMaxBytes) {
				t.Fatalf("base64 size=%d err=%v", size, err)
			}
		}
	}
}

func Test933BuildAndValidateRequest(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", videoGenerationsTaskPath, strings.NewReader(`{"model":"933-video2.0-mini","prompt":"test video","duration":4,"aspect_ratio":"21:9","seed":0,"watermark":false}`))
	c.Request.Header.Set("Content-Type", "application/json")
	info := &relaycommon.RelayInfo{OriginModelName: "933-video2.0-mini", TaskRelayInfo: &relaycommon.TaskRelayInfo{}, ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "933-video2.0-mini"}}
	a := &TaskAdaptor{}
	if err := a.ValidateRequestAndSetAction(c, info); err != nil {
		t.Fatal(err)
	}
	r, err := a.BuildRequestBody(c, info)
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	if err := common.DecodeJson(r, &body); err != nil {
		t.Fatal(err)
	}
	if body["resolution"] != "720p" || body["aspect_ratio"] != "21:9" || body["duration"] != float64(4) || body["seed"] != float64(0) || body["watermark"] != false {
		t.Fatalf("payload: %#v", body)
	}
}

func Test933GenerationDurationRange(t *testing.T) {
	for _, name := range video933Models {
		for duration := 4; duration <= 15; duration++ {
			for _, field := range []string{"duration", "seconds"} {
				body := map[string]any{}
				if field == "seconds" {
					body[field] = strconv.Itoa(duration)
				} else {
					body[field] = duration
				}
				if err := normalize933VideoRequest(body, name); err != nil {
					t.Fatalf("%s %s=%d: %v", name, field, duration, err)
				}
				if body["duration"] != duration {
					t.Fatalf("%s %s=%d normalized to %v", name, field, duration, body["duration"])
				}
			}
		}
		for _, value := range []any{0, 3, 16, 4.5, "NaN", "Infinity"} {
			for _, field := range []string{"duration", "seconds"} {
				if err := normalize933VideoRequest(map[string]any{field: value}, name); err == nil {
					t.Fatalf("%s accepted invalid %s=%v", name, field, value)
				}
			}
		}
		body := map[string]any{}
		if err := normalize933VideoRequest(body, name); err != nil {
			t.Fatal(err)
		}
		if body["duration"] != 5 {
			t.Fatal("default duration changed")
		}
	}
}
