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
| `GET /api/v1/public/scenes/{token}` | HTTP | Public | active | Read a shared scene by opaque token; ignores password query input; protected compatibility reads now use POST JSON `{password}` at the same path. Returns read-only scene JSON. The public viewer normalizes JSON-persisted Excalidraw collaborator maps before hydrating the canvas. |
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

Trash listing is owner-only and scene content is omitted. Scene asset reads filter file IDs to the authorized scene; an existing workspace file ID cannot fetch another scene's asset. Old public share routes remain compatibility adapters in A3, with passwords accepted only in POST JSON. The owner-only v1 Socket.IO adapter remains until B2.

The gateway resolves aliases before access checks, holds the workspace row lock through handler completion, and buffers HTTP success until database commit. Authorization failure or handler error rolls back the transaction. New member changes reject every owner target (including self-demotion/last-owner removal). No endpoint promotes another owner or transfers ownership.

A1 request-body protection: authenticated scoped requests are fully buffered before acquiring workspace authority. Scene bodies allow 26 MiB (25 MiB materialized-document allowance plus JSON/metadata envelope); other metadata bodies allow 1 MiB. Network body reading has a ten-second deadline, cleared after a complete read. Oversized bodies return structured `413 BODY_TOO_LARGE`; stalled bodies return `408 BODY_TIMEOUT`. Failure closes the request connection and never enters the authority transaction. Deadline support is required for requests with bodies; unsupported transports fail closed with 503. These limits apply only to scoped HTTP requests, not Socket.IO/WebSocket routes.

## Collaboration A2 invitation contract (implemented; release pending)

Owner-only GET/POST `/api/v1/workspaces/{id}/invitations` lists pending metadata or creates `{email,role:"editor"|"viewer"}`. Creation returns 201 `{invitation,inviteUrl}` with the approved browser route `/invite/:token`; only that response contains the raw token. `FRONTEND_URL` must be an HTTP(S) origin (an optional trailing slash is accepted); startup rejects credentials, path prefixes, query strings and fragments. DELETE `/api/v1/workspaces/{id}/invitations/{inviteId}` revokes a pending invitation. Workspace UUIDs and aliases are accepted. Metadata includes ID, workspace ID, normalized email, role, inviter, created/expiry/accepted/revoked times; SHA-256 hashes are never exposed.

POST `/api/v1/invitations/accept` requires login and `{token}`; the current active account email must match the trimmed lowercase invitation email. POST `/api/v1/invitations/register` is public and accepts `{token,email,password,name}`. A valid invite can create a password account, its `My Workspace` and the invited membership in one transaction. Existing accounts must log in; this endpoint never changes their password. Registration returns 201 with the new user (without password hash); accept returns 200 with `{workspaceId,userId,role,createdAt}`. Login remains the existing password endpoint. Email is trimmed/lowercased; name is trimmed, required, and at most 200 characters; registration passwords require at least eight characters and fit bcrypt’s 72-byte limit. Registration failures do not identify whether an email account exists.

Tokens contain 32 cryptographically random bytes, use URL-safe encoding and expire exactly seven days after creation (acceptance at the expiry timestamp fails). Store only SHA-256. Workspace authority is locked before invitation rows; recheck workspace, issuer ownership, invite state and active user inside the transaction. Consume once; lower-role invitations never downgrade an existing membership. Provisioning and invite consumption use A1's context-bound transaction. No outbound email or email-verification claim is introduced.

New invitation errors use `{code,message,requestId}`: 400 invalid input/link or generic registration rejection, 401 authentication required, 403 email mismatch/prohibited operation, 404 inaccessible workspace, 410 expired invite, 413 oversized body, 503 unavailable storage/authority. Disposable PostgreSQL proof covers the seven-day boundary, normalization, wrong account, two-connection concurrent consume, atomic rollback, no password reset, owner management, metadata/token redaction, default-workspace idempotency, aliases and migration rollback. HTTP DTOs reject unknown fields/trailing JSON and limit bodies to 4 KiB. A1’s 1 MiB prebuffer and ten-second read deadline run before authority locking for owner operations and acceptance/registration; oversized bodies return 413 and stalled bodies return 408. Browser invitation controls are C2; no browser UAT or deployment is claimed by A2.

`usecase.EnsureDefaultWorkspace(ctx, tx domain.UserProvisioningRepository, userID)` shares the default-personal-workspace behavior with `userctl`. `tx` is a transaction-scoped repository: invitation repositories reuse A1's context-bound GORM transaction, while `userctl` supplies a repository built from its existing GORM transaction. New personal workspaces receive a public alias and the A1 database trigger creates their owner membership. Creating and revoking invitations bumps the durable workspace ACL/audit; accepting uses the membership trigger. Migration 000011 stores invitations and ships with the remaining v2 migrations; never apply it ahead of reserved 000008–000010 in a deployed migration stream.

## Collaboration A3 public links and room credentials (implemented, release pending)

Owners manage independent links for an active scene with login JWT and X-Workspace-ID:

