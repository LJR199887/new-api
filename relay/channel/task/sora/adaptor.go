package sora

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel"
	taskcommon "github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// ============================
// Request / Response structures
// ============================

type ContentItem struct {
	Type     string    `json:"type"`                // "text" or "image_url"
	Text     string    `json:"text,omitempty"`      // for text type
	ImageURL *ImageURL `json:"image_url,omitempty"` // for image_url type
}

type ImageURL struct {
	URL string `json:"url"`
}

type responseTask struct {
	ID                 string             `json:"id"`
	TaskID             string             `json:"task_id,omitempty"`
	Object             string             `json:"object"`
	Model              string             `json:"model"`
	Status             string             `json:"status"`
	URL                string             `json:"url,omitempty"`
	VideoURL           string             `json:"video_url,omitempty"`
	Progress           float64            `json:"progress"`
	Created            int64              `json:"created,omitempty"`
	CreatedAt          int64              `json:"created_at"`
	CompletedAt        int64              `json:"completed_at,omitempty"`
	ExpiresAt          int64              `json:"expires_at,omitempty"`
	Seconds            string             `json:"seconds,omitempty"`
	Size               string             `json:"size,omitempty"`
	RemixedFromVideoID string             `json:"remixed_from_video_id,omitempty"`
	Error              *responseTaskError `json:"error,omitempty"`
}

type responseTaskError struct {
	Message string `json:"message"`
	Code    string `json:"code"`
}

func (e *responseTaskError) UnmarshalJSON(data []byte) error {
	switch common.GetJsonType(data) {
	case "object":
		type responseTaskErrorAlias responseTaskError
		return common.Unmarshal(data, (*responseTaskErrorAlias)(e))
	case "string":
		return common.Unmarshal(data, &e.Message)
	default:
		return nil
	}
}

// ============================
// Adaptor implementation
// ============================

type TaskAdaptor struct {
	taskcommon.BaseBilling
	ChannelType int
	apiKey      string
	baseURL     string
}

const videoGenerationsTaskPath = "/v1/video/generations"

func trimTaskPathQuery(path string) string {
	path = strings.TrimSpace(path)
	if idx := strings.Index(path, "?"); idx >= 0 {
		path = path[:idx]
	}
	return path
}

func usesVideoGenerationsTaskPath(path string) bool {
	path = trimTaskPathQuery(path)
	return path == videoGenerationsTaskPath || strings.HasPrefix(path, videoGenerationsTaskPath+"/")
}

func isVideoGenerationsTaskModel(model string) bool {
	model = strings.ToLower(strings.TrimSpace(model))
	return strings.HasPrefix(model, "veo") ||
		strings.Contains(model, "/veo") ||
		strings.HasPrefix(model, "sora-2") ||
		strings.HasPrefix(model, "sora2") ||
		model == "kling-v3" ||
		strings.HasPrefix(model, "seedance-2.0")
}

func usesVideoGenerationsTaskEndpoint(path string, modelNames ...string) bool {
	if !usesVideoGenerationsTaskPath(path) {
		return false
	}
	for _, modelName := range modelNames {
		if isVideoGenerationsTaskModel(modelName) {
			return true
		}
	}
	return false
}

func taskFetchRequestPath(body map[string]any) string {
	if body == nil {
		return ""
	}
	if requestPath, ok := body["request_path"].(string); ok {
		return requestPath
	}
	return ""
}

func taskFetchModel(body map[string]any, key string) string {
	if body == nil {
		return ""
	}
	if model, ok := body[key].(string); ok {
		return model
	}
	return ""
}

func relayInfoUpstreamModelName(info *relaycommon.RelayInfo) string {
	if info == nil || info.ChannelMeta == nil {
		return ""
	}
	return info.UpstreamModelName
}

func buildTaskFetchURL(baseURL string, body map[string]any) (string, error) {
	taskID, ok := body["task_id"].(string)
	if !ok {
		return "", fmt.Errorf("invalid task_id")
	}
	if usesVideoGenerationsTaskEndpoint(
		taskFetchRequestPath(body),
		taskFetchModel(body, "model"),
		taskFetchModel(body, "origin_model"),
	) {
		return fmt.Sprintf("%s%s/%s", baseURL, videoGenerationsTaskPath, taskID), nil
	}
	return fmt.Sprintf("%s/v1/videos/%s", baseURL, taskID), nil
}

func formatTaskProgress(progress float64) string {
	if progress == float64(int64(progress)) {
		return fmt.Sprintf("%d%%", int64(progress))
	}
	return fmt.Sprintf("%.1f%%", progress)
}

func stringifyBodyValue(value any) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func normalizeGrokVideoQuality(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "720p":
		return "high"
	case "480p":
		return "standard"
	case "high", "standard":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return strings.TrimSpace(value)
	}
}

func resolutionNameFromQuality(value string) string {
	switch normalizeGrokVideoQuality(value) {
	case "high":
		return "720p"
	case "standard":
		return "480p"
	default:
		return ""
	}
}

func qualityFromResolutionName(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "720p":
		return "high"
	case "480p":
		return "standard"
	default:
		return ""
	}
}

func appendGrokVideoImageReference(target []interface{}, value interface{}) []interface{} {
	if value == nil {
		return target
	}
	switch v := value.(type) {
	case string:
		if trimmed := strings.TrimSpace(v); trimmed != "" {
			target = append(target, trimmed)
		}
	case []string:
		for _, item := range v {
			target = appendGrokVideoImageReference(target, item)
		}
	case []interface{}:
		for _, item := range v {
			target = appendGrokVideoImageReference(target, item)
		}
	case map[string]interface{}:
		target = append(target, v)
	}
	return target
}

func isGrokImagineVideoModel(upstreamModel string) bool {
	return common.NormalizeGrokImagineModelName(upstreamModel) == "grok-imagine-video"
}

