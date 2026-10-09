import assert from 'node:assert/strict';
import test from 'node:test';
import { buildCreativeUpstreamPayload } from './creativeRequestPayload.js';

test('Creative Center keeps local fields out of all submission payloads', () => {
  for (const payload of [
    { model: 'gpt-4o', group: 'vip', messages: [{ role: 'user', content: 'hi' }] },
    { model: 'gpt-image-2', group: 'vip', request_id: 'image-1', user: 'creative-1', prompt: 'cat' },
    { model: 'video-2.5', group: 'vip', request_id: 'video-1', user: 'creative-2', prompt: 'dog' },
  ]) {
    const result = buildCreativeUpstreamPayload(payload);
    assert.equal(Object.hasOwn(result, 'group'), false);
    assert.equal(Object.hasOwn(result, 'request_id'), false);
    assert.equal(Object.hasOwn(result, 'user'), false);
    assert.equal(result.model, payload.model);
    assert.equal(payload.group, 'vip');
  }
});
