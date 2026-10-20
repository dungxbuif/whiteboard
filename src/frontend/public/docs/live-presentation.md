# Live presentation API

Presentation routes use login authentication except the public join endpoint.
Send `X-Workspace-ID` on scene-scoped host calls.

- `POST /api/v1/scenes/{sceneId}/presentations` with `{requestId}` creates an
  ephemeral host session and short-lived host credential.
- `POST /api/v1/scenes/{sceneId}/presentations/{sessionId}/links` with
  `{requestId}` creates a 24-hour, one-time-shown raw invite token. Replaying
  that request ID returns `REQUEST_ID_REUSED`; the service stores no raw token.
- `DELETE /api/v1/scenes/{sceneId}/presentations/{sessionId}/links/{linkId}`
  uses strict JSON `{requestId}` and revokes only that invite.
- `POST /api/v1/presentations/{token}/join` with `{displayName}` returns a
  five-minute pending credential.
- `POST /api/v1/scenes/{sceneId}/presentations/{sessionId}/admit` with
  `{connectionId,expectedRevision}` returns an audience credential after the
  private runtime confirms that exact waiting connection.

Send the pending token as `presentation_auth` on the existing `/collab/`
socket. Pending connections receive `presentation_waiting` only. They cannot
sync, send binary frames, obtain roster/media state or use asset routes.
Presentation controls carry server-derived connection IDs and session revision;
do not infer authority from their UI state.

Successful admission sends `presentation_admitted` only to the selected
pending socket. Its strict payload is `{sessionId,token,expiresAt}`. Use that
audience token for subsequent refresh, state-vector resync and scoped asset
requests. The host and other waiting connections never receive it. The token's
grant version must match the admitted expected revision.

An admitted audience credential can fetch scoped scene-asset downloads only
while the in-memory presentation session still contains that principal. It is
never accepted as an ordinary room or scene-session credential and cannot mint
upload capability. Every sync, fanout and asset request also rechecks the
host's current scene authority and `presentation_enabled` setting. Disabling
the feature ends the active presentation and invalidates audience access.

