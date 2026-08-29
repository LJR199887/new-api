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

func TestFilterOtherRatiosForVideo25DurationBilling(t *testing.T) {
	for _, modelName := range []string{
		"video-2.5",
		"video-2.5-480p",
		"minimax-h3-480p",
		"minimax-h3-768p",
		"minimax-h3-2k",
		"minimax-h3-4k",
		"wan3.0-480p",
		"wan3.0-720p",
		"wan3.0-1080p",
	} {
		t.Run(modelName, func(t *testing.T) {
			filtered := FilterOtherRatiosForBillingModel(modelName, map[string]float64{
				"seconds":    10,
				"size":       1.5,
				"resolution": 2,
			})

			assert.Equal(t, map[string]float64{
				"seconds": 10,
			}, filtered)
		})
	}
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
	appendTaskPricePatchDefault("minimax-h3")
	appendTaskPricePatchDefault("minimax-h3-480p")
	appendTaskPricePatchDefault("minimax-h3-768p")
	appendTaskPricePatchDefault("minimax-h3-2k")
	appendTaskPricePatchDefault("minimax-h3-4k")
	appendTaskPricePatchDefault("wan3.0-480p")
	appendTaskPricePatchDefault("wan3.0-720p")
	appendTaskPricePatchDefault("wan3.0-1080p")
	appendTaskPricePatchDefault("ko3")
	appendTaskPricePatchDefault("kling-o3")
	appendTaskPricePatchDefault("kling-video-o-3")
	appendTaskPricePatchDefault("video-2.5")
	appendTaskPricePatchDefault("video-2.5-480p")
	appendTaskPricePatchDefault("video-2.0")
	appendTaskPricePatchDefault("video-2.0-fast")
	appendTaskPricePatchDefault("video-2.0-mini")
	appendTaskPricePatchDefault("video-2.0-480p")
	appendTaskPricePatchDefault("video-2.0-fast-480p")
	appendTaskPricePatchDefault("video-2.0-mini-480p")
	appendTaskPricePatchDefault("video-2.0")

	assert.ElementsMatch(t, []string{
		"minimax-h3",
		"minimax-h3-480p",
		"minimax-h3-768p",
		"minimax-h3-2k",
		"minimax-h3-4k",
		"wan3.0-480p",
		"wan3.0-720p",
		"wan3.0-1080p",
		"ko3",
		"kling-o3",
		"kling-video-o-3",
		"kling-v3",
		"video-2.5",
		"video-2.5-480p",
		"video-2.0",
		"video-2.0-fast",
		"video-2.0-mini",
		"video-2.0-480p",
		"video-2.0-fast-480p",
		"video-2.0-mini-480p",
	}, constant.TaskPricePatches)
}
