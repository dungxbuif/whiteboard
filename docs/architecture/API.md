---
artifact_type: api_contract_index
id: API-MASTER
status: draft
owner: shared
human_fields: [contract_intent, approval]
ai_fields: [contract_rows, errors, auth_notes, versioning, linked_decisions]
shared_fields: [status, trace]
---

# API

## Field Ownership

- Human owns public contract intent and approval for contract changes.
- AI maintains contract rows, errors, auth notes, versioning, and linked decisions.

## API Surface

Document HTTP endpoints, RPC methods, events, CLI commands, or any other public contract.

| Contract | Type | Auth | Status | Notes |
| --- | --- | --- | --- | --- |
| `POST /api/v1/auth/login` | HTTP | Public | active | Accepts `{email,password}` and returns `{token}` for a password-authenticated user. |
| `GET, POST /api/v1/workspaces` | HTTP | JWT | active | List and create workspaces. |
| `GET, POST /api/v1/folders` | HTTP | JWT + workspace | active | List/create folders; `all=true` returns flat tree data. |
| `PATCH, DELETE /api/v1/folders/{id}` | HTTP | JWT + workspace | active | Folder move and trash lifecycle routes. |
| `GET, POST /api/v1/scenes` | HTTP | JWT + workspace | active | List/create scenes; `trash=true` lists deleted scenes. |
| `GET, PATCH, DELETE /api/v1/scenes/{id}` | HTTP | JWT + workspace | active | Read, update, and trash scene JSONB content. |
| `/w/{workspaceId}/scenes/{sceneId}` | Browser route | JWT + workspace ownership | active | Deep link to the authenticated Excalidraw editor; the frontend loads the scene through the workspace-scoped API. |
| `POST /api/v1/scenes/{id}/star` | HTTP | JWT + workspace | active | Explicitly set starred state; returns the updated scene. |
| `POST /api/v1/scenes/{id}/restore` | HTTP | JWT + workspace | active | Restore a trashed scene. |
| `POST /api/v1/folders/{id}/star` | HTTP | JWT + workspace | active | Toggle a folder's starred state. |
| `POST /api/v1/folders/{id}/restore` | HTTP | JWT + workspace | active | Restore a trashed folder tree. |
| `GET /api/v1/search` | HTTP | JWT + workspace | active | Search folder and scene names with `q`; `view=all|starred|trash` selects the dashboard view. |
| Scene `content.presentationOrder` | JSON field | JWT + workspace | active | Ordered Excalidraw frame IDs persisted with scene JSON for local presentation mode. |
| `PUT /api/v1/scenes/{id}/share` | HTTP | JWT + workspace | active | Enable/revoke a public link and optionally set a bcrypt-protected password. |
| `GET /api/v1/public/scenes/{token}` | HTTP | Public | active | Read a shared scene by opaque token; accepts `password` query for protected links and returns read-only scene JSON. The public viewer normalizes JSON-persisted Excalidraw collaborator maps before hydrating the canvas. |
| `POST /api/v1/ai/chat` | HTTP | JWT | active | Proxies text or image context to the configured OpenAI-compatible provider. |
| `POST /api/v1/ai/generate-diagram` | HTTP | JWT | active | Generates and validates Excalidraw JSON from text or a base64 image. |
| `POST /api/v1/ai/to-code` | HTTP | JWT | active | Converts a scene JSON payload into previewable React/HTML code. |
| `GET /api/v1/realtime/config` | HTTP | JWT | active | Returns the external TURN URL(s) and short-lived credentials; the backend signs credentials with `TURN_AUTH_SECRET`. |
| `/socket.io/` | Socket.IO | JWT query token | active | The handshake requires a valid JWT; each `scene:join` also checks that the token subject owns the requested workspace before broadcasting presence, cursor, scene update, or WebRTC signaling events. |

## Errors

| Error | Meaning | Consumer Impact |
| --- | --- | --- |
| `400 Bad Request` | Invalid UUID, body, title, content, or folder mapping | Request must be corrected. |
| `401 Unauthorized` | Missing/invalid JWT or invalid email/password | Consumer must authenticate again. |
| `404 Not Found` | Workspace-scoped entity does not exist | Consumer must discard stale entity reference. |
| `403 Forbidden` | Authenticated user does not own the selected workspace | Consumer must stop and must not retry with the same workspace. |
| `500 Internal Server Error` | Repository or infrastructure failure | Consumer may retry safely where appropriate. |

