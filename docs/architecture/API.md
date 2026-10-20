---
artifact_type: api_contract_index
id: API-MASTER
status: active
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

This index records the implemented contract. Collaboration v2 endpoints are
deployed on VM100; the release checklist distinguishes completed public checks
from the remaining operational canaries.

| Contract | Type | Auth | Status | Notes |
| --- | --- | --- | --- | --- |
| `POST /api/v1/auth/login` | HTTP | Public | active | Accepts `{email,password}` and returns `{token}` for a password-authenticated user. |
| `GET, POST /api/v1/workspaces` | HTTP | JWT | active | List and create workspaces. |
| `PATCH /api/v1/workspaces/{id}` | HTTP | Login JWT + current workspace owner | active locally | Rename only: `{name}` (trimmed, 3–100 characters); `200 Workspace`. UUID or workspace alias resolves to the same workspace; its ID does not change. |
| `GET, POST /api/v1/folders` | HTTP | JWT + workspace | active | List/create folders; `all=true` returns flat tree data. |
| `PATCH, DELETE /api/v1/folders/{id}` | HTTP | Login JWT + `X-Workspace-ID`; owner/editor for PATCH | active locally | PATCH accepts optional `{name,newParentId}`; omitted parent keeps its location, `null` moves to root, UUID selects a folder in this workspace. Name is trimmed, 1–100 characters. PATCH returns 204. Self/descendant and depth-invalid moves are rejected; missing parent returns 404. DELETE retains the trash lifecycle. |
| `GET, POST /api/v1/scenes` | HTTP | JWT + workspace | active | List/create scenes; `trash=true` lists deleted scenes. |
| `GET, PATCH, DELETE /api/v1/scenes/{id}` | HTTP | JWT + workspace | active | Read metadata/content, update legacy scene metadata/content, and trash scenes. Promoted v2 scenes reject legacy HTTP content PATCH and write through the durable collaboration writer. |
| `GET /api/v1/scenes/{id}/projection` | HTTP | Login JWT + `X-Workspace-ID` + current `scene:read` capability | active locally | `200` durable collaboration projection containing `content` and `sequence`, including acknowledged updates since the last compaction. UUID and public aliases resolve in the specified workspace. Every response, including 401/403/404/5xx, has `Cache-Control: no-store`. Service failure fails closed; no public-link or agent-key access. |
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
| `GET /api/v1/realtime/config` | HTTP | Scoped scene session or room token | active | Rechecks current scene access and returns external TURN URL(s) plus short-lived credentials; a login JWT alone is denied. |
| `/collab/` | Custom WebSocket | Room token in first `auth` control | active | One admitted socket carries durable Yjs updates, roster, cursor/selection, follow viewport and WebRTC signaling. Credentials never appear in the URL. `/socket.io/` is retired. |
| `GET /api/v1/scim/v2/ServiceProviderConfig`, `/ResourceTypes`, `/Schemas` | SCIM 2.0 | Dedicated `Bearer scim_…` token | active | Discovery endpoints for workspace provisioning. Tokens are workspace-scoped and never accepted as login JWTs. |
| `GET, POST /api/v1/scim/v2/Users`, `PATCH, PUT, DELETE /api/v1/scim/v2/Users/{id}` | SCIM 2.0 | Dedicated SCIM token | active | Provision viewers/editors, preserve manual provenance, reject owner adoption, and serialize changes with workspace locks. |
| `GET, POST /api/v1/scim/v2/Groups`, `PATCH, PUT, DELETE /api/v1/scim/v2/Groups/{id}` | SCIM 2.0 | Dedicated SCIM token | active | Group membership materializes managed membership sources; owner rows are protected. |
| `GET, POST /api/v1/workspaces/{id}/scim/token`, `DELETE .../scim/token` | HTTP | JWT + workspace owner | active | Rotate/revoke a token; the raw token is returned once. |
| `GET /api/v1/workspaces/{id}/libraries`, `POST .../libraries/import`, `PUT .../libraries/{libraryId}` | HTTP | JWT + workspace edit/manage | active | Import and publish immutable native `.excalidrawlib` revisions with bounded payloads. |
| `GET .../libraries/{libraryId}/revisions/{revision}/export`, `POST /api/v1/workspaces/{id}/scenes/{sceneId}/library-insertions` | HTTP | JWT + workspace/scene access | active | Export a revision or insert it into an authorized scene with sequence/idempotency checks. |

## Collection contract

Collections organize links to scenes a caller can already access. They never
grant scene access and deleting a collection never deletes or moves a scene.
Every request uses a login JWT and resolves the workspace path segment as a UUID
or public alias. Mutations require current workspace edit access and a UUID
`requestId`; updates also require the current decimal-string
`expectedRevision`.

| Method and path | Request / query | Success |
| --- | --- | --- |
| `GET /api/v1/workspaces/{workspace}/collections` | `cursor=<collection UUID>&limit=1..50` | `200 {items,nextCursor?}` after private-container and current scene-access filtering |
| `POST /api/v1/workspaces/{workspace}/collections` | `{requestId,name,visibility:"workspace"|"private"}` | `201 Collection` |
| `GET /api/v1/workspaces/{workspace}/collections/{id}` | — | `200 Collection` with currently accessible ordered scenes plus `mine` and `canSetPrivate` capabilities |
| `PATCH /api/v1/workspaces/{workspace}/collections/{id}` | `{requestId,expectedRevision,name,visibility}` | `200 Collection` |
| `DELETE /api/v1/workspaces/{workspace}/collections/{id}` | `{requestId,expectedRevision}` | `204`; membership rows only |
| `POST /api/v1/workspaces/{workspace}/collections/{id}/scenes` | `{requestId,expectedRevision,sceneId}` | `200 Collection` |
| `DELETE /api/v1/workspaces/{workspace}/collections/{id}/scenes/{sceneId}` | `{requestId,expectedRevision}` | `200 Collection`; current scene access is checked before membership |
| `PUT /api/v1/workspaces/{workspace}/collections/{id}/order` | `{requestId,expectedRevision,sceneIds:[...]}` | `200 Collection`; input is the exact currently accessible subset and stale hidden rows are pruned atomically |
| `/w/{workspace}/collections/{id}` | Browser route | Reloadable collection view; scene links remain canonical scene routes |

