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

Create and revoke require the caller to be the current runtime host. Both calls
bind the host's current presentation revision before the idempotent mutation and
again immediately before storage; host transfer or revision drift rejects the
operation.
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

## Existing-socket control protocol

The runtime emits strict `presentation_state` payloads:

`{sessionId,revision,hostConnectionId,hostPrincipalId,frameId,spotlight,locked,reactions,pending,admitted}`

Only the current host receives `pending`; all other admitted recipients receive
an empty pending list. The following controls use the same authenticated
`/collab/` socket. Every host mutation includes `sessionId` and decimal-string
`expectedRevision` and is rejected with `PRESENTATION_CONFLICT` when stale:

- `presentation_set_frame {frameId}` uses a stable native Excalidraw frame ID or
  `null`.
- `presentation_spotlight {connectionId}` accepts an admitted participant, the
  host, or `null`.
- `presentation_lock {locked}` changes audience viewport following only.
- `presentation_remove {connectionId}` denies a pending browser or removes an
  admitted one.
- `presentation_transfer {connectionId}` accepts only a currently authorized,
  connected signed-in editor.
- `presentation_leave` always remains available, including before admission and
  while view lock is active.

`presentation_react {sessionId,emoji}` is ephemeral and does not use a revision.
The allowlist is 👏, ❤️, 👍, 🎉 and 😂. The server permits one reaction per
connection per second and assigns a three-second expiry. Clients must discard
expired reactions using the server timestamp. The runtime bounds both admitted
audience state and the live reaction list at 100 entries so emitted state always
fits the strict client schema.

Host disconnect ends the in-memory session. Participant leave/removal clears a
matching spotlight. Concurrent host/frame operations have one authoritative
runtime revision. A removed `frameId` is rendered as a missing-frame state; a
client must not substitute another frame by array position.
Clients ignore lower revisions and all late state for a session already marked
terminal. Waiting-room Leave removes the guest from the host queue immediately.
Before a host receives a pending queue, the runtime freshly checks that host's
room authority, `presentation:host` feature, exact connection and session
revision. Failure ends the session without disclosing the pending identity.

Host start waits ten seconds for authoritative `presentation_state`. On timeout,
the client closes the initiating collaboration socket so the server detach path
ends any accepted session; retry is allowed only after transport reconnection.

The browser route is `/present/{token}`. It shows a display-name form, waiting,
admitted canvas, denied/removed/ended, and rejoin states. Pending UI mounts no
canvas and receives no scene, roster, voice, screen or asset data. Audience
credential loss returns to a fresh waiting-room join. The UI never starts voice
or screen capture automatically and relies on their existing browser permission
and autoplay prompts.
Rejected or ended `presentation_auth` also leaves Joining immediately and shows
the ended/error state with a fresh-rejoin action.

