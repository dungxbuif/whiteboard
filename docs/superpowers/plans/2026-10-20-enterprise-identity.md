# Domain, SSO, SCIM and Retention Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add verified domain capture, workspace OIDC SSO and SCIM provisioning, plus explicit retention with protected asset lifecycle.

**Architecture:** Go owns identity/session assurance and provisioning under existing workspace locks. Feature worker executes policy preview/purge; Node's canonical Yjs history remains protected. SSO sessions carry current workspace-bound assurance to every authority decision.

**Tech Stack:** Go/PostgreSQL, existing OAuth2 plus reviewed go-oidc/v3, DNS TXT, SCIM 2.0 JSON, private S3 and feature jobs.

**Spec:** [Detail design](../specs/2026-10-20-enterprise-parity-design.md); [programme](2026-10-20-enterprise-parity.md).

## Global Constraints

Programme constraints apply; approved for implementation. Depends on F1–F3;
R1/R2 additionally depend on comments/version/library reference models.
Workspace is the enterprise boundary. No automatic email-based identity merge,
global SCIM deletion, canonical Yjs purge, receipt truncation or bucket-only expiry.
Domain challenge32 random bytes/24h; OIDC transactions10min/exchange60s;
SCIM count<=100; retirement grace7days. Password/default workspace flow persists.

## Review Focus

- DNS/issuer changes, redirects and SSRF: I1/I2 exact-host and egress tests.
- OIDC replay or existing-email takeover: I2 identity-link/nonce/PKCE tests.
- SSO enforcement/deprovision during active room/assets/jobs: I2/I3 authority tests.
- Last owner/manual membership/cross-workspace loss: I3 provenance tests.
- Retention policy/hold/reference/backup race and S3 partial failure: R1/R2 tests.

## Task I1: Domain verification and explicit workspace capture

**Files:** Create backend `internal/domain/workspace_identity.go`,
`internal/usecase/domain_ucase.go`, `internal/repository/postgres/domain_repo.go`,
`internal/repository/dns/txt_resolver.go`,
`internal/delivery/http/handler/domain_handler.go`, tests and migration pairs;
create frontend `src/components/enterprise/domain-settings.tsx`, tests;
modify settings route registration in App and dashboard owner navigation.

**Interfaces:** `DomainService.Begin(ctx,p,workspaceID,domain,requestID)
(DomainChallenge,error)`, `Verify(ctx,p,workspaceID,id,requestID) (Domain,error)`,
`Capture(ctx,p,workspaceID,id,requestID) (MembershipResult,error)`;
`TXTResolver.LookupTXT(ctx,name) ([]string,error)` injectable for tests.
Produces owner `/workspaces/{id}/domains` create/verify/revoke routes, explicit
capture endpoint and `/w/{workspace}/settings/identity` UI.

- [ ] Write `TestDomainExactIDNAChallenge`, `TestDomainExpiredReplayAndConflict`,
  `TestCaptureDoesNotMoveExistingData`, `TestDomainRequiresOwner`: exact
  normalized unique domain, `_whiteboard-verification` TXT, challenge expiry24h,
  consumed challenge cannot claim a second workspace, challenge value not logged.
- [ ] Run focused domain Go/frontend tests (RED). Implement DNS verification
  and domain discovery returning workspace login hints without listing users.
  Bind domain activation to explicit owner connection configuration; join/capture
  needs verified identity and authenticated acceptance, never DNS evidence alone.
- [ ] Verify fresh/migration13 schema, DNS failure/timeouts, ownership reverify,
  conflicting claims, IDN and parent/subdomain separation. Real production DNS
  publication is an operator input; fake resolver tests do not prove public TXT.
- [ ] Reconcile identity API/ERD/setup/public docs and commit
  `feat(identity): verify workspace domains and explicit capture`.

## Task I2: OIDC SSO, safe linking and revocable session assurance

**Files:** Create backend `internal/domain/auth_session.go`,
`internal/usecase/sso_ucase.go`, `auth_session_ucase.go`,
`internal/repository/oidc/provider.go`,
`internal/repository/postgres/{sso_repo,auth_session_repo}.go`,
`internal/delivery/http/handler/sso_handler.go`, tests/migrations;
modify `auth_ucase.go`, `middleware/auth.go`, room/capability/access services,
`cmd/userctl/main.go`, frontend auth-store/admin-login, public API client and
enterprise settings; add dependency only after licence/version audit.

