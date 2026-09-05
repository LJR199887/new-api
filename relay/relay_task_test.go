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

func TestCalcTaskQuotaWithRatiosUsesPerSecondPrice(t *testing.T) {
	original := ratio_setting.ModelPriceBySeconds2JSONString()
	originalQuotaPerUnit := common.QuotaPerUnit
	defer func() {
		_ = ratio_setting.UpdateModelPriceBySecondsByJSONString(original)
		common.QuotaPerUnit = originalQuotaPerUnit
	}()

	common.QuotaPerUnit = 500
	require.NoError(t, ratio_setting.UpdateModelPriceBySecondsByJSONString(`{
		"video-2.5": {
			"per_second": 0.3
		}
	}`))

	info := &relaycommon.RelayInfo{
		OriginModelName: "video-2.5",
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
	})

	assert.Equal(t, int(3.0*common.QuotaPerUnit), quota)
	assert.Equal(t, 1.0, ratios["seconds"])
	assert.InDelta(t, 3.0, info.PriceData.ModelPrice, 1e-12)
	assert.Equal(t, "duration", info.PriceData.BillingType)
	assert.Equal(t, 10, info.PriceData.BillingSeconds)
	assert.InDelta(t, 0.3, info.PriceData.BillingUnitPrice, 1e-12)
	assert.InDelta(t, 3.0, info.PriceData.BillingTotalPrice, 1e-12)
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
				"per_second": 0.007
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
	assert.Equal(t, "duration", info.PriceData.BillingType)
	assert.Equal(t, 10, info.PriceData.BillingSeconds)
	assert.InDelta(t, 0.007, info.PriceData.BillingUnitPrice, 1e-12)
	assert.InDelta(t, 0.07, info.PriceData.BillingTotalPrice, 1e-12)
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

func TestCalc933TaskQuotaWithRatiosUsesPerSecondPrice(t *testing.T) {
	original := ratio_setting.ModelPriceBySeconds2JSONString()
	originalQuotaPerUnit := common.QuotaPerUnit
	defer func() {
		_ = ratio_setting.UpdateModelPriceBySecondsByJSONString(original)
		common.QuotaPerUnit = originalQuotaPerUnit
	}()

	common.QuotaPerUnit = 500
	require.NoError(t, ratio_setting.UpdateModelPriceBySecondsByJSONString(`{
		"933-video2.0-mini-480p": {
			"per_second": 0.3
		}
	}`))

	info := &relaycommon.RelayInfo{
		OriginModelName: "933-video2.0-mini-480p",
		PriceData: types.PriceData{
			BaseQuota: 100,
			Quota:     100,
			GroupRatioInfo: types.GroupRatioInfo{
				GroupRatio: 1,
			},
		},
	}

	quota, ratios := calcTaskQuotaWithRatios(nil, info, map[string]float64{
		"seconds": 5, "resolution": 100, "images": 9,
	})

	assert.Equal(t, int(1.5*common.QuotaPerUnit), quota)
	assert.Equal(t, 1.0, ratios["seconds"])
	assert.InDelta(t, 1.5, info.PriceData.ModelPrice, 1e-12)
	assert.Equal(t, "duration", info.PriceData.BillingType)
	assert.Equal(t, 5, info.PriceData.BillingSeconds)
	assert.InDelta(t, 0.3, info.PriceData.BillingUnitPrice, 1e-12)
	assert.InDelta(t, 1.5, info.PriceData.BillingTotalPrice, 1e-12)
}

func Test933AllModelsGroupPricingIgnoresResolutionAndReferenceCounts(t *testing.T) {
	secondsBackup := ratio_setting.ModelPriceBySeconds2JSONString()
	groupsBackup := ratio_setting.GroupModelPriceBySeconds2JSONString()
	resolutionBackup := ratio_setting.ModelPriceByResolution2JSONString()
	t.Cleanup(func() {
		_ = ratio_setting.UpdateModelPriceBySecondsByJSONString(secondsBackup)
		_ = ratio_setting.UpdateGroupModelPriceBySecondsByJSONString(groupsBackup)
		_ = ratio_setting.UpdateModelPriceByResolutionByJSONString(resolutionBackup)
	})
	for _, name := range []string{"933-video2.0", "933-video2.0-480p", "933-video2.0-mini", "933-video2.0-mini-480p"} {
		data, err := common.Marshal(map[string]any{name: map[string]float64{"per_second": 0.2}})
		require.NoError(t, err)
		require.NoError(t, ratio_setting.UpdateModelPriceBySecondsByJSONString(string(data)))
		data, err = common.Marshal(map[string]any{"vip": map[string]any{name: map[string]float64{"per_second": 0.1}}})
		require.NoError(t, err)
		require.NoError(t, ratio_setting.UpdateGroupModelPriceBySecondsByJSONString(string(data)))
		data, err = common.Marshal(map[string]any{name: map[string]float64{"720p": 99}})
		require.NoError(t, err)
		require.NoError(t, ratio_setting.UpdateModelPriceByResolutionByJSONString(string(data)))
		for _, seconds := range []int{4, 5} {
			for _, group := range []string{"default", "vip"} {
				c, _ := gin.CreateTestContext(httptest.NewRecorder())
				c.Set("task_request", relaycommon.TaskSubmitReq{ResolutionName: "720p"})
				info := &relaycommon.RelayInfo{OriginModelName: name, UsingGroup: group, PriceData: types.PriceData{BaseQuota: 100, GroupRatioInfo: types.GroupRatioInfo{GroupRatio: 2}}}
				quota, ratios := calcTaskQuotaWithRatios(c, info, map[string]float64{"seconds": float64(seconds), "resolution": 99, "images": 9, "audio": 3})
				price := 0.2 * float64(seconds) * 2
				if group == "vip" {
					price = 0.1 * float64(seconds)
				}
				assert.Equal(t, int(price*common.QuotaPerUnit), quota, name)
				assert.Len(t, ratios, 1)
				assert.Equal(t, "duration", info.PriceData.BillingType)
				assert.Equal(t, seconds, info.PriceData.BillingSeconds)
			}
		}
	}
}
