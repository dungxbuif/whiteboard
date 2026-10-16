# A — Access and sharing Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make workspace/scene permissions authoritative across HTTP, collab and agents.

**Architecture:** Extend Go Repository → Usecase → Handler and GORM migrations; use Casbin for action checks. Persist grants and audit in PostgreSQL; notify only IDs/versions.

**Tech Stack:** Go, GORM, Casbin v3, PostgreSQL, JWT, bcrypt.

**Spec:** [Approved collaboration design](../specs/2026-10-15-collaboration-v2-design.md); approved in conversation by “xác nhận”; implementation authorized by the user on 2026-10-16.

## Global Constraints

- Read [programme contracts](2026-10-16-collaboration-v2.md) before executing; its interface types, limits, dependency order and documentation requirements apply to every task.
- Custom Node/TypeScript WebSocket service; RelayHub is not involved.
- Workspace roles are `owner`, `editor`, and `viewer`; scene overrides are `editor` or `viewer`; management is owner-only.
- Invitation links: single-use, email-bound, seven-day expiry. Room JWT: ten minutes.
- Update frame: 1 MiB; materialized document: 25 MiB; 120 updates/minute/principal.
- Snapshot: first of 100 updates or five seconds. Voice: ten active participants; no product-level data-participant cap.
- Dashboard follows `references/devtool` styling; reuse its original colors and tokens without a monochrome override. Scene components retain native library styling.
- Retain TypeScript 7 toolchain, password login, default workspace provisioning, and OpenAI-compatible provider boundary.
- Every task updates its affected docs before implementation and reconciles them with verified behavior before commit.

## Review Focus

- Viewer uses PATCH/search/assets to bypass read-only status — A1.
- Two accepts consume one invitation — A2.
- Public password/token ends up in URLs or logs — A3.
- Scene file ID from a different scene in the same workspace leaks data — A4.
- Revocation overlaps a durable write or upload grant — A4/B2.

---

### A1: Membership policy and migration

**Files:** Create `src/backend/migrations/000007_collab_access.{up,down}.sql`, `internal/domain/access.go`, `internal/usecase/access_ucase.go`, `internal/repository/postgres/access_repo.go`, `internal/usecase/access_ucase_test.go`, `internal/repository/postgres/access_migration_test.go` (Go paths under `src/backend/`). Modify `domain/workspace.go`, `configs/casbin_model.conf`, workspace repository, tenant/authorize middleware, scene/folder/search handlers, router and main.

**Interfaces:** Produce programme `AccessService.Resolve`/`RequireOwner`; add `NewPublicID() (string,error)` in `internal/domain/public_id.go`, matching the old 21-character URL-safe NanoID behavior. Optional unique `public_id` aliases for workspaces/scenes preserve UUID PKs and old links; lookup resolves either alias or UUID before ACL.

- [x] **Step 1 — Record intended behavior:** update `docs/architecture/API.md`, `docs/architecture/ERD.md`, `docs/decisions/0009-collaboration-access.md` with this task's API/data/flow decision and the cases below.
- [x] **Step 2 — Add failing proof:** `TestResolveAccessMatrix`: owner edit/manage; viewer read only; editor cannot manage; scene override wins; removed member override denies. `TestAccessMigration`: member→editor, existing UUID links retained, inactive/deleted resources denied; aliases unique and lookup round-trips. `TestWorkspaceListAndSearchFilter`: no cross-workspace or forbidden scene results; folder move/delete cannot mutate a protected descendant.
- [x] **Step 3 — Verify failure:** run `rtk go test ./internal/usecase ./internal/repository/postgres ./internal/delivery/http/... -run 'Access|WorkspaceListAndSearchFilter' -count=1`; confirm the new assertion fails for missing behavior, not a broken fixture.
- [x] **Step 4 — Implement:** Add membership and scene override constraints, `acl_version` and audit event storage. Migrate active legacy shares in A3, not here. Backfill owners if absent; member→editor. Build Casbin policy using normalized authenticated principal and database-backed effective role. Add owner management routes from programme and reject self-demotion/last-owner removal. Public IDs are identifiers, never access capabilities. Record which utility code was reused and its provenance.
- [x] **Step 5 — Verify:** rerun `rtk go test ./internal/usecase ./internal/repository/postgres ./internal/delivery/http/... -run 'Access|WorkspaceListAndSearchFilter' -count=1`; require all listed cases to pass. Record actual results in the phase verification record; skipped integration tests do not count as passes.
- [x] **Step 6 — Reconcile and commit:** reconcile `docs/architecture/API.md`, `docs/architecture/ERD.md`, `docs/decisions/0009-collaboration-access.md`; stage only the named files and supporting lockfiles; commit `feat: enforce workspace and scene role policy`.

