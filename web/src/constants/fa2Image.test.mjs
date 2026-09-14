import assert from 'node:assert/strict';
import test from 'node:test';
import {
  FA2_IMAGE_MODELS,
  getFa2ImageModelSpec,
  getFa2ImageResolutionOptions,
  getFa2ImageAspectRatioOptions,
  isFa2ImageModel,
} from './fa2Image.js';

for (const name of ['gpt-image-2.5-flare', 'gpt-image-2.5-sunburst']) {
  test(`${name} supports three resolution tiers and 16 references`, () => {
    assert.ok(FA2_IMAGE_MODELS.includes(name));
    assert.ok(isFa2ImageModel(name));
    const spec = getFa2ImageModelSpec(name);
    assert.equal(spec.defaultResolution, '2K');
    assert.equal(spec.maxImages, 16);
    assert.deepEqual(
      getFa2ImageResolutionOptions(name).map(({ value }) => value),
      ['1K', '2K', '4K'],
    );
    assert.equal(getFa2ImageAspectRatioOptions(name).length, 14);
    assert.deepEqual(
      spec.aspectRatios,
      getFa2ImageModelSpec('gpt-image-2').aspectRatios,
    );
    assert.ok(Object.isFrozen(spec));
  });
}

test('existing image models retain reference limits', () => {
  for (const [name, count] of Object.entries({
    'gpt-image-2': 17,
    'nano-banana-pro': 10,
    'nano-banana2': 14,
    'seedream-5-0': 14,
  })) {
    assert.equal(getFa2ImageModelSpec(name).maxImages, count);
  }
  assert.equal(isFa2ImageModel('gpt-image-2.5-sunburs'), false);
});
