# Workspace retention

Workspace owners can preview retention before anything becomes eligible for
removal. The preview shows counts for activity, notifications, automatic
versions, and optional comment redaction, together with the earliest activation
date. Scheduling always waits at least seven days.

The default policy keeps activity for 90 days and notifications for 30 days,
and keeps the newest 100 automatic versions for each scene. Manual checkpoints,
pinned versions, and comments are kept indefinitely. Trash hard purge is not
available. Holds override every scheduled policy.

Changing a scheduled policy creates a new revision and cancels work based on an
older revision. Retention does not delete the collaborative drawing history or
its recovery receipts. Comment retention, when explicitly configured, replaces
old comment text with a redaction marker while preserving thread pagination.

