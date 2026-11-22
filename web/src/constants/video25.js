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

export const video25Resolution = (name) =>
  name.endsWith('-480p') ? '480p' : name.endsWith('-1080p') ? '1080p' : '720p';
