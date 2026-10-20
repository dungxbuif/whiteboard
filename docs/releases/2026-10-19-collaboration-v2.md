# Whiteboard collaboration v2 — 2026-10-19

Status: deployed to VM100; controlled reboot, runtime restart/replay, TURN
allocation and core public infrastructure canaries pass. The remaining product
acceptance matrix is listed below.

## User-visible changes

- Workspace owners can invite members, set workspace and scene access, and create independent public view or edit links.
- Scenes use durable realtime collaboration with native Excalidraw cursors, participant presence, follow mode, offline recovery and field-preserving conflict handling.
- Voice controls use WebRTC with short-lived TURN credentials. Muting, leaving and microphone denial clean up without interrupting document editing.
- Workspace owners can create scoped agent keys. MCP agents can create scenes, generate any of 19 editable diagram presets, patch/delete elements, inspect activity and export current PNG, SVG or Excalidraw JSON.
- The dashboard keeps the original app styling; the scene keeps native Excalidraw styling. UUID and short alias deep links remain directly reloadable.

## Technical and operational changes

- `/collab/` is a dedicated authenticated Node WebSocket runtime backed by durable Yjs updates, permanent request receipts, PostgreSQL authority fencing and bounded snapshot/history handling.
- Legacy Socket.IO server, client dependencies and nginx routing are removed. RelayHub is not used for document transport.
- Go retains permission authority and issues purpose-separated room, public-session, mutation and asset capabilities. Four dedicated v2 service secrets must remain distinct from the login JWT secret.
- MCP uses the same durable writer as browsers. HTTP/stdio calls are scoped, audited and bounded; the public adapter has persistent journal storage.
- Renderer startup tolerates Chromium's two-step `DevToolsActivePort` write while preserving the existing wall deadline and owned-process cleanup.
- Migrations 7–13 ship together. A fresh verified production backup is required immediately before migration. Promoted scenes require the documented replacement-scene recovery path; image-only rollback is unsafe.

## Verification

- Frontend: 25 files / 94 unit tests passed; production build passed.
- Collaboration runtime: 15 files / 150 tests passed against the guarded loopback test database; contracts, diagram and MCP workspace tests also passed.
- Backend: all Go packages passed.
- Real stack: 17/17 Playwright cases passed against actual Go, Node runtime, MCP, PostgreSQL, SeaweedFS S3 and browsers.
- Load: 50 principals,650/650 durable ACKs, p509.742ms, p9513.236ms, exact final convergence, peak runtime RSS399,097,856bytes on Apple M4 Pro ARM64.
- Presence/media: native follow/camera and30 cursor moves passed; generated audio was received in both browsers; mute/leave and permission-denial cleanup passed.

Local ARM64 acceptance is complemented by the Linux/amd64 deployment evidence
below. The release checklist records the remaining public-origin and
operational checks.

## Production deployment

Production runs the collaboration-v2 stack from application commit `772d3ea`
at `https://whiteboard.dungxbuif.com`. `COLLAB_V2_ENABLED=true`; the Go backend,
custom collaboration runtime, MCP adapter and nginx frontend are healthy.
Migrations 7–13 completed and the database reports version 13 with
`dirty=false`.

The frozen Linux/amd64 images are:

- Backend: `sha256:e0c6c51e31624401102a6bac4a555d99977748b2f9c6a321df6156e18fb0e167`.
- Collaboration: `sha256:ef235e1c3e3aa293072785c161660f526bf112c4daaeb8fac507333f8c7f8f69`.
- MCP: `sha256:0cd512501273886674680b5e9c14d87ee0b59802bf0399541120b07b32bd03de`.
- Frontend: `sha256:a516c748733261cba9cedef0f39381ec4747fe9c517481c4bb1e01b0d7056c5e` (release source `24b1cd0`, including the scene palette and reconciled public docs).

Four independent collaboration service secrets, each at least 32 bytes and
distinct from the login JWT secret, are installed without recording their
values. The fresh cutover dump restored cleanly with all 14 table counts
matching before migration. The v1 rollback bundle and exact image IDs remain
on VM100.

