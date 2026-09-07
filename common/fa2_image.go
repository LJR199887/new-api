package common

import "strings"

// Fa2ImageModelSpec describes the public fa2api image parameters supported by
// new-api. Returned slices are fresh values and may be safely modified.
type Fa2ImageModelSpec struct {
	DefaultResolution string
	Resolutions       []string
	AspectRatios      []string
	MaxImages         int
}

func GetFa2ImageModelSpec(modelName string) (Fa2ImageModelSpec, bool) {
	commonRatios := []string{"21:9", "16:9", "3:2", "4:3", "5:4", "1:1", "4:5", "3:4", "2:3", "9:16"}
	switch normalizeBillingModelName(modelName) {
	case "gpt-image-2":
		return Fa2ImageModelSpec{
			DefaultResolution: "2K",
			Resolutions:       []string{"1K", "2K", "4K"},
			AspectRatios:      []string{"3:1", "21:9", "2:1", "16:9", "3:2", "4:3", "5:4", "1:1", "4:5", "3:4", "2:3", "9:16", "1:2", "1:3"},
			MaxImages:         17,
		}, true
	case "nano-banana-pro":
		return Fa2ImageModelSpec{DefaultResolution: "1K", Resolutions: []string{"1K", "2K", "4K"}, AspectRatios: commonRatios, MaxImages: 10}, true
	case "nano-banana2":
		return Fa2ImageModelSpec{DefaultResolution: "1K", Resolutions: []string{"1K", "2K", "4K"}, AspectRatios: commonRatios, MaxImages: 14}, true
	case "seedream-5-0":
		return Fa2ImageModelSpec{
			DefaultResolution: "2K",
			Resolutions:       []string{"2K", "3K"},
			AspectRatios:      []string{"16:9", "4:3", "1:1", "3:4", "9:16"},
			MaxImages:         14,
		}, true
	default:
		return Fa2ImageModelSpec{}, false
	}
}

func IsFa2ImageModel(modelName string) bool {
	_, ok := GetFa2ImageModelSpec(modelName)
	return ok
}

func Fa2ImageModelSupportsResolution(modelName string, resolution string) bool {
	spec, ok := GetFa2ImageModelSpec(modelName)
	if !ok {
		return false
	}
	resolution = strings.ToUpper(strings.TrimSpace(resolution))
	for _, candidate := range spec.Resolutions {
		if resolution == candidate {
			return true
		}
	}
	return false
}

func Fa2ImageModelSupportsAspectRatio(modelName string, aspectRatio string) bool {
	spec, ok := GetFa2ImageModelSpec(modelName)
	if !ok {
		return false
	}
	aspectRatio = strings.TrimSpace(aspectRatio)
	for _, candidate := range spec.AspectRatios {
		if aspectRatio == candidate {
			return true
		}
	}
	return false
}
