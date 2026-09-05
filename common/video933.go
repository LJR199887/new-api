package common

// Is933VideoModel matches only the four fa2api video models supported here.
func Is933VideoModel(name string) bool {
	switch normalizeBillingModelName(name) {
	case "933-video2.0", "933-video2.0-480p", "933-video2.0-mini", "933-video2.0-mini-480p":
		return true
	}
	return false
}
