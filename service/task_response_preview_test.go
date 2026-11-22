package service

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
)

func TestTaskResponsePreviewSanitizesStoredPayload(t *testing.T) {
	task := &model.Task{Status: model.TaskStatusSuccess}
	task.SetData(map[string]any{
		"status":       "completed",
		"private_data": map[string]any{"value": "hidden-private-value"},
		"data": []any{map[string]any{
			"url":      "https://user:secret@example.com/result.png?token=hidden-token&format=png",
			"b64_json": "hidden-image-data",
		}},
	})

	body, truncated := BuildTaskResponsePreview(task)
	require.True(t, truncated)
	result := body.(map[string]any)
	require.Equal(t, "completed", result["status"])
	require.Equal(t, "[REDACTED]", result["private_data"])
	item := result["data"].([]any)[0].(map[string]any)
	require.Equal(t, "https://example.com/result.png?format=png&token=%5BREDACTED%5D", item["url"])
	require.Equal(t, "binary or base64 content", item["b64_json"].(map[string]any)["_omitted"])
}

func TestTaskResponsePreviewFallbackAndSizeLimit(t *testing.T) {
	t.Run("pending without response", func(t *testing.T) {
		body, truncated := BuildTaskResponsePreview(&model.Task{Status: model.TaskStatusSubmitted})
		require.Nil(t, body)
		require.False(t, truncated)
	})
	t.Run("failure without stored response", func(t *testing.T) {
		body, truncated := BuildTaskResponsePreview(&model.Task{
			Status: model.TaskStatusFailure, FailReason: "generation failed", Data: []byte(`{}`),
		})
		require.Nil(t, body)
		require.False(t, truncated)
	})
	t.Run("non-JSON response uses safe summary", func(t *testing.T) {
		body, truncated := BuildTaskResponsePreview(&model.Task{
			Status: model.TaskStatusFailure, FailReason: "upstream failed", Data: []byte(`<html>unstructured response</html>`),
		})
		require.True(t, truncated)
		result := body.(map[string]any)
		require.Equal(t, "stored response is not valid JSON", result["_omitted"])
		require.Equal(t, map[string]any{"message": "upstream failed"}, result["error"])
	})
	t.Run("oversized response preserves the result URL", func(t *testing.T) {
		task := &model.Task{
			Status:      model.TaskStatusSuccess,
			PrivateData: model.TaskPrivateData{ResultURL: "https://example.com/result.mp4"},
		}
		task.SetData(map[string]any{"b64_json": strings.Repeat("A", taskRequestSnapshotParseMaxBytes)})
		body, truncated := BuildTaskResponsePreview(task)
		require.True(t, truncated)
		result := body.(map[string]any)
		require.Equal(t, len(task.Data), result["original_size"])
		require.Equal(t, task.PrivateData.ResultURL, result["url"])
		require.NotContains(t, result, "b64_json")
	})
	t.Run("serialized fallback is also bounded", func(t *testing.T) {
		body, truncated := BuildTaskResponsePreview(&model.Task{
			Status: model.TaskStatusFailure, FailReason: strings.Repeat("\x00", taskRequestSnapshotStringMaxBytes),
			Data: []byte(`<html>upstream failure</html>`),
		})
		require.True(t, truncated)
		encoded, err := common.Marshal(body)
		require.NoError(t, err)
		require.LessOrEqual(t, len(encoded), taskRequestSnapshotMaxBytes)
	})
}

func TestTaskResponsePreviewOnlyShowsFinalResponses(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status model.TaskStatus
		reason string
		body   string
	}{
		{"submission", model.TaskStatusSubmitted, "", `{"status":"queued","progress":5}`},
		{"queued", model.TaskStatusQueued, "", `{"status":"queued","progress":5}`},
		{"in progress", model.TaskStatusInProgress, "", `{"status":"processing","progress":65}`},
		{"local timeout with old body", model.TaskStatusFailure, "任务超时（59分钟）", `{"status":"queued","progress":5}`},
		{"historical timeout without body", model.TaskStatusFailure, "任务超时（旧系统遗留任务，不进行退款，请联系管理员）", ""},
		{"FA2 local polling timeout", model.TaskStatusFailure, "fa2api image polling timed out: context deadline exceeded", `{"error":{"message":"timeout"}}`},
		{"other local failure with old body", model.TaskStatusFailure, "channel is unavailable", `{"data":{"status":"QUEUED"}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			task := &model.Task{Status: tc.status, FailReason: tc.reason, Data: []byte(tc.body)}
			body, truncated := BuildTaskResponsePreview(task)
			require.Nil(t, body)
			require.False(t, truncated)
			require.Equal(t, tc.body, string(task.Data))
		})
	}
	t.Run("actual upstream timeout failure still has a response", func(t *testing.T) {
		task := &model.Task{Status: model.TaskStatusFailure, FailReason: "upstream generation timed out"}
		task.SetData(map[string]any{"status": "failed", "error": map[string]any{"message": task.FailReason}})
		body, truncated := BuildTaskResponsePreview(task)
		require.Equal(t, map[string]any{"status": "failed", "error": map[string]any{"message": task.FailReason}}, body)
		require.False(t, truncated)
	})
}
