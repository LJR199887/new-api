import assert from 'node:assert/strict';
import test from 'node:test';
import {
  VIDEO_933_MODELS,
  VIDEO_933_DURATIONS,
  build933VideoParameters,
  is933VideoModel,
  video933SupportsFirstLast,
} from './video933.js';

test('video2.0 and A-video2.0 variants inherit the 933 options without first/last frames', () => {
  for (const prefix of ['', 'A-']) {
    for (const variant of ['video2.0', 'video2.0-fast', 'video2.0-mini']) {
      for (const [suffix, resolution] of [['', '720p'], ['-480p', '480p'], ['-1080p', '1080p']]) {
        const name = prefix + variant + suffix;
        assert.ok(is933VideoModel(name), name);
        assert.equal(video933SupportsFirstLast(name), false, name);
        assert.deepEqual(build933VideoParameters({ model: name, videoDuration: '15', aspectRatio: '21:9' }), {
          duration: 15,
          aspect_ratio: '21:9',
          resolution,
        }, name);
      }
    }
  }
  assert.equal(VIDEO_933_MODELS.size, 23);
  assert.equal(VIDEO_933_DURATIONS[0].value, '4');
  assert.equal(VIDEO_933_DURATIONS.at(-1).value, '15');
  assert.equal(video933SupportsFirstLast('933-video2.0'), true);
  assert.equal(is933VideoModel('video-2.0'), false);
  assert.equal(is933VideoModel('933-video2.0-fast'), false);
});
