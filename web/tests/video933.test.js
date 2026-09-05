import { describe, expect, test } from 'bun:test';
import { readFileSync } from 'node:fs';
import {
  VIDEO_933_MODELS,
  VIDEO_933_DURATIONS,
  is933VideoModel,
  video933Resolution,
  validate933References,
  build933VideoParameters,
} from '../src/constants/video933';

describe('933 video capabilities', () => {
  test('exactly four models, fixed resolution, 4–15 seconds', () => {
    expect(VIDEO_933_MODELS.size).toBe(4);
    expect(is933VideoModel('933-video2.0-fast')).toBe(false);
    expect(VIDEO_933_DURATIONS.map((v) => v.value)).toEqual(
      Array.from({ length: 12 }, (_, index) => String(index + 4)),
    );
    for (const name of VIDEO_933_MODELS) {
      expect(video933Resolution(name)).toBe(
        name.endsWith('-480p') ? '480p' : '720p',
      );
    }
  });
  test('per-item and aggregate limits are inclusive', () => {
    expect(() =>
      validate933References([
        { duration: 2 },
        { duration: 6 },
        { duration: 7 },
      ]),
    ).not.toThrow();
    expect(() => validate933References([{ duration: 15 }])).not.toThrow();
    for (const durations of [
      [1.99],
      [15.01],
      [NaN],
      [Infinity],
      [8, 8],
      [2, 2, 2, 2],
    ]) {
      expect(() =>
        validate933References(durations.map((duration) => ({ duration }))),
      ).toThrow();
    }
  });
  test('payload uses fa2api parameters for every model', () => {
    for (const model of VIDEO_933_MODELS) {
      for (const videoDuration of [
        ...Array.from({ length: 12 }, (_, index) => String(index + 4)),
        undefined,
      ]) {
        const params = build933VideoParameters({
          model,
          videoDuration,
          aspectRatio: '21:9',
        });
        expect(params).toEqual({
          duration: videoDuration === undefined ? 5 : Number(videoDuration),
          aspect_ratio: '21:9',
          resolution: video933Resolution(model),
        });
      }
    }
  });
  test('invalid generation durations fall back to the 5-second default', () => {
    for (const model of VIDEO_933_MODELS) {
      for (const videoDuration of [0, 3, 16, 4.5, NaN, Infinity, '', null]) {
        expect(build933VideoParameters({ model, videoDuration }).duration).toBe(
          5,
        );
      }
    }
  });
  test('all edited UI modules parse as JSX', () => {
    const transpiler = new Bun.Transpiler({ loader: 'jsx' });
    for (const file of [
      'src/pages/CreativeCenter/index.jsx',
      'src/components/playground/SettingsPanel.jsx',
      'src/helpers/api.js',
      'src/hooks/playground/useApiRequest.jsx',
      'src/pages/Setting/Ratio/hooks/useModelPricingEditorState.js',
    ]) {
      expect(() =>
        transpiler.transformSync(
          readFileSync(new URL('../' + file, import.meta.url), 'utf8'),
        ),
      ).not.toThrow();
    }
  });
  test('creative center keeps audio inside multimodal shared upload', () => {
    const source = readFileSync(
      new URL('../src/pages/CreativeCenter/index.jsx', import.meta.url),
      'utf8',
    );
    expect(source).not.toContain("value: 'audio_reference'");
    expect(source).toContain("params.referenceMode === 'multimodal'");
    expect(source).toContain("? 'audio/*' : ''");
    expect(source).toContain('CREATIVE_CENTER_AUDIO_UPLOAD_MAX_BYTES = 15 * 1024 * 1024');
    expect(source).toContain("return { maxCount: 5, minDuration: 1, maxDuration: 15, maxTotalDuration: 15 }");
    expect(source).toContain("return { maxCount: 3, minDuration: 1, maxDuration: 15, maxTotalDuration: 15 }");
    expect(source).toContain("return { maxCount: 3, minDuration: 2, maxDuration: 15, maxTotalDuration: 15 }");
  });
});
