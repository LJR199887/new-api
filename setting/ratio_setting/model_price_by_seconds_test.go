package ratio_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetModelPriceBySeconds(t *testing.T) {
	original := ModelPriceBySeconds2JSONString()
	defer func() {
		_ = UpdateModelPriceBySecondsByJSONString(original)
	}()

	require.NoError(t, UpdateModelPriceBySecondsByJSONString(`{
		"grok-imagine-video": {
			"6": 0.2,
			"10": 0.28
		},
		"video-2.5": {
			"per_second": 0.3
		}
	}`))

	price, ok := GetModelPriceBySeconds("grok-imagine-video", 6)
	require.True(t, ok)
	assert.Equal(t, 0.2, price)

	_, ok = GetModelPriceBySeconds("grok-imagine-video", 8)
	assert.False(t, ok)

	price, ok = GetModelPriceBySeconds("video-2.5", 10)
	require.True(t, ok)
	assert.InDelta(t, 3.0, price, 1e-12)

	price, ok = GetModelPriceBySeconds("video-2.5", 7)
	require.True(t, ok)
	assert.InDelta(t, 2.1, price, 1e-12)

	minPrice, ok := GetModelPriceBySecondsMin("video-2.5")
	require.True(t, ok)
	assert.InDelta(t, 0.3, minPrice, 1e-12)
}
