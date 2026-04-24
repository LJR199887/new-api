package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
)

func TestCalculateInviterRechargeReward(t *testing.T) {
	originalRatio := common.QuotaForInviterRechargeRatio
	defer func() {
		common.QuotaForInviterRechargeRatio = originalRatio
	}()

	tests := []struct {
		name  string
		ratio float64
		quota int
		want  int
	}{
		{name: "disabled", ratio: 0, quota: 500000, want: 0},
		{name: "simple percent", ratio: 10, quota: 500000, want: 50000},
		{name: "fractional percent floors", ratio: 12.5, quota: 3, want: 0},
		{name: "negative quota", ratio: 10, quota: -1, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			common.QuotaForInviterRechargeRatio = tt.ratio
			if got := calculateInviterRechargeReward(tt.quota); got != tt.want {
				t.Fatalf("calculateInviterRechargeReward(%d) = %d, want %d", tt.quota, got, tt.want)
			}
		})
	}
}
