package service

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/stretchr/testify/assert"
)

func TestBuildTaskConsumptionLogContentUsesDurationBillingLabel(t *testing.T) {
	info := &relaycommon.RelayInfo{
		OriginModelName: "video-2.5-480p",
		TaskRelayInfo: &relaycommon.TaskRelayInfo{
			Action: constant.TaskActionTextGenerate,
		},
		PriceData: types.PriceData{
			BillingType:    "duration",
			BillingSeconds: 4,
		},
	}

	content := buildTaskConsumptionLogContent(info)
	assert.Contains(t, content, "按时长计费（4 秒）")
	assert.NotContains(t, content, "按次计费")
}

func TestBuildTaskConsumptionLogContentKeepsPerCallLabel(t *testing.T) {
	originalPatches := constant.TaskPricePatches
	t.Cleanup(func() {
		constant.TaskPricePatches = originalPatches
	})
	constant.TaskPricePatches = []string{"video-2.0"}

	info := &relaycommon.RelayInfo{
		OriginModelName: "video-2.0",
		TaskRelayInfo: &relaycommon.TaskRelayInfo{
			Action: constant.TaskActionTextGenerate,
		},
	}

	content := buildTaskConsumptionLogContent(info)
	assert.Contains(t, content, "按次计费")
}
