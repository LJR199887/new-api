package common

import "strings"

var (
	GrokImagineModelAliases = map[string]string{
		"grok-imagine-1.0":       "grok-imagine-image",
		"grok-imagine-1.0-edit":  "grok-imagine-image-edit",
		"grok-imagine-1.0-video": "grok-imagine-video",
	}
	// OpenAIResponseOnlyModels is a list of models that are only available for OpenAI responses.
	OpenAIResponseOnlyModels = []string{
		"o3-pro",
		"o3-deep-research",
		"o4-mini-deep-research",
	}
	ImageGenerationModels = []string{
		"dall-e-3",
		"dall-e-2",
		"gpt-image-1",
		"gpt-image2",
		"exact:grok-imagine-image",
		"exact:grok-imagine-1.0",
		"exact:grok-imagine-1.0-fast",
		"prefix:imagen-",
		"flux-",
		"flux.1-",
	}
	ImageEditModels = []string{
		"exact:grok-imagine-image-edit",
		"exact:grok-imagine-1.0-edit",
	}
	OpenAIVideoModels = []string{
		"exact:grok-imagine-video",
		"exact:grok-imagine-1.0-video",
	}
	OpenAITextModels = []string{
		"gpt-",
		"o1",
		"o3",
		"o4",
		"chatgpt",
	}
)

func NormalizeGrokImagineModelName(modelName string) string {
	trimmedModelName := strings.TrimSpace(modelName)
	normalizedKey := strings.ToLower(trimmedModelName)
	if alias, ok := GrokImagineModelAliases[normalizedKey]; ok {
		return alias
	}
	return trimmedModelName
}

func GetGrokImagineModelNameCandidates(modelName string) []string {
	trimmedModelName := strings.TrimSpace(modelName)
	if trimmedModelName == "" {
		return nil
	}
	normalizedKey := strings.ToLower(trimmedModelName)
	if _, ok := GrokImagineModelAliases[normalizedKey]; !ok {
		isCanonicalGrokImagineModel := false
		for _, canonicalName := range GrokImagineModelAliases {
			if canonicalName == normalizedKey {
				isCanonicalGrokImagineModel = true
				break
			}
		}
		if !isCanonicalGrokImagineModel {
			return []string{trimmedModelName}
		}
	}
	candidates := make([]string, 0, 3)
	seen := map[string]bool{}
	appendCandidate := func(value string) {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" || seen[value] {
			return
		}
		seen[value] = true
		candidates = append(candidates, value)
	}
	appendCandidate(normalizedKey)
	normalized := strings.ToLower(NormalizeGrokImagineModelName(normalizedKey))
	appendCandidate(normalized)
	for legacyName, canonicalName := range GrokImagineModelAliases {
		if canonicalName == normalized {
			appendCandidate(legacyName)
		}
	}
	return candidates
}

func IsOpenAIResponseOnlyModel(modelName string) bool {
	for _, m := range OpenAIResponseOnlyModels {
		if strings.Contains(modelName, m) {
			return true
		}
	}
	return false
}

func IsImageGenerationModel(modelName string) bool {
	modelName = strings.ToLower(modelName)
	for _, m := range ImageGenerationModels {
		if strings.HasPrefix(m, "exact:") && modelName == strings.TrimPrefix(m, "exact:") {
			return true
		}
		if strings.Contains(modelName, m) {
			return true
		}
		if strings.HasPrefix(m, "prefix:") && strings.HasPrefix(modelName, strings.TrimPrefix(m, "prefix:")) {
			return true
		}
	}
	return false
}

func IsImageEditModel(modelName string) bool {
	modelName = strings.ToLower(modelName)
	for _, m := range ImageEditModels {
		if strings.HasPrefix(m, "exact:") && modelName == strings.TrimPrefix(m, "exact:") {
			return true
		}
		if strings.Contains(modelName, m) {
			return true
		}
		if strings.HasPrefix(m, "prefix:") && strings.HasPrefix(modelName, strings.TrimPrefix(m, "prefix:")) {
			return true
		}
	}
	return false
}

func IsOpenAIVideoModel(modelName string) bool {
	modelName = strings.ToLower(modelName)
	for _, m := range OpenAIVideoModels {
		if strings.HasPrefix(m, "exact:") && modelName == strings.TrimPrefix(m, "exact:") {
			return true
		}
		if strings.Contains(modelName, m) {
			return true
		}
		if strings.HasPrefix(m, "prefix:") && strings.HasPrefix(modelName, strings.TrimPrefix(m, "prefix:")) {
			return true
		}
	}
	return false
}

func IsOpenAITextModel(modelName string) bool {
	modelName = strings.ToLower(modelName)
	for _, m := range OpenAITextModels {
		if strings.Contains(modelName, m) {
			return true
		}
	}
	return false
}
