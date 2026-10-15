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
| `GET /api/v1/public/scenes/{token}` | HTTP | Public | active | Read a shared scene by opaque token; accepts `password` query for protected links and returns read-only scene JSON. |
| `POST /api/v1/ai/chat` | HTTP | JWT | active | Proxies text or image context to the configured OpenAI-compatible provider. |
| `POST /api/v1/ai/generate-diagram` | HTTP | JWT | active | Generates and validates Excalidraw JSON from text or a base64 image. |
| `POST /api/v1/ai/to-code` | HTTP | JWT | active | Converts a scene JSON payload into previewable React/HTML code. |
| `GET /api/v1/realtime/config` | HTTP | JWT | active | Returns the external TURN URL(s) and short-lived credentials; the backend signs credentials with `TURN_AUTH_SECRET`. |
| `/socket.io/` | Socket.IO | JWT query token | active | Scene rooms broadcast presence, cursor, scene update, and WebRTC signaling events. |

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

Password accounts are created or rotated with `go run ./cmd/userctl --email ...
--password ...` (or `/app/userctl` in the backend image). Passwords are stored as
bcrypt hashes and are never returned by the API. The command also creates a
`My Workspace` owned by the user when no non-deleted workspace exists.

All workspace-scoped routes require both a valid JWT and an `X-Workspace-ID`
owned by that user. Public share links use `/p/{token}` and intentionally skip
JWT access while enforcing the share permission and optional share password.

## Versioning

Phase 1 HTTP contracts are versioned under `/api/v1`.

## Linked Decisions

- TBD
