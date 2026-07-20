package service

import (
	"bytes"
	"mime/multipart"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTaskRequestSnapshotTestContext(t *testing.T, contentType string, body []byte) *gin.Context {
	t.Helper()
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest("POST", "/v1/video/async-generations", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", contentType)
	storage, err := common.CreateBodyStorage(body)
	require.NoError(t, err)
	c.Set(common.KeyBodyStorage, storage)
	t.Cleanup(func() {
		common.CleanupBodyStorage(c)
	})
	return c
}

func decodeTaskRequestSnapshotBody(t *testing.T, snapshotBody string) map[string]any {
	t.Helper()
	var body map[string]any
	require.NoError(t, common.UnmarshalJsonStr(snapshotBody, &body))
	return body
}

func TestBuildTaskRequestSnapshotSanitizesJSON(t *testing.T) {
	prompt := strings.Repeat("兔子在草地上玩耍", 500)
	rawBody, err := common.Marshal(map[string]any{
		"model":     "video-2.0-fast",
		"prompt":    prompt,
		"api_key":   "secret-key",
		"audio_url": "https://example.com/audio.mp3?token=abc&X-Amz-Signature=def",
		"file_data": "data:audio/mpeg;base64," + strings.Repeat("A", 4096),
	})
	require.NoError(t, err)
	c := newTaskRequestSnapshotTestContext(t, "application/json", rawBody)

	snapshot, err := BuildTaskRequestSnapshot(c, "task_test")
	require.NoError(t, err)
	require.NotNil(t, snapshot)
	body := decodeTaskRequestSnapshotBody(t, string(snapshot.Body))

	assert.Equal(t, "[REDACTED]", body["api_key"])
	assert.Equal(t, prompt, body["prompt"])
	parsedURL, err := url.Parse(body["audio_url"].(string))
	require.NoError(t, err)
	assert.Equal(t, "[REDACTED]", parsedURL.Query().Get("token"))
	assert.Equal(t, "[REDACTED]", parsedURL.Query().Get("X-Amz-Signature"))
	assert.Contains(t, body["file_data"], "_omitted")
	assert.True(t, snapshot.Truncated)
	assert.LessOrEqual(t, len(snapshot.Body), taskRequestSnapshotMaxBytes)
}

func TestBuildTaskRequestSnapshotUsesSummaryAboveParseLimit(t *testing.T) {
	rawBody := []byte(`{"model":"video-2.0-fast","payload":"` + strings.Repeat("x", taskRequestSnapshotParseMaxBytes) + `"}`)
	c := newTaskRequestSnapshotTestContext(t, "application/json", rawBody)
	c.Set("task_request", relaycommon.TaskSubmitReq{
		Model:       "video-2.0-fast",
		Prompt:      "summary prompt",
		Duration:    5,
		AspectRatio: "9:16",
	})

	snapshot, err := BuildTaskRequestSnapshot(c, "task_large")
	require.NoError(t, err)
	body := decodeTaskRequestSnapshotBody(t, string(snapshot.Body))

	assert.True(t, snapshot.Truncated)
	assert.Equal(t, "video-2.0-fast", body["model"])
	assert.Equal(t, "summary prompt", body["prompt"])
	assert.Equal(t, "request body exceeded 256 KiB parse limit", body["_omitted"])
}

func TestBuildTaskRequestSnapshotStoresMultipartMetadataOnly(t *testing.T) {
	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)
	require.NoError(t, writer.WriteField("model", "video-2.0-fast"))
	require.NoError(t, writer.WriteField("prompt", "multipart prompt"))
	filePart, err := writer.CreateFormFile("input_reference", "source.png")
	require.NoError(t, err)
	_, err = filePart.Write(bytes.Repeat([]byte{0xff}, 4096))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	c := newTaskRequestSnapshotTestContext(t, writer.FormDataContentType(), buffer.Bytes())
	snapshot, err := BuildTaskRequestSnapshot(c, "task_multipart")
	require.NoError(t, err)
	body := decodeTaskRequestSnapshotBody(t, string(snapshot.Body))
	fields := body["fields"].(map[string]any)
	files := body["files"].([]any)

	assert.Equal(t, "video-2.0-fast", fields["model"])
	assert.Equal(t, "multipart prompt", fields["prompt"])
	require.Len(t, files, 1)
	assert.Equal(t, "source.png", files[0].(map[string]any)["filename"])
	assert.NotContains(t, string(snapshot.Body), strings.Repeat("ff", 32))
}

func TestBuildTaskRequestSnapshotCompactsToStorageLimit(t *testing.T) {
	rawBody, err := common.Marshal(map[string]any{
		"model":    "video-2.0-fast",
		"prompt":   strings.Repeat("提示", 2500),
		"metadata": strings.Repeat("内容", 10000),
		"extra":    strings.Repeat("附加", 10000),
		"other":    strings.Repeat("其他", 10000),
	})
	require.NoError(t, err)
	c := newTaskRequestSnapshotTestContext(t, "application/json", rawBody)

	snapshot, err := BuildTaskRequestSnapshot(c, "task_compact")
	require.NoError(t, err)
	body := decodeTaskRequestSnapshotBody(t, string(snapshot.Body))

	assert.True(t, snapshot.Truncated)
	assert.LessOrEqual(t, len(snapshot.Body), taskRequestSnapshotMaxBytes)
	assert.Equal(t, "video-2.0-fast", body["model"])
	assert.NotEmpty(t, body["prompt"])
	assert.NotContains(t, body, "metadata")
}
