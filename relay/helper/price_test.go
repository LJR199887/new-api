package helper

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModelPriceHelperUsesSecondsPriceForChatCompatibleVideo(t *testing.T) {
	original := ratio_setting.ModelPriceBySeconds2JSONString()
	originalQuotaPerUnit := common.QuotaPerUnit
	defer func() {
		_ = ratio_setting.UpdateModelPriceBySecondsByJSONString(original)
		common.QuotaPerUnit = originalQuotaPerUnit
	}()

	common.QuotaPerUnit = 500
	require.NoError(t, ratio_setting.UpdateModelPriceBySecondsByJSONString(`{
		"veo31": {
			"4": 0.4,
			"8": 0.8
		}
	}`))

	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	duration := 8
	request := &dto.GeneralOpenAIRequest{
		Model:    "veo31",
		Duration: &duration,
	}
	info := &relaycommon.RelayInfo{
		OriginModelName: "veo31",
		UsingGroup:      "default",
		Request:         request,
	}

	priceData, err := ModelPriceHelper(c, info, 0, &types.TokenCountMeta{})

	require.NoError(t, err)
	assert.True(t, priceData.UsePrice)
	assert.Equal(t, 0.8, priceData.ModelPrice)
	assert.Equal(t, int(0.8*common.QuotaPerUnit), priceData.QuotaToPreConsume)
}

func TestModelPriceHelperUsesGroupResolutionPriceWithoutGroupRatio(t *testing.T) {
	originalGroupResolution := ratio_setting.GroupModelPriceByResolution2JSONString()
	originalGroupRatio := ratio_setting.GroupRatio2JSONString()
	originalQuotaPerUnit := common.QuotaPerUnit
	defer func() {
		_ = ratio_setting.UpdateGroupModelPriceByResolutionByJSONString(originalGroupResolution)
		_ = ratio_setting.UpdateGroupRatioByJSONString(originalGroupRatio)
		common.QuotaPerUnit = originalQuotaPerUnit
	}()

	common.QuotaPerUnit = 500
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{
		"default": 1,
		"vip": 0.5
	}`))
	require.NoError(t, ratio_setting.UpdateGroupModelPriceByResolutionByJSONString(`{
		"vip": {
			"nano-banana-pro": {
				"2K": 0.12
			}
		}
	}`))

	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	request := &dto.GeneralOpenAIRequest{
		Model:            "nano-banana-pro",
		OutputResolution: "2K",
	}
	info := &relaycommon.RelayInfo{
		OriginModelName: "nano-banana-pro",
		UsingGroup:      "default",
		UserGroup:       "vip",
		Request:         request,
	}

	priceData, err := ModelPriceHelper(c, info, 0, &types.TokenCountMeta{})

	require.NoError(t, err)
	assert.True(t, priceData.UsePrice)
	assert.True(t, priceData.GroupPriceOverride)
	assert.Equal(t, "vip", priceData.GroupPriceOverrideGroup)
	assert.Equal(t, 0.12, priceData.ModelPrice)
	assert.Equal(t, 1.0, priceData.GroupRatioInfo.GroupRatio)
	assert.Equal(t, int(0.12*common.QuotaPerUnit), priceData.QuotaToPreConsume)
}

func TestModelPriceHelperUsesGroupPerCallPriceWithoutGroupRatio(t *testing.T) {
	originalGroupPrice := ratio_setting.GroupModelPrice2JSONString()
	originalGroupRatio := ratio_setting.GroupRatio2JSONString()
	originalQuotaPerUnit := common.QuotaPerUnit
	defer func() {
		_ = ratio_setting.UpdateGroupModelPriceByJSONString(originalGroupPrice)
		_ = ratio_setting.UpdateGroupRatioByJSONString(originalGroupRatio)
		common.QuotaPerUnit = originalQuotaPerUnit
	}()

	common.QuotaPerUnit = 500
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{
		"default": 1,
		"vip": 0.5
	}`))
	require.NoError(t, ratio_setting.UpdateGroupModelPriceByJSONString(`{
		"vip": {
			"grok-imagine-image-edit": 0.02
		}
	}`))

	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	request := &dto.ImageRequest{
		Model: "grok-imagine-image-edit",
	}
	info := &relaycommon.RelayInfo{
		OriginModelName: "grok-imagine-image-edit",
		UsingGroup:      "default",
		UserGroup:       "vip",
		Request:         request,
	}

	priceData, err := ModelPriceHelper(c, info, 0, &types.TokenCountMeta{})

	require.NoError(t, err)
	assert.True(t, priceData.UsePrice)
	assert.True(t, priceData.GroupPriceOverride)
	assert.Equal(t, "vip", priceData.GroupPriceOverrideGroup)
	assert.Equal(t, 0.02, priceData.ModelPrice)
	assert.Equal(t, 1.0, priceData.GroupRatioInfo.GroupRatio)
	assert.Equal(t, int(0.02*common.QuotaPerUnit), priceData.QuotaToPreConsume)
}

