package relay

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShouldRealtimeFetchForRequestOnlyRefreshesPlayground(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name string
		path string
		want bool
	}{
		{name: "external v1 video", path: "/v1/video/generations/task-1", want: false},
		{name: "external openai video", path: "/v1/videos/task-1", want: false},
		{name: "creative center video", path: "/pg/video/generations/task-1", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest("GET", tt.path, nil)
			if got := shouldRealtimeFetchForRequest(c); got != tt.want {
				t.Fatalf("shouldRealtimeFetchForRequest() = %v, want %v", got, tt.want)
			}
		})
	}

	if shouldRealtimeFetchForRequest(nil) {
		t.Fatal("nil context must not trigger an upstream refresh")
	}
}

func TestExtractTaskClientRequestIDFromCreativeCenterHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/pg/video/async-generations", nil)
	c.Request.Header.Set("X-Creative-Center-Request-Id", "creative-request-123")
	if got := extractTaskClientRequestIDFromContext(c); got != "creative-request-123" {
		t.Fatalf("client request ID = %q", got)
	}
}

func TestTaskModel2DtoDoesNotExposeFailureReasonAsResultURL(t *testing.T) {
	task := &model.Task{
		TaskID:     "task_failed",
		Status:     model.TaskStatusFailure,
		FailReason: "video poll failed",
		PrivateData: model.TaskPrivateData{
			ResultURL: "https://example.com/stale-video.mp4",
		},
	}

	dtoTask := TaskModel2Dto(task)

	assert.Empty(t, dtoTask.ResultURL)
	assert.Equal(t, task.FailReason, dtoTask.FailReason)
}

func TestTaskModel2DtoKeepsLegacySuccessResultURLFallback(t *testing.T) {
	task := &model.Task{
		TaskID:     "task_success",
		Status:     model.TaskStatusSuccess,
		FailReason: "https://example.com/video.mp4",
	}

	dtoTask := TaskModel2Dto(task)

	assert.Equal(t, task.FailReason, dtoTask.ResultURL)
}

func TestIsSuccessfulTaskSubmitStatusAcceptsAny2xx(t *testing.T) {
	assert.True(t, isSuccessfulTaskSubmitStatus(200))
	assert.True(t, isSuccessfulTaskSubmitStatus(202))
	assert.True(t, isSuccessfulTaskSubmitStatus(299))
	assert.False(t, isSuccessfulTaskSubmitStatus(199))
	assert.False(t, isSuccessfulTaskSubmitStatus(300))
}

