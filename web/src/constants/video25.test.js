import assert from 'node:assert/strict';
import test from 'node:test';
import {
  VIDEO_25_MODELS,
  getVideo25ReferenceLimits,
  video25Resolution,
  video25SupportsFirstLast,
} from './video25.js';

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

test('A-Flash reference modes apply the new limits only to their allowed media', () => {
  for (const model of ['A-video2.5-flash', 'A-video2.5-flash-480p']) {
    assert.deepEqual(getVideo25ReferenceLimits(model, 'multi_image'), {
      images: 10, videos: null, audios: null,
    });
    assert.deepEqual(getVideo25ReferenceLimits(model, 'video_reference'), {
      images: null, videos: 5, audios: null,
    });
    assert.deepEqual(getVideo25ReferenceLimits(model, 'multimodal'), {
      images: 10, videos: 5, audios: 5,
    });
    assert.deepEqual(getVideo25ReferenceLimits(model, 'first_last'), {
      images: null, videos: null, audios: null,
    });
  }
});

test('other Video 2.5 Creative Center models keep their original reference limits', () => {
  for (const model of ['video2.5-flash', 'video2.5-flash-480p', 'A-video2.5', 'video-2.5']) {
    assert.deepEqual(getVideo25ReferenceLimits(model, 'multimodal'), {
      images: 30, videos: 10, audios: 10,
    });
  }
  assert.equal(getVideo25ReferenceLimits('video-2.5', 'first_last').images, 2);
});