Private collections are readable and mutable only by their creator. A workspace
owner may audit-delete an inaccessible private container by ID and revision,
without receiving its name or scene contents. A stale revision returns
`409 REVISION_CONFLICT`; changed reuse of a request ID returns
`409 REQUEST_ID_REUSED`; disabled policy returns `403 FEATURE_DISABLED`.
Only `canSetPrivate:true` callers may convert a workspace collection to private;
the response does not expose the creator's user ID. Invalid limits, including
non-decimal values, return `400 INVALID_INPUT`. Workspace names are unique
case-insensitively within the workspace; private names are unique
case-insensitively per creator and workspace. Name collisions return
`409 COLLECTION_NAME_CONFLICT`.

An exact mutation retry is bound to its operation as well as its principal and
payload. The service reauthorizes the workspace, collection visibility, and any
referenced scene while holding the workspace lock before returning a receipt;
it rebuilds collection bodies from current ACL-filtered data. A transport retry
must reuse its request ID until an HTTP response is received or the caller
explicitly cancels the attempt.

## Errors

| Error | Meaning | Consumer Impact |
| --- | --- | --- |
| `400 Bad Request` | Invalid UUID, body, title, content, or folder mapping | Request must be corrected. |
| `401 Unauthorized` | Missing/invalid JWT or invalid email/password | Consumer must authenticate again. |
| `404 Not Found` | Workspace-scoped entity does not exist | Consumer must discard stale entity reference. |
| `403 Forbidden` | Authenticated principal lacks the required current workspace/scene role or capability | Consumer must stop and must not retry with the same credential. |
| `500 Internal Server Error` | Repository or infrastructure failure | Consumer may retry safely where appropriate. |
| `429 Too Many Requests` | Login, invitation registration, or public scene credential attempts exhausted an IP or privacy-preserving identity bucket | Consumer must wait for the integer seconds in `Retry-After`; JSON is `{code:"RATE_LIMITED",message,requestId}`. |

Credential attempt limits use a five-minute fixed window in each API process.
Password login allows 30 attempts per client IP and 10 per normalized account;
invitation registration allows 20 per IP and 5 per invitation token; public
scene-session exchange and public snapshot POST share limits of 60 per IP and
15 per public token. The limiter never stores raw account or token identifiers.
Distributed/reverse-proxy limiting remains an independent deployment concern.
Implementation and regression evidence for credential attempt limits is in the
[authentication hardening verification](../work/verification/auth-hardening-2026-10-19.md).

Dashboard management uses the existing scene PATCH `{folderId}` contract and
`POST /api/v1/scenes/{id}/duplicate` with `{title,folderId}`. The duplicate
endpoint requires a login session, `X-Workspace-ID`, current scene edit access,
and workspace owner/editor membership; it returns `201` with the new scene.
The server reads a v2 scene through a signed collaboration export projection,
including acknowledged edits not yet compacted into the scene row. This bounded
export path supports scenes above the ordinary read projection's 500-element/
256 KiB limit. A legacy scene uses its stored content after normalizing it to
the promotion contract: only durable app state and scene-owned file metadata
are retained; renderer/session fields and inline file bytes are dropped. A
legacy file without a matching active scene-scoped `excalidraw_file` asset and
matching MIME cannot be copied. The v1 copy validator checks durable field
types and bounds against the collaboration scene contract before scene creation
and again after asset staging. Invalid element values, null required file
metadata, or an image
reference missing from `files` also rejects the copy. Referenced scene assets
are copied to new object keys and scene-scoped rows, with file IDs and image
references rewritten. The scene remains hidden until content and assets are
complete; failed copies use an independent cleanup context to remove staged
objects and rows. If object storage deletion fails, the hidden scene and stage
records remain for recovery instead of appearing as a partial copy. Migration
`000029` marks these rows `pending_duplicate`; Trash, search, scene Restore and
recursive folder Restore exclude them until the publish update clears the marker
and `deleted_at` atomically.
The server rechecks current scene and workspace edit access under the workspace
lock just before publication. A grant revoked during export or asset staging
prevents publication and triggers hidden-copy compensation.
Compensation first claims a pending row with `duplicate_cleanup_started`;
publication refuses claimed rows, while hard deletion refuses published rows.
After an ambiguous publish error, the server independently reads this state and
returns a committed scene instead of deleting it. If the read fails, it preserves
objects and returns `503 DUPLICATE_STATE_UNCERTAIN`. Migration `000029` backfills
staged old copies and refuses ambiguous near-creation soft deletes until old
backend writers are drained and candidates classified per the
[deployment runbook](../operations/DEPLOYMENT.md). The down migration refuses
to remove the marker while pending copies exist.
A cleanup failure returns `500 DUPLICATE_CLEANUP_FAILED` and logs
the hidden scene ID, phase and request ID without exposing storage details to
the caller.
Session and collaborator state is not copied. Browser `POST /api/v1/scenes`
with `{title,folderId,content}` remains for new scenes and imports, not duplication. These
metadata and content mutations are available only with current edit access;
workspace rename additionally requires owner membership. Folder PATCH rejects
invalid names/targets as `400 INVALID_INPUT`, and workspace PATCH rejects an
invalid name the same way. Access failures use current scoped 403/404 handling.
The dashboard treats `403 FEATURE_DISABLED` from Collections as a stable policy
state; other collection failures show Retry.
Dashboard management implementation and local regression evidence is in the
[dashboard management verification](../work/verification/dashboard-management-2026-10-21.md).

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

