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
| `rtk go test ./...` | `src/backend` | pass: 357 tests in 14 packages |
| `rtk npm run test:unit` | `src/frontend` | pass: 224 tests in 50 files |
| `rtk npm run build` | `src/frontend` | pass: TypeScript and Vite production build; existing vendor annotation and chunk-size warnings |
| `rtk npx playwright test e2e/dashboard-style.spec.ts e2e/dashboard.spec.ts --workers=1 --reporter=line` | `src/frontend` | pass: 18 browser tests |
| `rtk git diff --check` | repository root | pass |
| local Markdown link check for changed docs | repository root | pass after this note was created |

Browser tests exercise owner rename, folder rename/move, scene duplicate/move,
location lists, disabled Collections, dashboard presentation and native canvas
return. Go tests cover current scene-read authorization, durable un-compacted
projection, and `Cache-Control: no-store` on unauthenticated and service-error
projection responses. These are local automated checks, not VM100 canaries.

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
- ERD and migrations: unchanged because these controls reuse existing workspace,
  folder and scene data. Architecture boundaries and deployment configuration:
  unchanged; the projection route reads the existing durable writer. No new ADR
  was needed because the approved dashboard design already records this choice.
  Release changelog is unchanged until production rollout is verified.
