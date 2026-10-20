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
| `rtk go test ./...` | `src/backend` | pass: 371 tests in 14 packages without the PostgreSQL integration gate; full suite also passed with `ACCESS_TEST_DATABASE_URL` against a dedicated local database after publish-authorization and v1 copy fixes |
| `rtk npm run test:unit --workspace ./frontend` | `src` | pass: 233 tests in 51 files |
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
the removal of renderer/session state and inline bytes.

## Release boundary

Production image build, Compose configuration and migration gates, VM100 health,
and disposable-user public-origin management checks were **not run** in this
task. No deployment or push occurred. The pending checks are listed in the
[release checklist](../../operations/RELEASE_CHECKLIST.md). Current public
dashboard docs describe the new controls as the local release candidate; the
existing deployed-status statement refers to the prior dashboard baseline.

## Docs review

- Internal API index, release checklist, and this verification note reflect the
  implementation, role checks, projection authentication/cache semantics, and
  pending operations.
- Human guides in `docs/user/dashboard.md` and
  `src/frontend/public/docs/dashboard.md` explain the controls and error states.
  `src/frontend/public/llms.txt` points agents to the new plain Markdown API
  reference.
- Migration `000029` adds pending-copy and cleanup-claim state. It has **not**
  been applied in production; the old backend must be drained and ambiguous
  soft-deleted rows classified per the [deployment runbook](../../operations/DEPLOYMENT.md).
  The authenticated duplicate route reads the existing durable writer. The
  approved design and plan now record the corrected server-side copy contract.
  Release changelog is unchanged until production rollout is verified.
