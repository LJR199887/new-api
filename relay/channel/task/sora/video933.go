package sora

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"
)

const video933ImageMaxBytes = 20 << 20

// Reuse the pure-Go media parsers. Never trust client-supplied duration metadata.
// Spool one bounded file at a time rather than holding up to six media files in RAM.
func probe933Media(ctx context.Context, source string, maxBytes int64) (float64, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	resp, err := service.DoDownloadRequestContext(ctx, source)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("reference media HTTP %d", resp.StatusCode)
	}
	if resp.ContentLength > maxBytes {
		return 0, fmt.Errorf("reference media exceeds %dMB", maxBytes>>20)
	}
	f, err := os.CreateTemp("", "new-api-933-media-*")
	if err != nil {
		return 0, err
	}
	defer os.Remove(f.Name())
	defer f.Close()
	n, err := io.Copy(f, io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return 0, err
	}
	if n > maxBytes {
		return 0, fmt.Errorf("reference media exceeds %dMB", maxBytes>>20)
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return 0, err
	}
	u, _ := url.Parse(source)
	ext := strings.ToLower(path.Ext(u.Path))
	switch strings.Split(resp.Header.Get("Content-Type"), ";")[0] {
	case "video/mp4", "audio/mp4", "video/quicktime":
		ext = ".mp4"
	case "audio/mpeg":
		ext = ".mp3"
	case "audio/wav", "audio/x-wav", "audio/wave":
		ext = ".wav"
	case "audio/flac", "audio/x-flac":
		ext = ".flac"
	case "video/webm", "audio/webm":
		ext = ".webm"
	case "audio/ogg", "application/ogg":
		ext = ".ogg"
	}
	if ext == ".mov" {
		ext = ".mp4"
	}
	return common.GetAudioDuration(ctx, f, ext)
}

func check933MediaDurations(ctx context.Context, body map[string]any) error {
	for _, kind := range []string{"video", "audio"} {
		maxBytes := int64(200 << 20)
		if kind == "audio" {
			maxBytes = 15 << 20
		}
		urls, _ := body[kind+"_urls"].([]string)
		refs := make([]map[string]any, 0, len(urls))
		for _, source := range urls {
			duration, err := probe933Media(ctx, source, maxBytes)
			if err != nil {
				return fmt.Errorf("probe %s reference: %w", kind, err)
			}
			refs = append(refs, map[string]any{"duration": duration})
		}
		if err := validateReferenceDurations(stringifyBodyValue(body["model"]), kind, refs, 2, 15, 15); err != nil {
			return err
		}
	}
	return nil
}

func video933URL(value any, image bool) (string, error) {
	s, ok := value.(string)
	if !ok || strings.TrimSpace(s) == "" {
		return "", fmt.Errorf("reference must be a non-empty URL string")
	}
	s = strings.TrimSpace(s)
	if image && strings.HasPrefix(s, "data:image/") {
		header, data, found := strings.Cut(s, ",")
		if !found || !strings.HasSuffix(header, ";base64") {
			return "", fmt.Errorf("image must be a base64 Data URL")
		}
		if err := check933ImageSize(base64.NewDecoder(base64.StdEncoding, strings.NewReader(data))); err != nil {
			return "", err
		}
		return s, nil
	}
	u, err := url.Parse(s)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Hostname() == "" || u.User != nil {
		return "", fmt.Errorf("reference must be an HTTP(S) URL%s", map[bool]string{true: " or image Data URL"}[image])
	}
	return s, nil
}

func check933ImageSize(r io.Reader) error {
	n, err := io.Copy(io.Discard, io.LimitReader(r, video933ImageMaxBytes+1))
	if err != nil {
		return fmt.Errorf("read reference image: %w", err)
	}
	if n > video933ImageMaxBytes {
		return fmt.Errorf("image too large, max 20MB")
	}
	if n == 0 {
		return fmt.Errorf("reference image is empty")
	}
	return nil
}

// Validate downloaded bytes, not Content-Length or the encoded string length.
// The existing download service applies the administrator's SSRF/proxy policy.
func check933RemoteImages(ctx context.Context, body map[string]any) error {
	images, _ := body["image_urls"].([]string)
	for _, key := range []string{"start_image_url", "end_image_url"} {
		if s, ok := body[key].(string); ok {
			images = append(images, s)
		}
	}
	for _, s := range images {
		if strings.HasPrefix(s, "data:") {
			continue
		}
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		resp, err := service.DoDownloadRequestContext(ctx, s)
		if err != nil {
			cancel()
			return fmt.Errorf("download reference image: %w", err)
		}
		err = func() error {
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("reference image HTTP %d", resp.StatusCode)
			}
			if resp.ContentLength > video933ImageMaxBytes {
				return fmt.Errorf("image too large, max 20MB")
			}
			return check933ImageSize(resp.Body)
		}()
		cancel()
		if err != nil {
			return err
		}
	}
	return nil
}