- [ADR 0008 — OpenAI-compatible AI provider](../decisions/0008-openai-compatible-ai-provider.md)
- [ADR 0009 — collaboration access](../decisions/0009-collaboration-access.md)
- [ADR 0010 — durable Yjs writer](../decisions/0010-durable-yjs-writer.md)
- [ADR 0011 — native MCP](../decisions/0011-native-mcp.md)
- [ADR 0012 — legacy dashboard presentation](../decisions/0012-legacy-dashboard-presentation.md)

## Collaboration A1 contract (deployed; complete production roles canary pending)

Approved programme A1 replaces owner-only tenancy with database-backed owner/editor/viewer membership. Editors write; viewers read; owner-only management cannot remove or demote any owner. Scene editor/viewer overrides require membership and affect content only. Workspace and scene UUIDs remain canonical; optional 21-character URL-safe aliases resolve before ACL checks and never grant access. Removed members and deleted resources are denied. Trash metadata/restore is owner-only. Folder mutations require edit access to every descendant scene. Workspace lists contain current memberships; search and scene lists use the same current policy.

Owner routes: GET `/workspaces/{id}/members`, PATCH/DELETE `/workspaces/{id}/members/{userId}`, PUT/DELETE `/scenes/{id}/members/{userId}`. PATCH/PUT accepts `{role:"editor"|"viewer"}`. All routes require login; scene routes retain the workspace header. New policy errors use `{code,message,requestId}`.

Verified on 2026-10-16: 79 backend tests pass with real disposable PostgreSQL enabled, including migration/rollback, aliases, access matrix, removed/soft-deleted members, search/list isolation, protected descendants, owner immutability, audit/version changes, and two-connection write/revoke serialization. Public-session and invitation behavior belongs to A2/A3.

A1 DTOs preserve existing fields and add `publicId?: string` to Workspace/Scene, and `aclVersion: number` to Workspace. Effective role/access is currently internal to `AccessService`; C2 must add an explicit response contract before hiding editor/owner actions. `ownerId` remains the legacy primary owner; membership is the authority. Member GET returns `[{workspaceId,userId,role,createdAt}]`; PATCH/DELETE/override writes return 204. Management body errors return 400; policy failures return the structured error above.

Trash listing is owner-only and scene content is omitted. Scene asset reads
filter file IDs to the authorized scene; an existing workspace file ID cannot
fetch another scene's asset. Old public share routes remain compatibility
adapters in A3, with passwords accepted only in POST JSON. The v1 Socket.IO
adapter and proxy route are removed; /collab/ is the only realtime transport.

The gateway resolves aliases before access checks, holds the workspace row lock through handler completion, and buffers HTTP success until database commit. Authorization failure or handler error rolls back the transaction. New member changes reject every owner target (including self-demotion/last-owner removal). No endpoint promotes another owner or transfers ownership.

A1 request-body protection: authenticated scoped requests are fully buffered before acquiring workspace authority. Scene bodies allow 26 MiB (25 MiB materialized-document allowance plus JSON/metadata envelope); other metadata bodies allow 1 MiB. Network body reading has a ten-second deadline, cleared after a complete read. Oversized bodies return structured `413 BODY_TOO_LARGE`; stalled bodies return `408 BODY_TIMEOUT`. Failure closes the request connection and never enters the authority transaction. Deadline support is required for requests with bodies; unsupported transports fail closed with 503. These limits apply to scoped HTTP requests; the custom WebSocket applies its own bounded protocol/frame limits.

## Collaboration A2 invitation contract (deployed; production acceptance canary pending)

Owner-only GET/POST `/api/v1/workspaces/{id}/invitations` lists pending metadata or creates `{email,role:"editor"|"viewer"}`. Creation returns 201 `{invitation,inviteUrl}` with the approved browser route `/invite/:token`; only that response contains the raw token. `FRONTEND_URL` must be an HTTP(S) origin (an optional trailing slash is accepted); startup rejects credentials, path prefixes, query strings and fragments. DELETE `/api/v1/workspaces/{id}/invitations/{inviteId}` revokes a pending invitation. Workspace UUIDs and aliases are accepted. Metadata includes ID, workspace ID, normalized email, role, inviter, created/expiry/accepted/revoked times; SHA-256 hashes are never exposed.

POST `/api/v1/invitations/accept` requires login and `{token}`; the current active account email must match the trimmed lowercase invitation email. POST `/api/v1/invitations/register` is public and accepts `{token,email,password,name}`. A valid invite can create a password account, its `My Workspace` and the invited membership in one transaction. Existing accounts must log in; this endpoint never changes their password. Registration returns 201 with the new user (without password hash); accept returns 200 with `{workspaceId,userId,role,createdAt}`. Login remains the existing password endpoint. Email is trimmed/lowercased; name is trimmed, required, and at most 200 characters; registration passwords require at least eight characters and fit bcrypt’s 72-byte limit. Registration failures do not identify whether an email account exists.

Tokens contain 32 cryptographically random bytes, use URL-safe encoding and expire exactly seven days after creation (acceptance at the expiry timestamp fails). Store only SHA-256. Workspace authority is locked before invitation rows; recheck workspace, issuer ownership, invite state and active user inside the transaction. Consume once; lower-role invitations never downgrade an existing membership. Provisioning and invite consumption use A1's context-bound transaction. No outbound email or email-verification claim is introduced.