func normalizeGrokVideoSeconds(value string) string {
	switch strings.TrimSpace(value) {
	case "6", "10":
		return strings.TrimSpace(value)
	default:
		return "10"
	}
}

func normalizeGrokVideoRequest(bodyMap map[string]interface{}, upstreamModel string) {
	if !isGrokImagineVideoModel(upstreamModel) {
		return
	}

	quality := normalizeGrokVideoQuality(stringifyBodyValue(bodyMap["quality"]))
	resolutionName := stringifyBodyValue(bodyMap["resolution_name"])
	preset := stringifyBodyValue(bodyMap["preset"])
	seconds := stringifyBodyValue(bodyMap["seconds"])
	duration := stringifyBodyValue(bodyMap["duration"])

	if videoConfig, ok := bodyMap["video_config"].(map[string]interface{}); ok {
		if resolutionName == "" {
			resolutionName = stringifyBodyValue(videoConfig["resolution_name"])
		}
		if preset == "" {
			preset = stringifyBodyValue(videoConfig["preset"])
		}
	}

	if quality == "" {
		quality = qualityFromResolutionName(resolutionName)
	}
	if resolutionName == "" {
		resolutionName = resolutionNameFromQuality(quality)
	}

	if quality != "" {
		bodyMap["quality"] = quality
	}
	if seconds == "" && duration != "" {
		bodyMap["seconds"] = normalizeGrokVideoSeconds(duration)
	}
	imageReferences := make([]interface{}, 0)
	imageReferences = appendGrokVideoImageReference(imageReferences, bodyMap["image_reference"])
	imageReferences = appendGrokVideoImageReference(imageReferences, bodyMap["image"])
	imageReferences = appendGrokVideoImageReference(imageReferences, bodyMap["images"])
	if len(imageReferences) > 0 {
		bodyMap["image_reference"] = imageReferences
	}
	delete(bodyMap, "image")
	delete(bodyMap, "images")
	if resolutionName != "" {
		bodyMap["resolution_name"] = resolutionName
	}
	if preset != "" {
		bodyMap["preset"] = preset
	}
	if seconds == "" {
		seconds = stringifyBodyValue(bodyMap["seconds"])
	}
	bodyMap["seconds"] = normalizeGrokVideoSeconds(seconds)
	if size := grokVideoSizeValue(bodyMap); size != "" {
		bodyMap["size"] = size
	}
	if resolutionName != "" || preset != "" {
		videoConfig := map[string]interface{}{}
		if resolutionName != "" {
			videoConfig["resolution_name"] = resolutionName
		}
		if preset != "" {
			videoConfig["preset"] = preset
		}
		bodyMap["video_config"] = videoConfig
	}
}

func grokVideoSizeValue(bodyMap map[string]interface{}) string {
	size := stringifyBodyValue(bodyMap["size"])
	if size == "" {
		if videoConfig, ok := bodyMap["video_config"].(map[string]interface{}); ok {
			size = stringifyBodyValue(videoConfig["size"])
		}
	}
	if size != "" {
		return size
	}
	width := stringifyBodyValue(bodyMap["width"])
	height := stringifyBodyValue(bodyMap["height"])
	if width != "" && height != "" {
		return width + "x" + height
	}
	return ""
}

func firstGrokVideoFieldValue(bodyMap map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if value := stringifyBodyValue(bodyMap[key]); value != "" {
			return value
		}
	}
	if videoConfig, ok := bodyMap["video_config"].(map[string]interface{}); ok {
		for _, key := range keys {
			if value := stringifyBodyValue(videoConfig[key]); value != "" {
				return value
			}
		}
	}
	return ""
}

func appendGrokVideoReferenceSource(target []string, value interface{}) []string {
	if len(target) >= 5 || value == nil {
		return target
	}
	switch v := value.(type) {
	case string:
		if trimmed := strings.TrimSpace(v); trimmed != "" {
			target = append(target, trimmed)
		}
	case []string:
		for _, item := range v {
			target = appendGrokVideoReferenceSource(target, item)
			if len(target) >= 5 {
				break
			}
		}
	case []interface{}:
		for _, item := range v {
			target = appendGrokVideoReferenceSource(target, item)
			if len(target) >= 5 {
				break
			}
		}
	case map[string]interface{}:
		target = appendGrokVideoReferenceSource(target, v["url"])
		target = appendGrokVideoReferenceSource(target, v["image"])
		target = appendGrokVideoReferenceSource(target, v["image_url"])
	case map[string]string:
		target = appendGrokVideoReferenceSource(target, v["url"])
		target = appendGrokVideoReferenceSource(target, v["image"])
		target = appendGrokVideoReferenceSource(target, v["image_url"])
	}
	return target
}

func collectGrokVideoReferenceSources(bodyMap map[string]interface{}) []string {
	sources := make([]string, 0, 5)
	for _, key := range []string{"input_reference", "input_reference[]", "image_reference", "image_reference[]", "image", "image[]", "images", "images[]", "image_url"} {
		sources = appendGrokVideoReferenceSource(sources, bodyMap[key])
		if len(sources) >= 5 {
			break
		}
	}
	return sources
}

func grokVideoReferenceFilename(contentType string, index int) string {
	extensions, _ := mime.ExtensionsByType(contentType)
	ext := ".png"
	if len(extensions) > 0 {
		ext = extensions[0]
	}
	if ext == ".jpe" {
		ext = ".jpg"
	}
	return fmt.Sprintf("reference-%d%s", index+1, ext)
}

