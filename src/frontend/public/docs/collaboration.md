# Workspace invitations — Collaboration v2

Status: implemented on the development branch; deployment and browser invitation controls are pending.

Owners create an email-bound editor or viewer invitation and copy the `/invite/:token` link to the recipient. The app sends no email. A link works once and expires exactly seven days after creation; accepting at the expiry timestamp fails. Owners can list pending metadata and revoke invitations. A list cannot recover the raw link.

Existing account: log in with the invited email, then accept. New account: register with a matching email, name and password; the account, personal workspace and invited membership commit together. Email is trimmed and lowercased. Passwords require at least eight characters and at most 72 UTF-8 bytes. A typed email and link do not establish verified email ownership. Existing passwords are never changed by invitation registration. A lower-role invitation never reduces an existing membership.

## Agent-readable API

All paths begin `/api/v1`. Login JWTs remain required for authenticated routes. Workspace route IDs accept UUIDs or public aliases; invitation management takes workspace scope from the path.

| Method and route | Authorization | Request | Success |
| --- | --- | --- | --- |
| POST `/workspaces/{id}/invitations` | Owner | `{email,role:"editor"\|"viewer"}` | 201 `{invitation,inviteUrl}` |
| GET `/workspaces/{id}/invitations` | Owner | none | 200 pending metadata array |
| DELETE `/workspaces/{id}/invitations/{inviteId}` | Owner | none | 204 |
| POST `/invitations/accept` | Login | `{token}` | 200 `{workspaceId,userId,role,createdAt}` |
| POST `/invitations/register` | Public, valid invite | `{token,email,password,name}` | 201 User; then use existing password login |

Metadata: `id`, `workspaceId`, `email`, `role`, `inviterId`, `createdAt`, `expiresAt`, optional `acceptedAt`/`revokedAt`. Pending listing excludes consumed, revoked and expired invitations. Only creation returns a raw token URL. No API returns a token hash or password hash.

Errors are `{code,message,requestId}`. 400 invalid input/link or generic registration rejection; 401 login required; 403 wrong email or prohibited operation; 404 inaccessible workspace; 410 expired invitation; 413 oversized input; 408 stalled input; 503 storage/authority unavailable. Registration errors do not disclose whether an account exists. DTOs reject unknown fields and trailing JSON; JSON body limit is 4 KiB, with a ten-second prebuffer deadline before authority locking.

Runtime `FRONTEND_URL` must be an HTTP(S) origin without credentials, a path prefix, query or fragment; optional trailing slash is accepted.

## Public scene links and room sessions (A3, release pending)

Owner routes require login JWT and X-Workspace-ID. IDs accept UUIDs or public aliases.

| Method and route | Input | Output |
| --- | --- | --- |
| GET `/scenes/{id}/share-links` | none | `{id,access,active,expiresAt}[]` |
| POST `/scenes/{id}/share-links` | `{access:"view"\|"edit",password?,expiresAt?}` | `{id,access,active,expiresAt,token}`; raw token once |
| DELETE `/scenes/{id}/share-links/{linkId}` | none | 204 |
| POST `/public/scene-session` | `{token,password?,displayName?}` | SceneSession |
| POST `/scenes/{id}/room-token` | bearer login or scene-session | `{token,expiresAt,sceneSession?}` |
| POST `/public/scenes/{token}` | legacy `{password}` | read-only scene |

SceneSession: `{token,expiresAt,workspaceId,sceneId,access,principalId,displayName}`. Public access is exactly one scene. Edit guests require a trimmed 1–100 character display name. Passwords require at least six characters and at most 72 UTF-8 bytes. Expiry is a future RFC3339 timestamp. Link metadata never exposes credentials or hashes. Legacy GET ignores password query parameters; use POST JSON.

Room refresh validates current membership/link and scene/workspace status. For guests, it returns a renewed sceneSession with the same guest ID; replace both credentials before expiry. Credentials last at most ten minutes and never outlive link expiry. Expired sessions require link/password exchange. Password/link changes invalidate old sessions. A room token is not a login or scene-session token.

Room JWT: HS256 only; iss whiteboard-api; aud whiteboard-collab; purpose room; sub=principalId; principalType user|public; principalId, grantId, displayName, workspaceId, sceneId, access view|edit, aclVersion decimal string, iat/exp/jti. Public grantVersion is the decimal bigint link revision and principalId starts guest:. User grantId is membership:<workspace UUID>:<user UUID>; public grantId is the link UUID. Scene-session aud is whiteboard-scene-session and purpose scene-session. Go and collab require dedicated COLLAB_JWT_SECRET, at least 32 bytes and distinct from JWT_SECRET.

Errors retain `{code,message,requestId}`: 400 invalid input, 401 password/session challenge, 403 forbidden, 404 inaccessible/revoked/expired resource, 413 oversized JSON, 503 unavailable authority. Body limit is 4 KiB. Clients must keep public-session credentials separate from their login credential and must not log them. Browser sharing/editor controls and runtime integration are pending.

Browser credential isolation: public scene reads use POST `/public/scenes/{token}` with JSON `{password?}`, no account Authorization header and no password query. A public401 does not clear `auth_token`. Scene-session credentials remain separate from account login credentials. Locally verified; deployment pending.
