export const VIDEO_933_MODELS = new Set([
  '933-video2.0',
  '933-video2.0-480p',
  '933-video2.0-mini',
  '933-video2.0-mini-480p',
]);

export const is933VideoModel = (name) => VIDEO_933_MODELS.has(name);
export const video933Resolution = (name) =>
  name.endsWith('-480p') ? '480p' : '720p';
export const VIDEO_933_DURATIONS = [4, 5].map((n) => ({
  label: `${n}s`,
  value: String(n),
}));

export function build933VideoParameters(inputs) {
  return {
    duration: [4, 5].includes(Number(inputs.videoDuration))
      ? Number(inputs.videoDuration)
      : 5,
    aspect_ratio: inputs.aspectRatio || '16:9',
    resolution: video933Resolution(inputs.model),
  };
}

export function validate933References(refs) {
  if (refs.length > 3) throw new Error('最多 3 个参考素材');
  let total = 0;
  for (const ref of refs) {
    if (
      !Number.isFinite(ref.duration) ||
      ref.duration < 2 ||
      ref.duration > 15
    ) {
      throw new Error('单个参考素材时长必须为 2–15 秒');
    }
    total += ref.duration;
  }
  if (total > 15) throw new Error('同类参考素材总时长不能超过 15 秒');
}

export function read933MediaDuration(source, kind) {
  return new Promise((resolve, reject) => {
    const media = document.createElement(kind);
    const objectURL =
      typeof source === 'string' ? null : URL.createObjectURL(source);
    const finish = (error) => {
      clearTimeout(timer);
      const duration = media.duration;
      media.onloadedmetadata = null;
      media.onerror = null;
      media.removeAttribute('src');
      media.load();
      if (objectURL) URL.revokeObjectURL(objectURL);
      if (error) reject(error);
      else resolve(duration);
    };
    const timer = setTimeout(
      () => finish(new Error('读取素材时长超时')),
      15000,
    );
    media.preload = 'metadata';
    media.onloadedmetadata = () => finish();
    media.onerror = () =>
      finish(new Error('无法读取素材时长，请检查链接或文件格式'));
    media.src = objectURL || source;
  });
}