New invitation errors use `{code,message,requestId}`: 400 invalid input/link or generic registration rejection, 401 authentication required, 403 email mismatch/prohibited operation, 404 inaccessible workspace, 410 expired invite, 413 oversized body, 503 unavailable storage/authority. Disposable PostgreSQL proof covers the seven-day boundary, normalization, wrong account, two-connection concurrent consume, atomic rollback, no password reset, owner management, metadata/token redaction, default-workspace idempotency, aliases and migration rollback. HTTP DTOs reject unknown fields/trailing JSON and limit bodies to 4 KiB. A1’s 1 MiB prebuffer and ten-second read deadline run before authority locking for owner operations and acceptance/registration; oversized bodies return 413 and stalled bodies return 408. Browser invitation controls are C2; no browser UAT or deployment is claimed by A2.

`usecase.EnsureDefaultWorkspace(ctx, tx domain.UserProvisioningRepository, userID)` shares the default-personal-workspace behavior with `userctl`. `tx` is a transaction-scoped repository: invitation repositories reuse A1's context-bound GORM transaction, while `userctl` supplies a repository built from its existing GORM transaction. New personal workspaces receive a public alias and the A1 database trigger creates their owner membership. Creating and revoking invitations bumps the durable workspace ACL/audit; accepting uses the membership trigger. Migration 000011 stores invitations and ships with the remaining v2 migrations; never apply it ahead of reserved 000008–000010 in a deployed migration stream.

## Collaboration A3 public links and room credentials (deployed; guest mutation/revocation canary pending)

Owners manage independent links for an active scene with login JWT and X-Workspace-ID:

| Method and path | Request | Success |
| --- | --- | --- |
| GET `/api/v1/scenes/{id}/share-links` | none | 200 array `{id,access,allowComments,active,expiresAt}` |
| POST `/api/v1/scenes/{id}/share-links` | `{access:"view"\|"edit",allowComments,password?,expiresAt?}` | 201 `{id,access,allowComments,active,expiresAt,token}`; token returned once |
| DELETE `/api/v1/scenes/{id}/share-links/{linkId}` | none | 204; independent/idempotent revoke |
| POST `/api/v1/public/scene-session` | `{token,password?,displayName?}` | 200 SceneSession below |
| POST `/api/v1/scenes/{id}/room-token` | no body; bearer login JWT or scene-session | 200 `{token,expiresAt,sceneSession?}` |
| POST `/api/v1/public/scenes/{token}` | legacy `{password}` | 200 legacy view scene after current authority check |

SceneSession is `{token,expiresAt,workspaceId,sceneId,access,principalId,displayName}`. Edit exchange requires a trimmed displayName of 1–100 characters. Passwords require at least six characters, at most 72 UTF-8 bytes; expiry must be future RFC3339. Creation/listing never exposes hashes. Public sessions are restricted to one scene and cannot act as login credentials or manage permissions. Guests mint room tokens without a workspace header; if supplied, the header must match their session workspace. Scene paths accept UUID or alias. Login callers supply X-Workspace-ID and retain current membership/override checks.

Room JWTs use dedicated COLLAB_JWT_SECRET, HS256, issuer whiteboard-api, audience whiteboard-collab and purpose room, with iat/exp/jti and at most ten minutes lifetime. Exact claims: sub=principalId, principalType=user|public, principalId, grantId, displayName, workspaceId, sceneId, access, aclVersion (decimal string). Public tokens also carry grantVersion (decimal string link revision); public IDs start `guest:` and have no user UUID. User grantId is `membership:<workspace UUID>:<user UUID>`; public grantId is its link UUID. Scene-session JWTs use audience whiteboard-scene-session and purpose scene-session. Login middleware rejects both credential purposes even with UUID subjects.

Each refresh checks current scene/workspace status, membership or link access/expiry/revocation/revision under the workspace authority lock. Public refresh returns a renewed sceneSession with the same guest identity; clients replace it before expiry. Both JWT expiries are capped to link expiry. Expired sessions require a fresh link/password exchange. Password or link changes invalidate earlier public sessions; unrelated workspace changes do not invalidate the link revision. Link mutations increment workspace ACL version, audit and notify within one transaction.

Errors use `{code,message,requestId}`: 400 INVALID_INPUT, 401 SHARE_PASSWORD_REQUIRED or UNAUTHENTICATED, 403 FORBIDDEN, 404 NOT_FOUND (including revoked/expired links), 413 BODY_TOO_LARGE, 503 AUTHORITY_UNAVAILABLE. Sharing bodies are limited to 4 KiB with unknown-field/trailing-JSON rejection. Credential responses disable caching. The legacy GET path ignores password queries; middleware strips its query and redacts RequestURI before logging.

Migration 000008 preserves active/inactive legacy view links and bcrypt hashes. Legacy PUT share updates synchronize only their mapped link; new links store SHA-256 digests only. Down migration refuses multiple links and any new/edit/expiring grant that the legacy table cannot represent. See [public-link authority and migration](public-links.md) and [deployment](../operations/DEPLOYMENT.md). Browser controls and live collab-runtime behavior remain C2/B integration; no v2 deployment is claimed.

## Enterprise C1 comment contract

Comment metadata is served by the Go API and is never stored in Yjs scene
content. Login callers provide `X-Workspace-ID`; public callers provide the
existing scene-session bearer. Both forms resolve the URL scene inside the
credential scope and recheck current feature/access authority for every read
and mutation.

