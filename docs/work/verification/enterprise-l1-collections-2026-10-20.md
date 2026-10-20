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

- `go test ./internal/usecase ./internal/delivery/http/handler`: 168 tests pass.
- Guarded `go test ./internal/repository/postgres -run TestCollection -count=1`:
  privacy, moderation delete, composite references, stable reorder, conflict,
  exact retry/changed reuse and migration up/down pass on PostgreSQL.
- Guarded `go test ./...`: all backend packages pass; `go vet ./...` reports no
  issues.
- `npm run test:unit --workspace ./frontend -- --run`: 40 files and 151 tests
  pass; the focused collection file contains four passing cases.
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