func TestModelPriceHelperFallsBackToSecondsMinPrice(t *testing.T) {
	original := ratio_setting.ModelPriceBySeconds2JSONString()
	originalQuotaPerUnit := common.QuotaPerUnit
	defer func() {
		_ = ratio_setting.UpdateModelPriceBySecondsByJSONString(original)
		common.QuotaPerUnit = originalQuotaPerUnit
	}()

	common.QuotaPerUnit = 500
	require.NoError(t, ratio_setting.UpdateModelPriceBySecondsByJSONString(`{
		"veo31": {
			"4": 0.4,
			"8": 0.8
		}
	}`))

	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	request := &dto.GeneralOpenAIRequest{
		Model: "veo31",
	}
	info := &relaycommon.RelayInfo{
		OriginModelName: "veo31",
		UsingGroup:      "default",
		Request:         request,
	}

	priceData, err := ModelPriceHelper(c, info, 0, &types.TokenCountMeta{})

	require.NoError(t, err)
	assert.True(t, priceData.UsePrice)
	assert.Equal(t, 0.4, priceData.ModelPrice)
	assert.Equal(t, int(0.4*common.QuotaPerUnit), priceData.QuotaToPreConsume)
}

func Test933RequiresExplicitPerSecondPricing(t *testing.T) {
	original := ratio_setting.ModelPriceBySeconds2JSONString()
	t.Cleanup(func() { _ = ratio_setting.UpdateModelPriceBySecondsByJSONString(original) })
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	info := &relaycommon.RelayInfo{OriginModelName: "933-video2.0", UsingGroup: "default"}
	require.NoError(t, ratio_setting.UpdateModelPriceBySecondsByJSONString(`{"933-video2.0":{"5":0.5}}`))
	_, err := ModelPriceHelperPerCall(c, info)
	require.Error(t, err)
	require.NoError(t, ratio_setting.UpdateModelPriceBySecondsByJSONString(`{"933-video2.0":{"per_second":0.2}}`))
	price, err := ModelPriceHelperPerCall(c, info)
	require.NoError(t, err)
	assert.Equal(t, 0.2, price.ModelPrice)
}

func TestFa2ImageModelsRequireExactResolutionPricing(t *testing.T) {
	original := ratio_setting.ModelPriceByResolution2JSONString()
	originalQuotaPerUnit := common.QuotaPerUnit
	t.Cleanup(func() {
		_ = ratio_setting.UpdateModelPriceByResolutionByJSONString(original)
		common.QuotaPerUnit = originalQuotaPerUnit
	})

	common.QuotaPerUnit = 500
	require.NoError(t, ratio_setting.UpdateModelPriceByResolutionByJSONString(`{
		"gpt-image-2":{"1K":0.1,"2K":0.2,"4K":0.4},
		"nano-banana-pro":{"1K":0.05},
		"nano-banana2":{"1K":0.06},
		"seedream-5-0":{"2K":0.3,"3K":0.45}
	}`))

	tests := []struct {
		model      string
		resolution string
		price      float64
	}{
		{"gpt-image-2", "4K", 0.4},
		{"nano-banana-pro", "1K", 0.05},
		{"nano-banana2", "1K", 0.06},
		{"seedream-5-0", "3K", 0.45},
	}
	for _, test := range tests {
		t.Run(test.model, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			info := &relaycommon.RelayInfo{
				OriginModelName: test.model,
				UsingGroup:      "default",
				Request: &dto.ImageRequest{
					Model:            test.model,
					OutputResolution: test.resolution,
				},
			}
			priceData, err := ModelPriceHelper(c, info, 0, &types.TokenCountMeta{})
			require.NoError(t, err)
			require.True(t, priceData.UsePrice)
			require.Equal(t, test.price, priceData.ModelPrice)
			require.Equal(t, int(test.price*float64(common.QuotaPerUnit)), priceData.QuotaToPreConsume)
			require.Equal(t, priceData, info.PriceData)
		})
	}

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	info := &relaycommon.RelayInfo{
		OriginModelName: "seedream-5-0",
		UsingGroup:      "default",
		Request:         &dto.ImageRequest{Model: "seedream-5-0", OutputResolution: "4K"},
	}
	_, err := ModelPriceHelper(c, info, 0, &types.TokenCountMeta{})
	require.ErrorContains(t, err, "requires ModelPriceByResolution pricing for 4K")
}