### A2: Atomic invite acceptance and password onboarding

Required boundary assertions in the invitation fixture:

```go
require.Equal(t, 7*24*time.Hour, invite.ExpiresAt.Sub(invite.CreatedAt))
require.Error(t, acceptAt(invite.ExpiresAt))
require.Equal(t, 1, successfulConcurrentAccepts)
require.Equal(t, originalPasswordHash, reloadedExistingUser.PasswordHash)
```

**Files:** Create `migrations/000011_workspace_invitations.{up,down}.sql`, `internal/domain/invitation.go`, `internal/usecase/invitation_ucase.go`, `internal/repository/postgres/invitation_repo.go`, `internal/delivery/http/handler/invitation_handler.go` and matching `_test.go`; modify user provisioning usecase, `cmd/userctl/main.go`, router/main. Migration 11 is reserved for this omitted invitation table; 8–10 remain reserved for sharing, collab documents and agent keys. All migrations ship together in v2; disposable fixtures rebuild from the complete available set.

**Interfaces:** `CreateInvitation(ctx,owner,workspace,email,role) (InviteResult,error)`; `AcceptInvitation(ctx,user,token) (Membership,error)`; `RegisterWithInvitation(ctx,token,RegistrationInput) (User,error)`; `EnsureDefaultWorkspace(ctx,tx,userID) error` shared with userctl.

- [x] **Step 1 — Record intended behavior:** update `docs/architecture/API.md`, `docs/user/collaboration.md` (create) with this task's API/data/flow decision and the cases below.
- [x] **Step 2 — Add failing proof:** `TestInviteExpiryAndEmail`: expires exactly at seven days, lowercase/trim email normalization, wrong user rejected. `TestConcurrentInviteAccept`: one successful consume. `TestRegisterInviteAtomic`: account, personal workspace and invited membership either all commit or none; existing email requires login and password hash never changes.
- [x] **Step 3 — Verify failure:** run `rtk go test ./internal/usecase ./internal/repository/postgres ./internal/delivery/http/handler -run 'Invite|RegisterInvite' -count=1`; confirm the new assertion fails for missing behavior, not a broken fixture.
- [x] **Step 4 — Implement:** Generate cryptographic tokens, store SHA-256 only, bind role/email/workspace, hash passwords with existing policy. Lock invitation row for consume; do not downgrade existing membership via a lower-role invite. Return copyable invite URL only on creation; owner can list metadata/revoke. Never claim email ownership verified merely from typed email. No outbound email.
- [x] **Step 5 — Verify:** rerun `rtk go test ./internal/usecase ./internal/repository/postgres ./internal/delivery/http/handler -run 'Invite|RegisterInvite' -count=1`; require all listed cases to pass. Record actual results in the phase verification record; skipped integration tests do not count as passes.
- [x] **Step 6 — Reconcile and commit:** reconcile `docs/architecture/API.md`, `docs/user/collaboration.md` (create); stage only the named files and supporting lockfiles; commit `feat: add single-use workspace invitations`.

### A3: Independent public view/edit links and room sessions

**Files:** Create migration `000008_share_links.up.sql` and `.down.sql`, `internal/domain/share_link.go`, `internal/usecase/share_link_ucase.go`, `internal/repository/postgres/share_link_repo.go`, `internal/delivery/http/handler/room_handler.go` and tests; modify sharing handler, domain permission compatibility adapter and router.

**Interfaces:** Produce `ExchangePublicSession(ctx,token,password,displayName) (SceneSession,error)` and `MintRoomToken(ctx,principal,SceneRef) (RoomToken,error)`; use exact programme claims. List returns IDs/access/expiry only, raw secret once on create.

