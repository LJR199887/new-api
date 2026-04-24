package gemini

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
)

const maxVeoImageSize = 20 * 1024 * 1024 // 20 MB

// ExtractMultipartImage reads the first `input_reference` file from a multipart
// form upload and returns a VeoImageInput. Returns nil if no file is present.
func ExtractMultipartImage(c *gin.Context, info *relaycommon.RelayInfo) *VeoImageInput {
	mf, err := c.MultipartForm()
	if err != nil {
		return nil
	}
	files, exists := mf.File["input_reference"]
	if !exists || len(files) == 0 {
		return nil
	}
	fh := files[0]
	if fh.Size > maxVeoImageSize {
		return nil
	}
	file, err := fh.Open()
	if err != nil {
		return nil
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return nil
	}

	mimeType := fh.Header.Get("Content-Type")
	if mimeType == "" || mimeType == "application/octet-stream" {
		mimeType = http.DetectContentType(fileBytes)
	}

	markTaskActionGenerate(info)
	return &VeoImageInput{
		BytesBase64Encoded: base64.StdEncoding.EncodeToString(fileBytes),
		MimeType:           mimeType,
	}
}

func markTaskActionGenerate(info *relaycommon.RelayInfo) {
	if info == nil {
		return
	}
	if info.TaskRelayInfo == nil {
		info.TaskRelayInfo = &relaycommon.TaskRelayInfo{}
	}
	info.Action = constant.TaskActionGenerate
}

// ResolveImageInput resolves Veo image input from multipart upload or request
// fields such as image/image_url/images/input_reference/image_reference.
func ResolveImageInput(c *gin.Context, info *relaycommon.RelayInfo, req relaycommon.TaskSubmitReq) (*VeoImageInput, error) {
	if img := ExtractMultipartImage(c, info); img != nil {
		return img, nil
	}

	candidates := collectImageCandidates(req)
	if len(candidates) == 0 {
		return nil, nil
	}

	proxyURL := ""
	if info != nil && info.ChannelMeta != nil {
		proxyURL = info.ChannelSetting.Proxy
	}
	for _, candidate := range candidates {
		parsed, err := ParseImageInput(candidate, proxyURL)
		if err != nil {
			return nil, err
		}
		if parsed != nil {
			markTaskActionGenerate(info)
			return parsed, nil
		}
	}
	return nil, fmt.Errorf("invalid image input: expected multipart file, data URI, base64, or accessible image URL")
}

func collectImageCandidates(req relaycommon.TaskSubmitReq) []string {
	candidates := make([]string, 0, len(req.Images)+4)
	appendCandidate := func(value string) {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			candidates = append(candidates, trimmed)
		}
	}

	appendCandidate(req.Image)
	appendCandidate(req.ImageURL)
	appendCandidate(req.InputReference)
	for _, image := range req.Images {
		appendCandidate(image)
	}
	collectRawImageReferenceCandidates(req.ImageReference, &candidates)
	return candidates
}

func collectRawImageReferenceCandidates(raw []byte, candidates *[]string) {
	if len(raw) == 0 {
		return
	}
	var parsed interface{}
	if err := common.Unmarshal(raw, &parsed); err != nil {
		return
	}
	appendImageReferenceCandidates(parsed, candidates)
}

func appendImageReferenceCandidates(value interface{}, candidates *[]string) {
	switch v := value.(type) {
	case string:
		if trimmed := strings.TrimSpace(v); trimmed != "" {
			*candidates = append(*candidates, trimmed)
		}
	case []interface{}:
		for _, item := range v {
			appendImageReferenceCandidates(item, candidates)
		}
	case map[string]interface{}:
		for _, key := range []string{"url", "image_url", "image"} {
			if candidate := strings.TrimSpace(common.Interface2String(v[key])); candidate != "" {
				*candidates = append(*candidates, candidate)
				return
			}
		}
	}
}

