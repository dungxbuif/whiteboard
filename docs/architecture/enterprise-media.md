# Enterprise screen media

Status: M1 implemented; production rollout remains a separate release step.

Screen sharing uses the admitted `/collab/` control channel and remains separate
from microphone voice. A room admits at most ten distinct media connections
across voice and screen use, counting a connection once when it uses both. One
authenticated editor connection may reserve the screen publisher role for 30
seconds before capture starts. Current scene authority is checked again before
admission and before every peer signal.

Publishing also requires the workspace `screen:publish` feature decision. The
collaboration runtime asks the private Go authority with the current publisher
room token before publisher reservation, post-capture activation, any receiver
join, either direction of signaling, and either side stopping. Go verifies the
room identity and current FeatureService policy; screen
sharing is disabled by default. A denial or unavailable decision fails closed,
ends an existing screen session before any signal can be routed, and leaves the
drawing connection available.

The private check is `POST /internal/authority/features`, authenticated with the
existing collab service bearer secret. Its strict request is `{ roomToken,
workspaceId, sceneId, action: "screen:publish" }`. Success returns `204` with
`Cache-Control: no-store`; an invalid service credential or stale room token
returns `401`, a mismatched room/scene identity returns `403`, disabled policy returns
`403 FEATURE_DISABLED`, and an unavailable feature service returns `503`. The
endpoint returns no workspace policy metadata.

The server owns connection and screen-session identifiers. An empty
`screen_request` reserves or joins the current screen session; after successful
capture, the publisher confirms the same request with its server-issued
`sessionId`, which clears the 30-second reservation. `screen_admission` returns only
bounded routing metadata, `screen_signal` relays bounded SDP/ICE to an admitted
same-room peer, and `screen_stop` releases a receiver or the whole publisher
session. Screen errors do not close the drawing transport.

Browser capture has two explicit steps: request admission, then select **Start
sharing**. The second click calls `getDisplayMedia` with video limited to
1280×720 at 15 fps and `audio:false`. Cancelled, late, ended, disconnected, or
revoked captures stop every track and close all screen peer connections without
changing voice state.

Automated verification covers the voice/screen union limit, the eleventh connection
continuing to draw, one publisher, reservation expiry, cross-room signal denial,
default-disabled reservation denial, enabled-to-disabled publisher revocation,
late capture cleanup and existing voice behavior. The packaging proof generates
video frames locally and over a relay-only two-network peer path; the latter
requires the documented isolated gateway browser before release evidence can be
recorded.