- [x] **Step 1 — Record intended behavior:** update `docs/architecture/API.md`, `docs/user/collaboration.md`, `docs/operations/DEPLOYMENT.md` with this task's API/data/flow decision and the cases below.
- [x] **Step 2 — Add failing proof:** `TestMigratePublicLinks`: existing active links still resolve as view, password hash preserved, inactive stay inactive. `TestPublicSessionScope`: view rejects updates, edit requires display name, token cannot select another scene. `TestRoomTokenClaims`: wrong alg/aud/issuer/expiry rejected. `TestRevokeActiveGrant`: version increment + audit committed together.
- [x] **Step 3 — Verify failure:** run `rtk go test ./internal/usecase ./internal/repository/postgres ./internal/delivery/http/... -run 'Public|RoomToken|RevokeActiveGrant|MigratePublicLinks' -count=1`; confirm the new assertion fails for missing behavior, not a broken fixture.
- [x] **Step 4 — Implement:** Hash old share tokens in a staged migration; retain legacy lookup compatibility until retirement. Remove new password-in-query use and redact old route query logs during transition. Expose multiple independently revocable links and owner-only scene override/share routes. Refresh checks current membership/link state. Publish ACL change after commit; log no secret. Down migration refuses to collapse multiple links into one.
- [x] **Step 5 — Verify:** rerun `rtk go test ./internal/usecase ./internal/repository/postgres ./internal/delivery/http/... -run 'Public|RoomToken|RevokeActiveGrant|MigratePublicLinks' -count=1`; require all listed cases to pass. Record actual results in the phase verification record; skipped integration tests do not count as passes.
- [x] **Step 6 — Reconcile and commit:** reconcile `docs/architecture/API.md`, `docs/user/collaboration.md`, `docs/operations/DEPLOYMENT.md`; stage only the named files and supporting lockfiles; commit `feat: add scoped public editing and room tokens`.

### A4: Asset capabilities and mutation authority

**Files:** Create `internal/domain/capability.go`, `internal/usecase/capability_ucase.go`, `internal/delivery/http/handler/capability_handler.go` and tests; modify asset usecase/repository/handler, realtime config handler, router/main.

**Interfaces:** Produce `AuthorizeCommand(ctx,principal,SceneMutation) (SignedGrant,error)`; `CreateAssetCapability(ctx,principal,SceneRef,[]AssetUploadRequest) (AssetGrant,error)`; internal authority service returns current ACL version with a bounded decision. Grant binds scene, principal, requestId, payloadHash and action.

- [x] **Step 1 — Record intended behavior:** update `docs/architecture/API.md`, `docs/architecture/ERD.md`, `docs/operations/ENVIRONMENTS.md` with this task's API/data/flow decision and the cases below.
- [x] **Step 2 — Add failing proof:** `TestSceneAssetIsolation`: foreign scene file IDs rejected even in same workspace. `TestPublicUploadCapability`: view denied, edit allowed only target scene, expired/revoked denied, file size/MIME validated. `TestAuthorityUnavailable`: no mutation grant on authority failure. `TestRevokeRace`: mutation cannot commit after revoked access under shared lock/version protocol.
- [x] **Step 3 — Verify failure:** run `rtk go test ./internal/usecase ./internal/delivery/http/handler ./internal/repository/postgres -run 'SceneAssetIsolation|PublicUploadCapability|Authority|RevokeRace' -count=1`; confirm the new assertion fails for missing behavior, not a broken fixture.
- [x] **Step 4 — Implement:** Use transactionally current ACL. Scope presign-view query by workspace AND scene. Public upload audit uses guest principal with nullable user UUID. Capability TTL cannot exceed parent room session. Mint TURN REST credentials after scene access check; never expose shared TURN secret. Internal endpoints require service authentication and are not publicly proxied. Document that already-issued object-store signed URLs last until expiry; use short TTL rather than claiming immediate invalidation.
- [x] **Step 5 — Verify:** rerun `rtk go test ./internal/usecase ./internal/delivery/http/handler ./internal/repository/postgres -run 'SceneAssetIsolation|PublicUploadCapability|Authority|RevokeRace' -count=1`; require all listed cases to pass. Record actual results in the phase verification record; skipped integration tests do not count as passes.
- [x] **Step 6 — Reconcile and commit:** reconcile `docs/architecture/API.md`, `docs/architecture/ERD.md`, `docs/operations/ENVIRONMENTS.md`; stage only the named files and supporting lockfiles; commit `feat: bind uploads and internal mutations to scene grants`.