// Read one representation only; conflicting aliases must never bypass limits.
func video933References(body map[string]any, keys []string, image bool) ([]map[string]any, error) {
	var refs []map[string]any
	selected := ""
	for _, key := range keys {
		raw, exists := body[key]
		if !exists || raw == nil {
			continue
		}
		var values []any
		switch v := raw.(type) {
		case []any:
			values = v
		case []string:
			for _, s := range v {
				values = append(values, s)
			}
		case []map[string]any:
			for _, s := range v {
				values = append(values, s)
			}
		default:
			values = []any{v}
		}
		if len(values) == 0 {
			continue
		}
		if selected != "" {
			return nil, fmt.Errorf("use only one of %s", strings.Join(keys, ", "))
		}
		selected = key
		for _, value := range values {
			entry := map[string]any{}
			if obj, ok := value.(map[string]any); ok {
				for k, v := range obj {
					entry[k] = v
				}
				value = obj["url"]
			}
			s, err := video933URL(value, image)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", key, err)
			}
			entry["url"] = s
			refs = append(refs, entry)
		}
	}
	return refs, nil
}

// fa2api expects aspect_ratio/resolution and *_urls, not Adobe size/start_frame.
func normalize933VideoRequest(body map[string]any, modelName string) error {
	duration := stringifyBodyValue(body["duration"])
	seconds := stringifyBodyValue(body["seconds"])
	if duration != "" && seconds != "" && duration != seconds {
		return fmt.Errorf("duration and seconds disagree")
	}
	if duration == "" {
		duration = seconds
	}
	if duration == "" {
		duration = "5"
	}
	if err := validateGenerationDuration(modelName, duration, "", 4, 15); err != nil {
		return err
	}
	metadata, _ := body["metadata"].(map[string]any)
	resolution := "720p"
	if strings.HasSuffix(modelName, "-480p") {
		resolution = "480p"
	}
	for _, raw := range []any{body["resolution"], body["resolution_name"], metadata["resolution"]} {
		if s := stringifyBodyValue(raw); s != "" && s != resolution {
			return fmt.Errorf("unsupported resolution: %s requires %s", modelName, resolution)
		}
	}
	ratio := stringifyBodyValue(body["aspect_ratio"])
	if ratio == "" {
		ratio = stringifyBodyValue(metadata["ratio"])
	}
	if ratio == "" {
		ratio = seedanceAspectRatioFromSize(stringifyBodyValue(body["size"]))
	}
	if ratio == "" {
		ratio = "16:9"
	}
	switch ratio {
	case "16:9", "9:16", "4:3", "3:4", "1:1", "21:9":
	default:
		return fmt.Errorf("unsupported aspect_ratio for %s", modelName)
	}
	images, err := video933References(body, []string{"image_urls", "image_url", "images", "image", "input_reference", "reference_image", "reference_images", "input_image", "input_images"}, true)
	if err != nil {
		return err
	}
	if len(images) > 9 {
		return fmt.Errorf("%s supports at most 9 image references", modelName)
	}
	start, err := video933References(body, []string{"start_image_url", "start_frame"}, true)
	if err != nil {
		return err
	}
	end, err := video933References(body, []string{"end_image_url", "end_frame"}, true)
	if err != nil {
		return err
	}
	if len(start) > 1 || len(end) > 1 || len(start) != len(end) {
		return fmt.Errorf("frame mode requires exactly one start and one end image")
	}
	for _, kind := range []string{"video", "audio"} {
		keys := []string{kind + "_reference", kind + "_urls", kind + "_url", "reference_" + kind + "_urls", "reference_" + kind + "_url"}
		refs, err := video933References(body, keys, false)
		if err != nil {
			return err
		}
		if len(refs) > 3 {
			return fmt.Errorf("%s supports at most 3 %s references", modelName, kind)
		}
		// Validate supplied hints early; check933MediaDurations probes actual bytes.
		if err := validateReferenceDurations(modelName, kind, refs, 2, 15, 15); err != nil {
			return err
		}
		if len(start) > 0 && len(refs) > 0 {
			return fmt.Errorf("frame mode cannot be combined with references")
		}
		for _, key := range keys {
			delete(body, key)
		}
		if len(refs) > 0 {
			urls := make([]string, 0, len(refs))
			for _, ref := range refs {
				urls = append(urls, ref["url"].(string))
			}
			body[kind+"_urls"] = urls
		}
	}
	if len(start) > 0 && len(images) > 0 {
		return fmt.Errorf("frame mode cannot be combined with references")
	}
	if mode := stringifyBodyValue(body["reference_mode"]); (mode == "first_last" || mode == "i2v_start_end") && len(start) != 1 {
		return fmt.Errorf("frame mode requires both start_image_url and end_image_url")
	}
	for _, key := range []string{"image_guidance", "guidances", "reference_assets", "messages", "extra_body"} {
		if body[key] != nil {
			return fmt.Errorf("%s is unsupported; use image_urls/video_urls/audio_urls", key)
		}
	}
	for _, key := range []string{"count", "n"} {
		if value, ok := body[key]; ok && stringifyBodyValue(value) != "1" {
			return fmt.Errorf("%s must be 1", key)
		}
	}
	for _, key := range []string{"seconds", "metadata", "size", "quality", "resolution_name", "image", "images", "image_url", "input_reference", "start_frame", "end_frame", "reference_mode", "stream", "reference_image", "reference_images", "input_image", "input_images"} {
		delete(body, key)
	}
	if len(images) > 0 {
		urls := make([]string, 0, len(images))
		for _, ref := range images {
			urls = append(urls, ref["url"].(string))
		}
		body["image_urls"] = urls
	}
	if len(start) == 1 {
		body["start_image_url"], body["end_image_url"] = start[0]["url"], end[0]["url"]
	}
	body["model"], body["duration"], body["resolution"], body["aspect_ratio"] = modelName, soraDurationBodyValue(duration), resolution, ratio
	return nil
}
