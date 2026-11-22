// The task resolver reads the database; only a terminal record can replace
// the live status request. Active records are useful for recovering the ID.
export function planCreativeVideoTaskRefresh(queryTaskId, cachedTaskState) {
  const taskId = queryTaskId || cachedTaskState?.taskId || '';
  const terminalState =
    cachedTaskState?.status === 'completed' || cachedTaskState?.status === 'failed'
      ? cachedTaskState
      : null;

  return {
    queryTaskId: taskId,
    terminalState,
    shouldRefresh: Boolean(taskId) && !terminalState,
  };
}
