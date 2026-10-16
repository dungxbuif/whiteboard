# ADR 0010: One durable Yjs writer and isolated validation

Status: accepted for implementation; deployment pending.

## Decision

The custom Node/TypeScript service is the only content writer for v2 scenes. Browser, MCP, AI and reorder operations share its validation and durable append path; Go continues to own identity, permissions, grants and ACL versions. The initial release has one active service process. RelayHub and untested multi-process fanout are excluded.

Seed `scenes.content` once under a database lock and change writer mode atomically. Legacy content PATCH returns `409 COLLAB_REQUIRED` for v2 scenes while metadata PATCH remains available. Durable sequence numbers are decimal strings outside PostgreSQL. Principal-scoped idempotency receipts survive log compaction. Later B2 work owns persistence, grant/ACL rechecks, locking and acknowledgement; B1 alone does not supply durable storage.

B1 represents elements as per-field Y.Maps and order as a Y.Array, preserves native tombstones and bindings, validates the complete resulting scene, and rejects removal/retyping of known IDs. It does not rebuild accepted documents from materialized JSON. Candidate updates execute against a cloned snapshot in a worker, never against the room document. Validation is necessarily asynchronous; the caller holds its room mutation serialization boundary across validation and persistence. A successful validation result is not authorization and is not a durable receipt.

## Consequences and verification

Strict serializable schemas exclude renderer-local state, executable embeds and inline asset bytes. Documents larger than 25 MiB or frames larger than 1 MiB fail. Worker jobs have a two-second deadline, 256 MiB old-generation/16 MiB young-generation V8 heaps and a 4 MiB stack; failure terminates the worker and leaves the live document untouched. A separate 64 MiB retained snapshot budget fails with `SNAPSHOT_HISTORY_LIMIT`, requiring controlled recovery while preserving pending edits, durable updates and dedup receipts. Ordinary log pruning does not guarantee that `gc:false` Yjs history shrinks; no automatic re-seed or epoch switch is authorized.

Tests cover invalid resulting documents, actual resource failures, independent concurrent fields, preserved tombstones/references, trailing update bytes, unresolved dependencies, exact byte limits, and the typed history recovery error. Full real-stack durability, reconnect/revoke and rollback tests belong to B2/E and remain pending. Runtime documentation is internal at this stage; user-facing collaboration guides remain pending release integration.

Primary references: [Yjs update API](https://docs.yjs.dev/api/document-updates), [Node worker limits](https://nodejs.org/api/worker_threads.html), installed Excalidraw 0.18.1 declarations. Dependency versions and measured test evidence are recorded in the B1 execution report.
