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
returns the original version; a changed retry is rejected.

List uses `?cursor={versionId}&limit=50` and returns `{items,nextCursor?}`. Get
`/{versionId}` returns canonical content and an `assets` map containing freshly
authorized image view URLs. Patch `/{versionId}` accepts
`{requestId,expectedRevision,label?,pinned?}` and cannot change captured content.

Content is at most 25 MiB. File assets are pinned atomically from the same scene.
Compaction cannot alter a version. Automatic checkpoints wait five dirty minutes
and are at least one minute apart. Diff and restore are reserved for V2.
