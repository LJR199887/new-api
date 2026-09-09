package sora

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
)

func Test933ForwardsReferenceURLsWithoutDownloading(t *testing.T) {
	// If local validation accidentally returns, even unreachable/failed media
	// endpoints must expose the regression instead of silently passing the test.
	setting := system_setting.GetFetchSetting()
	old := *setting
	setting.EnableSSRFProtection = false
	t.Cleanup(func() { *setting = old })
	service.InitHttpClient()
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		http.Error(w, "media must only be fetched by the upstream", http.StatusServiceUnavailable)
	}))
	defer server.Close()
	imageURL := server.URL + "/image.png?signature=keep"
	videoURL := server.URL + "/video.mp4?signature=keep"
	audioURL := server.URL + "/audio.mp3?signature=keep"
	cases := []struct {
		name string
		refs map[string]any
		want map[string]any
	}{
		{"image", map[string]any{"image_urls": []string{imageURL}}, map[string]any{"image_urls": []any{imageURL}}},
		{"video", map[string]any{"video_urls": []string{videoURL}}, map[string]any{"video_urls": []any{videoURL}}},
		{"audio", map[string]any{"audio_urls": []string{audioURL}}, map[string]any{"audio_urls": []any{audioURL}}},
		{"frames", map[string]any{"start_frame": imageURL, "end_frame": imageURL + "&frame=end"}, map[string]any{"start_image_url": imageURL, "end_image_url": imageURL + "&frame=end"}},
		{"mixed aliases", map[string]any{
			"image_url":       imageURL,
			"video_reference": []any{map[string]any{"url": videoURL, "duration": 8}},
			"audio_reference": []any{map[string]any{"url": audioURL, "duration": 14}},
		}, map[string]any{"image_urls": []any{imageURL}, "video_urls": []any{videoURL}, "audio_urls": []any{audioURL}}},
	}
	for _, name := range video933Models {
		for _, tc := range cases {
			t.Run(name+"/"+tc.name, func(t *testing.T) {
				input := map[string]any{"model": name, "prompt": "test video", "duration": 4, "seed": 0, "watermark": false}
				for key, value := range tc.refs {
					input[key] = value
				}
				raw, err := common.Marshal(input)
				if err != nil {
					t.Fatal(err)
				}
				c, _ := gin.CreateTestContext(httptest.NewRecorder())
				c.Request = httptest.NewRequest(http.MethodPost, videoGenerationsTaskPath, bytes.NewReader(raw))
				c.Request.Header.Set("Content-Type", "application/json")
				t.Cleanup(func() { common.CleanupBodyStorage(c) })
				info := &relaycommon.RelayInfo{OriginModelName: name, TaskRelayInfo: &relaycommon.TaskRelayInfo{}, ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: name}}
				a := &TaskAdaptor{}
				if err := a.ValidateRequestAndSetAction(c, info); err != nil {
					t.Fatal(err)
				}
				body, err := a.BuildRequestBody(c, info)
				if err != nil {
					t.Fatalf("reference passthrough failed: %v", err)
				}
				var got map[string]any
				if err := common.DecodeJson(body, &got); err != nil {
					t.Fatal(err)
				}
				for key, want := range tc.want {
					if !reflect.DeepEqual(got[key], want) {
						t.Fatalf("%s = %#v, want %#v", key, got[key], want)
					}
				}
				for _, key := range []string{"image_url", "start_frame", "end_frame", "video_reference", "audio_reference"} {
					if _, exists := got[key]; exists {
						t.Fatalf("unnormalized reference field %q leaked", key)
					}
				}
				if got["model"] != name || got["duration"] != float64(4) || got["seed"] != float64(0) || got["watermark"] != false {
					t.Fatalf("generation parameters changed: %#v", got)
				}
			})
		}
	}
	if got := requests.Load(); got != 0 {
		t.Fatalf("new-api downloaded reference media %d times", got)
	}
}
