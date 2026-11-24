const LEGACY_VIDEO_25_MODELS = new Set([
  'video-2.5',
  'video-2.5-480p',
  'video-2.5-1080p',
]);

export const VIDEO_25_MODELS = new Set([
  ...LEGACY_VIDEO_25_MODELS,
  'video2.5',
  'video2.5-480p',
  'video2.5-1080p',
  'video2.5-flash',
  'video2.5-flash-480p',
  'A-video2.5',
  'A-video2.5-480p',
  'A-video2.5-1080p',
  'A-video2.5-flash',
  'A-video2.5-flash-480p',
]);

export const video25SupportsFirstLast = (name) =>
  LEGACY_VIDEO_25_MODELS.has(name);

const A_FLASH_VIDEO_25_MODELS = new Set([
  'A-video2.5-flash',
  'A-video2.5-flash-480p',
]);

export function getVideo25ReferenceLimits(model, mode) {
  const isAFlash = A_FLASH_VIDEO_25_MODELS.has(model);
  const hasImages = mode === 'multi_image' || mode === 'multimodal';
  const hasVideos = mode === 'video_reference' || mode === 'multimodal';
  const hasFrames = mode === 'first_last' && video25SupportsFirstLast(model);
  return {
    images: hasFrames ? 2 : hasImages ? (isAFlash ? 10 : 30) : null,
    videos: hasVideos ? (isAFlash ? 5 : 10) : null,
    audios: mode === 'multimodal' ? (isAFlash ? 5 : 10) : null,
  };
}

export const video25Resolution = (name) =>
  name.endsWith('-480p') ? '480p' : name.endsWith('-1080p') ? '1080p' : '720p';
