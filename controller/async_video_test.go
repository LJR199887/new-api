package controller

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestCreativeVideoStatusRefreshesQueuedTaskWithoutBackgroundPolling(t *testing.T) {
	const upstreamID = "kv-video25-test"
	const videoURL = "https://example.com/final.mp4"
	var upstreamCalls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamCalls.Add(1)
		if r.Method != http.MethodGet || r.URL.Path != "/v1/video/generations/"+upstreamID {
			t.Errorf("unexpected upstream request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte(`{"status":"completed","progress":100,"video_url":"` + videoURL + `"}`))
	}))
	defer upstream.Close()

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "video-status.db")), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	oldDB, oldFactory := model.DB, service.GetTaskAdaptorFunc
	model.DB = db
	service.InitHttpClient()
	service.GetTaskAdaptorFunc = func(platform constant.TaskPlatform) service.TaskPollingAdaptor {
		return relay.GetTaskAdaptor(platform)
	}
	t.Cleanup(func() {
		model.DB, service.GetTaskAdaptorFunc = oldDB, oldFactory
		_ = sqlDB.Close()
	})
	require.NoError(t, db.AutoMigrate(&model.Task{}, &model.Channel{}))
	baseURL := upstream.URL
	ch := &model.Channel{Type: constant.ChannelTypeOpenAI, BaseURL: &baseURL, Key: "test-key"}
	require.NoError(t, db.Create(ch).Error)
	task := &model.Task{
		TaskID: "task_queued_video25", UserId: 10, ChannelId: ch.Id, Platform: "1",
		Status: model.TaskStatusQueued, Progress: "5%",
		Properties: model.Properties{
			OriginModelName: "A-video2.5-flash-480p", UpstreamModelName: "video2.5-flash-480p",
		},
		PrivateData: model.TaskPrivateData{
			UpstreamTaskID: upstreamID, UpstreamRequestPath: "/v1/video/generations",
			BillingContext: &model.TaskBillingContext{PerCallBilling: true},
		},
	}
	task.SetData(map[string]any{"status": "queued", "progress": 5})
	require.NoError(t, db.Create(task).Error)

	writer := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(writer)
	c.Request = httptest.NewRequest(http.MethodGet, "/pg/video/async-generations/"+task.TaskID, nil)
	c.Params = gin.Params{{Key: "task_id", Value: task.TaskID}}
	c.Set("id", task.UserId)
	RelayAsyncVideoFetch(c)

	require.Equal(t, http.StatusOK, writer.Code, writer.Body.String())
	require.Equal(t, int32(1), upstreamCalls.Load())
	var stored model.Task
	require.NoError(t, db.First(&stored, task.ID).Error)
	require.Equal(t, model.TaskStatus(model.TaskStatusSuccess), stored.Status)
	require.Equal(t, "100%", stored.Progress)
	require.Equal(t, videoURL, stored.GetResultURL())
}

