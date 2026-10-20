# Enterprise L1 collections verification — 2026-10-20

## Intended behavior

Collections are named scene lists that never move a scene and never grant scene
access. Workspace collections are visible to current workspace members; private
collections are visible only to their creator. Workspace owners may delete a
private collection for moderation without receiving its name, contents, or
scene counts. Editors and owners may create and manage collections while the
workspace collection feature is enabled.

The canonical dashboard route is
`/w/{workspace}/collections/{collection}`. Dashboard styling follows the
existing legacy surface and collection navigation remains separate from the
folder tree.

## Technical decision

Migration `000022` adds `collections` and `collection_scenes` with composite
workspace foreign keys, revision guards, stable integer positions, and no
cascade from a collection membership to a scene. Workspace-scoped idempotency
uses the existing `feature_mutation_receipts` table with
`collection:manage`; collection writes execute under the existing workspace
authority lock and recheck feature policy and current membership before commit.

List/detail SQL filters private visibility and current scene ACL before counts,
ordering, and pagination. The API never returns inaccessible scene identifiers.

## Verification plan

- RED/GREEN unit cases for private visibility, owner moderation delete, no
  access grant, foreign-workspace rejection, revision conflicts, stable order,
  and idempotent retries.
- Apply migration `000022` in the guarded disposable PostgreSQL schema and run
  repository/usecase/handler tests.
- Frontend unit and real-stack coverage for legacy styling, create/update/delete,
  add/remove/reorder, search, keyboard/mobile use, and deep-link reload.
- Run full guarded Go tests, frontend tests/build, `go vet`, and `git diff --check`.

## Evidence

RED was observed independently for the usecase contract (undefined collection
domain/service), HTTP handler (undefined handler), collection components
(missing modules), and collection-name filtering (missing accessible filter).
Each focused test was then made GREEN before the next layer was added.

- `go test ./internal/usecase ./internal/delivery/http/handler`: 198 tests pass.
- Guarded `go test ./internal/repository/postgres -run TestCollection -count=1`:
  privacy, moderation delete, composite references, stable reorder, conflict,
  exact retry/changed reuse and migration up/down pass on PostgreSQL.
- Guarded `go test ./...`: all backend packages pass; `go vet ./...` reports no
  issues.
- `npm run test:unit --workspace ./frontend -- --run`: the pre-concurrency gate
  passed 41 files and 153 tests; collection API regressions prove timeout retry,
  definitive response clearing, and explicit cancel.
- `npm run build --workspace ./frontend`: TypeScript and the production Vite
  build pass.
- Real-stack Playwright `enterprise-collections.spec.ts`: one browser scenario
  passes on a 390x844 viewport, covering keyboard create, filter, deep-link
  reload, stable membership removal without folder movement, private denial and
  viewer control hiding. The review rerun also verifies creator capabilities,
  noncreator privacy controls and structured malformed-limit rejection.

Production router/server composition is intentionally owned by the programme
integrator. Commit `71163a5` wires `CollectionService`, event auditing,
`CollectionHandler`, and authenticated collection routes; its focused route
test passed before the real-stack run above.

## Review corrections

The fresh L1 review found four contract gaps. The corrected behavior is:

- Reorder treats the currently accessible scene subset as the complete ordered
  input. Under the workspace lock it prunes stale membership rows whose scenes
  are deleted or no longer visible to the current principal, then validates and
  reorders the remaining exact set. Hidden rows cannot permanently block edits.
- Remove requires current scene access before checking collection membership and
  repeats that check under the workspace lock. Guessed inaccessible scene IDs
  therefore cannot reveal whether a membership row exists.
- Collection responses expose `mine` and `canSetPrivate` capabilities without a
  creator identifier. A noncreator editor may manage a workspace collection but
  cannot convert it to private; the edit dialog does not offer that option.
- A malformed collection page limit is an invalid request and returns structured
  `400 INVALID_INPUT` rather than entering generic infrastructure error mapping.

Regression proof adds a real PostgreSQL collection with live and soft-deleted
members, verifies filtered Get/List counts and pagination, prunes the hidden row
during subset reorder, verifies removal authorization before membership lookup,
and verifies revoked workspace access reveals no collection data.

## Receipt and retry hardening

Fresh review requires collection receipts to remain idempotency records rather
than authorization snapshots. Every receipt replay stays inside the workspace
lock, repeats current collection visibility and relevant scene checks, and
rebuilds collection response content through current ACL filtering. Replays for
deleted/private/revoked resources fail without returning stored content.

Migration `000022` binds `collection:manage` receipts to a concrete collection
operation (`create`, `update`, `delete`, `add_scene`, `remove_scene`, or
`reorder`). The operation participates in receipt identity, so one request UUID
cannot replay an Add as a Remove even when their payload fields match.

The browser collection client keeps a generated request UUID for the exact
operation and payload after a transport failure with no HTTP response. Retrying
that operation reuses the UUID; a definitive HTTP response clears it, and an
explicit cancel clears retained attempts. This covers timeout-after-commit
without creating a second mutation.

Workspace collection names are case-insensitively unique per workspace. Private
collection names are case-insensitively unique per creator and workspace.
PostgreSQL unique violations from those two indexes map to stable
`409 COLLECTION_NAME_CONFLICT` responses.

## Hardening verification

RED was observed for the missing collection operation discriminator and stable
name-conflict error, and for the missing browser `createCollectionClient`
retry state. GREEN evidence after implementation:

- Guarded `go test ./internal/repository/postgres -run TestCollection -count=1`
  passes real PostgreSQL checks for cross-operation request reuse, private and
  deleted collection replay denial, deleted-scene replay denial, migration
  up/down, and both case-insensitive uniqueness scopes.
- Guarded `go test ./...` passes all backend packages in 28.065 seconds;
  `go vet ./...` reports no issues.
- Focused frontend collection tests pass 8/8; TypeScript plus the production
  Vite build pass. A later shared-worktree full rerun reached 157 passing tests
  and one unrelated concurrent history-panel copy mismatch; L1 collection tests
  remained green and the V2 owner was notified.
- The focused real-stack collection browser command is environment-blocked
  before application setup because the local PostgreSQL role `test` does not
  exist. The runner reports one failed fixture and zero passed product tests;
  no browser assertion is claimed from this rerun. The earlier 1/1 collection
  scenario remains the latest completed browser evidence.