**Interfaces:** `SSOService.Begin(ctx,connectionID,returnPath) (AuthorizationRedirect,error)`,
`Callback(ctx,state,code,browserBinding) (OneTimeExchange,error)`,
`Exchange(ctx,code,browserBinding) (LoginResult,error)`,
`Link(ctx,authenticatedUserID,externalIdentity,requestID) error`;
`SessionService.RequireWorkspaceAssurance(ctx,principal,workspaceID) error`,
`RevokeWorkspaceSessions(ctx,workspaceID,userID,connectionID) error`.
Login JWT adds `sid`; room/mutation capability claims bind original source
session/connection so stale SSO status is rechecked, not only login JWT expiry.
Existing no-sid JWTs can access password-allowed workspaces during transition;
they fail enforced-workspace access until a fresh SSO session.

- [ ] Add `TestOIDCStateNoncePKCEReplay`, `TestOIDCWrongIssuerAudienceExpiry`,
  `TestSameEmailCannotLinkAccount`, `TestIssuerDiscoverySSRFAndRedirect`,
  `TestPasswordSessionCannotEnterEnforcedWorkspace`,
  `TestSSORevokedSessionCannotRenewRoomOrAssets`, `TestBreakGlassOwnerRecovery`.
  Confirm valid flow uses issuer+subject, verified email, exact redirect, single
  10min transaction and60s exchange; no bearer credential in callback URL.
- [ ] Run focused auth/SSO Go tests (RED). Use official go-oidc verification and
  OAuth2 PKCE; do not re-enable retired Google endpoints. Store provider secret
  encrypted with backend-only key, one-time exchange hashed/browser-bound;
  session revocation fails closed on unavailable authority.
- [ ] Implement optional/enforced connection modes with recent owner
  reauthentication and working break-glass test before enforcement. Break-glass
  comes from operator CLI, cannot be provisioned via SCIM, and is audited.
  Keep password login usable for other workspaces; disable/capture rules are
  workspace-specific. Owner config must explicitly decide public-link policy;
  existing public links never inherit member SSO assurance.
- [ ] Add disposable OIDC-provider browser fixture for complete redirects,
  link/relogin/revoke/role checks; prove room, command, export, media and asset
  denial after revocation. Real IdP UAT requires its issuer/client/secret later.
  New accepted identities use existing transactional user/default-My-Workspace
  provisioning; repeated callbacks cannot create duplicate workspaces.
- [ ] Review auth/assurance ADR, migration/secret rotation/recovery runbooks,
  human and public setup; commit `feat(sso): add workspace OIDC and session assurance`.

## Task I3: SCIM Users/Groups, role provenance and deprovision

**Files:** Create backend `internal/domain/scim.go`,
`internal/usecase/scim_ucase.go`, `internal/repository/postgres/scim_repo.go`,
`internal/delivery/http/handler/scim_handler.go`, `scim_handler_test.go`,
`internal/delivery/http/middleware/scim_auth.go`, integration tests/migrations;
create owner SCIM token/settings UI and public SCIM schema/reference JSON files.

**Interfaces:** `SCIMService.CreateUser/GetUser/ListUsers/ReplaceUser/PatchUser/
DeleteUser`, corresponding group methods, `SCIMPage {schemas,totalResults,
startIndex,itemsPerPage,Resources}` and spec SCIM error/ETag models.
`SCIMCredentials` hashes dedicated per-workspace tokens with explicit revoke;
user/agent/login keys cannot authenticate SCIM routes. `MembershipSource` binds
connection+externalId/group and distinguishes manual grants.

- [ ] Add `TestSCIMDiscoveryPagingAndFilters`, `TestSCIMPatchAddReplaceRemove`,
  `TestSCIMIfMatchAndRetry`, `TestSCIMDisableRevokesOnlyManagedWorkspace`,
  `TestSCIMProtectsOwnerAndManualGrant`, `TestSCIMGroupNeverCreatesOwner`:
  startIndex1/count<=100, supported equality filters, invalid paths SCIM400,
  wrong ETag412, bulk false, token scope cross-workspace denial.
- [ ] Run scoped SCIM tests (RED). Implement RFC7643/7644 JSON resources,
  PUT/PATCH/DELETE, stable UUID mapping/unique externalId, soft-deprovision and
  atomic membership+session revoke+ACL bump+event. Effective managed role is
  strongest active mapped group editor/viewer; no group grants owner.
- [ ] Preserve independently manual memberships unless owner explicitly adopts
  them into SCIM management. Show retained manual access in deprovision results;
  never falsely report all access removed. Removed SCIM-only membership fails
  new requests and existing room refresh/media immediately via ACL invalidation.