func grokVideoReferenceBytes(source string) ([]byte, string, error) {
	source = strings.TrimSpace(source)
	if source == "" {
		return nil, "", fmt.Errorf("empty reference source")
	}

	var contentType string
	var base64Data string
	var err error
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		contentType, base64Data, err = service.GetImageFromUrl(source)
	} else {
		contentType, base64Data, err = service.DecodeBase64FileData(source)
	}
	if err != nil {
		return nil, "", err
	}
	raw, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return nil, "", fmt.Errorf("decode reference image failed: %w", err)
	}
	if contentType == "" || contentType == "application/octet-stream" {
		contentType = http.DetectContentType(raw)
	}
	return raw, contentType, nil
}

func writeGrokVideoReferenceFiles(writer *multipart.Writer, sources []string) error {
	for index, source := range sources {
		raw, contentType, err := grokVideoReferenceBytes(source)
		if err != nil {
			return fmt.Errorf("prepare grok video reference %d failed: %w", index+1, err)
		}
		header := make(textproto.MIMEHeader)
		header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="input_reference[]"; filename="%s"`, grokVideoReferenceFilename(contentType, index)))
		header.Set("Content-Type", contentType)
		part, err := writer.CreatePart(header)
		if err != nil {
			return err
		}
		if _, err := part.Write(raw); err != nil {
			return err
		}
	}
	return nil
}

func buildGrokVideoMultipartBody(bodyMap map[string]interface{}, upstreamModelName string) (io.Reader, string, error) {
	normalizeGrokVideoRequest(bodyMap, upstreamModelName)

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	fields := map[string]string{
		"model":           upstreamModelName,
		"prompt":          firstGrokVideoFieldValue(bodyMap, "prompt", "input"),
		"seconds":         normalizeGrokVideoSeconds(firstGrokVideoFieldValue(bodyMap, "seconds", "duration")),
		"size":            grokVideoSizeValue(bodyMap),
		"resolution_name": firstGrokVideoFieldValue(bodyMap, "resolution_name"),
		"preset":          firstGrokVideoFieldValue(bodyMap, "preset"),
	}
	for _, key := range []string{"model", "prompt", "seconds", "size", "resolution_name", "preset"} {
		if value := strings.TrimSpace(fields[key]); value != "" {
			if err := writer.WriteField(key, value); err != nil {
				return nil, "", err
			}
		}
	}
	if err := writeGrokVideoReferenceFiles(writer, collectGrokVideoReferenceSources(bodyMap)); err != nil {
		return nil, "", err
	}
	if err := writer.Close(); err != nil {
		return nil, "", err
	}
	return &buf, writer.FormDataContentType(), nil
}

func isSoraVideoModel(upstreamModel string) bool {
	upstreamModel = strings.ToLower(strings.TrimSpace(upstreamModel))
	return strings.HasPrefix(upstreamModel, "sora-2") || strings.HasPrefix(upstreamModel, "sora2")
}

func isKlingV3VideoModel(upstreamModel string) bool {
	return strings.EqualFold(strings.TrimSpace(upstreamModel), "kling-v3")
}

func isSeedanceVideoModel(upstreamModel string) bool {
	upstreamModel = strings.ToLower(strings.TrimSpace(upstreamModel))
	return strings.HasPrefix(upstreamModel, "seedance-2.0")
}

func usesImageURLVideoGenerationsModel(upstreamModel string) bool {
	upstreamModel = strings.ToLower(strings.TrimSpace(upstreamModel))
	return isSoraVideoModel(upstreamModel) || strings.HasPrefix(upstreamModel, "veo") || isKlingV3VideoModel(upstreamModel)
}

func seedanceAspectRatioFromSize(value string) string {
	switch strings.TrimSpace(value) {
	case "1280x720", "1920x1080":
		return "16:9"
	case "1112x834", "1664x1248":
		return "4:3"
	case "960x960", "1024x1024", "1440x1440":
		return "1:1"
	case "834x1112", "1248x1664":
		return "3:4"
	case "720x1280", "1080x1920":
		return "9:16"
	case "1470x630", "2208x944":
		return "21:9"
	default:
		return ""
	}
}

func firstSeedanceImageValue(bodyMap map[string]interface{}) string {
	for _, key := range []string{"image", "image_url", "input_reference"} {
		if value := stringifyBodyValue(bodyMap[key]); value != "" {
			return value
		}
	}
	for _, key := range []string{"images", "image_urls"} {
		switch values := bodyMap[key].(type) {
		case []interface{}:
			for _, value := range values {
				if candidate := stringifyBodyValue(value); candidate != "" {
					return candidate
				}
			}
		case []string:
			for _, value := range values {
				if candidate := strings.TrimSpace(value); candidate != "" {
					return candidate
				}
			}
		}
	}
	return ""
}

func seedanceBaseDimensionFromResolution(value string) int {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "", "720p":
		return 720
	case "480p":
		return 480
	case "1080p":
		return 1080
	default:
		if strings.HasSuffix(value, "p") {
			if parsed, err := strconv.Atoi(strings.TrimSuffix(value, "p")); err == nil && parsed > 0 {
				return parsed
			}
		}
	}
	return 0
}

func seedanceSizeFromAspectRatioAndResolution(ratio string, resolution string) string {
	parts := strings.Split(strings.TrimSpace(ratio), ":")
	if len(parts) != 2 {
		return ""
	}

	ratioWidth, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil || ratioWidth <= 0 {
		return ""
	}
	ratioHeight, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil || ratioHeight <= 0 {
		return ""
	}

	baseDimension := seedanceBaseDimensionFromResolution(resolution)
	if baseDimension <= 0 {
		return ""
	}

	width := baseDimension
	height := baseDimension
	if ratioWidth > ratioHeight {
		width = (baseDimension*ratioWidth + ratioHeight/2) / ratioHeight
	} else if ratioHeight > ratioWidth {
		height = (baseDimension*ratioHeight + ratioWidth/2) / ratioWidth
	}

	return fmt.Sprintf("%dx%d", width, height)
}

func normalizeSeedanceVideoRequest(bodyMap map[string]interface{}, upstreamModel string) {
	if !isSeedanceVideoModel(upstreamModel) {
		return
	}

	metadata, _ := bodyMap["metadata"].(map[string]interface{})
	ratio := stringifyBodyValue(metadata["ratio"])
	if ratio == "" {
		ratio = stringifyBodyValue(bodyMap["aspect_ratio"])
	}
	if ratio == "" {
		ratio = seedanceAspectRatioFromSize(stringifyBodyValue(bodyMap["size"]))
	}

	resolution := stringifyBodyValue(metadata["resolution"])
	if resolution == "" {
		resolution = stringifyBodyValue(bodyMap["resolution"])
	}
	if resolution == "" {
		resolution = stringifyBodyValue(bodyMap["resolution_name"])
	}
	if resolution == "" {
		resolution = resolutionNameFromQuality(stringifyBodyValue(bodyMap["quality"]))
	}

	duration := stringifyBodyValue(bodyMap["duration"])
	if duration == "" {
		duration = stringifyBodyValue(bodyMap["seconds"])
	}

	size := stringifyBodyValue(bodyMap["size"])
	if size == "" {
		size = seedanceSizeFromAspectRatioAndResolution(ratio, resolution)
	}

	if duration != "" {
		bodyMap["duration"] = soraDurationBodyValue(duration)
	} else {
		delete(bodyMap, "duration")
	}
	if size != "" {
		bodyMap["size"] = size
	}
	if image := firstSeedanceImageValue(bodyMap); image != "" {
		bodyMap["image_url"] = image
	}

	bodyMap["model"] = upstreamModel

	delete(bodyMap, "seconds")
	delete(bodyMap, "quality")
	delete(bodyMap, "resolution_name")
	delete(bodyMap, "video_config")
	delete(bodyMap, "image")
	delete(bodyMap, "image_urls")
	delete(bodyMap, "images")
	delete(bodyMap, "input_reference")
	delete(bodyMap, "reference_mode")
	delete(bodyMap, "async")
	delete(bodyMap, "aspect_ratio")
	delete(bodyMap, "resolution")
	delete(bodyMap, "metadata")
}

func defaultVideoGenerationsReferenceMode(upstreamModel string) string {
	upstreamModel = strings.ToLower(strings.TrimSpace(upstreamModel))
	if !strings.HasPrefix(upstreamModel, "veo") {
		return ""
	}
	if strings.Contains(upstreamModel, "ref") {
		return "image"
	}
	return "frame"
}

func soraSizeFromAspectRatio(value string) string {
	switch strings.TrimSpace(value) {
	case "16:9":
		return "1280x720"
	case "9:16":
		return "720x1280"
	default:
		return ""
	}
}

func soraAspectRatioFromSize(value string) string {
	switch strings.TrimSpace(value) {
	case "1280x720", "1792x1024":
		return "16:9"
	case "720x1280", "1024x1792":
		return "9:16"
	default:
		return ""
	}
}

func soraDurationBodyValue(value string) interface{} {
	if duration, err := strconv.Atoi(strings.TrimSpace(value)); err == nil {
		return duration
	}
	return strings.TrimSpace(value)
}

func normalizeKlingV3Duration(value string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 5, nil
	}
	duration, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("duration must be an integer between 3 and 15")
	}
	if duration < 3 || duration > 15 {
		return 0, fmt.Errorf("duration must be between 3 and 15 for kling-v3")
	}
	return duration, nil
}

func normalizeKlingV3AspectRatio(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "16:9", nil
	}
	switch value {
	case "16:9", "9:16":
		return value, nil
	default:
		return "", fmt.Errorf("aspect_ratio must be 16:9 or 9:16 for kling-v3")
	}
}

func collectKlingV3ImageURLs(bodyMap map[string]interface{}) []string {
	sources := make([]string, 0, 2)
	for _, key := range []string{"image_urls", "image_url", "image", "input_reference", "images", "image_reference"} {
		sources = appendGrokVideoReferenceSource(sources, bodyMap[key])
		if len(sources) >= 2 {
			break
		}
	}
	if len(sources) > 2 {
		return sources[:2]
	}
	return sources
}

func normalizeKlingV3VideoRequest(bodyMap map[string]interface{}) error {
	duration := stringifyBodyValue(bodyMap["duration"])
	if duration == "" {
		duration = stringifyBodyValue(bodyMap["seconds"])
	}
	normalizedDuration, err := normalizeKlingV3Duration(duration)
	if err != nil {
		return err
	}

	aspectRatio := stringifyBodyValue(bodyMap["aspect_ratio"])
	if aspectRatio == "" {
		aspectRatio = soraAspectRatioFromSize(stringifyBodyValue(bodyMap["size"]))
	}
	normalizedAspectRatio, err := normalizeKlingV3AspectRatio(aspectRatio)
	if err != nil {
		return err
	}

	imageURLs := collectKlingV3ImageURLs(bodyMap)
	bodyMap["duration"] = normalizedDuration
	bodyMap["aspect_ratio"] = normalizedAspectRatio
	bodyMap["async"] = true
	generateAudio := bodyMap["generate_audio"]
	if generateAudio == nil {
		generateAudio = bodyMap["generateAudio"]
	}
	if stringifyBodyValue(generateAudio) == "" {
		generateAudio = true
	}
	bodyMap["generate_audio"] = generateAudio
	bodyMap["generateAudio"] = generateAudio
	if len(imageURLs) > 0 {
		bodyMap["image_url"] = imageURLs[0]
		if len(imageURLs) > 1 {
			imageURLValues := make([]interface{}, 0, len(imageURLs))
			for _, imageURL := range imageURLs {
				imageURLValues = append(imageURLValues, imageURL)
			}
			bodyMap["image_urls"] = imageURLValues
		}
	}
	delete(bodyMap, "seconds")
	delete(bodyMap, "size")
	delete(bodyMap, "input_reference")
	delete(bodyMap, "image")
	delete(bodyMap, "images")
	delete(bodyMap, "image_reference")
	delete(bodyMap, "resolution")
	delete(bodyMap, "reference_mode")
	return nil
}

func hasMultipartFieldValue(values map[string][]string, key string) bool {
	items := values[key]
	for _, item := range items {
		if strings.TrimSpace(item) != "" {
			return true
		}
	}
	return false
}

func firstNonEmptyMultipartValue(values map[string][]string, keys ...string) string {
	for _, key := range keys {
		for _, value := range values[key] {
			if trimmed := strings.TrimSpace(value); trimmed != "" {
				return trimmed
			}
		}
	}
	return ""
}

func isGrokVideoReferenceFileField(fieldName string) bool {
	switch strings.TrimSpace(fieldName) {
	case "image", "image[]", "images", "images[]", "image_reference", "image_reference[]", "input_reference", "input_reference[]":
		return true
	default:
		return false
	}
}

func normalizeSoraVideoRequest(bodyMap map[string]interface{}, upstreamModel string) {
	if !usesImageURLVideoGenerationsModel(upstreamModel) {
		return
	}
	if isKlingV3VideoModel(upstreamModel) {
		return
	}

	duration := stringifyBodyValue(bodyMap["duration"])
	aspectRatio := stringifyBodyValue(bodyMap["aspect_ratio"])
	seconds := stringifyBodyValue(bodyMap["seconds"])
	size := stringifyBodyValue(bodyMap["size"])
	imageURL := stringifyBodyValue(bodyMap["image_url"])
	inputReference := stringifyBodyValue(bodyMap["input_reference"])
	image := stringifyBodyValue(bodyMap["image"])
	referenceMode := stringifyBodyValue(bodyMap["reference_mode"])

	if duration == "" {
		if seconds != "" {
			duration = seconds
		} else {
			duration = "4"
		}
	}
	if aspectRatio == "" {
		if mapped := soraAspectRatioFromSize(size); mapped != "" {
			aspectRatio = mapped
		} else {
			aspectRatio = "9:16"
		}
	}

	bodyMap["duration"] = soraDurationBodyValue(duration)
	bodyMap["aspect_ratio"] = aspectRatio
	bodyMap["async"] = true
	delete(bodyMap, "seconds")
	delete(bodyMap, "size")

	if imageURL == "" {
		switch {
		case inputReference != "":
			imageURL = inputReference
		case image != "":
			imageURL = image
		default:
			if images, ok := bodyMap["images"].([]interface{}); ok && len(images) > 0 {
				imageURL = stringifyBodyValue(images[0])
			}
		}
	}
	if imageURL != "" {
		bodyMap["image_url"] = imageURL
	}
	if referenceMode == "" && imageURL != "" {
		referenceMode = defaultVideoGenerationsReferenceMode(upstreamModel)
	}
	if referenceMode != "" {
		bodyMap["reference_mode"] = referenceMode
	}
	delete(bodyMap, "input_reference")
	delete(bodyMap, "image")
	delete(bodyMap, "images")
}

func extractVideoURL(respBody []byte) string {
	for _, path := range []string{
		"url",
		"video_url",
		"metadata.url",
		"data.url",
		"data.video_url",
		"data.0.url",
		"data.0.video_url",
		"result.url",
		"result.video_url",
		"result.data.url",
		"result.data.video_url",
		"result.data.0.url",
		"result.data.0.video_url",
		"task.url",
		"task.video_url",
		"task.data.url",
		"task.data.video_url",
		"task.data.0.url",
		"task.data.0.video_url",
		"output.video_url",
		"task_result.videos.0.url",
	} {
		if url := strings.TrimSpace(gjson.GetBytes(respBody, path).String()); url != "" {
			return url
		}
	}
	return ""
}

func firstTaskStringValue(respBody []byte, paths ...string) string {
	for _, path := range paths {
		if value := strings.TrimSpace(gjson.GetBytes(respBody, path).String()); value != "" {
			return value
		}
	}
	return ""
}

func firstTaskFloatValue(respBody []byte, paths ...string) (float64, bool) {
	for _, path := range paths {
		result := gjson.GetBytes(respBody, path)
		if !result.Exists() {
			continue
		}
		switch result.Type {
		case gjson.Number:
			return result.Float(), true
		case gjson.String:
			if value, err := strconv.ParseFloat(strings.TrimSpace(result.String()), 64); err == nil {
				return value, true
			}
		}
	}
	return 0, false
}

func firstTaskInt64Value(respBody []byte, paths ...string) (int64, bool) {
	if value, ok := firstTaskFloatValue(respBody, paths...); ok {
		return int64(value), true
	}
	return 0, false
}

func extractTaskFailureReason(respBody []byte) string {
	for _, path := range []string{
		"error.message",
		"data.error.message",
		"result.error.message",
		"task.error.message",
		"error",
		"data.error",
		"result.error",
		"task.error",
		"message",
		"data.message",
		"result.message",
		"task.message",
	} {
		result := gjson.GetBytes(respBody, path)
		if !result.Exists() {
			continue
		}
		switch result.Type {
		case gjson.String:
			if value := strings.TrimSpace(result.String()); value != "" {
				return value
			}
		case gjson.JSON:
			if value := strings.TrimSpace(result.Get("message").String()); value != "" {
				return value
			}
		}
	}
	return ""
}

func fillResponseTaskFallbacks(respBody []byte, task *responseTask) {
	if task == nil {
		return
	}

	if task.TaskID == "" {
		task.TaskID = firstTaskStringValue(respBody,
			"task_id",
			"data.task_id",
			"data.id",
			"result.task_id",
			"result.id",
			"task.task_id",
			"task.id",
		)
	}
	if task.ID == "" {
		task.ID = firstTaskStringValue(respBody,
			"id",
			"task_id",
			"data.id",
			"data.task_id",
			"result.id",
			"result.task_id",
			"task.id",
			"task.task_id",
		)
	}
	if task.Object == "" {
		task.Object = firstTaskStringValue(respBody, "object", "data.object", "result.object", "task.object")
	}
	if task.Model == "" {
		task.Model = firstTaskStringValue(respBody, "model", "data.model", "result.model", "task.model")
	}
	if task.Status == "" {
		task.Status = firstTaskStringValue(respBody, "status", "data.status", "result.status", "task.status")
	}
	if task.URL == "" {
		task.URL = extractVideoURL(respBody)
	}
	if task.VideoURL == "" {
		task.VideoURL = firstTaskStringValue(
			respBody,
			"video_url",
			"data.video_url",
			"result.video_url",
			"task.video_url",
		)
		if task.VideoURL == "" {
			task.VideoURL = task.URL
		}
	}
	if task.URL == "" {
		task.URL = task.VideoURL
	}
	if task.Progress == 0 {
		if progress, ok := firstTaskFloatValue(respBody, "progress", "data.progress", "result.progress", "task.progress"); ok {
			task.Progress = progress
		}
	}
	if task.CreatedAt == 0 {
		if createdAt, ok := firstTaskInt64Value(respBody, "created_at", "data.created_at", "result.created_at", "task.created_at"); ok {
			task.CreatedAt = createdAt
		}
	}
	if task.Created == 0 {
		if created, ok := firstTaskInt64Value(respBody, "created", "data.created", "result.created", "task.created"); ok {
			task.Created = created
		}
	}
	if task.CompletedAt == 0 {
		if completedAt, ok := firstTaskInt64Value(respBody, "completed_at", "data.completed_at", "result.completed_at", "task.completed_at"); ok {
			task.CompletedAt = completedAt
		}
	}
	if task.ExpiresAt == 0 {
		if expiresAt, ok := firstTaskInt64Value(respBody, "expires_at", "data.expires_at", "result.expires_at", "task.expires_at"); ok {
			task.ExpiresAt = expiresAt
		}
	}
	if task.Seconds == "" {
		task.Seconds = firstTaskStringValue(respBody, "seconds", "data.seconds", "result.seconds", "task.seconds")
	}
	if task.Size == "" {
		task.Size = firstTaskStringValue(respBody, "size", "data.size", "result.size", "task.size")
	}
	if task.Error == nil {
		if reason := extractTaskFailureReason(respBody); reason != "" {
			task.Error = &responseTaskError{
				Message: reason,
				Code:    firstTaskStringValue(respBody, "error.code", "data.error.code", "result.error.code", "task.error.code"),
			}
		}
	} else if task.Error.Message == "" {
		task.Error.Message = extractTaskFailureReason(respBody)
	}
}

func (a *TaskAdaptor) Init(info *relaycommon.RelayInfo) {
	a.ChannelType = info.ChannelType
	a.baseURL = info.ChannelBaseUrl
	a.apiKey = info.ApiKey
}

func validateRemixRequest(c *gin.Context) *dto.TaskError {
	var req relaycommon.TaskSubmitReq
	if err := common.UnmarshalBodyReusable(c, &req); err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}
	if strings.TrimSpace(req.Prompt) == "" {
		return service.TaskErrorWrapperLocal(fmt.Errorf("field prompt is required"), "invalid_request", http.StatusBadRequest)
	}
	// 存储原始请求到 context，与 ValidateMultipartDirect 路径保持一致
	c.Set("task_request", req)
	return nil
}

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) (taskErr *dto.TaskError) {
	if info.Action == constant.TaskActionRemix {
		return validateRemixRequest(c)
	}
	return relaycommon.ValidateMultipartDirect(c, info)
}

// EstimateBilling 根据用户请求的 seconds 计算 OtherRatios。
func (a *TaskAdaptor) EstimateBilling(c *gin.Context, info *relaycommon.RelayInfo) map[string]float64 {
	// remix 路径的 OtherRatios 已在 ResolveOriginTask 中设置
	if info.Action == constant.TaskActionRemix {
		return nil
	}

	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil
	}
	if isKlingV3VideoModel(info.UpstreamModelName) {
		return nil
	}

	seconds, _ := strconv.Atoi(req.Seconds)
	if seconds == 0 {
		seconds = req.Duration
	}
	if isGrokImagineVideoModel(info.UpstreamModelName) {
		if seconds != 6 && seconds != 10 {
			seconds = 10
		}
	} else if isSeedanceVideoModel(info.UpstreamModelName) {
		if seconds <= 0 {
			seconds = 5
		}
	} else if seconds <= 0 {
		seconds = 4
	}

	return map[string]float64{
		"seconds": float64(seconds),
	}
}

func (a *TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	if info != nil && info.TaskRelayInfo != nil && info.Action == constant.TaskActionRemix {
		return fmt.Sprintf("%s/v1/videos/%s/remix", a.baseURL, info.OriginTaskID), nil
	}
	if info != nil && usesVideoGenerationsTaskEndpoint(info.RequestURLPath, relayInfoUpstreamModelName(info), info.OriginModelName) {
		return fmt.Sprintf("%s%s", a.baseURL, videoGenerationsTaskPath), nil
	}
	return fmt.Sprintf("%s/v1/videos", a.baseURL), nil
}

// BuildRequestHeader sets required headers.
func (a *TaskAdaptor) BuildRequestHeader(c *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error {
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	req.Header.Set("Content-Type", c.Request.Header.Get("Content-Type"))
	return nil
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	storage, err := common.GetBodyStorage(c)
	if err != nil {
		return nil, errors.Wrap(err, "get_request_body_failed")
	}
	cachedBody, err := storage.Bytes()
	if err != nil {
		return nil, errors.Wrap(err, "read_body_bytes_failed")
	}
	contentType := c.GetHeader("Content-Type")

	if strings.HasPrefix(contentType, "application/json") {
		var bodyMap map[string]interface{}
		if err := common.Unmarshal(cachedBody, &bodyMap); err == nil {
			upstreamModelName := common.NormalizeGrokImagineModelName(info.UpstreamModelName)
			bodyMap["model"] = upstreamModelName
			if isGrokImagineVideoModel(upstreamModelName) {
				body, formContentType, err := buildGrokVideoMultipartBody(bodyMap, upstreamModelName)
				if err != nil {
					return nil, err
				}
				c.Request.Header.Set("Content-Type", formContentType)
				return body, nil
			}
			if isSeedanceVideoModel(upstreamModelName) {
				normalizeSeedanceVideoRequest(bodyMap, upstreamModelName)
			}
			normalizeGrokVideoRequest(bodyMap, upstreamModelName)
			if isKlingV3VideoModel(upstreamModelName) {
				if err := normalizeKlingV3VideoRequest(bodyMap); err != nil {
					return nil, err
				}
			}
			normalizeSoraVideoRequest(bodyMap, upstreamModelName)
			if newBody, err := common.Marshal(bodyMap); err == nil {
				c.Request.Header.Set("Content-Type", "application/json")
				return bytes.NewReader(newBody), nil
			}
		}
		return bytes.NewReader(cachedBody), nil
	}

	if strings.Contains(contentType, "multipart/form-data") {
		formData, err := common.ParseMultipartFormReusable(c)
		if err != nil {
			return bytes.NewReader(cachedBody), nil
		}
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)
		upstreamModelName := common.NormalizeGrokImagineModelName(info.UpstreamModelName)
		isGrokVideo := isGrokImagineVideoModel(upstreamModelName)
		isKlingV3Video := isKlingV3VideoModel(upstreamModelName)
		writer.WriteField("model", upstreamModelName)
		hasSeconds := false
		hasDuration := false
		durationValue := ""
		hasSize := false
		sizeValue := ""
		hasAspectRatio := false
		aspectRatioValue := ""
		for key, values := range formData.Value {
			if key == "model" {
				continue
			}
			if isGrokVideo && (key == "duration" || key == "async" || key == "video_config") {
				if key == "duration" && len(values) > 0 && durationValue == "" {
					durationValue = strings.TrimSpace(values[0])
				}
				continue
			}
			if key == "seconds" && len(values) > 0 && strings.TrimSpace(values[0]) != "" {
				hasSeconds = true
			}
			if key == "duration" && len(values) > 0 && strings.TrimSpace(values[0]) != "" {
				hasDuration = true
			}
			if key == "duration" && len(values) > 0 && durationValue == "" {
				durationValue = strings.TrimSpace(values[0])
			}
			if key == "size" && len(values) > 0 && strings.TrimSpace(values[0]) != "" {
				hasSize = true
				if sizeValue == "" {
					sizeValue = strings.TrimSpace(values[0])
				}
			}
			if key == "aspect_ratio" && len(values) > 0 && strings.TrimSpace(values[0]) != "" {
				hasAspectRatio = true
			}
			if key == "aspect_ratio" && len(values) > 0 && aspectRatioValue == "" {
				aspectRatioValue = strings.TrimSpace(values[0])
			}
			if usesImageURLVideoGenerationsModel(upstreamModelName) && (key == "seconds" || key == "size") {
				continue
			}
			if isKlingV3Video && (key == "resolution" || key == "reference_mode") {
				continue
			}
			for _, v := range values {
				writer.WriteField(key, v)
			}
		}
		if isGrokImagineVideoModel(upstreamModelName) && !hasSeconds && durationValue != "" {
			writer.WriteField("seconds", durationValue)
		}
		if usesImageURLVideoGenerationsModel(upstreamModelName) {
			if !hasDuration {
				if durationValue == "" {
					if isKlingV3Video {
						durationValue = "5"
					} else {
						durationValue = "4"
					}
				}
				writer.WriteField("duration", durationValue)
			}
			if !hasAspectRatio {
				if aspectRatioValue == "" && hasSize {
					aspectRatioValue = soraAspectRatioFromSize(sizeValue)
				}
				if aspectRatioValue == "" {
					if isKlingV3Video {
						aspectRatioValue = "16:9"
					} else {
						aspectRatioValue = "9:16"
					}
				}
				writer.WriteField("aspect_ratio", aspectRatioValue)
			}
			writer.WriteField("async", "true")
			if isKlingV3Video && !hasMultipartFieldValue(formData.Value, "generate_audio") {
				writer.WriteField("generate_audio", "true")
			}
			if isKlingV3Video && !hasMultipartFieldValue(formData.Value, "generateAudio") {
				generateAudioValue := firstNonEmptyMultipartValue(formData.Value, "generate_audio", "generateAudio")
				if generateAudioValue == "" {
					generateAudioValue = "true"
				}
				writer.WriteField("generateAudio", generateAudioValue)
			}
			if firstAsyncImageValue := firstNonEmptyMultipartValue(formData.Value, "image_url", "input_reference", "image"); firstAsyncImageValue != "" {
				if firstAsyncImageValue != "" && !hasMultipartFieldValue(formData.Value, "image_url") {
					writer.WriteField("image_url", firstAsyncImageValue)
				}
				if !isKlingV3Video && !hasMultipartFieldValue(formData.Value, "reference_mode") {
					if referenceMode := defaultVideoGenerationsReferenceMode(upstreamModelName); referenceMode != "" {
						writer.WriteField("reference_mode", referenceMode)
					}
				}
			}
		}
		for fieldName, fileHeaders := range formData.File {
			targetFieldName := fieldName
			if isGrokVideo && isGrokVideoReferenceFileField(fieldName) {
				targetFieldName = "input_reference[]"
			}
			for _, fh := range fileHeaders {
				f, err := fh.Open()
				if err != nil {
					continue
				}
				ct := fh.Header.Get("Content-Type")
				if ct == "" || ct == "application/octet-stream" {
					buf512 := make([]byte, 512)
					n, _ := io.ReadFull(f, buf512)
					ct = http.DetectContentType(buf512[:n])
					// Re-open after sniffing so the full content is copied below
					f.Close()
					f, err = fh.Open()
					if err != nil {
						continue
					}
				}
				h := make(textproto.MIMEHeader)
				h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, targetFieldName, fh.Filename))
				h.Set("Content-Type", ct)
				part, err := writer.CreatePart(h)
				if err != nil {
					f.Close()
					continue
				}
				io.Copy(part, f)
				f.Close()
			}
		}
		writer.Close()
		c.Request.Header.Set("Content-Type", writer.FormDataContentType())
		return &buf, nil
	}

	return common.ReaderOnly(storage), nil
}

// DoRequest delegates to common helper.
func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

// DoResponse handles upstream response, returns taskID etc.
func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (taskID string, taskData []byte, taskErr *dto.TaskError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		taskErr = service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
		return
	}
	_ = resp.Body.Close()

	// Parse Sora response
	var dResp responseTask
	if err := common.Unmarshal(responseBody, &dResp); err != nil {
		taskErr = service.TaskErrorWrapper(errors.Wrapf(err, "body: %s", responseBody), "unmarshal_response_body_failed", http.StatusInternalServerError)
		return
	}
	fillResponseTaskFallbacks(responseBody, &dResp)

	upstreamID := dResp.TaskID
	if upstreamID == "" {
		upstreamID = dResp.ID
	}
	if upstreamID == "" {
		if parsedTask, parseErr := a.ParseTaskResult(responseBody); parseErr == nil && parsedTask != nil && parsedTask.Status == model.TaskStatusSuccess && parsedTask.Url != "" {
			if dResp.Object == "" {
				dResp.Object = "video"
			}
			if dResp.Status == "" {
				dResp.Status = "completed"
			}
			if dResp.URL == "" {
				dResp.URL = parsedTask.Url
			}
			if dResp.VideoURL == "" {
				dResp.VideoURL = dResp.URL
			}
			if dResp.Progress == 0 {
				dResp.Progress = 100
			}
			if dResp.CreatedAt == 0 {
				dResp.CreatedAt = parsedTask.CreatedAt
			}
		} else {
			taskErr = service.TaskErrorWrapper(fmt.Errorf("task_id is empty"), "invalid_response", http.StatusInternalServerError)
			return
		}
	}
	if dResp.URL == "" {
		dResp.URL = extractVideoURL(responseBody)
	}
	if dResp.VideoURL == "" {
		dResp.VideoURL = dResp.URL
	}
	if dResp.URL == "" {
		dResp.URL = dResp.VideoURL
	}

	// 使用公开 task_xxxx ID 返回给客户端
	dResp.ID = info.PublicTaskID
	dResp.TaskID = info.PublicTaskID
	c.JSON(http.StatusOK, dResp)
	return upstreamID, responseBody, nil
}

// FetchTask fetch task status
func (a *TaskAdaptor) FetchTask(baseUrl, key string, body map[string]any, proxy string) (*http.Response, error) {
	uri, err := buildTaskFetchURL(baseUrl, body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+key)

	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, fmt.Errorf("new proxy http client failed: %w", err)
	}
	return client.Do(req)
}

func (a *TaskAdaptor) GetModelList() []string {
	return ModelList
}

func (a *TaskAdaptor) GetChannelName() string {
	return ChannelName
}

func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	resTask := responseTask{}
	if err := common.Unmarshal(respBody, &resTask); err != nil {
		return nil, errors.Wrap(err, "unmarshal task result failed")
	}
	fillResponseTaskFallbacks(respBody, &resTask)
	createdAt := resTask.CreatedAt
	if createdAt == 0 {
		createdAt = resTask.Created
	}

	taskResult := relaycommon.TaskInfo{
		Code:        0,
		CreatedAt:   createdAt,
		CompletedAt: resTask.CompletedAt,
	}

	status := strings.ToLower(strings.TrimSpace(resTask.Status))
	if status == "" && resTask.URL != "" {
		status = "completed"
	}

	switch status {
	case "queued", "pending":
		taskResult.Status = model.TaskStatusQueued
	case "processing", "in_progress", "running":
		taskResult.Status = model.TaskStatusInProgress
	case "completed":
		if resTask.URL == "" {
			taskResult.Status = model.TaskStatusFailure
			taskResult.Reason = "video result url is empty"
		} else {
			taskResult.Status = model.TaskStatusSuccess
			taskResult.Url = resTask.URL
		}
	case "failed", "cancelled":
		taskResult.Status = model.TaskStatusFailure
		if resTask.Error != nil {
			taskResult.Reason = resTask.Error.Message
		} else {
			taskResult.Reason = extractTaskFailureReason(respBody)
		}
		if taskResult.Reason == "" {
			taskResult.Reason = "task failed"
		}
	default:
	}
	if resTask.Progress > 0 && resTask.Progress < 100 {
		taskResult.Progress = formatTaskProgress(resTask.Progress)
	}

	return &taskResult, nil
}

func (a *TaskAdaptor) ConvertToOpenAIVideo(task *model.Task) ([]byte, error) {
	data := task.Data
	var err error
	if data, err = sjson.SetBytes(data, "id", task.TaskID); err != nil {
		return nil, errors.Wrap(err, "set id failed")
	}
	if gjson.GetBytes(data, "task_id").Exists() {
		if data, err = sjson.SetBytes(data, "task_id", task.TaskID); err != nil {
			return nil, errors.Wrap(err, "set task_id failed")
		}
	}
	return data, nil
}
