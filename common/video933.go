package common

// Is933VideoModel matches the supported fa2api video variants.
func Is933VideoModel(name string) bool {
	switch normalizeBillingModelName(name) {
	case "933-video2.0", "933-video2.0-480p", "933-video2.0-1080p", "933-video2.0-mini", "933-video2.0-mini-480p":
		return true
	}
	return false
}
