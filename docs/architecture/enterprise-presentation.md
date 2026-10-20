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
the token grant version against the host's expected revision before moving the
connection to a presentation-specific view-only sync path. The audience bearer
is delivered only to that exact pending socket in `presentation_admitted`; it
is never echoed to the host or another waiting connection. Initial sync,
channel-1 state-vector resync, durable refresh, broadcast fanout and asset
download all use presentation read authority. Each check resolves the active
in-memory audience and then rechecks the host's current scene edit authority and
`presentation_enabled` flag in Go. No audience path exchanges the credential
for ordinary room authority. Admitted viewers may download scene-scoped assets
while this chain remains valid, but cannot upload assets.

Active sessions are intentionally ephemeral. They are serialized by the single
collaboration runtime owner, are lost on restart and end immediately when the
host disconnects. A feature-policy invalidation ends affected sessions and
revokes their waiting/audience runtime state before further sync, fanout or
asset access. Stored invite revocation does not change membership, public links
or other credentials. The private `/internal/presentations/resolve` endpoint is
service-authenticated and must never be proxied publicly.

Join throttling uses the direct peer address captured before proxy header
rewrites. Public peers cannot evade the 60/minute/IP budget with forged
`X-Forwarded-For` or `X-Real-IP` values; trusted proxy CIDRs remain the only
source of forwarded client identity.

Host and one-time invite credentials are omitted from feature-mutation receipts.
Only invite hashes are durable; replaying a link-creation request ID returns
`REQUEST_ID_REUSED` because the raw token cannot be reconstructed safely.

