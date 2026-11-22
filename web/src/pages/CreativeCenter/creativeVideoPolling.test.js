import assert from 'node:assert/strict';
import test from 'node:test';
import { planCreativeVideoTaskRefresh } from './creativeVideoPolling.js';

test('an exact queued or generating record does not short-circuit live polling', () => {
  for (const status of ['submitted', 'queued', 'generating']) {
    assert.deepEqual(
      planCreativeVideoTaskRefresh('task_local', {
        taskId: 'task_local',
        status,
        progress: 5,
      }),
      { queryTaskId: 'task_local', terminalState: null, shouldRefresh: true },
    );
  }
});

test('recover the local task ID from a nonterminal lookup and poll it', () => {
  const plan = planCreativeVideoTaskRefresh('', {
    taskId: 'task_recovered',
    status: 'queued',
    progress: 5,
  });
  assert.equal(plan.queryTaskId, 'task_recovered');
  assert.equal(plan.shouldRefresh, true);
  assert.equal(plan.terminalState, null);
});

test('terminal results from the database are reused without another query', () => {
  for (const status of ['completed', 'failed']) {
    const cached = { taskId: 'task_local', status };
    assert.deepEqual(planCreativeVideoTaskRefresh('', cached), {
      queryTaskId: 'task_local',
      terminalState: cached,
      shouldRefresh: false,
    });
  }
});

test('a status request requires an ID, not a successful database lookup', () => {
  assert.equal(planCreativeVideoTaskRefresh('task_local', null).shouldRefresh, true);
  assert.equal(planCreativeVideoTaskRefresh('', null).shouldRefresh, false);
});
