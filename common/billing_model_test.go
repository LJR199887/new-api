package common

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/stretchr/testify/assert"
)

func TestFilterOtherRatiosForDurationOnlyModel(t *testing.T) {
	filtered := FilterOtherRatiosForBillingModel("grok-imagine-video", map[string]float64{
		"seconds":    6,
		"size":       1.666667,
		"resolution": 1.5,
	})

	assert.Equal(t, map[string]float64{
		"seconds": 6,
	}, filtered)
}

func TestFilterOtherRatiosForResolutionOnlyModel(t *testing.T) {
	filtered := FilterOtherRatiosForBillingModel("nano-banana-pro", map[string]float64{
		"resolution":        2,
		"quality":           1.5,
		"output_resolution": 4,
		"n":                 3,
	})

	assert.Empty(t, filtered)
}

func TestAppendTaskPricePatchDefault(t *testing.T) {
	original := constant.TaskPricePatches
	t.Cleanup(func() {
		constant.TaskPricePatches = original
	})

	constant.TaskPricePatches = []string{"kling-v3"}
	appendTaskPricePatchDefault("video-2.0")
	appendTaskPricePatchDefault("video-2.0-fast")
	appendTaskPricePatchDefault("video-2.0")

	assert.ElementsMatch(t, []string{
		"kling-v3",
		"video-2.0",
		"video-2.0-fast",
	}, constant.TaskPricePatches)
}