| Method and path | Request / query | Success |
| --- | --- | --- |
| GET `/api/v1/scenes/{id}/comments` | `cursor=<thread UUID>&limit=1..50` (default 50) | 200 `{items,nextCursor}` |
| POST `/api/v1/scenes/{id}/comments` | `{requestId,anchor:{elementId?,x,y},body}` | 201 thread |
| GET `/api/v1/scenes/{id}/comments/{threadId}` | `messageCursor=<message UUID>&messageLimit=1..50` (default 50) | 200 thread with at most 50 messages and optional `nextMessageCursor` |
| POST `/api/v1/scenes/{id}/comments/{threadId}/replies` | `{requestId,body}` | 201 message |
| PATCH `/api/v1/scenes/{id}/comments/{threadId}/messages/{messageId}` | `{requestId,expectedRevision,body}` | 200 message |
| DELETE `/api/v1/scenes/{id}/comments/{threadId}/messages/{messageId}` | strict JSON `{requestId,expectedRevision}` | 204 |
| PATCH `/api/v1/scenes/{id}/comments/{threadId}` | `{requestId,expectedRevision,resolved?,subscribed?}` | 200 thread |

Bodies are plain UTF-8 text containing 1–4,000 Unicode code points and are
rendered as text by clients. Revisions are positive canonical decimal strings.
Request IDs are UUIDs bound to principal, workspace, scene, action and exact
payload; an exact retry returns the stored result while changed reuse returns
`409 REQUEST_ID_REUSED`. Stale revisions return `409 REVISION_CONFLICT`.
Foreign thread/message IDs return 404 without revealing another tenant or
scene. Responses expose safe author display names and `mine`, never accept an
author/display name in mutation bodies, and never expose stored principal IDs.
Message pages use stable `(created_at,id)` order. Pass `nextMessageCursor` as
the next request's `messageCursor` until it is omitted; cursors are bound to
the requested workspace, scene and thread, so a cursor copied from another
thread returns the same 404 as an inaccessible resource.

Authors may edit or redact their own messages. Workspace owners may redact any
message; authenticated owners/editors may resolve or reopen threads. Redaction
keeps the message and revision with an empty body plus `deletedAt`. Authenticated
authors are subscribed on creation/reply; guests have no workspace inbox.

Authenticated members use `GET /api/v1/notifications?cursor=<notification
UUID>&limit=1..50` and `POST /api/v1/notifications/{id}/read` with strict JSON
`{requestId}`. Pages contain actionable
`/w/{workspace}/scenes/{scene}?thread={thread}` links. Delivery is unique per
`(user,event)`, excludes the actor, and checks current scene/comment authority;
list and mark-read repeat that check. Public scene-session principals cannot
use the private inbox.
Writes are capped at 30 committed mutations per principal and scene per minute.
Mutation, subscription, idempotency receipt, cursor-only event/outbox row and
comment data share the workspace-lock transaction. Events contain only IDs,
revision, status and request ID, never the body.

## A4 mutation and scene assets (deployed; production storage canary passed)

Private Go mutation decisions and public scene assets follow the [authority contract](mutation-authority.md). The private listener authenticates a service credential plus a scoped room token; it is absent from public API routing. Scene-session upload/view routes use current access and bounded object-store URLs; public view sessions cannot upload. [Human guide](../user/scene-assets.md) and [agent reference](../../src/frontend/public/docs/scene-assets.md) describe the same contract. Verification includes PostgreSQL revoke races, scene isolation, expiry and authority failure.

| Method | Endpoint | Credential / behavior |
| --- | --- | --- |
| POST | `/api/v1/scenes/{id}/assets/upload-capability` | Login JWT + workspace header, or target scene-session/room token; `{assets:[...]}` → `{urls,expiresAt}`; edit only |
| POST | `/api/v1/public/scenes/{id}/assets/presign-upload` | Scoped scene-session/room token; same edit-only request/response |
| POST | `/api/v1/public/scenes/{id}/assets/presign-view` | Scoped scene-session/room token; `{fileIds:[...]}` → `{urls}`; view/edit |
| GET | `/api/v1/realtime/config` | Scoped scene-session/room token required; current scene access → TURN REST credentials; login alone denied |
| POST | `/internal/authority/decisions` | Private listener only; dedicated service bearer + room token, exact binding in [protocol](mutation-authority.md) |

Existing login `/api/v1/assets/presign-upload` remains compatible with the new upload limits and short URLs. New scoped upload responses include `headers.Content-Type`; PUT the original Blob so the browser supplies exact Content-Length. MIME allowlist is PNG/JPEG/GIF/WebP; 1 byte–10 MiB; at most 100 unique file IDs. Existing SVG/AVIF references remain readable. URL expiry is bounded by credential/link expiry; already-issued object-store URLs last until their expiration despite revocation. New Go authority decision lifetimes are <=5s. Runtime reverse-proxy and object-store integration validation remains E-owned.


## Scoped agent keys and MCP (deployed; authenticated read/revocation canary passed)

Owner login routes: `GET/POST /api/v1/workspaces/{id}/agent-keys`, `DELETE /api/v1/workspaces/{id}/agent-keys/{keyId}`. POST accepts `{scopes:string[],sceneIds:UUID[]}` and returns metadata plus `key` once. Lists/revocations never return the raw key or digest. Each key targets one workspace; empty sceneIds means all its scenes. Explicit scene targets cannot include `scene:create`. Scopes: `workspace:read`, `scene:read`, `scene:create`, `scene:write`, `scene:move`, `scene:trash`, `scene:export`; no scope implies another. Key mutations audit key IDs and bump the workspace ACL transactionally.

Dedicated bearer routes (agent keys are rejected by login middleware):

| Route | Requirement / response |
| --- | --- |
| `GET /api/v1/agent/auth` | Current active key, owner membership and workspace; `{principalId,keyId}` |
| `GET /api/v1/agent/workspaces?cursor&limit` | `workspace:read`; `{items,nextCursor?}` |
| `GET /api/v1/agent/workspaces/{workspaceId}/scenes?cursor&limit` | `scene:read`; scoped metadata page |
| `POST /api/v1/agent/workspaces/{workspaceId}/scenes` | `scene:create`, unrestricted workspace target, current edit; `{title}` creates `{elements:[],appState:{},files:{}}` |
| `GET /api/v1/agent/workspaces/{workspaceId}/scenes/{sceneId}` | `scene:read`; at most 500 elements/256 KiB, else `413 PROJECTION_TOO_LARGE` |
| `POST /api/v1/agent/workspaces/{workspaceId}/scenes/{sceneId}/room-token` | Current read or write scope; ten-minute scoped room credential for private authority bridge |

