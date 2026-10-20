# Enterprise C3 activity verification — 2026-10-20

Scope: authorized workspace and scene activity browsing over the existing F2
event projection. This task does not alter MCP `get_scene_activity`.

## Intended contract

- `GET /api/v1/workspaces/{workspace}/activity` and
  `GET /api/v1/scenes/{scene}/activity` require login authentication and the
  current `activity:read` feature policy.
- Pages contain `items`, an opaque stable cursor, and optional `retainedSince`.
  The retained boundary is omitted for an authorized empty feed. Scene
  visibility and owner-only security/admin filtering happen before the SQL
  limit, so hidden rows cannot change counts or cursors.
- Actor display names are resolved only for authorized rows. Historical actor
  IDs, safe event metadata, and redacted system/agent labels remain content and
  credential free.
- The dashboard route is `/w/{workspace}/activity`; the scene activity panel
  floats over the existing full-height canvas and uses the shared panel
  coordinator. Mobile, Escape, focus return, empty and error states remain
  accessible.

## Technical decision

`ActivityService` owns current authorization and feature-policy checks.
`ActivityRepository` performs keyset paging with a stable `(occurred_at,id)`
cursor, current membership/scene visibility predicates, owner-only event
predicates, and optional canonical scene scope before `LIMIT`. The existing
append/projector path remains unchanged.

Workspace and scene cursor indexes use
`(workspace_id[,scene_id],occurred_at DESC,id DESC)` so authorization filters
run before the keyset limit without replacing the F2 projection indexes.

`retainedSince`, when present, is the oldest currently retained event timestamp
in the same authorized scope. R1 will later enforce the 90-day deletion policy;
C3 exposes the actual retained boundary rather than claiming a cutoff that has
not run. Empty and fully filtered feeds omit it instead of serializing Go's zero
time.

## Verification plan

- Go unit/repository/handler: filters before pagination, mixed-source equal-time
  ordering, scene scope, owner-only activity, stable cursor, retained boundary,
  current revocation, invalid input and no unsafe display payload.
- Frontend: workspace/scene fetch headers, dashboard deep link, full-canvas
  overlay, mobile layout, Escape/focus through the shared coordinator, empty and
  error states.
- Real stack: edit/comment/share/agent source mix, duplicated source projection,
  revoke while paging, and retained cutoff. The public VM100 canary remains E1.

## Implementation evidence

The C3 routes are wired in the production server and the browser exposes the
workspace activity route plus an authenticated scene overlay. A failed refresh
or older-page request clears cached entries, cursors and the retained boundary,
so a membership revocation cannot leave earlier rows visible. The shared panel
coordinator captures Escape before Excalidraw and returns focus to the activity
trigger without changing canvas bounds.

TDD failures were observed for the absent activity domain/handler/panel, an
expired credential accepted by the workspace service, retained rows after a
paging revocation, and Escape consumed before the panel coordinator. The final
checks on 2026-10-20 were:

- Guarded PostgreSQL repository activity tests: pass, including 60 hidden rows
  before the visible limit, owner-only rows, deleted actor labels, equal-time
  mixed-source cursor paging and `retainedSince`.
- Activity use-case and handler tests: pass, including canonical aliases,
  tamper rejection, feature denial, public denial and expired credentials.
- Frontend unit suite: **35 files / 136 tests passed**; production build passed.
- Dedicated production-stack Playwright case: **1/1 passed in 32.7 s** against
  real Go, PostgreSQL, Node collaboration runtime and Chromium. It covers
  edit/comment/share/agent kinds, idempotent duplicate projection, mobile
  nonowner filtering, revoke while paging, retained boundary, canvas size,
  Escape and focus return.
- Full guarded Go run passed every C3 package. The final repository-wide run
  had one concurrent M2 presentation-authority test failure; C3 did not modify
  that handler and its focused/full-package checks remain green.

Review follow-up: an empty or fully filtered repository scope returns no
retained timestamp. A handler regression requires the JSON field to be absent,
and the matching frontend regression requires the empty panel to omit the
**Available since** label. The backend regression first failed with Go's
`0001-01-01T00:00:00Z` serialization before the optional contract was applied.
The production-stack browser case also logs into a distinct empty workspace and
verifies the empty panel end to end; the expanded case passed **1/1 in 35.7 s**.

