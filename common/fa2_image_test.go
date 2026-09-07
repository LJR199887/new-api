package common

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFa2ImageModelSpecs(t *testing.T) {
	tests := []struct {
		model             string
		defaultResolution string
		maxImages         int
	}{
		{"gpt-image-2", "2K", 17},
		{"nano-banana-pro", "1K", 10},
		{"nano-banana2", "1K", 14},
		{"seedream-5-0", "2K", 14},
	}

	for _, test := range tests {
		t.Run(test.model, func(t *testing.T) {
			spec, ok := GetFa2ImageModelSpec(test.model)
			require.True(t, ok)
			require.Equal(t, test.defaultResolution, spec.DefaultResolution)
			require.Equal(t, test.maxImages, spec.MaxImages)
			require.True(t, Fa2ImageModelSupportsResolution(test.model, test.defaultResolution))
			require.True(t, Fa2ImageModelSupportsAspectRatio(test.model, "1:1"))
			require.True(t, IsResolutionOnlyBillingModel(test.model))
		})
	}

	require.False(t, IsFa2ImageModel("gpt-image2"))
	require.False(t, IsFa2ImageModel("seedream-5.0"))
	require.False(t, Fa2ImageModelSupportsResolution("seedream-5-0", "1K"))
	require.False(t, Fa2ImageModelSupportsAspectRatio("seedream-5-0", "21:9"))
}