func TestTaskPricingFollowsSavedConfiguration(t *testing.T) {
	fixed := ratio_setting.ModelPrice2JSONString()
	groupFixed := ratio_setting.GroupModelPrice2JSONString()
	seconds := ratio_setting.ModelPriceBySeconds2JSONString()
	groupSeconds := ratio_setting.GroupModelPriceBySeconds2JSONString()
	resolution := ratio_setting.ModelPriceByResolution2JSONString()
	groupResolution := ratio_setting.GroupModelPriceByResolution2JSONString()
	modelRatios := ratio_setting.ModelRatio2JSONString()
	t.Cleanup(func() {
		_ = ratio_setting.UpdateModelPriceByJSONString(fixed)
		_ = ratio_setting.UpdateGroupModelPriceByJSONString(groupFixed)
		_ = ratio_setting.UpdateModelPriceBySecondsByJSONString(seconds)
		_ = ratio_setting.UpdateGroupModelPriceBySecondsByJSONString(groupSeconds)
		_ = ratio_setting.UpdateModelPriceByResolutionByJSONString(resolution)
		_ = ratio_setting.UpdateGroupModelPriceByResolutionByJSONString(groupResolution)
		_ = ratio_setting.UpdateModelRatioByJSONString(modelRatios)
	})
	require.NoError(t, ratio_setting.UpdateGroupModelPriceByJSONString(`{}`))
	require.NoError(t, ratio_setting.UpdateGroupModelPriceBySecondsByJSONString(`{}`))
	require.NoError(t, ratio_setting.UpdateGroupModelPriceByResolutionByJSONString(`{}`))

	for _, name := range []string{"A-video2.0", "A-video2.0-480p", "933-video2.0", "video2.5", "grok-imagine-video", "kling-v3", "ordinary-async-model"} {
		t.Run(name, func(t *testing.T) {
			set := func(update func(string) error, value any) {
				t.Helper()
				data, err := common.Marshal(value)
				require.NoError(t, err)
				require.NoError(t, update(string(data)))
			}
			check := func(group string, duration int, mode string, wantPrice float64, billedSeconds int) {
				t.Helper()
				c, _ := gin.CreateTestContext(httptest.NewRecorder())
				c.Set("task_request", relaycommon.TaskSubmitReq{Duration: duration, Resolution: common.Video933Resolution(name)})
				info := &relaycommon.RelayInfo{OriginModelName: name, UsingGroup: "default", UserGroup: group}
				requestSeconds := duration
				if requestSeconds == 0 {
					requestSeconds = 5
				}
				price, err := helper.ModelPriceHelperPerCall(c, info, requestSeconds)
				require.NoError(t, err)
				info.PriceData = price
				require.True(t, info.TaskPriceFinalizedAtSubmit())
				require.Equal(t, mode, price.BillingType)
				require.Equal(t, billedSeconds, price.BillingSeconds)
				require.InDelta(t, wantPrice, price.ModelPrice, 1e-10)
				multiplier := price.GroupRatioInfo.GroupRatio
				if group == "vip" {
					require.True(t, price.GroupPriceOverride)
					multiplier = 1
				}
				require.Equal(t, int(wantPrice*common.QuotaPerUnit*multiplier), price.Quota)
				require.Empty(t, price.OtherRatios)
			}

			clear := map[string]any{}
			set(ratio_setting.UpdateModelPriceBySecondsByJSONString, clear)
			set(ratio_setting.UpdateModelPriceByResolutionByJSONString, clear)
			set(ratio_setting.UpdateModelPriceByJSONString, map[string]float64{name: 2.5})
			check("default", 4, "per-request", 2.5, 0)
			check("default", 15, "per-request", 2.5, 0)

			set(ratio_setting.UpdateModelPriceByJSONString, clear)
			set(ratio_setting.UpdateModelRatioByJSONString, map[string]float64{name: 2})
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			info := &relaycommon.RelayInfo{OriginModelName: name, UsingGroup: "default"}
			price, err := helper.ModelPriceHelperPerCall(c, info, 15)
			require.NoError(t, err)
			info.PriceData = price
			require.Equal(t, "per-token", price.BillingType)
			require.Equal(t, 2.0, price.ModelRatio)
			require.False(t, info.TaskPriceFinalizedAtSubmit())

			set(ratio_setting.UpdateModelPriceByJSONString, clear)
			set(ratio_setting.UpdateModelPriceBySecondsByJSONString, map[string]any{name: map[string]float64{"per_second": 0.25}})
			check("default", 4, "duration", 1, 4)
			check("default", 15, "duration", 3.75, 15)
			check("default", 0, "duration", 1.25, 5)

			set(ratio_setting.UpdateGroupModelPriceByJSONString, map[string]any{"vip": map[string]float64{name: 2}})
			check("vip", 15, "per-request", 2, 0)
			set(ratio_setting.UpdateGroupModelPriceByJSONString, clear)

			set(ratio_setting.UpdateModelPriceBySecondsByJSONString, clear)
			set(ratio_setting.UpdateModelPriceByResolutionByJSONString, map[string]any{name: map[string]float64{common.Video933Resolution(name): 1.5}})
			check("default", 15, "resolution", 1.5, 0)

			set(ratio_setting.UpdateModelPriceByResolutionByJSONString, clear)
			set(ratio_setting.UpdateModelPriceByJSONString, map[string]float64{name: 2.5})
			check("default", 15, "per-request", 2.5, 0)
		})
	}
}
