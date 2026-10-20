# Version history API

Use `/api/v1/scenes/{sceneId}/versions` with login bearer and
`X-Workspace-ID`. Public scene-session credentials are denied.

Create body:

```json
{"requestId":"UUID","expectedSequence":"42","label":"Before launch","pinned":true}
```

The response contains `id`, `documentId`, decimal-string `sequence`, SHA-256
`hash`, optional `label`, `pinned`, `kind`, `creator`, `metadataRevision` and
timestamps. Stale sequence returns `409 CONFLICT`. The same request ID and body
returns the original version; a changed retry is rejected. A missing private
writer bridge returns retryable `503 UNAVAILABLE`. Policy-disable races fail
closed with `403 FEATURE_DISABLED`, even after a create grant was issued.

List uses `?cursor={versionId}&limit=50` and returns `{items,nextCursor?}`. Get
`/{versionId}` returns canonical content and an `assets` map containing freshly
authorized image view URLs. Patch `/{versionId}` accepts
`{requestId,expectedRevision,label?,pinned?}` and cannot change captured content.

Content is at most 25 MiB. File assets are pinned atomically from the same scene.
Compaction cannot alter a version. Automatic checkpoints wait five dirty minutes
and are at least one minute apart. The default-false `versionsEnabled` policy
creates no automatic version or event, and disabled workspaces are omitted from
the durable backlog.

Diff a selected version against an exact current sequence with
`GET /api/v1/scenes/{sceneId}/versions/{versionId}/diff?againstSequence=42`.
The response is `{addedIds,removedIds,changedIds,changedAppStateFields}`. A
sequence race returns `409 REVISION_CONFLICT`.

Restore at `POST /api/v1/scenes/{sceneId}/versions/{versionId}/restore` with:

```json
{"requestId":"UUID","expectedSequence":"42"}
```

The result is `{requestId,sequence,stateVector,preRestoreVersionId}`. The signed
`version:restore` mutation binds workspace, scene, target version, request and
sequence. Node rechecks edit authority and policy under the writer lock. Success
keeps `documentId`, creates one pinned pre-restore version, then atomically commits
one validated Yjs update, receipt, audit row and safe events. Stale input commits
nothing. Identical retry returns the same receipt and recovery checkpoint.
An uncertain HTTP result must be retried with the same request ID and exact
`expectedSequence` pair. Only `409 CONFLICT` waits for a newer durable sequence;
after rebase it uses a new request ID. Other 409 codes are definitive. Reusing an
ID that already captured a checkpoint returns
`REQUEST_ID_REUSED` and commits nothing. Authorized room recipients receive the
durable sequence after the corresponding update frame so their next request is
based on current state.
`VERSION_INCOMPATIBLE` rejects another document generation. Missing image pins
reject the whole restore. File records referenced by later tombstoned image
elements remain available for an offline edit to revive. Queued offline updates
remain and may replay later. Never-sent pending edits persist per-transaction
base/desired intent; after sync the browser waits for local durability and
atomically replaces the safe creation-ordered pending rows with one causal rebase update. A restore
therefore cannot win through a client-ID or IndexedDB key-order tie-break.
If that atomic replacement observes another local writer, no raw update is sent;
the pending rows remain recoverable and the client requires a fresh reconnect.
IndexedDB schema v3 preserves legacy rows during upgrade. Already-sent requests
and legacy rows without intent retain their original identity and bytes, disable
automatic replay, and remain available in the recovery export. Restore validation
accepts internal deltas up to the 64 MiB snapshot ceiling while client mutation
frames remain capped at 1 MiB. Oversized restore fanout uses 512 KiB durable sync
chunks followed by the committed sequence; the room queue retains only a small
sequence marker while it reloads the authorized durable state. Acknowledgement clears both the
collaboration-local undo stack and Excalidraw's native history.
