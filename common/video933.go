package common

var video933ModelNames = func() []string {
	names := []string{
		"933-video2.0", "933-video2.0-480p", "933-video2.0-1080p",
		"933-video2.0-mini", "933-video2.0-mini-480p",
	}
	for _, prefix := range []string{"", "A-"} {
		for _, variant := range []string{"video2.0", "video2.0-fast", "video2.0-mini"} {
			for _, suffix := range []string{"", "-480p", "-1080p"} {
				names = append(names, prefix+variant+suffix)
			}
		}
	}
	return names
}()

var video933ModelSet = func() map[string]struct{} {
	models := make(map[string]struct{}, len(video933ModelNames))
	for _, name := range video933ModelNames {
		models[normalizeBillingModelName(name)] = struct{}{}
	}
	return models
}()

// Video933ModelNames returns the model IDs using the 933 video contract.
func Video933ModelNames() []string {
	return append([]string(nil), video933ModelNames...)
}

// Is933VideoModel matches the original 933 models and the compatible Video 2.0 variants.
func Is933VideoModel(name string) bool {
	_, ok := video933ModelSet[normalizeBillingModelName(name)]
	return ok
}
