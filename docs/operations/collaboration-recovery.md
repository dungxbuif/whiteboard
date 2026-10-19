# Collaboration recovery: verified replacement scene

Status: detail design approved by the release controller on 2026-10-16; bounded disposable rehearsal implemented, verification recorded separately. This is the E3 recovery slice, not production deployment or full E3 acceptance.

Trace: [release plan](../superpowers/plans/2026-10-16-collaboration-release.md), [programme contracts](../superpowers/plans/2026-10-16-collaboration-v2.md), [approved design](../superpowers/specs/2026-10-15-collaboration-v2-design.md), [verification](../work/verification/collaboration-recovery.md).

## Decision before implementation

Existing binary updates bind to the scene UUID, without a document epoch precondition. Disabling new promotions does not stop existing v2 documents. Changing writer_mode alone also leaves the existing collab document active. Migration 9 cascades document deletion into updates, receipts and audit. An ordinary image rollback or down migration cannot safely recover current state.

The approved bounded recovery creates a replacement scene with a NEW scene UUID and, on later promotion, a NEW documentId. Keep the original scene trashed and retain its collab document, ordered log, receipts, audit and original promotion JSON. Do not restore the original scene into the active service: Trash restore would deliberately reactivate that old identity. Old queues remain associated with that original scene; never rewrite their sceneId, namespace or documentId. Recovery does not promise the same URL.

The reusable operator module only verifies and writes offline checkpoint archives. The executable mutation rehearsal is test-only and strictly requires 127.0.0.1:55439/devtool_a1_test with randomized fixture schemas. It never accepts a production reset target.

## Intended procedure and safety boundaries

1. Put public, browser and private mutation ingress into maintenance; stop/drain the collab process and background compactor. Stop old HTTP writers too. Acquire the existing runtime ownership and exclusive handoff barriers; this drains any transaction still crossing the writer boundary and prevents a replacement runtime starting during the checkpoint operation.
2. Export the complete checkpoint using exportRecovery: documentId, latest durable sequence, snapshot and snapshot sequence, ordered remaining updates, receipts, mutation audit, original promotion JSON/version, and latest materialized content. Seal the full JSON with SHA-256, validate individual snapshot/update hashes and sequence continuity, reconstruct Yjs, compare semantic materialization, check receipt/audit correspondence, and compare the latest receipt state vector with the reconstructed document. Write a private exclusive file, fsync it, fsync its parent directory, read it back and verify it. A checksum detects corruption; it is not a signature or encrypted backup. Store the archive outside the database failure domain in production operations.
3. Re-read the source checkpoint under workspace/scene locks before mutation and require exact archive equality. Refuse unsupported recovery if any scene assets/files/image elements or scene member overrides exist. This bounded procedure must not broaden restricted access or create broken old-scene asset references. Unsupported scenes remain unavailable/read-only while an operator designs authorized asset metadata remapping or override preservation. Keep all existing history.
4. Materialize the verified latest content into a new v1 scene in the same workspace, with preserved element IDs and presentation order. Trash the old scene atomically. Inherit workspace access only when there are no scene overrides. Do not copy public links, guest sessions, or scene-scoped agent grants. Owners explicitly update links, invitations and targeted keys after review.
5. Verify old-scene admission and queued write denial, then enable the current release for the replacement. Its first promotion must have a distinct documentId and equivalent native content. Do not run an old image against the newer permission schema.
6. Restore the original archive only into an isolated schema/database for forensic recovery; compare the acknowledged history and reconstruct content again. Keep the original active-database rows and archive until retention is explicitly authorized.

If drain, archive, integrity, access or content verification fails, keep the service in maintenance/read-only recovery. Do not delete acknowledged history or retry by resetting tables.

## Verification plan

Use the real PostgreSQL migrations, Persistence, runtime ownership, WebSocket runtime and filesystem. Prove latest content after a compacted update and a remaining ordered update; retain both receipts and audit events; drain the writer and acquire exclusive barriers; write/re-read a durable archive; recover to v1 then promote to a new identity; reject an old queue without modifying it; restore original history into another fixture schema. Negative checks cover corrupt hashes, missing log/receipt/audit, changed promotion content, stale checkpoint, live writer, unsafe DB URL, assets and scene overrides. No production mutation, image rollback, database down migration or full user-facing history feature is included.

Internal docs: this runbook and evidence record are affected. Public documentation needs a concise limitation in the controller-owned collaboration guide and machine-readable Markdown surface; the recovery module adds no HTTP API or user-facing automatic restore feature. Central context/backlog/validation/release reconciliation belongs to the release controller.

## Executable scope

[Rehearsal instructions](../../tests/recovery/README.md) run the offline archive verifier and disposable integration proof. The reusable module is `src/collab/src/recovery.ts`: `sealCheckpoint`, `verifyArchive`, `writeArchive`, and `readArchive`. Its archive format is `collab-recovery-v1`, containing the unmodified format-1 persistence export plus a canonical whole-export SHA-256. Files are exclusive-create, mode 0600, limited to 256 MiB, fsynced and read back. There is no HTTP recovery endpoint or production database mutation CLI.

The helper under `src/collab/tests/recovery-fixture.ts` is deliberately test-only. It refuses an active runtime owner, takes the handoff barrier, locks workspace then scene, compares the archive with the current export and refuses assets/overrides before any scene mutation. It restores the archived identity into a separate randomized fixture schema, never over an existing scene. The test database fixture runs actual migrations 1–9 relevant to this persistence contract. This is a scene-checkpoint restore equivalence test, not a substitute for a full database/object-storage backup or the deployment controller's migration 1–13 validation.

Public documentation should state: “Recovery currently uses a verified replacement scene with a new link; assets and scene-specific access overrides require an operator mapping procedure. Existing queued edits stay associated with their original scene.” The controller owns publishing this limitation to human and AI-readable release surfaces.
