# Disposable recovery rehearsal

From the repository root, with the workspace dependencies and contract build already installed by the release controller:

```sh
rtk proxy env COLLAB_TEST_DATABASE_URL='postgres://test:test@127.0.0.1:5432/whiteboard_test?sslmode=disable' node tests/recovery/rehearse.mjs
```

Replace `LOCAL_TEST_USER` with your local PostgreSQL test account. The script refuses any other host, port or database, and query parameters that redirect the connection. It compiles the collab workspace using installed dependencies, then runs the recovery unit/integration tests. It never installs packages, changes a production database, performs a down migration, or deploys an image. Each integration test uses randomly named `b2_*` schemas and drops only those fixtures.

Optionally set `RECOVERY_EVIDENCE_DIR` to a new empty private directory. The main drill writes an exclusive, verified `checkpoint.json` and a `result.json` containing the archive digest, old/new scene and document identities, sequence and history counts. Existing evidence files are never overwritten. These are synthetic fixtures; keep real operator recovery archives private and outside the database failure domain.

The main drill uses PostgreSQL, the actual runtime owner lock, WebSocket admission/acknowledgement, Persistence and Yjs. It stores a native rectangle, commits an edit, compacts, commits another edit through the socket, stops the writer, archives and verifies snapshot+log+receipts+audit+original promotion JSON, creates a replacement v1 scene, denies the old identity, promotes the replacement and restores the original archive in a second schema. Asset metadata and scene overrides are refusal cases, not supported restoration cases. The browser IndexedDB behavior is covered separately by B3; this drill keeps its old pending packet unchanged and proves the server refuses its old scene scope.

See the [operator contract](../../docs/operations/collaboration-recovery.md) and [evidence](../../docs/work/verification/collaboration-recovery.md). This bounded rehearsal does not implement production recovery automation or same-URL rollback.
