package service

import (
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
)

const (
	taskRequestSnapshotMaxBytes       = 64 * 1024
	taskRequestSnapshotParseMaxBytes  = 256 * 1024
	taskRequestSnapshotStringMaxBytes = 24 * 1024
	taskRequestSnapshotMaxArrayItems  = 100
)

var taskRequestSnapshotSensitiveFields = map[string]struct{}{
	"authorization": {},
	"apikey":        {},
	"accesstoken":   {},
	"refreshtoken":  {},
	"token":         {},
	"secret":        {},
	"password":      {},
	"credential":    {},
	"signature":     {},
	"sig":           {},
	"key":           {},
}

func BuildTaskRequestSnapshot(c *gin.Context, taskID string) (*model.TaskRequestSnapshot, error) {
	if c == nil || c.Request == nil || strings.TrimSpace(taskID) == "" {
		return nil, nil
	}

	storage, err := common.GetBodyStorage(c)
	if err != nil {
		return nil, err
	}
	originalSize := storage.Size()
	contentType := strings.TrimSpace(c.GetHeader("Content-Type"))
	if saved, ok := c.Get("_original_multipart_ct"); ok {
		if originalContentType, valid := saved.(string); valid && originalContentType != "" {
			contentType = originalContentType
		}
	}
	mediaType, _, mediaTypeErr := mime.ParseMediaType(contentType)
	if mediaTypeErr != nil {
		mediaType = strings.TrimSpace(strings.Split(contentType, ";")[0])
	}

	var body any
	truncated := false
	switch {
	case strings.EqualFold(mediaType, "multipart/form-data"):
		body, err = buildMultipartTaskRequestSnapshot(storage, contentType, &truncated)
	case strings.EqualFold(mediaType, "application/x-www-form-urlencoded"):
		body, err = buildFormTaskRequestSnapshot(storage, &truncated)
	default:
		body, err = buildJSONTaskRequestSnapshot(c, storage, &truncated)
	}
	if err != nil {
		return nil, err
	}

	body = sanitizeTaskRequestSnapshotValue(body, "", &truncated)
	payload, err := common.Marshal(body)
	if err != nil {
		return nil, err
	}
	if len(payload) > taskRequestSnapshotMaxBytes {
		truncated = true
		body = compactTaskRequestSnapshotBody(body, originalSize)
		payload, err = common.Marshal(body)
		if err != nil {
			return nil, err
		}
	}
	if len(payload) > taskRequestSnapshotMaxBytes {
		truncated = true
		payload, err = common.Marshal(map[string]any{
			"_omitted":      "sanitized request exceeded 64 KiB",
			"original_size": originalSize,
		})
		if err != nil {
			return nil, err
		}
	}

	requestPath := ""
	if c.Request.URL != nil {
		requestPath = c.Request.URL.Path
	}
	return &model.TaskRequestSnapshot{
		CreatedAt:    time.Now().Unix(),
		TaskID:       strings.TrimSpace(taskID),
		Method:       c.Request.Method,
		RequestPath:  requestPath,
		ContentType:  mediaType,
		Body:         model.TaskRequestSnapshotPayload(payload),
		OriginalSize: originalSize,
		Truncated:    truncated,
	}, nil
}

func PersistTaskRequestSnapshot(snapshot *model.TaskRequestSnapshot) error {
	return model.InsertTaskRequestSnapshot(snapshot)
}

func CaptureTaskRequestSnapshot(c *gin.Context, taskID string) error {
	snapshot, err := BuildTaskRequestSnapshot(c, taskID)
	if err != nil || snapshot == nil {
		return err
	}
	return PersistTaskRequestSnapshot(snapshot)
}

func buildJSONTaskRequestSnapshot(c *gin.Context, storage common.BodyStorage, truncated *bool) (any, error) {
	if storage.Size() > taskRequestSnapshotParseMaxBytes {
		*truncated = true
		return summarizeTaskRequest(c, storage.Size()), nil
	}
	bodyBytes, err := storage.Bytes()
	if err != nil {
		return nil, err
	}
	if len(bodyBytes) == 0 {
		return map[string]any{}, nil
	}
	var body any
	if err := common.Unmarshal(bodyBytes, &body); err != nil {
		return map[string]any{
			"_omitted": "request body is not valid JSON",
			"size":     len(bodyBytes),
		}, nil
	}
	return body, nil
}