Public-origin verification currently includes authenticated deep-link reload,
dashboard/settings/share/invitation/agent-key UI, an actual saved-and-reloaded
canvas edit with cleanup, the `whiteboard` S3 bucket write/read/delete canary,
public-view mutation rejection, public-edit add/delete acknowledgements at
sequences 1 and 2, owner-role preservation on self-invite and effective pending
invite revocation. Mobile navigation and overflow checks pass. Native MCP
initialized with 13 tools, generated and projected a nine-element diagram, and
downloaded JSON and SVG exports over public HTTPS. The temporary MCP key was
revoked and then rejected with HTTP 401. Anonymous MCP and workspace API
requests also return 401.

Public `/internal`, `/api/internal` and encoded/normalized variants return 404.
Nginx access logging is disabled; inspected local nginx/backend/collab/MCP and
recent Cloudflared logs contain no credential-bearing URL. Cloudflare dashboard
retention policy was not available from VM100. A scoped production room token
and realtime config return 200 with `Cache-Control: no-store`, the configured
`turn:103.82.21.202:5349` endpoint and short-lived credential fields. An
unauthenticated STUN Allocate receives the expected 401 challenge with the
configured realm/nonce. Secret values were not printed.

The initial production create exposed a Compose/YAML tmpfs parsing defect. The
manifest correction in `d885975` preserves each complete mount option; runtime
inspection confirms the expected `/tmp` sizes/modes on collaboration, MCP and
frontend containers. See [tmpfs evidence](../work/verification/compose-tmpfs-vm100-2026-10-19.md).

Commit `772d3ea` makes nginx re-resolve backend, collaboration and MCP service
names through Docker DNS. With no frontend restart, production moved the
collaboration service from `172.30.0.76` to `172.30.0.84`; public WebSocket
upgrade, authenticated admission/sync and the visible `People (1)` / `Saved`
state passed. See [DNS recovery evidence](../work/verification/nginx-docker-dns-recovery-2026-10-19.md).

Homelab commit `9722490` reconciles the VM100 template, adds a semantic Compose
regression test, and updates recovery tests, architecture, runbook, deployment
evidence and its infrastructure release note. The Homelab repository has no
configured remote, so this is a local commit.

The controlled VM reboot started at 16:29:38 local time. The installed recovery
unit waited for network metadata and restored all four healthy services without
manual intervention. `homelab-net` and `edge-http-net` returned as Swarm overlay
networks, `whiteboard-app` as a local bridge, migration 13 stayed clean, public
health returned successfully, `/internal` returned 404 and anonymous `/mcp`
returned 401.

A freshly issued production TURN credential completed authenticated UDP
Allocate and returned a relayed address; Refresh with lifetime zero released it.
No credential value is recorded.

The public browser media canary opened two scene peers over HTTPS, observed
`connected` WebRTC state, received approximately 60,037 bytes / 750 audio
packets on each peer and exchanged nine signaling frames. Both browsers ran on
the same host, so this canary selected host/host candidates; the independent
cross-network relay-only TURN proof remains in the packaging evidence. The
production scene canary then drew a rectangle, reached `Saved`, completed
semantic Undo and Redo with `Saved` acknowledgements, and undid the final edit
to leave the user's scene unchanged. Temporary scenes, links, users and agent
keys were cleaned up.

Release sign-off is complete for the shipped collaboration scope.
Live AI provider
UAT is externally blocked until the user supplies `AI_API_KEY`. Retired Google
OAuth credential revocation remains an external Google Cloud operation; the
values are absent from reachable Git history.

## Documentation

Human guides cover collaboration, public links, recovery and agents. Stable agent-readable surfaces are published under `/docs/*.md` and `/llms.txt`; API, protocol, environment and deployment contracts remain in the repository documentation.

Docs review: requirements/validation, architecture/API/ERD/integrations,
deployment/runbooks, context/backlog/roadmap/checklist/changelog and public
Markdown/`llms.txt` were reconciled to the deployed boundary. This release
does not add another API or schema beyond the already reviewed v2 contracts and
migrations, so it does not require a new ADR. Operational corrections and
their verification are linked from the release checklist.