All discovery filters precede pagination (limit 1–50, UUID cursor, no global totals). Agent principal is `agent:<key UUID>`, grantId is that key UUID, UserID is nil and Owner is always false. A4 independently requires `scene:read` for read decisions and `scene:write` for update/command decisions; a write-only room credential does not confer read admission. Current key revocation, owner membership/status and targets are rechecked under workspace lock. Agents do not use asset/TURN capabilities or management endpoints.

MCP `/mcp` uses the official SDK Streamable HTTP lifecycle with exact Origin allowlist and authenticated principal-bound sessions. The remote stdio adapter requires `WHITEBOARD_API_URL`, `WHITEBOARD_PUBLIC_URL` and `WHITEBOARD_AGENT_KEY`; stdout contains only MCP frames. PAT/header and stdio compatibility are supported; OAuth discovery/PKCE is not claimed. See [agent guide](../user/agents.md) and [ADR 0011](../decisions/0011-native-mcp.md).

### Asset capability expiry and renewal clarification

Upload capabilities preserve the absolute parent credential/current public-link deadline through repository lookup and signing. A renewal of the same scene/file preserves its asset ID, storage key and original uploader audit only when MIME, byte size, SHA-256 and dimensions are identical. Conflicting metadata returns `409 ASSET_CONFLICT` instead of replacing the existing object identity.

### D1 coordinated audit clarification

Agent operations write authoritative `go_operation` audit records in PostgreSQL (migration 13), with server-derived key/principal, key workspace, requested workspace/scene, exact tool/scope, result/failure class, latency and UUID correlation. Successful `create_scene` and audit insertion commit atomically. Optional canonical UUID `X-Agent-Call-ID` is diagnostic only; caller principal/result authority is never accepted.

MCP also writes `adapter_observed` journal events for validation, denial and unreachable upstream. HTTP requires absolute persistent `MCP_AUDIT_PATH`; stdio defaults to `~/.local/state/whiteboard-mcp/audit.jsonl`. Client-controlled stdio journals must be collected for complete adapter history. One process owns each path; mode 0600, default active file plus two 1 MiB archives. Pre-operation journal failure fails closed; completion-write failure returns `AUDIT_COMPLETION_UNCERTAIN` and correlation. Inspect authoritative state before retrying scene creation. D3 idempotent mutations may retry identical stable request ID/bytes. No credentials, arguments, document contents or raw exception strings are recorded. Malformed MCP `tools/call` envelopes return JSON-RPC `-32602 INVALID_INPUT` with `error.data.agentCallId` and are audited before SDK validation; valid tool results retain `_meta.agentCallId`. This clarification was supplied by D1 during C2 API ownership handoff.

## Collaboration C2 browser contract (implemented locally; release pending)

The browser consumes A2/A3 without inferring permissions from `ownerId`. Workspace list responses include the requesting member's computed `role: owner|editor|viewer`; workspace creation returns `role: owner`. The repository reads role from current membership, and this projection is read-only with no schema migration. Authenticated scene create/list/get/update/star responses include `access: view|edit` and `owner: boolean`, resolved through `AccessService` under the existing workspace authority transaction. Scene overrides are reflected in these fields. Missing permission fields fail closed in browser controls. UUID/public aliases are mapped to canonical workspace IDs and resolved by server ACL for scenes.

Browser routes: `/w/:id` (workspace dashboard), `/w/:id/settings/members`, `/w/:id/settings/agents`, `/w/:id/scenes/:sceneId`, `/invite/:token`, `/p/:token`. Route generations invalidate stale scene opens; returning home cancels the prior open without erasing the loaded scene list. Explicit workspace selection navigates to the selected workspace dashboard. All files/folder/Starred/owner-only Trash/search navigation leaves settings while retaining its folder/view/query; Dashboard resets those selections. URL-to-store synchronization remains authoritative for direct links and browser history. Search scene results navigate to a scene route. Settings components issue no management requests for nonowners.

`acceptInvite(token:string): Promise<Membership>` consumes POST `/invitations/accept`. Registration uses the isolated public client, then ordinary password login; no scene credential becomes an account token. Invitation tokens stay in the URL/component only, never local/session storage, and successful consume replaces the invitation URL.

`createPublicSession({token,password?,displayName?}): Promise<SceneSession>` consumes POST `/public/scene-session` using the isolated client. `SceneSession = {token,expiresAt,workspaceId,sceneId,access,principalId,displayName}` is held in the B3 provider and same-tab sessionStorage keyed by public link, separate from account authentication. `renewPublicRoom(session)` POSTs `/scenes/:sceneId/room-token` with only the explicit scene-session bearer; it returns the room token, expiry and optional renewed sceneSession. These functions live in frontend `lib/collaboration-api.ts`. Public requests never run the account 401 interceptor.

POST `/public/scenes/:token` now accepts `{password?,displayName?}`. The
application reads new and migrated links by exchanging the link, deriving its
canonical scope from the signed scene session, and rechecking current authority
and credential expiry under the workspace lock before reading the snapshot. The
supplied display name permits reading an edit-link snapshot; IDs/access are
never accepted from the caller. The earlier legacy-only permission lookup could
not read newly created independent links and was replaced in runtime wiring.
Existing legacy constructor compatibility remains for old tests/integrations.
GET/query passwords remain unsupported.

