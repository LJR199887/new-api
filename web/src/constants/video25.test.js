import assert from 'node:assert/strict';
import test from 'node:test';
import { VIDEO_25_MODELS, video25Resolution, video25SupportsFirstLast } from './video25.js';

test('new video2.5 models use suffix resolution and do not support first/last frames', () => {
  const variants = {
    'video2.5': '720p',
    'video2.5-480p': '480p',
    'video2.5-1080p': '1080p',
    'video2.5-flash': '720p',
    'video2.5-flash-480p': '480p',
  };
  for (const [model, resolution] of Object.entries(variants)) {
    for (const name of [model, `A-${model}`]) {
      assert.ok(VIDEO_25_MODELS.has(name), name);
      assert.equal(video25Resolution(name), resolution, name);
      assert.equal(video25SupportsFirstLast(name), false, name);
    }
  }

  for (const name of ['video-2.5', 'video-2.5-480p', 'video-2.5-1080p']) {
    assert.ok(VIDEO_25_MODELS.has(name), name);
    assert.equal(video25SupportsFirstLast(name), true, name);
  }
  assert.equal(VIDEO_25_MODELS.size, 13);
});
