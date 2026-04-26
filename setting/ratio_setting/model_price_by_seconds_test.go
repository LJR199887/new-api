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
		}
	}`))

	price, ok := GetModelPriceBySeconds("grok-imagine-video", 6)
	require.True(t, ok)
	assert.Equal(t, 0.2, price)

	_, ok = GetModelPriceBySeconds("grok-imagine-video", 8)
	assert.False(t, ok)
}