func buildFormTaskRequestSnapshot(storage common.BodyStorage, truncated *bool) (any, error) {
	if storage.Size() > taskRequestSnapshotParseMaxBytes {
		*truncated = true
		return map[string]any{
			"_omitted": "form body exceeded 256 KiB parse limit",
			"size":     storage.Size(),
		}, nil
	}
	bodyBytes, err := storage.Bytes()
	if err != nil {
		return nil, err
	}
	values, err := url.ParseQuery(string(bodyBytes))
	if err != nil {
		return nil, err
	}
	fields := make(map[string]any, len(values))
	for key, items := range values {
		if len(items) == 1 {
			fields[key] = items[0]
		} else {
			fields[key] = items
		}
	}
	return fields, nil
}

func buildMultipartTaskRequestSnapshot(storage common.BodyStorage, contentType string, truncated *bool) (any, error) {
	_, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return nil, err
	}
	boundary := params["boundary"]
	if boundary == "" {
		return nil, fmt.Errorf("multipart boundary is missing")
	}

	position, _ := storage.Seek(0, io.SeekCurrent)
	if _, err := storage.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	defer func() {
		_, _ = storage.Seek(position, io.SeekStart)
	}()

	fields := make(map[string]any)
	files := make([]any, 0)
	reader := multipart.NewReader(storage, boundary)
	for {
		part, nextErr := reader.NextPart()
		if nextErr == io.EOF {
			break
		}
		if nextErr != nil {
			return nil, nextErr
		}

		fieldName := part.FormName()
		if fileName := part.FileName(); fileName != "" {
			files = append(files, map[string]any{
				"field":        fieldName,
				"filename":     fileName,
				"content_type": part.Header.Get("Content-Type"),
			})
			_ = part.Close()
			continue
		}

		valueBytes, readErr := io.ReadAll(io.LimitReader(part, taskRequestSnapshotStringMaxBytes+1))
		_ = part.Close()
		if readErr != nil {
			return nil, readErr
		}
		if len(valueBytes) > taskRequestSnapshotStringMaxBytes {
			*truncated = true
			valueBytes = valueBytes[:taskRequestSnapshotStringMaxBytes]
			for len(valueBytes) > 0 && !utf8.Valid(valueBytes) {
				valueBytes = valueBytes[:len(valueBytes)-1]
			}
		}
		appendTaskRequestSnapshotField(fields, fieldName, string(valueBytes))
	}

	return map[string]any{
		"fields": fields,
		"files":  files,
	}, nil
}

func appendTaskRequestSnapshotField(fields map[string]any, key string, value string) {
	if current, exists := fields[key]; exists {
		switch values := current.(type) {
		case []any:
			fields[key] = append(values, value)
		default:
			fields[key] = []any{current, value}
		}
		return
	}
	fields[key] = value
}

func summarizeTaskRequest(c *gin.Context, originalSize int64) map[string]any {
	summary := map[string]any{
		"_omitted": "request body exceeded 256 KiB parse limit",
		"size":     originalSize,
	}
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return summary
	}
	addTaskRequestSummaryValue(summary, "model", req.Model)
	addTaskRequestSummaryValue(summary, "prompt", req.Prompt)
	addTaskRequestSummaryValue(summary, "request_id", req.RequestId)
	addTaskRequestSummaryValue(summary, "image_url", req.ImageURL)
	addTaskRequestSummaryValue(summary, "image_urls", req.ImageURLs)
	addTaskRequestSummaryValue(summary, "images", req.Images)
	addTaskRequestSummaryValue(summary, "aspect_ratio", req.AspectRatio)
	addTaskRequestSummaryValue(summary, "size", req.Size)
	addTaskRequestSummaryValue(summary, "seconds", req.Seconds)
	addTaskRequestSummaryValue(summary, "duration", req.Duration)
	addTaskRequestSummaryValue(summary, "resolution", req.Resolution)
	addTaskRequestSummaryValue(summary, "quality", req.Quality)
	return summary
}

func addTaskRequestSummaryValue(summary map[string]any, key string, value any) {
	switch typed := value.(type) {
	case string:
		if strings.TrimSpace(typed) != "" {
			summary[key] = typed
		}
	case []string:
		if len(typed) > 0 {
			summary[key] = typed
		}
	case int:
		if typed != 0 {
			summary[key] = typed
		}
	}
}

func sanitizeTaskRequestSnapshotValue(value any, fieldName string, truncated *bool) any {
	if isSensitiveTaskRequestSnapshotField(fieldName) {
		return "[REDACTED]"
	}
	switch typed := value.(type) {
	case map[string]any:
		result := make(map[string]any, len(typed))
		for key, item := range typed {
			result[key] = sanitizeTaskRequestSnapshotValue(item, key, truncated)
		}
		return result
	case []any:
		limit := len(typed)
		if limit > taskRequestSnapshotMaxArrayItems {
			limit = taskRequestSnapshotMaxArrayItems
			*truncated = true
		}
		result := make([]any, 0, limit+1)
		for index := 0; index < limit; index++ {
			result = append(result, sanitizeTaskRequestSnapshotValue(typed[index], fieldName, truncated))
		}
		if len(typed) > limit {
			result = append(result, fmt.Sprintf("[%d additional items omitted]", len(typed)-limit))
		}
		return result
	case []string:
		items := make([]any, len(typed))
		for index, item := range typed {
			items[index] = item
		}
		return sanitizeTaskRequestSnapshotValue(items, fieldName, truncated)
	case string:
		return sanitizeTaskRequestSnapshotString(typed, fieldName, truncated)
	default:
		return value
	}
}

