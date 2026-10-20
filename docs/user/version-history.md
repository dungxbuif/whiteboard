# Version history

Version history saves a durable scene copy at one exact collaboration sequence.
Editors can create named checkpoints and pin important entries. Viewers can
inspect history when member history is enabled. Public-link guests cannot.
When workspace version history is disabled, neither manual nor automatic
checkpoints are created.

A conflict means somebody edited after the history panel loaded. Reload and
create with a new request ID. Retrying the same unchanged request is safe and
returns the original checkpoint.

Referenced images are pinned with the checkpoint. Missing, deleted or foreign
scene images make the whole create fail. A scene larger than 25 MiB cannot be
versioned. Opening a version mints fresh image URLs; stored URLs and credentials
are never part of the checkpoint. Automatic checkpoints wait five dirty minutes and occur at most once
per minute; retention later limits them to 100 per scene. Manual and pinned
versions remain until an explicit policy removes them.

V1 provides create, list, inspect, label and pin. Diff and restore arrive in V2.