AI requests are limited to a 2 MiB JSON body and prompts are limited to 12,000 bytes. `AI_API_KEY` is read only by the backend.
TURN credentials are minted for a one-hour lifetime and must never be logged or
persisted by the browser. `TURN_AUTH_SECRET` is shared only between the backend
and the Coturn relay.

The frontend toolchain pins TypeScript 7.0.2. TypeScript 7 removed the legacy
`baseUrl` compiler option, so the app keeps its `@/*` path mapping without that
option and uses `globalThis` for browser test globals.

Workspace and scene primary keys remain UUIDs. A1 adds nullable unique public aliases for new resources and accepts either UUID or alias in the workspace header and scene URL. Existing UUID links remain valid. Alias generation ports the old server NanoID alphabet and default length; see [ADR 0009](../decisions/0009-collaboration-access.md). Aliases do not grant permission.

Password accounts are created or rotated with `go run ./cmd/userctl --email ...
--password ...` (or `/app/userctl` in the backend image). Passwords are stored as
bcrypt hashes and are never returned by the API. The command also creates a
`My Workspace` owned by the user when no non-deleted workspace exists.

The deployed v1 workspace routes use ownership. The A1 implementation replaces this with a valid JWT and current database membership for the workspace selected by `X-Workspace-ID`. Public share links use `/p/{token}` and intentionally skip
JWT access while enforcing the share permission and optional share password.

## Versioning

Phase 1 HTTP contracts are versioned under `/api/v1`.

## Linked Decisions

- TBD

## Collaboration A1 branch contract (implemented; release pending)

Approved programme A1 replaces owner-only tenancy with database-backed owner/editor/viewer membership. Editors write; viewers read; owner-only management cannot remove or demote any owner. Scene editor/viewer overrides require membership and affect content only. Workspace and scene UUIDs remain canonical; optional 21-character URL-safe aliases resolve before ACL checks and never grant access. Removed members and deleted resources are denied. Trash metadata/restore is owner-only. Folder mutations require edit access to every descendant scene. Workspace lists contain current memberships; search and scene lists use the same current policy.

Owner routes: GET `/workspaces/{id}/members`, PATCH/DELETE `/workspaces/{id}/members/{userId}`, PUT/DELETE `/scenes/{id}/members/{userId}`. PATCH/PUT accepts `{role:"editor"|"viewer"}`. All routes require login; scene routes retain the workspace header. New policy errors use `{code,message,requestId}`.

Verified on 2026-10-16: 79 backend tests pass with real disposable PostgreSQL enabled, including migration/rollback, aliases, access matrix, removed/soft-deleted members, search/list isolation, protected descendants, owner immutability, audit/version changes, and two-connection write/revoke serialization. Public-session and invitation behavior belongs to A2/A3.

A1 DTOs preserve existing fields and add `publicId?: string` to Workspace/Scene, and `aclVersion: number` to Workspace. Effective role/access is currently internal to `AccessService`; C2 must add an explicit response contract before hiding editor/owner actions. `ownerId` remains the legacy primary owner; membership is the authority. Member GET returns `[{workspaceId,userId,role,createdAt}]`; PATCH/DELETE/override writes return 204. Management body errors return 400; policy failures return the structured error above.

Trash listing is owner-only and scene content is omitted. Scene asset reads filter file IDs to the authorized scene; an existing workspace file ID cannot fetch another scene's asset. Old public share routes and the owner-only v1 Socket.IO adapter remain legacy until A3/B2; no new password query route was added.

The gateway resolves aliases before access checks, holds the workspace row lock through handler completion, and buffers HTTP success until database commit. Authorization failure or handler error rolls back the transaction. New member changes reject every owner target (including self-demotion/last-owner removal). No endpoint promotes another owner or transfers ownership.

A1 request-body protection: authenticated scoped requests are fully buffered before acquiring workspace authority. Scene bodies allow 26 MiB (25 MiB materialized-document allowance plus JSON/metadata envelope); other metadata bodies allow 1 MiB. Network body reading has a ten-second deadline, cleared after a complete read. Oversized bodies return structured `413 BODY_TOO_LARGE`; stalled bodies return `408 BODY_TIMEOUT`. Failure closes the request connection and never enters the authority transaction. Deadline support is required for requests with bodies; unsupported transports fail closed with 503. These limits apply only to scoped HTTP requests, not Socket.IO/WebSocket routes.