- [ ] Run real HTTP recorded IdP-style provisioning sequences plus active-room
  deprovision and other-workspace preservation. Test last owner protection and
  default My Workspace idempotency. Reconcile IdP setup, SCIM discovery schemas,
  API/security docs; commit `feat(scim): provision workspace memberships safely`.

## Task R1: Retention policy preview, holds and metadata purge

**Files:** Create backend `internal/domain/retention.go`,
`internal/usecase/retention_ucase.go`, `internal/repository/postgres/retention_repo.go`,
`internal/delivery/http/handler/retention_handler.go`, tests/migrations;
register F3 handlers; create frontend `src/components/enterprise/retention-settings.tsx`
and tests; create `docs/operations/enterprise-retention.md`.

**Interfaces:** `RetentionService.Preview(ctx,p,workspaceID,PolicyInput)
(RetentionPreview,error)`, `Schedule(ctx,p,workspaceID,previewID,expectedRevision,
requestID) (RetentionPolicy,error)`, `SetHold(ctx,p,ref,versionID?,held,requestID) error`,
`PurgeBatch(ctx,now,limit) (PurgeResult,error)`.
Preview binds proposed policy hash+revision, counts and earliest destructive date;
all jobs re-read effective policy and holds under locks before mutation.

- [ ] Add `TestRetentionDefaultCutoffs`, `TestRetentionSevenDayActivation`,
  `TestHoldWinsAgainstClaimedPurge`, `TestPolicyRevisionCancelsStaleJob`,
  `TestRetentionNeverDeletesCanonicalHistory`: activity90days, notifications30,
  auto versions beyond100; manual/pinned/comments indefinitely unless configured,
  Trash hard purge disabled; no collab receipt/log/tombstone SQL deletion.
- [ ] Run focused real-DB retention tests (RED). Implement policy preview and
  explicit schedule, 7day delay, hold records, bounded metadata purge/idempotency,
  version-reference removal and preserved minimal purge audit. For any live
  comment retention, leave redaction markers/consistent thread pagination.
- [ ] Browser test preview/effect counts, hold/unhold/review policy, keyboard/
  mobile and restricted owner route. Dry-run worker report contains counts/IDs
  only, never raw comment/version content. Do not enable deletion on production.
- [ ] Reconcile ERD/retention/user/public docs and commit
  `feat(retention): preview and enforce workspace retention policies`.

## Task R2: Reference-aware object lifecycle and backup protection

**Files:** Create backend `internal/domain/asset_reference.go`,
`internal/usecase/asset_gc_ucase.go`,
`internal/repository/postgres/asset_reference_repo.go`, tests/migrations;
modify asset repository/metadata writers, V1/L2 pin transactions and worker;
extend backup/restore tooling and E1 packaging fixtures.

**Interfaces:** `AssetReferences.Pin/Unpin(ctx,workspaceID,assetID,sourceKind,
sourceID) error` shares the source transaction;
`AssetGC.MarkCandidates(ctx,now,limit)`, `DeleteDue(ctx,now,limit) (GCResult,error)`;
`BackupProtection.Publish(ctx,manifestID,assetIDs,expiresAt) error`.
Per-object lifecycle is `live -> pending_delete -> deleted`; any valid new pin
under the same lock cancels pending deletion; grace7days and current hold/
policy/backup/reference validation precede S3 DeleteObject.

- [ ] Add `TestAssetGCBlockedBySceneVersionLibraryAndBackup`,
  `TestNewPinCancelsPendingDeletion`, `TestDeleteCrashRetryIdempotent`,
  `TestBackupRestoreIncludesProtectedObjects`, `TestCrossWorkspaceGCIsolation`.
  Assert source references block deletion, failed delete retry keeps metadata,
  shared object ownership cannot bypass tenant checks and held content survives.
- [ ] Run scoped S3/DB integration tests (RED). Implement DB lock ordering and
  conservative reference accounting, staged-copy cleanup, per-object lease and
  idempotent tombstone; publication of new refs must serialize with deletion.
  If reference inventory cannot be proved complete, skip deletion with an alert.
- [ ] Verify immutable DB+S3 backup manifest and restore, reference counts,
  version/library exports after purge, worker kill/restart and bucket lifecycle
  inventory. No bucket rule may expire objects still pinned by the application.
- [ ] Update Homelab storage/backup/worker runbook during E2 rollout and
  public retention limits; commit `feat(storage): protect referenced assets during retention`.
