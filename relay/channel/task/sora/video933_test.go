package sora

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/system_setting"
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
			for _, duration := range []int{0, 4, 5} {
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
		{"duration": 0}, {"duration": 3}, {"duration": 6}, {"duration": 4.5}, {"duration": 4, "seconds": "5"},
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

func wav933Fixture(seconds int) []byte {
	buf := new(bytes.Buffer)
	buf.WriteString("RIFF")
	_ = binary.Write(buf, binary.LittleEndian, uint32(36+seconds*16000))
	buf.WriteString("WAVEfmt ")
	for _, value := range []any{uint32(16), uint16(1), uint16(1), uint32(8000), uint32(16000), uint16(2), uint16(16)} {
		_ = binary.Write(buf, binary.LittleEndian, value)
	}
	buf.WriteString("data")
	_ = binary.Write(buf, binary.LittleEndian, uint32(seconds*16000))
	buf.Write(make([]byte, seconds*16000))
	return buf.Bytes()
}

func Test933ActualDownloadedMediaLimits(t *testing.T) {
	setting := system_setting.GetFetchSetting()
	old := *setting
	setting.EnableSSRFProtection = false
	t.Cleanup(func() { *setting = old })
	service.InitHttpClient()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/image":
			w.Header().Set("Content-Length", "20971521")
		case "/chunked":
			w.WriteHeader(200)
			w.(http.Flusher).Flush()
			_, _ = io.Copy(w, bytes.NewReader(make([]byte, video933ImageMaxBytes+1)))
		default:
			seconds := 5
			if r.URL.Path == "/short.wav" {
				seconds = 1
			}
			if r.URL.Path == "/long.wav" {
				seconds = 16
			}
			w.Header().Set("Content-Type", "audio/wav")
			_, _ = w.Write(wav933Fixture(seconds))
		}
	}))
	defer server.Close()
	for _, source := range []string{server.URL + "/image", server.URL + "/chunked"} {
		if err := check933RemoteImages(context.Background(), map[string]any{"image_urls": []string{source}}); err == nil {
			t.Fatal("oversize download accepted")
		}
	}
	for _, kind := range []string{"video", "audio"} {
		for _, suffix := range []string{"/short.wav", "/long.wav"} {
			if err := check933MediaDurations(context.Background(), map[string]any{kind + "_urls": []string{server.URL + suffix}}); err == nil {
				t.Fatal("invalid actual duration accepted")
			}
		}
		urls := []string{server.URL + "/1.wav", server.URL + "/2.wav", server.URL + "/3.wav"}
		if err := check933MediaDurations(context.Background(), map[string]any{kind + "_urls": urls}); err != nil {
			t.Fatal(err)
		}
		urls = append(urls, server.URL+"/4.wav")
		if err := check933MediaDurations(context.Background(), map[string]any{kind + "_urls": urls}); err == nil {
			t.Fatal("total > 15 accepted")
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