| Method and path | Request | Success |
| --- | --- | --- |
| GET `/api/v1/scenes/{id}/share-links` | none | 200 array `{id,access,active,expiresAt}` |
| POST `/api/v1/scenes/{id}/share-links` | `{access:"view"\|"edit",password?,expiresAt?}` | 201 `{id,access,active,expiresAt,token}`; token returned once |
| DELETE `/api/v1/scenes/{id}/share-links/{linkId}` | none | 204; independent/idempotent revoke |
| POST `/api/v1/public/scene-session` | `{token,password?,displayName?}` | 200 SceneSession below |
| POST `/api/v1/scenes/{id}/room-token` | no body; bearer login JWT or scene-session | 200 `{token,expiresAt,sceneSession?}` |
| POST `/api/v1/public/scenes/{token}` | legacy `{password}` | 200 legacy view scene after current authority check |

SceneSession is `{token,expiresAt,workspaceId,sceneId,access,principalId,displayName}`. Edit exchange requires a trimmed displayName of 1–100 characters. Passwords require at least six characters, at most 72 UTF-8 bytes; expiry must be future RFC3339. Creation/listing never exposes hashes. Public sessions are restricted to one scene and cannot act as login credentials or manage permissions. Guests mint room tokens without a workspace header; if supplied, the header must match their session workspace. Scene paths accept UUID or alias. Login callers supply X-Workspace-ID and retain current membership/override checks.

Room JWTs use dedicated COLLAB_JWT_SECRET, HS256, issuer whiteboard-api, audience whiteboard-collab and purpose room, with iat/exp/jti and at most ten minutes lifetime. Exact claims: sub=principalId, principalType=user|public, principalId, grantId, displayName, workspaceId, sceneId, access, aclVersion (decimal string). Public tokens also carry grantVersion (decimal string link revision); public IDs start `guest:` and have no user UUID. User grantId is `membership:<workspace UUID>:<user UUID>`; public grantId is its link UUID. Scene-session JWTs use audience whiteboard-scene-session and purpose scene-session. Login middleware rejects both credential purposes even with UUID subjects.

Each refresh checks current scene/workspace status, membership or link access/expiry/revocation/revision under the workspace authority lock. Public refresh returns a renewed sceneSession with the same guest identity; clients replace it before expiry. Both JWT expiries are capped to link expiry. Expired sessions require a fresh link/password exchange. Password or link changes invalidate earlier public sessions; unrelated workspace changes do not invalidate the link revision. Link mutations increment workspace ACL version, audit and notify within one transaction.

Errors use `{code,message,requestId}`: 400 INVALID_INPUT, 401 SHARE_PASSWORD_REQUIRED or UNAUTHENTICATED, 403 FORBIDDEN, 404 NOT_FOUND (including revoked/expired links), 413 BODY_TOO_LARGE, 503 AUTHORITY_UNAVAILABLE. Sharing bodies are limited to 4 KiB with unknown-field/trailing-JSON rejection. Credential responses disable caching. The legacy GET path ignores password queries; middleware strips its query and redacts RequestURI before logging.

Migration 000008 preserves active/inactive legacy view links and bcrypt hashes. Legacy PUT share updates synchronize only their mapped link; new links store SHA-256 digests only. Down migration refuses multiple links and any new/edit/expiring grant that the legacy table cannot represent. See [public-link authority and migration](public-links.md) and [deployment](../operations/DEPLOYMENT.md). Browser controls and live collab-runtime behavior remain C2/B integration; no v2 deployment is claimed.

## A4 mutation and scene assets (backend implemented; release integration pending)

Private Go mutation decisions and public scene assets follow the [authority contract](mutation-authority.md). The private listener authenticates a service credential plus a scoped room token; it is absent from public API routing. Scene-session upload/view routes use current access and bounded object-store URLs; public view sessions cannot upload. [Human guide](../user/scene-assets.md) and [agent reference](../../src/frontend/public/docs/scene-assets.md) describe the same contract. Verification includes PostgreSQL revoke races, scene isolation, expiry and authority failure.

| Method | Endpoint | Credential / behavior |
| --- | --- | --- |
| POST | `/api/v1/scenes/{id}/assets/upload-capability` | Login JWT + workspace header, or target scene-session/room token; `{assets:[...]}` → `{urls,expiresAt}`; edit only |
| POST | `/api/v1/public/scenes/{id}/assets/presign-upload` | Scoped scene-session/room token; same edit-only request/response |
| POST | `/api/v1/public/scenes/{id}/assets/presign-view` | Scoped scene-session/room token; `{fileIds:[...]}` → `{urls}`; view/edit |
| GET | `/api/v1/realtime/config` | Scoped scene-session/room token required; current scene access → TURN REST credentials; login alone denied |
| POST | `/internal/authority/decisions` | Private listener only; dedicated service bearer + room token, exact binding in [protocol](mutation-authority.md) |

Existing login `/api/v1/assets/presign-upload` remains compatible with the new upload limits and short URLs. New scoped upload responses include `headers.Content-Type`; PUT the original Blob so the browser supplies exact Content-Length. MIME allowlist is PNG/JPEG/GIF/WebP; 1 byte–10 MiB; at most 100 unique file IDs. Existing SVG/AVIF references remain readable. URL expiry is bounded by credential/link expiry; already-issued object-store URLs last until their expiration despite revocation. New Go authority decision lifetimes are <=5s. Runtime reverse-proxy and object-store integration validation remains E-owned.
