---
artifact_type: integrations_doc
id: INTEGRATIONS
status: active
owner: shared
human_fields: [provider_choice, credential_approval, business_owner]
ai_fields: [systems, contracts, secrets_names, failure_modes]
shared_fields: [status, trace]
---

# Integrations

## Field Ownership

- Human owns provider choices, credential approval, and business owner.
- AI documents systems, contracts, secret names, and failure modes.

## External Systems

| System | Purpose | Contract | Owner | Status |
| --- | --- | --- | --- | --- |
| PostgreSQL 16 | Users, workspaces, ACL, scenes, durable Yjs state, receipts and audits | SQL migrations 1–13 and repository transactions | Whiteboard operator | active; production at migration 13, clean |
| S3-compatible object storage | Immutable raster scene assets | AWS S3 signing/API; scoped capability endpoints | Whiteboard operator | active; production `whiteboard` bucket write/read/delete canary passed |
| Coturn | STUN/TURN relay for WebRTC voice | TURN REST credentials issued after current scene authorization | Homelab gateway operator | active release candidate; tested gateway IP required until split DNS is fixed |
| OpenAI-compatible provider | Chat, diagram generation, image reasoning and diagram-to-code | Backend-only OpenAI-compatible HTTP API | Product owner | implementation active; live production UAT blocked on key |
| Chromium renderer | Isolated PNG/SVG export for MCP | Local Playwright CDP process with network denial and bounded lifecycle | Whiteboard MCP service | deployed in the frozen Linux/amd64 MCP image; public authenticated read canary passed, write/export remains |

## Credentials And Secrets

Document required secret names only. Do not store secret values.

- `DATABASE_URL`
- `JWT_SECRET`
- `COLLAB_JWT_SECRET`
- `AUTHORITY_SERVICE_SECRET`
- `MUTATION_GRANT_SECRET`
- `COLLAB_INTERNAL_SECRET`
- `S3_ACCESS_KEY`, `S3_SECRET_KEY`
- `TURN_AUTH_SECRET`
- `AI_API_KEY` when live AI features are enabled
- `WHITEBOARD_AGENT_KEY` for remote MCP clients

## Failure Modes

- PostgreSQL unavailability makes authorization and durable writes fail closed;
  collaboration readiness drops rather than accepting stale authority.
- Object storage failure retains scene/file metadata and exposes retry/recovery;
  a failed PUT is never treated as a completed asset.
- TURN/DNS failure leaves document collaboration available while voice relay
  fails; use the tested gateway IP until hostname resolution is proven.
- Missing `AI_API_KEY` returns a provider-configuration error without exposing
  a key to the browser.
- MCP audit/journal or authority failure prevents mutations; uncertain
  completion requires receipt/current-state inspection before retry.
- Renderer timeout or crash returns a bounded render failure and cleans the
  owned process/temp directory.

## Collaboration v2 presence and voice

The custom Node WebSocket shares the admitted scene scope for Yjs, roster, cursor/selection, follow and WebRTC signaling. Go room credentials and current ACL remain authoritative; a server connection UUID identifies each tab independently of its principal. Voice has ten active connections per scene; rejected voice admission leaves document collaboration available. Camera/follow/voice state is ephemeral. Browser media uses API-issued short-lived Coturn credentials, never credentials embedded in the frontend. See the [B4 contract and verification record](../work/verification/collaboration-presence.md).

A 25-second server protocol heartbeat maintains the bounded nginx tunnel and removes nonresponsive connections by the following interval. Disconnect/revoke cleanup releases voice admission and clears followers. Browser integration is verified with native same-user tabs, follow/camera, permission denial, generated audio, mute and leave cleanup. The legacy Socket.IO client/server/proxy path is retired; `/collab/` is the sole collaboration socket. VM100 Docker DNS restart/replay and authenticated TURN relay allocation pass; public two-peer browser media remains in the final product matrix.

Ingress limits separate shared-IP transport resource protection from operations: 2,000 frames/second/IP, 64 MiB/minute byte refill with 8 MiB burst; authentication60/minute/IP; cursor/viewport share30/second/connection; other ephemeral controls120/minute/connection. Document writes retain120/minute/principal and1 MiB frames. Raw received bytes are charged before decoding; limiter entries expire and active entry counts are bounded. These limits accommodate normal cursor traffic from50 clients sharing one IP without consuming their authentication or durable-write budgets.
