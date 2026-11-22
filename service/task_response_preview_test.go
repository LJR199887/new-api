package service

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
)

func TestTaskResponsePreviewSanitizesStoredPayload(t *testing.T) {
	task := &model.Task{}
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
		require.Equal(t, map[string]any{"error": map[string]any{"message": "generation failed"}}, body)
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
		})
		require.True(t, truncated)
		encoded, err := common.Marshal(body)
		require.NoError(t, err)
		require.LessOrEqual(t, len(encoded), taskRequestSnapshotMaxBytes)
	})
}
