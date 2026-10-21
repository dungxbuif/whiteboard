# Dashboard management parity verification — 2026-10-21

Scope: owner workspace rename; owner/editor folder rename/move and scene
move/duplicate; Collections disabled/retry UI; authenticated durable scene
projection. The approved [design](../../superpowers/specs/2026-10-21-dashboard-management-design.md)
and [plan](../../superpowers/plans/2026-10-21-dashboard-management-parity.md)
record the intended behavior before implementation. Task implementation and
review reports are under `.superpowers/sdd/2026-10-21-dashboard-management-parity-plans/`.

## Local evidence

All commands ran in the dashboard-management-parity worktree on 2026-10-21.

| Command | Location | Result |
| --- | --- | --- |
| `rtk go test ./...` | `src/backend` | pass: 400 tests in 14 packages without the PostgreSQL integration gate; full suite also passed with `ACCESS_TEST_DATABASE_URL` against a dedicated local database after repairable asset-error mapping |
| `rtk npm run test:unit --workspace ./frontend` | `src` | pass: 237 tests in 51 files |
| `rtk npm run build --workspace ./frontend` | `src` | pass: TypeScript and Vite production build; existing vendor annotation and chunk-size warnings |
| `rtk npx playwright test e2e/dashboard-style.spec.ts e2e/dashboard.spec.ts --workers=1 --reporter=line` | `src/frontend` | pass: 18 browser tests |
| `rtk git diff --check` | repository root | pass |
| local Markdown link check for changed docs | repository root | pass after this note was created |

Browser tests exercise owner rename, folder rename/move, scene duplicate/move,
location lists, disabled Collections, dashboard presentation and native canvas
return. Go tests cover current scene-read authorization, durable un-compacted
projection, and `Cache-Control: no-store` on unauthenticated and service-error
projection responses. The added tests cover commit-then-error duplicate
publication, state-uncertain preservation, pending copy exclusion from restore,
and conditional cleanup. PostgreSQL-backed Restore/Publish and asset cleanup
tests are gated by `ACCESS_TEST_DATABASE_URL` and passed against a dedicated
local PostgreSQL test database for this follow-up.
The Restore/Publish fixture starts two goroutines but uses a single database
connection, so it checks the final outcome, not truly parallel transactions.
A separate migration fixture passed ambiguous-row abort/DDL rollback and
staged-copy backfill under the same database gate. The earlier real-stack
Playwright suite could not start because its expected local PostgreSQL role
`test` is absent. These are local automated checks, not VM100 canaries.
Delayed-staging unit tests cover scene/workspace access revocation before
duplicate publication; a PostgreSQL fixture rejects legacy inline images with
no scene-scoped asset row. V1 canonicalization tests cover durable fields and
the removal of renderer/session state and inline bytes. Latest usecase tests
reject missing image-file references, incomplete image shape and missing file
creation time; PostgreSQL tests reject thumbnail asset rows and MIME mismatches
before object copying. The latest v1 tests reject typed-field errors (`x`,
`groupIds`, `created:null`), unsafe protocol-relative links, and a staged
result without an asset ID; the strict validator runs before
hidden-scene creation and again before publication.
Latest checks also reject null or unsafe nested `customData`, extra fields in
bound entries/bindings, and out-of-range freedraw pressure before or after
staging.
Depth-boundary checks accept 29 metadata containers and reject 30, both on
the source and after staging, matching collaboration's scene-root 32-level
limit.
Delayed-staging tests accept an unchanged destination folder and compensate a
copy when the folder was trashed or moved before publication. Deferred frontend
tests protect a new workspace's folder list from late star/trash completions in
the old workspace. Scene fallback tests keep the root list limited to root
scenes after successful move/duplicate responses when the reload fails.
The import regression rejects embeddable elements that the durable scene
contract cannot copy. A duplicate route regression maps a missing source
asset to `422 INVALID_SCENE_ASSET` instead of an authority outage.
The import test now checks the exact web-embed removal message and confirms
iframe continues to use the generic invalid-element message. Frontend unit
tests and build passed after this UX change; backend code was unchanged.

## Release boundary

The candidate was deployed to VM100 after the local gates above. Compose
configuration and the five production images built successfully. Migration
`000029` reached version `29` cleanly after the ambiguous legacy-row recovery;
one qualifying hidden legacy scene was preserved and its original deletion
state restored. Backend, collaboration, feature worker, MCP, and frontend are
running, and the public health endpoint, root shell, and scene deep-link shell
returned `200`. The complete production record is in the [VM100 release
note](../../releases/2026-10-21-dashboard-management-vm100.md).

Credentialed role-by-role browser UAT, provider checks, and cross-network media
remain operator-owned follow-up. They are not inferred from public HTTP or
container health.

The existing authenticated browser session also loaded the production workspace
settings Members page. This confirms the session and dashboard shell are live;
it does not replace the owner/editor/viewer mutation canaries.

## Docs review

- Internal API index, release checklist, and this verification note reflect the
  implementation, role checks, projection authentication/cache semantics, and
  pending operations.
- Human guides in `docs/user/dashboard.md` and
  `src/frontend/public/docs/dashboard.md` explain the controls and error states.
  `src/frontend/public/llms.txt` points agents to the new plain Markdown API
  reference.
- Migration `000029` adds pending-copy and cleanup-claim state. It is applied in
  production at clean version `29`; the old backend was drained and one
  ambiguous legacy row was classified and restored per the [deployment
  runbook](../../operations/DEPLOYMENT.md).
  The authenticated duplicate route reads the existing durable writer. The
  approved design and plan now record the corrected server-side copy contract.
  The [changelog](../../releases/CHANGELOG.md) has an Unreleased
  release-candidate entry; a production release entry awaits rollout proof.
