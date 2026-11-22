package common

import "strings"

var video25ModelNames = [...]string{
	"video-2.5",
	"video-2.5-480p",
	"video-2.5-1080p",
	"video2.5",
	"video2.5-480p",
	"video2.5-1080p",
	"video2.5-flash",
	"video2.5-flash-480p",
	"A-video2.5",
	"A-video2.5-480p",
	"A-video2.5-1080p",
	"A-video2.5-flash",
	"A-video2.5-flash-480p",
}

// Video25ModelNames returns the configured model IDs with their original casing.
func Video25ModelNames() []string {
	return append([]string(nil), video25ModelNames[:]...)
}

func IsVideo25Model(name string) bool {
	name = strings.TrimSpace(name)
	for _, model := range video25ModelNames {
		if strings.EqualFold(name, model) {
			return true
		}
	}
	return false
}

func Video25Resolution(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	switch {
	case strings.HasSuffix(name, "-480p"):
		return "480p"
	case strings.HasSuffix(name, "-1080p"):
		return "1080p"
	default:
		return "720p"
	}
}
