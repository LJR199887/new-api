package service

import (
	"bytes"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
)

// BuildTaskResponsePreview is a read-only view of the final task response.
// Task.Data may still contain the submission or an intermediate poll response.
func BuildTaskResponsePreview(task *model.Task) (any, bool) {
	if task == nil || (task.Status != model.TaskStatusSuccess && task.Status != model.TaskStatusFailure) {
		return nil, false
	}
	if task.Status == model.TaskStatusFailure && isLocalTaskTimeout(task.FailReason) {
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
		if isIntermediateTaskResponse(value) {
			return nil, false
		}
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
		// A local failure reason alone is not a returned response.
		if task.Status != model.TaskStatusSuccess {
			return nil, false
		}
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

func isLocalTaskTimeout(reason string) bool {
	reason = strings.TrimSpace(reason)
	if reason == legacyTaskTimeoutReason || strings.HasPrefix(reason, constant.Fa2ImagePollingTimeoutReason+": ") {
		return true
	}
	minutes, ok := strings.CutPrefix(reason, taskTimeoutReasonPrefix)
	if !ok {
		return false
	}
	minutes, ok = strings.CutSuffix(minutes, taskTimeoutReasonSuffix)
	if !ok {
		return false
	}
	value, err := strconv.Atoi(minutes)
	return err == nil && value > 0
}

func isIntermediateTaskResponse(body map[string]any) bool {
	status, _ := body["status"].(string)
	status = strings.ToLower(strings.TrimSpace(status))
	if status == "" {
		for _, key := range []string{"data", "result", "task"} {
			if nested, ok := body[key].(map[string]any); ok {
				value, _ := nested["status"].(string)
				if value = strings.ToLower(strings.TrimSpace(value)); value != "" {
					status = value
					break
				}
			}
		}
	}
	switch status {
	case "not_start", "submitted", "queued", "pending", "processing", "running", "in_progress":
		return true
	default:
		return false
	}
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
