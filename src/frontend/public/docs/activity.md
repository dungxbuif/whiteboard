# Activity API

Login-authenticated members can list authorized activity:

- `GET /api/v1/workspaces/{workspace}/activity?limit=50&cursor=...`
- `GET /api/v1/scenes/{scene}/activity?limit=50&cursor=...` with
  `X-Workspace-ID`.

The response is `{items,nextCursor?,retainedSince?}`. `retainedSince` is absent
when the authorized feed is empty. Treat `nextCursor` as an opaque stable token
and pass it back unchanged. Limits are 1–50. Items contain
the event ID, workspace/optional scene ID, server-resolved actor kind/name,
bounded event kind, subject ID, timestamp and allowlisted safe metadata.

Authorization, scene visibility, owner-only security/admin filtering and the
workspace activity feature flag are evaluated before pagination. A revoked or
deleted subject is absent rather than counted and hidden afterward. Public
scene-session and agent credentials are denied. This browser endpoint does not
change MCP `get_scene_activity`.

