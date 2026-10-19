# ADR 0009: Database-backed collaboration access

Status: approved; A1 implemented and verified locally, pending review/deployment (2026-10-16).

Trace: [phase](../work/phases/PHASE-COLLAB-V2.md), [programme](../superpowers/plans/2026-10-16-collaboration-v2.md), [API](../architecture/API.md), [ERD](../architecture/ERD.md).

Workspace is the only organization boundary. Workspace owners manage membership, invitations, sharing and scene overrides. Editors edit and viewers read. Scene overrides replace the member content role, including editor-to-viewer downgrade, but never remove ownership or authorize management. Owners remain owners; owner membership cannot be removed/demoted through these endpoints. Removal deletes dependent overrides.

Go resolves normalized user identity from JWT and current database state, then Casbin evaluates the effective role and action. UUIDs remain primary keys; public aliases use NanoID-compatible 21-character cryptographically random URL-safe identifiers. Identifiers are not credentials. Public/agent identity must resolve through a scoped grant; A1 fails closed for these kinds until their owning tasks integrate them.

One workspace `acl_version` fences all scene authority. Mutations lock workspace before scene; authorization and authorized HTTP writes share a transaction. Membership/override changes bump the durable version, insert audit metadata, and emit `acl_changed` notifications. Realtime consumers must compare the version under the same workspace lock before committing an append.

Deleted scene content is inaccessible. Owners may list trash metadata and restore deleted resources; other roles cannot. Folder mutation must preserve every descendant's effective scene access.

Verification plan: access matrix, real PostgreSQL migration/backfill and alias round trips, current-member listing/search isolation, protected descendants, owner protection, durable ACL/audit changes. No production database will be used.

## Verified implementation and provenance

`NewPublicID` ports the exact alphabet, default length 21 and six-bit masking from `/Users/dungxbuif/workspace/devtool/references/server/src/common/lib/nanoid.ts` (used by `references/server/src/common/database/postgre/base.entity.ts`). Go uses `crypto/rand`; it does not copy the JavaScript buffer pool. Existing aliases are optional and UUID rows are retained. This corrects the older API note that reported no app-owned helper.

Production loads the Casbin model from the existing required `CASBIN_MODEL_PATH` and builds a fixed role/action matrix from current database role results; there is no root bypass or persisted user-to-role policy cache. Alias lookup is followed by ACL lookup, and a canonical scene UUID is placed in context for handlers.

PostgreSQL repositories use the transaction bound by `WithWorkspaceLock`; write/permission checks and descendant checks therefore serialize against owner revocation. The response is buffered until commit. Failed handlers roll back. `collab_bump_acl` is shared with later grant implementations. User/workspace soft-delete changes also advance authority versions.

Actual evidence: original migration/list assertions failed for member mapping, missing owner membership, and missing member workspaces; alias integration exposed Chi retaining original route parameters and was corrected with canonical context. Cross-scene asset reading and missing soft-delete grant invalidation had dedicated failing regressions. Final full suite: 79 pass, with both DATABASE_URL and ACCESS_TEST_DATABASE_URL set to a local disposable PostgreSQL database; no integration skips. The two-connection test holds authority, observes a lock timeout on concurrent revocation, rolls back a scene write, then revokes and proves subsequent access denied.

Public/agent grant resolvers, invitations, room tokens, activity API, and v1 realtime replacement remain their owning A2–A4/B tasks. Effective-access DTOs for C2 also require an explicit response contract; this task exposes aliases/version while keeping response compatibility. Public guides and agent-readable rollout docs are reconciled by C2/E3 alongside the complete shipped flow; this branch is not deployed.

## A1 review fix intent: stalled request bodies

Review found that authority was locked before JSON decoders consumed client-controlled request bodies. A viewer could stall an asset request and hold the workspace lock, blocking owner revocation. The verified fix reads authenticated scoped bodies with a byte cap and network read deadline before obtaining authority, then replays the buffered body inside the existing authorization/mutation transaction. Preserve the 25 MiB materialized-document allowance by permitting a 26 MiB scene request envelope; cap other metadata bodies at 1 MiB. Scope deadlines to these HTTP bodies so Socket.IO/WebSocket upgrades are unchanged. Verify with a real socket stalled during upload, concurrent owner revocation, timeout/size errors, and the existing atomicity test.

Review-fix proof: real TCP client sends incomplete asset JSON while a viewer; concurrent HTTP owner revocation finishes within a one-second client bound, and the stalled request returns 408 after its ten-second read deadline. Before the fix the same owner request timed out. Scoped regression suite: 14 passed, including byte caps (known and chunked lengths), an accepted 25 MiB-plus-envelope body, deadline reset, unsupported-transport failure, and existing DB authority lock/rollback checks. No whole-suite rerun was needed for this bounded middleware change.