Share UI uses GET/POST `/scenes/:sceneId/share-links` and DELETE `/scenes/:sceneId/share-links/:linkId`. Owners can opt each link into comments independently of view/edit access; omitted `allowComments` defaults to false. Tokens are consumed only from creation responses; lists contain metadata including the stored comment policy. Member/override/invitation actions use A2 routes. Scene override selectors apply explicit updates and do not claim to display current override metadata (no read endpoint exists).

`PublicSceneEditorProps` in frontend `components/public-view.tsx` exposes
`{scene,session,onSessionChange,onAccessLost}`. It renders a guarded snapshot
until authenticated room admission. Live edit, renewal, revocation, offline
replay and real API+DB+WebSocket browser behavior now pass the 17/17 local
real-stack suite. Public-origin VM100 proof remains the E3 release gate.

## C3 member activity feed (implemented locally; release pending)

Authenticated members use `GET /api/v1/workspaces/{workspace}/activity` or
`GET /api/v1/scenes/{scene}/activity` with `X-Workspace-ID` for scene aliases.
Both accept `cursor` and `limit` (1–50) and return
`{items,nextCursor?,retainedSince?}`. `retainedSince` is omitted when the
authorized scope contains no retained events. The cursor is opaque and binds the stable
descending `(occurredAt,id)` boundary.

Current membership, live-scene visibility, feature policy and owner-only event
rules are applied before pagination. Nonowners never receive `security.*` or
`acl.*` events. Actor names are server-resolved only after authorization; raw
comment bodies, canvas content, credentials and source provenance are absent.
Guests and agent keys cannot use these member routes. The existing MCP scene
activity contract is unchanged.


## Durable agent operations and export (deployed; production write/export canary pending)

D3 replaces materialized-snapshot reads with current durable projection: `sequence` is a decimal bigint string, accompanied by `documentId`, `stateVector` and native content. `contentVersion` is not a mutation precondition. The normal read retains its500element/256KiB bound. Exact scopes are independent; write-only keys do not receive read access.

Routes below share `/api/v1/agent/workspaces/{workspaceId}/scenes/{sceneId}` and bearer agent authentication:

| Suffix/method | Scope and contract |
| --- | --- |
| `POST /commands` | `scene:write`; strict `{tool,mutationBase64}` with canonical base64 of original UTF-8 mutation bytes. Mutation includes stable requestId, decimal expectedSequence and targeted commands. |
| `GET /activity?cursor=0&limit=50` | `scene:read`; numeric ascending durable sequence, redacted metadata and decimal next cursor, at most50events. |
| `GET /export-projection` | `scene:export`; current durable projection,25MiB document/26MiB response envelope. |
| `GET /export-assets/{fileId}?assetId={uuid}` | `scene:export`; current same-scene asset metadata, bounded raster bytes and authority recheck after storage I/O. No caller storage URL. |

Go signs exact mutation bytes through A4, then invokes the private Node writer without holding a Go authority transaction across HTTP. Node rechecks current authority under its writer locks. An acknowledged receipt means durable append and audit committed. Retry only identical bytes and requestId for an uncertain result; changed bytes/identity conflict. A stale sequence is an explicit conflict and is never silently rebased over human changes. The optional bridge being unavailable returns503 rather than an old snapshot. Go operation audit and the B2 durable receipt remain distinct from adapter-observed journals.

MCP adds native diagram catalog/schema resources, `generate_diagram`, `patch_elements`, `delete_elements`, `get_scene_activity` and `export_scene` alongside discovery/create tools. Nineteen typed diagram presets and the restricted native Mermaid subset emit editable elements; unsupported syntax returns a typed alternative. See the [catalog](../user/diagram-catalog.md) and [complete tool guide](../user/agents.md) for schemas/examples.

`POST /mcp/exports` supports the remote stdio export flow. `GET /mcp/artifacts/{id}` requires the same bearer principal and current independent export scope; the URL alone grants nothing. JSON/SVG/PNG artifacts have a captured decimal revision, five-minute expiry and bounded in-memory cache. Restart/expiry requires re-export. Both endpoints retain Origin checks and audit handling. Public scene/artifact links derive from explicit `WHITEBOARD_PUBLIC_URL`, never an internal API origin. See [renderer operations](../operations/mcp.md) for limits, fonts, process boundaries and release checks.

## V2 immutable scene versions and durable restore (implemented and wired; release pending)

Authenticated members use the Go API; public scene-session guests are denied
history. Responses use decimal-string sequences and metadata revisions.

| Method | Endpoint | Contract |
| --- | --- | --- |
| GET | `/api/v1/scenes/{id}/versions?cursor&limit` | `version:read`; newest first, limit 1–50, UUID cursor |
| POST | `/api/v1/scenes/{id}/versions` | `version:create`; `{requestId,expectedSequence,label?,pinned}`; immutable summary or 409 stale sequence |
| GET | `/api/v1/scenes/{id}/versions/{versionId}` | `version:read`; immutable canonical content and metadata |
| PATCH | `/api/v1/scenes/{id}/versions/{versionId}` | `version:create`; `{requestId,expectedRevision,label?,pinned?}`; label/pin only |
| GET | `/api/v1/scenes/{id}/versions/{versionId}/diff?againstSequence={sequence}` | `version:read`; stable element/app-state semantic diff at one exact sequence |
| POST | `/api/v1/scenes/{id}/versions/{versionId}/restore` | `version:restore`; `{requestId,expectedSequence}`; receipt plus `preRestoreVersionId` |

Go checks F1 policy and current scene authority, signs the canonical create
payload, then calls private Node `POST /internal/scenes/{sceneId}/versions`.
Node verifies the binding under the writer transaction and atomically stores
the exact document ID, sequence, content/hash, same-scene asset pins, safe event
and outbox row. Content is limited to 25 MiB. Identical retries return the
original version; changed retries are rejected.

