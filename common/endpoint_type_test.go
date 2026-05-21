package common

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
)

func TestGetEndpointTypesByChannelTypeRecognizesGrokImagineImageModels(t *testing.T) {
	tests := []struct {
		name     string
		channel  int
		model    string
		expected constant.EndpointType
	}{
		{
			name:     "grok imagine image generation",
			channel:  constant.ChannelTypeXai,
			model:    "grok-imagine-image",
			expected: constant.EndpointTypeImageGeneration,
		},
		{
			name:     "grok imagine 1.0 fast generation",
			channel:  constant.ChannelTypeXai,
			model:    "grok-imagine-1.0-fast",
			expected: constant.EndpointTypeImageGeneration,
		},
		{
			name:     "grok imagine image edit",
			channel:  constant.ChannelTypeXai,
			model:    "grok-imagine-image-edit",
			expected: constant.EndpointTypeImageEdit,
		},
		{
			name:     "grok imagine video on xai",
			channel:  constant.ChannelTypeXai,
			model:    "grok-imagine-video",
			expected: constant.EndpointTypeOpenAIVideo,
		},
		{
			name:     "grok imagine video stays video",
			channel:  constant.ChannelTypeSora,
			model:    "grok-imagine-video",
			expected: constant.EndpointTypeOpenAIVideo,
		},
		{
			name:     "ko3 video generation",
			channel:  constant.ChannelTypeSora,
			model:    "ko3",
			expected: constant.EndpointTypeOpenAIVideo,
		},
		{
			name:     "ko3 compatibility alias",
			channel:  constant.ChannelTypeSora,
			model:    "kling-video-o-3",
			expected: constant.EndpointTypeOpenAIVideo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetEndpointTypesByChannelType(tt.channel, tt.model)
			if len(got) == 0 {
				t.Fatalf("expected endpoint types for %s", tt.model)
			}
			if got[0] != tt.expected {
				t.Fatalf("expected first endpoint %s, got %s", tt.expected, got[0])
			}
		})
	}
}
