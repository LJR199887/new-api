package ratio_setting

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetGroupModelPrice(t *testing.T) {
	original := GroupModelPrice2JSONString()
	defer func() {
		_ = UpdateGroupModelPriceByJSONString(original)
	}()

	require.NoError(t, UpdateGroupModelPriceByJSONString(`{
		"vip": {
			"grok-imagine-image-edit": 0.02
		}
	}`))

	price, ok := GetGroupModelPrice("vip", "grok-imagine-image-edit")
	require.True(t, ok)
	require.Equal(t, 0.02, price)

	_, ok = GetGroupModelPrice("default", "grok-imagine-image-edit")
	require.False(t, ok)
}

func TestGetGroupModelPriceBySeconds(t *testing.T) {
	original := GroupModelPriceBySeconds2JSONString()
	defer func() {
		_ = UpdateGroupModelPriceBySecondsByJSONString(original)
	}()

	require.NoError(t, UpdateGroupModelPriceBySecondsByJSONString(`{
		"vip": {
			"grok-imagine-video": {
				"6": 0.05,
				"10": 0.07
			},
			"video-2.5": {
				"per_second": 0.3
			}
		}
	}`))

	price, ok := GetGroupModelPriceBySeconds("vip", "grok-imagine-video", 10)
	require.True(t, ok)
	require.Equal(t, 0.07, price)

	_, ok = GetGroupModelPriceBySeconds("default", "grok-imagine-video", 10)
	require.False(t, ok)

	price, ok = GetGroupModelPriceBySeconds("vip", "video-2.5", 10)
	require.True(t, ok)
	require.InDelta(t, 3.0, price, 1e-12)
}

func TestGetGroupModelPriceByResolution(t *testing.T) {
	original := GroupModelPriceByResolution2JSONString()
	defer func() {
		_ = UpdateGroupModelPriceByResolutionByJSONString(original)
	}()

	require.NoError(t, UpdateGroupModelPriceByResolutionByJSONString(`{
		"vip": {
			"nano-banana-pro": {
				"1K": 0.07,
				"2K": 0.12
			}
		}
	}`))

	price, ok := GetGroupModelPriceByResolution("vip", "nano-banana-pro", "2k")
	require.True(t, ok)
	require.Equal(t, 0.12, price)

	_, ok = GetGroupModelPriceByResolution("vip", "nano-banana-pro", "4K")
	require.False(t, ok)
}
