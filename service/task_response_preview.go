package service

import (
	"bytes"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

// BuildTaskResponsePreview returns a bounded view of the latest stored task
// response. Older tasks may have only a result URL or a failure reason.
func BuildTaskResponsePreview(task *model.Task) (any, bool) {
	if task == nil {
		return nil, false
	}

	raw := bytes.TrimSpace(task.Data)
	if len(raw) > taskRequestSnapshotParseMaxBytes {
		return summarizeTaskResponse(task, len(raw), "response too large to preview"), true
	}
	var body any
	if len(raw) > 0 {
		if err := common.Unmarshal(raw, &body); err != nil {
			return summarizeTaskResponse(task, len(raw), "stored response is not valid JSON"), true
		}
	}
	switch value := body.(type) {
	case map[string]any:
		if len(value) == 0 {
			body = nil
		}
	case []any:
		if len(value) == 0 {
			body = nil
		}
	case string:
		if strings.TrimSpace(value) == "" {
			body = nil
		}
	}
	if body == nil {
		fallback := taskResponseFallback(task)
		if len(fallback) == 0 {
			return nil, false
		}
		body = fallback
	}

	truncated := false
	body = sanitizeTaskRequestSnapshotValue(body, "", &truncated)
	if payload, err := common.Marshal(body); err != nil || len(payload) > taskRequestSnapshotMaxBytes {
		return summarizeTaskResponse(task, len(raw), "sanitized response exceeded 64 KiB"), true
	}
	return body, truncated
}

func taskResponseFallback(task *model.Task) map[string]any {
	fallback := map[string]any{}
	switch task.Status {
	case model.TaskStatusSuccess:
		if resultURL := strings.TrimSpace(task.GetResultURL()); resultURL != "" {
			fallback["url"] = resultURL
		}
	case model.TaskStatusFailure:
		if reason := strings.TrimSpace(task.FailReason); reason != "" {
			fallback["error"] = map[string]any{"message": reason}
		}
	}
	return fallback
}

func summarizeTaskResponse(task *model.Task, originalSize int, reason string) any {
	summary := taskResponseFallback(task)
	summary["_omitted"] = reason
	summary["original_size"] = originalSize
	truncated := true
	preview := sanitizeTaskRequestSnapshotValue(summary, "", &truncated)
	if payload, err := common.Marshal(preview); err == nil && len(payload) <= taskRequestSnapshotMaxBytes {
		return preview
	}
	return map[string]any{"_omitted": reason, "original_size": originalSize}
}
