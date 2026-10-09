// These fields are only used for Creative Center routing and local task tracking.
// Never include them in a provider-facing request body.
export const buildCreativeUpstreamPayload = (payload) => {
  const upstreamPayload = { ...payload };
  delete upstreamPayload.group;
  delete upstreamPayload.request_id;
  delete upstreamPayload.user;
  return upstreamPayload;
};
