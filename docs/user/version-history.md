# Version history

Version history saves a durable scene copy at one exact collaboration sequence.
Editors can create named checkpoints and pin important entries. Viewers can
inspect history when member history is enabled. Public-link guests cannot.
When workspace version history is disabled, neither manual nor automatic
checkpoints are created.

A conflict means somebody edited after the history panel loaded. Reload and
retry from the current sequence with a new request ID. Retrying the same
unchanged request is safe and returns the original checkpoint or restore.

Referenced images are pinned with the checkpoint. Missing, deleted or foreign
scene images make the whole create fail. A scene larger than 25 MiB cannot be
versioned. Opening a version mints fresh image URLs; stored URLs and credentials
are never part of the checkpoint. Automatic checkpoints wait five dirty minutes and occur at most once
per minute; retention later limits them to 100 per scene. Manual and pinned
versions remain until an explicit policy removes them.

Open **Version history** from the canvas controls to filter checkpoints, compare
element and canvas-setting changes, label or pin entries, and restore a prior
version. Restore always creates a pinned **Before restore** checkpoint first.
It keeps the same live document identity and writes one ordinary durable update,
so collaborators receive it through the normal room stream.

Restore requires the exact current collaboration sequence. If the scene changed,
review the latest diff and confirm again. Local undo history clears only after the
restore acknowledgement. Durable queued edits are retained; an offline device can
replay them later as newer edits, so coordinate with offline collaborators before
restoring an important scene.