Diff reads the bounded current Node projection and rejects a raced sequence.
Restore uses private Node `POST /internal/scenes/{sceneId}/version-restore` with a
signed hash over workspace, scene, target version, request and sequence. Under
the writer transaction Node rechecks policy/edit authority, validates content and
asset pins, stores one pinned pre-restore checkpoint, then commits one Yjs update,
receipt, mutation audit and safe events. The document UUID and stable element maps
remain unchanged; later elements become tombstones. A stale sequence writes
nothing. Identical retry returns the original receipt/checkpoint.

The owner-fenced checkpointer waits five dirty minutes and captures at most once
per scene per minute. R1 owns the 100 automatic-version retention policy;
manual versions are preserved until an owner configures a day limit, while
pinned versions are always preserved. See [ADR 0014](../decisions/0014-version-writer-ownership.md),
[human guide](../user/version-history.md), and [agent reference](../../src/frontend/public/docs/version-history.md).

## I2 workspace OIDC SSO and session assurance (implemented locally; wiring pending)

| Method | Endpoint | Contract |
| --- | --- | --- |
| POST | `/api/v1/sso/begin` | Strict `{connectionId,returnPath}`; creates a hashed, browser-bound 10-minute state/nonce/PKCE transaction and returns the provider authorization URL. |
| GET | `/api/v1/sso/callback` | Consumes state and provider code; redirects with only a 60-second opaque exchange code and safe relative return path. |
| POST | `/api/v1/sso/exchange` | Strict `{code}` plus HttpOnly browser binding; consumes once and returns the login result. |
| POST | `/api/v1/sso/link/begin` | Authenticated strict `{connectionId,returnPath,requestId}`; binds the current user and retry ID before the verified callback. |
| GET | `/api/v1/workspaces/{id}/sso` | Owner-only connection metadata; provider secret is never returned. |
| PUT | `/api/v1/workspaces/{id}/sso` | Owner-only exact-retry configuration with revision, explicit public-link policy, recent reauth and tested recovery before enforcement. |

OIDC identity is exact issuer plus subject. Verified email is bootstrap profile
data only and never auto-links an existing user. Member authority uses JWT
`sid`, `auth_method` and `auth_time`, then rechecks the current enterprise login
session and workspace connection. Missing `sid` is accepted only where password
access remains allowed. Room and mutation capabilities bind the source session
so renewal cannot outlive revocation. Recovery tokens additionally bind
`recovery_workspace_id`, expire after 15 minutes, and are accepted only for that
workspace while SSO is enforced. Link callback and exchange recheck their source
session and current connection under the workspace lock.

## M3 live presentation control plane (implemented locally; release verification pending)

Live presentation control traffic extends the single `/collab/` WebSocket.
Pending browsers authenticate with `presentation_auth`; admission upgrades that
same connection to a view-only audience grant before `ready` and scene sync.
The runtime never sends scene, roster, media or asset data to a pending socket.

`presentation_state` is the authoritative snapshot. Its integer `revision`
serializes frame ID, spotlight, viewport lock, participant removal/admission and
host transfer. Every durable host control carries the current revision as a
decimal string. A transfer target is re-authorized as a connected signed-in
editor immediately before the revision changes. Reactions are the only
unrevisioned presentation control: their fixed allowlist, per-connection
one-second limit and server-issued three-second expiry keep them ephemeral.
Link creation and revocation resolve the current runtime host revision before
the idempotent mutation and re-resolve the same host and revision immediately
before persistence. Host transfer or any intervening presentation revision
rejects the link mutation.

Audience state always strips the host's pending queue. Host disconnect or
current-authority loss ends the in-memory session; participant departure clears
a matching spotlight. The browser follows native frame IDs and treats a deleted
frame as an explicit missing target. View lock operates only on the camera and
never changes F1 scene authority.

Before the runtime sends a pending queue to its host, it freshly checks that
host's scene access and `presentation:host` feature, then binds the send to the
same attached host connection and exact session revision. Failure ends the
session before any pending identity is disclosed. If host start has no
authoritative state acknowledgement within ten seconds, the browser closes the
initiating collaboration socket so server detach clears a possibly accepted
session before retry.

The public `/present/{token}` route performs HTTP join and then opens the normal
collaboration transport with `presentation_auth`. Admission returns a private
audience token only to that socket. Credential loss unmounts the canvas and
requires a fresh waiting-room join. See the [user guide](../user/live-presentation.md)
and [machine-readable protocol reference](../../src/frontend/public/docs/live-presentation.md).
Clients apply presentation revisions monotonically and ignore late state for a
terminal session. A rejected pending socket authentication transitions to an
ended/error screen with an explicit fresh-rejoin action.

## R1 workspace retention (implemented locally; wiring pending, production activation disabled)

Owner-authenticated routes expose the current policy, a persisted dry-run
preview, explicit schedule, and holds under
`/api/v1/workspaces/{workspace}/retention`. Preview is non-destructive and binds
the canonical proposed policy plus current policy revision. Schedule and hold
requests require a UUID request ID; exact retries return the stored response,
while a changed principal, action, or body returns `409 REQUEST_ID_REUSED`.

Scheduled policies activate no sooner than seven days. The worker re-locks the
workspace and re-reads the current revision, feature flag, and holds. It removes
bounded activity/inbox metadata, unpinned automatic-version metadata beyond the
limit, optionally expires old unpinned manual versions, and optionally redacts
old comment bodies in place. A scene hold protects supported metadata for that
scene. Activity deletion cannot cascade-delete a notification that remains
inside its separate cutoff or under hold. It never targets pinned versions,
scene content, collaboration updates, receipts, tombstones, or object storage.
See the [operator runbook](../operations/enterprise-retention.md)
and [machine-readable reference](../../src/frontend/public/docs/retention.md).
