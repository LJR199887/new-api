package gemini

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
)

func TestResolveImageInputSupportsRemoteImageURL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service.InitHttpClient()
	originalMaxFileDownloadMB := constant.MaxFileDownloadMB
	constant.MaxFileDownloadMB = 20
	defer func() {
		constant.MaxFileDownloadMB = originalMaxFileDownloadMB
	}()
	fetchSetting := system_setting.GetFetchSetting()
	originalAllowPrivateIP := fetchSetting.AllowPrivateIp
	originalAllowedPorts := append([]string(nil), fetchSetting.AllowedPorts...)
	fetchSetting.AllowPrivateIp = true
	defer func() {
		fetchSetting.AllowPrivateIp = originalAllowPrivateIP
		fetchSetting.AllowedPorts = originalAllowedPorts
	}()

	const pngBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+aK1cAAAAASUVORK5CYII="
	imageBytes, err := base64.StdEncoding.DecodeString(pngBase64)
	if err != nil {
		t.Fatalf("DecodeString failed: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(imageBytes)
	}))
	defer server.Close()
	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("url.Parse failed: %v", err)
	}
	fetchSetting.AllowedPorts = append(fetchSetting.AllowedPorts, serverURL.Port())

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/", nil)

	info := &relaycommon.RelayInfo{}
	req := relaycommon.TaskSubmitReq{
		Image: server.URL + "/reference.png",
	}

	imageInput, err := ResolveImageInput(ctx, info, req)
	if err != nil {
		t.Fatalf("ResolveImageInput returned error: %v", err)
	}
	if imageInput == nil {
		t.Fatal("expected image input, got nil")
	}
	if imageInput.MimeType != "image/png" {
		t.Fatalf("expected mime type image/png, got %q", imageInput.MimeType)
	}
	if imageInput.BytesBase64Encoded == "" {
		t.Fatal("expected base64 encoded image bytes")
	}
	if info.Action != constant.TaskActionGenerate {
		t.Fatalf("expected action %q, got %q", constant.TaskActionGenerate, info.Action)
	}
}
