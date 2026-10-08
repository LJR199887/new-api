package sora

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// video-2.5 uses fa2api's URL-based video-generation contract, not the
// start_frame/video_reference contract used by older Seedance relays.
func normalizeVideo25Request(body map[string]interface{}, model string) error {
	prompt := strings.TrimSpace(stringifyBodyValue(body["prompt"]))
	if len([]rune(prompt)) < 3 {
		return fmt.Errorf("prompt must contain at least 3 characters for video-2.5")
	}

	images, err := video25URLs(body, "image_urls", "images", "image_url", "image", "input_reference")
	if err != nil {
		return err
	}
	videos, err := video25URLs(body, "video_urls", "video_reference", "video_url")
	if err != nil {
		return err
	}
	audios, err := video25URLs(body, "audio_urls", "audio_reference", "audio_url")
	if err != nil {
		return err
	}
	if len(images) > 30 || len(videos) > 10 || len(audios) > 10 {
		return fmt.Errorf("video-2.5 supports at most 30 images, 10 videos and 10 audios")
	}

	start, err := video25FrameURL(body, "start_image_url", "start_frame")
	if err != nil {
		return err
	}
	end, err := video25FrameURL(body, "end_image_url", "end_frame")
	if err != nil {
		return err
	}
	if (start != "") != (end != "") {
		return fmt.Errorf("start_image_url and end_image_url must be supplied together")
	}
	if start != "" && (len(images) > 0 || len(videos) > 0 || len(audios) > 0) {
		return fmt.Errorf("first/last frames cannot be combined with other references")
	}
	if len(audios) > 0 && len(images) == 0 && len(videos) == 0 {
		return fmt.Errorf("audio reference requires at least one image or video reference")
	}

	mode := "t2v"
	switch {
	case start != "":
		mode = "i2v_start_end"
	case len(images) > 0:
		mode = "i2v_ref"
	case len(videos) > 0:
		mode = "t2v_video_ref"
	}
	explicitMode := strings.TrimSpace(stringifyBodyValue(body["generation_mode"]))
	if explicitMode != "" && explicitMode != mode {
		return fmt.Errorf("generation_mode does not match supplied reference assets: expected %s", mode)
	}

	metadata, _ := body["metadata"].(map[string]interface{})
	ratio := strings.TrimSpace(stringifyBodyValue(body["aspect_ratio"]))
	if ratio == "" {
		ratio = strings.TrimSpace(stringifyBodyValue(metadata["ratio"]))
	}
	if ratio == "" {
		ratio = seedanceAspectRatioFromSize(stringifyBodyValue(body["size"]))
	}
	if ratio == "" {
		ratio = "9:16"
	}
	switch ratio {
	case "16:9", "9:16", "4:3", "3:4", "1:1", "21:9":
	default:
		return fmt.Errorf("unsupported aspect_ratio %q for video-2.5", ratio)
	}

	resolution := "720p"
	if model == "video-2.5-480p" {
		resolution = "480p"
	}
	providedResolution := strings.TrimSpace(stringifyBodyValue(body["resolution"]))
	if providedResolution == "" {
		providedResolution = strings.TrimSpace(stringifyBodyValue(metadata["resolution"]))
	}
	if providedResolution != "" && providedResolution != resolution {
		return fmt.Errorf("unsupported resolution %q for %s: expected %s", providedResolution, model, resolution)
	}

	duration := 5
	if mode == "i2v_ref" || mode == "i2v_start_end" {
		duration = 4
	}
	requestedDuration := strings.TrimSpace(stringifyBodyValue(body["duration"]))
	if requestedDuration == "" {
		requestedDuration = strings.TrimSpace(stringifyBodyValue(body["seconds"]))
	}
	if requestedDuration != "" {
		duration, err = strconv.Atoi(requestedDuration)
		if err != nil || duration < 4 || duration > 30 {
			return fmt.Errorf("duration must be an integer between 4 and 30 for video-2.5")
		}
	}

	// Rebuild only the documented upstream payload. In particular, never send
	// legacy reference objects or stale dimension metadata in place of URLs.
	for key := range body {
		delete(body, key)
	}
	body["model"] = model
	body["prompt"] = prompt
	body["duration"] = duration
	body["aspect_ratio"] = ratio
	body["resolution"] = resolution
	if explicitMode != "" {
		body["generation_mode"] = explicitMode
	}
	if len(images) > 0 {
		body["image_urls"] = images
	}
	if len(videos) > 0 {
		body["video_urls"] = videos
	}
	if len(audios) > 0 {
		body["audio_urls"] = audios
	}
	if start != "" {
		body["start_image_url"] = start
		body["end_image_url"] = end
	}
	return nil
}

func video25URLs(body map[string]interface{}, keys ...string) ([]string, error) {
	for _, key := range keys {
		raw, ok := body[key]
		if !ok || raw == nil {
			continue
		}
		var entries []any
		switch values := raw.(type) {
		case []any:
			entries = values
		case []string:
			for _, value := range values {
				entries = append(entries, value)
			}
		default:
			entries = []any{raw}
		}
		if len(entries) == 0 {
			continue
		}
		urls := make([]string, 0, len(entries))
		for index, entry := range entries {
			value := entry
			if object, ok := entry.(map[string]any); ok {
				value = object["url"]
			}
			candidate, ok := value.(string)
			if !ok || !video25ValidURL(candidate) {
				return nil, fmt.Errorf("%s[%d] must be an HTTP(S) or Data URL", key, index)
			}
			urls = append(urls, strings.TrimSpace(candidate))
		}
		return urls, nil
	}
	return nil, nil
}

func video25FrameURL(body map[string]interface{}, canonical, legacy string) (string, error) {
	for _, key := range []string{canonical, legacy} {
		value := body[key]
		if value == nil {
			continue
		}
		if key == canonical && stringifyBodyValue(value) == "" {
			continue
		}
		if key == legacy {
			entries := normalizeSeedanceReferenceFrameEntries(value)
			if len(entries) != 1 {
				return "", fmt.Errorf("%s requires exactly one image", key)
			}
			value = entries[0]["url"]
		}
		candidate, ok := value.(string)
		if !ok || !video25ValidURL(candidate) {
			return "", fmt.Errorf("%s must be an HTTP(S) or Data URL", key)
		}
		return strings.TrimSpace(candidate), nil
	}
	return "", nil
}

func video25ValidURL(value string) bool {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "data:") {
		return strings.Contains(value, ";base64,")
	}
	parsed, err := url.Parse(value)
	return err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != ""
}
