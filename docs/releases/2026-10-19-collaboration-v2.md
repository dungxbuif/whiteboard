# Whiteboard collaboration v2 — 2026-10-19

Status: release candidate accepted locally; VM100 deployment evidence pending.

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

- Frontend: 25 files / 93 unit tests passed; production build passed.
- Collaboration/contracts/diagram/MCP workspace: 15 files / 149 tests passed.
- Backend: all Go packages passed.
- Real stack: 17/17 Playwright cases passed against actual Go, Node runtime, MCP, PostgreSQL, SeaweedFS S3 and browsers.
- Load: 50 principals,650/650 durable ACKs, p509.742ms, p9513.236ms, exact final convergence, peak runtime RSS399,097,856bytes on Apple M4 Pro ARM64.
- Presence/media: native follow/camera and30 cursor moves passed; generated audio was received in both browsers; mute/leave and permission-denial cleanup passed.

Local ARM64 acceptance does not replace VM100 Linux/amd64 and public-origin evidence. The release checklist records the remaining image, backup, migration, public browser and operational checks.

## Documentation

Human guides cover collaboration, public links, recovery and agents. Stable agent-readable surfaces are published under `/docs/*.md` and `/llms.txt`; API, protocol, environment and deployment contracts remain in the repository documentation.