func TestShouldRefreshAsyncVideoTask(t *testing.T) {
	oldGraceMinutes := constant.TaskNotFoundGraceMinutes
	constant.TaskNotFoundGraceMinutes = 10
	defer func() {
		constant.TaskNotFoundGraceMinutes = oldGraceMinutes
	}()
	now := time.Now().Unix()

	tests := []struct {
		name string
		task *model.Task
		want bool
	}{
		{
			name: "nil task",
			task: nil,
			want: false,
		},
		{
			name: "terminal success task does not refresh",
			task: &model.Task{
				Status:    model.TaskStatusSuccess,
				ChannelId: 1,
				PrivateData: model.TaskPrivateData{
					UpstreamTaskID: "upstream-task",
				},
			},
			want: false,
		},
		{
			name: "terminal transient failure task remains terminal",
			task: &model.Task{
				Status:     model.TaskStatusFailure,
				Action:     constant.TaskActionGenerate,
				ChannelId:  1,
				SubmitTime: now - 60,
				FailReason: "upstream returned error",
				Properties: model.Properties{
					OriginModelName: "video-2.0-fast",
				},
				PrivateData: model.TaskPrivateData{
					UpstreamTaskID: "upstream-task",
				},
			},
			want: false,
		},
		{
			name: "terminal real failure task does not refresh",
			task: &model.Task{
				Status:     model.TaskStatusFailure,
				Action:     constant.TaskActionGenerate,
				ChannelId:  1,
				SubmitTime: now - 60,
				FailReason: "invalid image_url: 404",
				Properties: model.Properties{
					OriginModelName: "video-2.0-fast",
				},
				PrivateData: model.TaskPrivateData{
					UpstreamTaskID: "upstream-task",
				},
			},
			want: false,
		},
		{
			name: "missing channel does not refresh",
			task: &model.Task{
				Status: model.TaskStatusInProgress,
				PrivateData: model.TaskPrivateData{
					UpstreamTaskID: "upstream-task",
				},
			},
			want: false,
		},
		{
			name: "missing upstream id does not refresh",
			task: &model.Task{
				Status:    model.TaskStatusInProgress,
				ChannelId: 1,
			},
			want: false,
		},
		{
			name: "in progress task with upstream id refreshes",
			task: &model.Task{
				Status:    model.TaskStatusInProgress,
				ChannelId: 1,
				PrivateData: model.TaskPrivateData{
					UpstreamTaskID: "upstream-task",
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldRefreshAsyncVideoTask(tt.task); got != tt.want {
				t.Fatalf("shouldRefreshAsyncVideoTask() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShouldRefreshAsyncVideoForRequestOnlyRefreshesPlayground(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name string
		path string
		want bool
	}{
		{name: "external v1 async video", path: "/v1/video/async-generations/task-1", want: false},
		{name: "creative center async video", path: "/pg/video/async-generations/task-1", want: true},
		{name: "nil context", path: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			if tt.path != "" {
				c.Request = httptest.NewRequest("GET", tt.path, nil)
			}
			if got := shouldRefreshAsyncVideoForRequest(c); got != tt.want {
				t.Fatalf("shouldRefreshAsyncVideoForRequest() = %v, want %v", got, tt.want)
			}
		})
	}

	if shouldRefreshAsyncVideoForRequest(nil) {
		t.Fatal("nil context must not trigger an upstream refresh")
	}
}

func TestReadAsyncVideoTaskRequestPreservesAspectRatioAndResolution(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest("POST", "/v1/video/async-generations", strings.NewReader(`{
		"model": "seedance-2.0",
		"prompt": "make a short video",
		"duration": 4,
		"aspect_ratio": "9:16",
		"resolution": "720p"
	}`))
	c.Request.Header.Set("Content-Type", "application/json")

	req := readAsyncVideoTaskRequest(c, []byte(`{
		"model": "seedance-2.0",
		"prompt": "make a short video",
		"duration": 4,
		"aspect_ratio": "9:16",
		"resolution": "720p"
	}`))

	if req.AspectRatio != "9:16" {
		t.Fatalf("AspectRatio = %q, want 9:16", req.AspectRatio)
	}
	if req.Resolution != "720p" {
		t.Fatalf("Resolution = %q, want 720p", req.Resolution)
	}
}

func TestInitAsyncVideoTaskStoresClientRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name             string
		bodyRequestID    string
		creativeHeaderID string
		headerID         string
		want             string
	}{
		{
			name:          "body request id wins",
			bodyRequestID: "creative-request-body",
			headerID:      "creative-request-header",
			want:          "creative-request-body",
		},
		{
			name:             "creative center header fallback",
			creativeHeaderID: "creative-request-private-header",
			want:             "creative-request-private-header",
		},
		{
			name:     "header fallback",
			headerID: "creative-request-header",
			want:     "creative-request-header",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest("POST", "/v1/video/async-generations", nil)
			c.Set(common.RequestIdKey, "internal-request-id")
			if tt.headerID != "" {
				c.Request.Header.Set("X-Request-Id", tt.headerID)
			}
			if tt.creativeHeaderID != "" {
				c.Request.Header.Set("X-Creative-Center-Request-Id", tt.creativeHeaderID)
			}

			task := initAsyncVideoTask(c, relaycommon.TaskSubmitReq{
				Model:     "sora2",
				Prompt:    "make a short video",
				RequestId: tt.bodyRequestID,
			})

			if got := task.PrivateData.ClientRequestId; got != tt.want {
				t.Fatalf("ClientRequestId = %q, want %q", got, tt.want)
			}
			if got := task.PrivateData.RequestId; got != "internal-request-id" {
				t.Fatalf("RequestId = %q, want internal request id", got)
			}
		})
	}
}
