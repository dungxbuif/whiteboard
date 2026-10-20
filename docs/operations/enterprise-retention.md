# Enterprise retention operations

## Safety contract

Retention is an owner-controlled, two-step operation. `Preview` evaluates a
proposed policy against current metadata and persists a hash-bound preview.
`Schedule` accepts only that preview at the current policy revision and sets an
activation time at least seven days in the future. Workers always re-read the
current policy revision and legal holds while holding the workspace lock; a
replacement policy therefore invalidates work queued for an older revision.

Defaults are activity metadata after 90 days, inbox notifications after 30
days, and unpinned automatic versions beyond the newest 100 per scene. Manual
or pinned versions and comments are retained indefinitely unless an owner
explicitly supplies a comment retention period. Trash hard purge is disabled.
Canonical collaboration updates, receipts, tombstones and durable logs are not
retention targets.

Holds are workspace-scoped and may protect an activity event, notification,
version, comment, or an entire metadata class. A hold is rechecked in the same
transaction that claims deletion. Comment retention redacts author content in
place and preserves the comment row and thread order. Version retention removes
only metadata and its reference rows; object deletion belongs to the separate
reference-aware R2 lifecycle.

## Rollout and recovery

Keep `retentionEnabled=false` in production until the owner has reviewed the
preview for seven days and backup/reference evidence is complete. The worker
dry-run output contains identifiers and counts only. It never includes comment
bodies or version content. Disable the feature flag to stop new schedules and
worker claims; already stored policies and holds remain for audit and recovery.

Every mutation uses a caller-supplied request ID. Exact retries return the
stored response. Reusing an ID with a different principal, action, or canonical
body returns `REQUEST_ID_REUSED`.

