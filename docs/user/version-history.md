# Version history

Version history saves a durable scene copy at one exact collaboration sequence.
Editors can create named checkpoints and pin important entries. Viewers can
inspect history when member history is enabled. Public-link guests cannot.
When workspace version history is disabled, neither manual nor automatic
checkpoints are created.

A conflict means somebody edited after the history panel loaded. The restore
dialog waits for the latest collaboration sequence before allowing another
confirmation, then uses a new request ID. If a network or server interruption
makes the outcome uncertain, retrying reuses the same request ID so it returns
the original result instead of restoring twice.

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
review the latest diff and confirm again. The app advances that sequence when a
collaborator's durable update arrives. Local and canvas undo histories clear only
after the restore acknowledgement. Durable queued edits are retained; an offline
device can replay them later as newer edits. Image metadata needed by a later
tombstoned element is retained so that offline edit can safely revive the image.
Coordinate with offline collaborators before restoring an important scene.
