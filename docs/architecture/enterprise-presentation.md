# Live presentation admission

Live presentation credentials are separate from login, public scene-session and
ordinary collaboration room credentials. A workspace editor starts an in-memory
session through a short-lived `presentation-host` token. Invite rows contain a
SHA-256 token hash, workspace/scene/session binding, revision, expiry and revoke
state; the raw token appears only in the create response.

Joining an invite produces a five-minute `presentation-pending` credential.
The collaboration runtime registers that connection in a session-local queue
of at most 100 participants and sends only its own waiting status. Pending
connections have no scene principal or room membership: binary sync/update,
presence, roster, assets, voice and screen controls fail before data access.

Admission is two-sided. Go asks the private collaboration listener to resolve
the active host, exact waiting connection and expected session revision, then
issues a ten-minute `presentation-audience` credential bound to the same
workspace, scene, session and pending principal. Node verifies that binding and
revision again before moving the connection to a presentation-specific
view-only sync path. Each durable refresh checks both the active in-memory
audience session and the captured workspace ACL version; it never exchanges the
audience token for an ordinary room credential. Admitted viewers may request
scene-scoped asset downloads while the audience session is active, but cannot
upload assets. Host presentation controls recheck the current Go feature and
scene authority through the private authority listener.

Active sessions are intentionally ephemeral. They are serialized by the single
collaboration runtime owner, are lost on restart and end immediately when the
host disconnects. Stored invite revocation does not change membership, public
links or other credentials. The private `/internal/presentations/resolve`
endpoint is service-authenticated and must never be proxied publicly.

Host and one-time invite credentials are omitted from feature-mutation receipts.
Only invite hashes are durable; replaying a link-creation request ID returns
`REQUEST_ID_REUSED` because the raw token cannot be reconstructed safely.