func sanitizeTaskRequestSnapshotString(value string, fieldName string, truncated *bool) any {
	if isBinaryTaskRequestSnapshotField(fieldName) || strings.HasPrefix(strings.TrimSpace(value), "data:") ||
		(!isPromptTaskRequestSnapshotField(fieldName) && looksLikeBase64TaskRequestValue(value)) {
		*truncated = true
		return map[string]any{
			"_omitted": "binary or base64 content",
			"size":     len(value),
		}
	}
	value = redactSensitiveTaskRequestURL(value)
	if len(value) <= taskRequestSnapshotStringMaxBytes {
		return value
	}
	*truncated = true
	return truncateTaskRequestSnapshotUTF8(value, taskRequestSnapshotStringMaxBytes) + "...[truncated]"
}

func normalizeTaskRequestSnapshotFieldName(fieldName string) string {
	replacer := strings.NewReplacer("_", "", "-", "", ".", "", " ", "")
	return strings.ToLower(replacer.Replace(strings.TrimSpace(fieldName)))
}

func isSensitiveTaskRequestSnapshotField(fieldName string) bool {
	normalized := normalizeTaskRequestSnapshotFieldName(fieldName)
	if _, sensitive := taskRequestSnapshotSensitiveFields[normalized]; sensitive {
		return true
	}
	for _, fragment := range []string{"authorization", "apikey", "accesstoken", "refreshtoken", "password", "credential", "signature"} {
		if strings.Contains(normalized, fragment) {
			return true
		}
	}
	return strings.HasSuffix(normalized, "token") || strings.HasSuffix(normalized, "secret")
}

func isPromptTaskRequestSnapshotField(fieldName string) bool {
	normalized := normalizeTaskRequestSnapshotFieldName(fieldName)
	return normalized == "prompt" || normalized == "negativeprompt"
}

func isBinaryTaskRequestSnapshotField(fieldName string) bool {
	normalized := normalizeTaskRequestSnapshotFieldName(fieldName)
	return strings.Contains(normalized, "base64") ||
		strings.Contains(normalized, "b64json") ||
		strings.Contains(normalized, "filedata") ||
		strings.Contains(normalized, "binarydata")
}

func looksLikeBase64TaskRequestValue(value string) bool {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) < 1024 || len(trimmed)%4 != 0 {
		return false
	}
	sample := trimmed
	if len(sample) > 4096 {
		sample = sample[:4096]
	}
	valid := 0
	for _, char := range sample {
		if (char >= 'A' && char <= 'Z') || (char >= 'a' && char <= 'z') ||
			(char >= '0' && char <= '9') || char == '+' || char == '/' || char == '=' ||
			char == '-' || char == '_' || char == '\r' || char == '\n' {
			valid++
		}
	}
	return valid*100/len(sample) >= 98
}

func redactSensitiveTaskRequestURL(value string) string {
	parsed, err := url.Parse(value)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return value
	}
	query := parsed.Query()
	changed := false
	for key := range query {
		if isSensitiveTaskRequestSnapshotField(key) {
			query.Set(key, "[REDACTED]")
			changed = true
		}
	}
	if changed {
		parsed.RawQuery = query.Encode()
	}
	return parsed.String()
}

func truncateTaskRequestSnapshotUTF8(value string, maxBytes int) string {
	if len(value) <= maxBytes {
		return value
	}
	value = value[:maxBytes]
	for len(value) > 0 && !utf8.ValidString(value) {
		value = value[:len(value)-1]
	}
	return value
}

func compactTaskRequestSnapshotBody(body any, originalSize int64) map[string]any {
	source, _ := body.(map[string]any)
	fields := source
	if nested, ok := source["fields"].(map[string]any); ok {
		fields = nested
	}
	result := map[string]any{
		"_truncated":    true,
		"original_size": originalSize,
	}
	for _, key := range []string{
		"model", "prompt", "request_id", "duration", "seconds", "size",
		"aspect_ratio", "resolution", "quality", "image_url", "audio_url",
		"image_reference", "audio_reference",
	} {
		if value, exists := fields[key]; exists {
			result[key] = value
		}
	}
	if files, exists := source["files"]; exists {
		result["files"] = files
	}
	return result
}
