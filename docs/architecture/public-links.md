# Public links and room sessions (Collaboration v2 A3)

Status: implemented and locally verified on merged master; production remains
v1 until VM100 cutover. Trace:
[programme](../superpowers/plans/2026-10-16-collaboration-v2.md),
[phase](../work/phases/PHASE-COLLAB-V2.md), [ACL
decision](../decisions/0009-collaboration-access.md).

## Decision and implementation

Owners create independent view/edit links per active scene. New raw tokens are 32 random bytes encoded base64url and returned once; PostgreSQL stores SHA-256 hex digests. Lists contain only link ID, access, expiry and active status. Passwords are bcrypt hashes; password input travels only in POST JSON, never query strings. Edit exchange requires a trimmed display name (1–100 characters). Public principals are `guest:<random>` and never authenticated user UUIDs.

Migration 000008 copies every legacy scene_permissions row with access=view, preserves password hash and active state, and keeps legacy token storage during the compatibility stage. Legacy writes synchronize only their mapped link, so independent links remain independent. Revoking a migrated link disables its legacy lookup too. Services acquire the workspace row lock before link mutations. Database triggers bump durable ACL version, append audit, and issue commit-bound NOTIFY in that transaction. A monotonic bigint link revision advances on every update, so rapid changes cannot collide. Down refuses multiple links or new hash-only/edit/expiring links that legacy storage cannot represent; it never silently discards grants.

Public POST /api/v1/public/scene-session accepts token/password/displayName. The session is scene scoped and expires within ten minutes. POST /api/v1/scenes/{id}/room-token accepts a login or scene-session token; every call checks current scene/workspace/link or membership under the workspace row lock. No public principal receives workspace list, management, folder or user authority. Refresh uses the same endpoint and revalidates current state. For a public principal it returns a renewed sceneSession alongside the room token; the client replaces both before expiry. Guest identity/name remain stable, and both expiries are capped to link expiry. Expired scene sessions must re-exchange the link.

Room claims: HS256 only, iss=whiteboard-api, aud=whiteboard-collab, purpose=room; sub=principalId; principalType=user|public; principalId, grantId, displayName, workspaceId, sceneId, access=view|edit, aclVersion as decimal string, public grantVersion as a decimal bigint link revision, iat, exp (ten minutes), jti. Scene sessions use aud=whiteboard-scene-session and purpose=scene-session with matching scoped claims. Dedicated COLLAB_JWT_SECRET is required (minimum 32 bytes, distinct from JWT_SECRET); no fallback. Login middleware rejects scoped-purpose/issuer/audience tokens even when their subject parses as a UUID. User grantId is `membership:<workspace UUID>:<user UUID>`; public grantId is the share-link UUID. Public AccessService resolution requires the current revision, matching scene/workspace and active/unexpired link; it never uses UserID. Password changes invalidate prior sessions through the link revision.

## Alternatives and integration

A shared mutable scene token was rejected because revocation must be independent. Public UserID substitution was rejected because it could inherit membership or owner privileges. Public resolution uses an optional repository capability and fails closed without it. Room signing checks canonical UUID references under the existing workspace-first transaction; it returns only after commit.

A2 handed off router/main/API/user documentation after its snapshot. A3 wires existing constructors through optional sharing/room handlers without changing A2 invitation routes. Central context/backlog/validation/releases and browser UAT belong to the controller/C2/E3. This Markdown is the stable agent-readable contract until public guides ship.

## Verification plan

Real disposable PostgreSQL proves legacy active/inactive/password preservation, independent revocation, audit/version atomicity and rollback refusal. Unit and HTTP tests prove view/edit scope, required display name, current-state refresh, secret-free lists, purpose separation and wrong alg/aud/issuer/expiry denial. Target command in src/backend: `rtk go test ./internal/usecase ./internal/repository/postgres ./internal/delivery/http/... -run 'Public|RoomToken|RevokeActiveGrant|MigratePublicLinks' -count=1`, with ACCESS_TEST_DATABASE_URL set to guarded loopback *_test database. Full backend suite follows integration. No production mutations.

## Verified implementation details

Public errors use `{code,message,requestId}`; 400 malformed input, 401 password/session challenge, 403 prohibited operation, 404 inaccessible/revoked/expired scope, 413 oversized body, 503 unavailable authority. DTOs reject unknown fields and trailing JSON, and cap sharing JSON at 4 KiB. New credential responses set Cache-Control: no-store. Legacy protected scene reads use POST JSON at the existing path; GET ignores password queries. Production legacy reads also use the current public resolver under a workspace lock.

Credential SQL operations suppress GORM parameter logging. Router middleware removes the entire legacy query before downstream handlers/logging and replaces the token in RequestURI. URL.Path remains for routing; reverse-proxy/access tracing must separately redact that legacy path at release. Never log raw Authorization or JSON credentials.

The first full-suite run caught the new revision ORM field defaulting to zero instead of the database default one. The explicit GORM default now preserves the positive revision constraint. Final command/count evidence is in the A3 execution report and central verification record. Browser public-editor UI, collaboration-runtime disconnect behavior, proxy configuration and rollout remain C2/B/E integration work.

Refresh checks credential expiry again after the workspace lock is acquired;
`Principal.CredentialExpiresAt` and `SceneSessionIdentity.ExpiresAt` retain the
verified JWT expiry for downstream capability bounds. Link expiry queries use
PostgreSQL clock_timestamp(), not transaction-start now(), so a transaction
that waits or runs across expiry cannot keep an expired link valid.

## Guest comment policy

`allowComments` is independent from link `access`. A view link may comment and
an edit link may remain comment-disabled. Both the workspace
`commentsEnabled` policy and the link's current `allowComments=true` revision
must hold for guest comment reads and writes. The comment route parses the
existing scene-session token, binds the URL scene and optional workspace header
to its exact scope, then `FeatureService` rechecks the active, unexpired link
and current revision under the workspace lock. Revocation or a revision change
therefore denies a mutation that was prepared from an older session before any
comment, receipt, subscription or event commits.

The request body contains only comment input; it never accepts guest identity
or display name. Author identity and the safe display-name snapshot come from
the verified session. Guests are not inserted into `users`, cannot subscribe
to a private inbox, and never gain canvas edit authority from comment
permission. Comment bodies remain in comment storage and authenticated API
responses; realtime invalidations and workspace events contain cursor/ID
metadata only.
