package controller

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestTaskRequestDetailIncludesStoredResponse(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "task-details.db")), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	previousDB := model.DB
	model.DB = db
	t.Cleanup(func() {
		model.DB = previousDB
		_ = sqlDB.Close()
	})
	require.NoError(t, db.AutoMigrate(&model.Task{}, &model.TaskRequestSnapshot{}))

	for _, recorded := range []bool{true, false} {
		taskID := "task_with_request"
		if !recorded {
			taskID = "task_without_request"
		}
		task := &model.Task{
			TaskID:      taskID,
			Status:      model.TaskStatusSuccess,
			PrivateData: model.TaskPrivateData{Key: "internal-key"},
		}
		task.SetData(map[string]any{"data": []any{map[string]any{"url": "https://example.com/result.png"}}})
		require.NoError(t, db.Create(task).Error)
		if recorded {
			recordAsyncImageUpstreamTaskID(task, " upstream-123 ")
			recordAsyncImageUpstreamTaskID(task, " ")
			var stored model.Task
			require.NoError(t, db.First(&stored, task.ID).Error)
			require.Equal(t, "upstream-123", stored.PrivateData.UpstreamTaskID)
			require.Equal(t, "internal-key", stored.PrivateData.Key)
			require.Equal(t, task.Status, stored.Status)
			publicResponse, err := common.Marshal(buildAsyncImageTaskResponse(&stored))
			require.NoError(t, err)
			require.NotContains(t, string(publicResponse), "upstream-123")
			require.NoError(t, model.InsertTaskRequestSnapshot(&model.TaskRequestSnapshot{
				TaskID: taskID, Method: http.MethodPost, Body: `{"prompt":"test prompt"}`,
			}))
		}

		writer := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(writer)
		c.Request = httptest.NewRequest(http.MethodGet, "/api/task/"+taskID+"/request", nil)
		c.Params = gin.Params{{Key: "task_id", Value: taskID}}
		GetTaskRequestSnapshot(c)

		var response struct {
			Success bool           `json:"success"`
			Data    map[string]any `json:"data"`
		}
		require.NoError(t, common.Unmarshal(writer.Body.Bytes(), &response))
		require.True(t, response.Success, writer.Body.String())
		require.Equal(t, taskID, response.Data["task_id"])
		require.Equal(t, recorded, response.Data["recorded"])
		if recorded {
			require.Equal(t, map[string]any{"prompt": "test prompt"}, response.Data["body"])
			require.Equal(t, "upstream-123", response.Data["upstream_task_id"])
		} else {
			require.NotContains(t, response.Data, "body")
			require.NotContains(t, response.Data, "upstream_task_id")
		}
		require.Equal(t, map[string]any{"data": []any{map[string]any{"url": "https://example.com/result.png"}}}, response.Data["response_body"])
		require.Equal(t, false, response.Data["response_truncated"])
		require.NotContains(t, writer.Body.String(), "internal-key")
	}
}
