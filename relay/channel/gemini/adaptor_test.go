package gemini

import (
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

func TestConvertImageRequestUsesExplicitAspectRatio(t *testing.T) {
	adaptor := &Adaptor{}
	converted, err := adaptor.ConvertImageRequest(nil, &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "imagen-4.0-generate-001",
		},
	}, dto.ImageRequest{
		Model:            "nano-banana",
		Prompt:           "make a poster",
		Size:             "1024x1024",
		AspectRatio:      "9:16",
		OutputResolution: "2K",
	})
	if err != nil {
		t.Fatalf("ConvertImageRequest returned error: %v", err)
	}

	req, ok := converted.(dto.GeminiImageRequest)
	if !ok {
		t.Fatalf("expected GeminiImageRequest, got %T", converted)
	}
	if req.Parameters.AspectRatio != "9:16" {
		t.Fatalf("expected explicit aspect ratio 9:16, got %q", req.Parameters.AspectRatio)
	}
	if req.Parameters.ImageSize != "2K" {
		t.Fatalf("expected output_resolution to set imageSize=2K, got %q", req.Parameters.ImageSize)
	}
}

func TestConvertImageRequestFallsBackToSizeAspectRatio(t *testing.T) {
	adaptor := &Adaptor{}
	converted, err := adaptor.ConvertImageRequest(nil, &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "imagen-4.0-generate-001",
		},
	}, dto.ImageRequest{
		Model:  "nano-banana",
		Prompt: "make a poster",
		Size:   "1792x1024",
	})
	if err != nil {
		t.Fatalf("ConvertImageRequest returned error: %v", err)
	}

	req, ok := converted.(dto.GeminiImageRequest)
	if !ok {
		t.Fatalf("expected GeminiImageRequest, got %T", converted)
	}
	if req.Parameters.AspectRatio != "16:9" {
		t.Fatalf("expected size fallback aspect ratio 16:9, got %q", req.Parameters.AspectRatio)
	}
}
