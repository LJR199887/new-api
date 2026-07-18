package relay

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/types"
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

func TestCalcTaskQuotaWithRatiosUsesMappedSecondsPrice(t *testing.T) {
	original := ratio_setting.ModelPriceBySeconds2JSONString()
	originalQuotaPerUnit := common.QuotaPerUnit
	defer func() {
		_ = ratio_setting.UpdateModelPriceBySecondsByJSONString(original)
		common.QuotaPerUnit = originalQuotaPerUnit
	}()

	common.QuotaPerUnit = 500
	require.NoError(t, ratio_setting.UpdateModelPriceBySecondsByJSONString(`{
		"grok-imagine-video": {
			"10": 0.2
		}
	}`))

	info := &relaycommon.RelayInfo{
		OriginModelName: "grok-imagine-video",
		PriceData: types.PriceData{
			BaseQuota: 100,
			Quota:     100,
			GroupRatioInfo: types.GroupRatioInfo{
				GroupRatio: 1,
			},
		},
	}

	quota, ratios := calcTaskQuotaWithRatios(nil, info, map[string]float64{
		"seconds": 10,
		"size":    1.666667,
	})

	assert.Equal(t, int(0.2*common.QuotaPerUnit), quota)
	assert.Equal(t, 1.0, ratios["seconds"])
	_, hasSize := ratios["size"]
	assert.False(t, hasSize)
	assert.Equal(t, 0.2, info.PriceData.ModelPrice)
}

func TestCalcTaskQuotaWithRatiosUsesGroupMappedSecondsPriceWithoutGroupRatio(t *testing.T) {
	original := ratio_setting.GroupModelPriceBySeconds2JSONString()
	originalQuotaPerUnit := common.QuotaPerUnit
	defer func() {
		_ = ratio_setting.UpdateGroupModelPriceBySecondsByJSONString(original)
		common.QuotaPerUnit = originalQuotaPerUnit
	}()

	common.QuotaPerUnit = 500
	require.NoError(t, ratio_setting.UpdateGroupModelPriceBySecondsByJSONString(`{
		"vip": {
			"grok-imagine-video": {
				"10": 0.07
			}
		}
	}`))

	info := &relaycommon.RelayInfo{
		OriginModelName: "grok-imagine-video",
		UsingGroup:      "default",
		UserGroup:       "vip",
		PriceData: types.PriceData{
			BaseQuota: 100,
			Quota:     100,
			GroupRatioInfo: types.GroupRatioInfo{
				GroupRatio: 0.5,
			},
		},
	}

	quota, ratios := calcTaskQuotaWithRatios(nil, info, map[string]float64{
		"seconds": 10,
		"size":    1.666667,
	})

	assert.Equal(t, int(0.07*common.QuotaPerUnit), quota)
	assert.Equal(t, 1.0, ratios["seconds"])
	_, hasSize := ratios["size"]
	assert.False(t, hasSize)
	assert.Equal(t, 0.07, info.PriceData.ModelPrice)
	assert.True(t, info.PriceData.GroupPriceOverride)
	assert.Equal(t, "vip", info.PriceData.GroupPriceOverrideGroup)
}

func TestCalcTaskQuotaWithRatiosFallsBackToLinearSeconds(t *testing.T) {
	original := ratio_setting.ModelPriceBySeconds2JSONString()
	defer func() {
		_ = ratio_setting.UpdateModelPriceBySecondsByJSONString(original)
	}()

	require.NoError(t, ratio_setting.UpdateModelPriceBySecondsByJSONString(`{}`))

	info := &relaycommon.RelayInfo{
		OriginModelName: "grok-imagine-video",
		PriceData: types.PriceData{
			BaseQuota: 100,
			Quota:     100,
		},
	}

	quota, ratios := calcTaskQuotaWithRatios(nil, info, map[string]float64{
		"seconds": 12,
		"size":    1.5,
	})

	assert.Equal(t, 1200, quota)
	assert.Equal(t, 12.0, ratios["seconds"])
	_, hasSize := ratios["size"]
	assert.False(t, hasSize)
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
