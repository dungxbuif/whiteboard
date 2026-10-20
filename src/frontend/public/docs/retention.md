# Retention API and limits

Retention is available only to workspace owners and only when the workspace
retention feature is enabled.

- `POST /api/v1/workspaces/{workspace}/retention/previews` creates a hash-bound
  dry-run preview.
- `POST /api/v1/workspaces/{workspace}/retention/schedule` schedules the exact
  preview for activation no sooner than seven days later.
- `PUT /api/v1/workspaces/{workspace}/retention/holds` adds or removes a hold.
- `GET /api/v1/workspaces/{workspace}/retention/policy` returns the current
  policy and revision.

Mutation bodies include a UUID `requestId`. Exact retries are idempotent;
rebinding an ID returns `409 REQUEST_ID_REUSED`. Defaults are 90 activity days,
30 notification days, and 100 automatic versions per scene. Omitted
`manualVersionDays` and `commentsDays` keep manual versions and comments
indefinitely. Pinned versions are always kept. `trashHardPurge` must be false.
Hold kind `scene` accepts a scene UUID and protects supported metadata belonging
to that scene. Collaboration logs, update receipts, and tombstones are not
deleted by this API.