// ParseImageInput parses an image string (HTTP URL, data URI, or raw base64)
// into a VeoImageInput. Returns nil if the input is empty.
func ParseImageInput(imageStr string, proxyURL string) (*VeoImageInput, error) {
	imageStr = strings.TrimSpace(imageStr)
	if imageStr == "" {
		return nil, nil
	}

	if strings.HasPrefix(imageStr, "http://") || strings.HasPrefix(imageStr, "https://") {
		return downloadImageInput(imageStr, proxyURL)
	}

	if strings.HasPrefix(imageStr, "data:") {
		parsed := parseDataURI(imageStr)
		if parsed == nil {
			return nil, fmt.Errorf("invalid data URI image input")
		}
		return parsed, nil
	}

	raw, err := base64.StdEncoding.DecodeString(imageStr)
	if err != nil {
		return nil, fmt.Errorf("invalid image input, expected base64/data URI/HTTP URL: %w", err)
	}
	return &VeoImageInput{
		BytesBase64Encoded: imageStr,
		MimeType:           http.DetectContentType(raw),
	}, nil
}

func downloadImageInput(imageURL string, proxyURL string) (*VeoImageInput, error) {
	if proxyURL == "" {
		mimeType, data, err := service.GetImageFromUrl(imageURL)
		if err != nil {
			return nil, err
		}
		return &VeoImageInput{
			BytesBase64Encoded: data,
			MimeType:           mimeType,
		}, nil
	}

	fetchSetting := system_setting.GetFetchSetting()
	if err := common.ValidateURLWithFetchSetting(
		imageURL,
		fetchSetting.EnableSSRFProtection,
		fetchSetting.AllowPrivateIp,
		fetchSetting.DomainFilterMode,
		fetchSetting.IpFilterMode,
		fetchSetting.DomainList,
		fetchSetting.IpList,
		fetchSetting.AllowedPorts,
		fetchSetting.ApplyIPFilterForDomain,
	); err != nil {
		return nil, fmt.Errorf("request reject: %v", err)
	}

	client, err := service.GetHttpClientWithProxy(proxyURL)
	if err != nil {
		return nil, fmt.Errorf("new proxy http client failed: %w", err)
	}
	resp, err := client.Get(imageURL)
	if err != nil {
		return nil, fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download image: HTTP %d", resp.StatusCode)
	}
	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/octet-stream" && !strings.HasPrefix(contentType, "image/") {
		return nil, fmt.Errorf("invalid content type: %s, required image/*", contentType)
	}
	if resp.ContentLength > maxVeoImageSize {
		return nil, fmt.Errorf("image size %d exceeds maximum allowed size of %d bytes", resp.ContentLength, maxVeoImageSize)
	}

	limitReader := io.LimitReader(resp.Body, maxVeoImageSize)
	buffer := &bytes.Buffer{}
	written, err := io.Copy(buffer, limitReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read image data: %w", err)
	}
	if written >= maxVeoImageSize {
		return nil, fmt.Errorf("image size exceeds maximum allowed size of %d bytes", maxVeoImageSize)
	}

	data := base64.StdEncoding.EncodeToString(buffer.Bytes())
	if contentType == "application/octet-stream" {
		contentType = http.DetectContentType(buffer.Bytes())
	}
	return &VeoImageInput{
		BytesBase64Encoded: data,
		MimeType:           contentType,
	}, nil
}

func parseDataURI(uri string) *VeoImageInput {
	// data:image/png;base64,iVBOR...
	rest := uri[len("data:"):]
	idx := strings.Index(rest, ",")
	if idx < 0 {
		return nil
	}
	meta := rest[:idx]
	b64 := rest[idx+1:]
	if b64 == "" {
		return nil
	}

	mimeType := "application/octet-stream"
	parts := strings.SplitN(meta, ";", 2)
	if len(parts) >= 1 && parts[0] != "" {
		mimeType = parts[0]
	}

	return &VeoImageInput{
		BytesBase64Encoded: b64,
		MimeType:           mimeType,
	}
}
