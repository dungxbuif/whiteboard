# ADR 0010: One durable Yjs writer and isolated validation

Status: accepted for implementation; deployment pending.

## Decision

The custom Node/TypeScript service is the only content writer for v2 scenes. Browser, MCP, AI and reorder operations share its validation and durable append path; Go continues to own identity, permissions, grants and ACL versions. The initial release has one active service process. RelayHub and untested multi-process fanout are excluded.

Seed `scenes.content` once under a database lock and change writer mode atomically. Legacy content PATCH returns `409 COLLAB_REQUIRED` for v2 scenes while metadata PATCH remains available. Durable sequence numbers are decimal strings outside PostgreSQL. Principal-scoped idempotency receipts survive log compaction. B2 implements persistence, grant/ACL rechecks, locking and acknowledgement; deployment verification remains separate.

B1 represents elements as per-field Y.Maps and order as a Y.Array, preserves native tombstones and bindings, validates the complete resulting scene, and rejects removal/retyping of known IDs. It does not rebuild accepted documents from materialized JSON. Candidate updates execute against a cloned snapshot in a worker, never against the room document. Validation is necessarily asynchronous; the caller holds its room mutation serialization boundary across validation and persistence. A successful validation result is not authorization and is not a durable receipt.

## Consequences and verification

Strict serializable schemas exclude renderer-local state, executable embeds and inline asset bytes. Documents larger than 25 MiB or frames larger than 1 MiB fail. Worker jobs have a two-second deadline, 256 MiB old-generation/16 MiB young-generation V8 heaps and a 4 MiB stack; failure terminates the worker and leaves the live document untouched. A separate 64 MiB retained snapshot budget fails with `SNAPSHOT_HISTORY_LIMIT`, requiring controlled recovery while preserving pending edits, durable updates and dedup receipts. Ordinary log pruning does not guarantee that `gc:false` Yjs history shrinks; no automatic re-seed or epoch switch is authorized.

Tests cover invalid resulting documents, actual resource failures, independent concurrent fields, preserved tombstones/references, trailing update bytes, unresolved dependencies, exact byte limits, and the typed history recovery error. B2 tests real PostgreSQL and WebSockets, commit/connection faults, ACL invalidation, receipts through compaction, legacy promotion and lossless history export. Real Go/browser/deployment integration and controlled rollback rehearsal remain E-owned. Runtime documentation is internal at this stage; user-facing collaboration guides remain pending release integration.

Primary references: [Yjs update API](https://docs.yjs.dev/api/document-updates), [Node worker limits](https://nodejs.org/api/worker_threads.html), installed Excalidraw 0.18.1 declarations. Dependency versions and measured test evidence are recorded in the B1 execution report.

## B2 implementation decision

Store `scenes.writer_mode` (`v1` or `v2`) separately from the durable document row. Lock workspace then scene before initial seeding and every append. Migration 000009 adds document snapshots, ordered updates, permanent receipts and mutation audit. A Go content update atomically checks writer_mode=v1; metadata writes never include the content column. Go supplies fresh signed five-second authority decisions before Node acquires its workspace lock. Node verifies actual-byte hashes, parent principal/scope, ACL version, expected sequence and expiry under the lock before append or replay.

Controls, command envelope and operational recovery are defined in the protocol and operations documents. History-limit handling preserves the current document identity, durable receipts and offline queues; only lossless export is automated. No automatic history reset is authorized. Actual fault and race test results will be recorded in task-B2-report.md.

Promotion adapts legacy renderer JSON without weakening the v2 validator. Before writer ownership changes, retain exact original JSONB and content version as an immutable promotion checkpoint. Preserve native elements/bindings/order, allowlist only shared durable app-state fields, and resolve every file ID through active scene_assets rows scoped by workspace and scene to the canonical asset UUID. Strip renderer dataURL fields only after that mapping succeeds. Missing/mismatched assets or invalid native data roll back the entire transaction and retain writer_mode=v1. The original renderer state remains recoverable from the checkpoint; the v2 scene contains only validated durable state.
