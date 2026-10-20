# Durable collaboration operations (pre-release)

The service runs exactly one active Node process. Build from `src` with `npm ci` and `npm run build --workspace ./collab`; run `npm start --workspace ./collab`. Worker files must accompany `dist/server.js`; never copy that file alone.

Configuration: `DATABASE_URL` (PostgreSQL), `COLLAB_V2_ENABLED` (exact true/false, default false; permits new promotions only), `COLLAB_JWT_SECRET` and `MUTATION_GRANT_SECRET` (distinct >=32-byte secrets matching Go), `AUTHORITY_SERVICE_SECRET` (Go private listener), `AUTHORITY_URL` (default http://127.0.0.1:8081), `COLLAB_INTERNAL_SECRET` (distinct >=32-byte private command credential), `COLLAB_HOST` (default 127.0.0.1), `COLLAB_PORT` (default 8082), `COLLAB_INTERNAL_HOST` (default 127.0.0.1), `COLLAB_INTERNAL_PORT` (default 8083), `COLLAB_ALLOWED_ORIGINS` (comma-separated exact browser origins; required), `COLLAB_UPDATES_PER_MINUTE` (default 120), `COLLAB_MAX_BUFFERED_BYTES` (default 4194304). The service enforces one owner per database/schema using a session advisory lock. Do not proxy the internal listener. Do not log tokens, command bodies or update bytes. Public `/healthz` checks liveness; `/readyz` checks database and ACL listener health. Secrets must all differ. Migrations 1–13 must be applied before starting.

An ack means update bytes, receipt and audit committed together. A disconnect or unavailable error does not imply the update failed; preserve and replay the same request ID and bytes. Different bytes or principal with the same ID fail. Receipts survive compaction. Browser callers should merge/coalesce local bursts before sending; the server never merges request IDs because each is an independent retry identity. Rejected or rate-limited updates stay in the client queue.

`SNAPSHOT_HISTORY_LIMIT` means retained gc:false history exceeded 64 MiB. Stop writes, preserve client queues and acknowledged receipts, and export the complete durable checkpoint using the persistence `exportRecovery` API (snapshot, remaining ordered updates, receipts, audit, materialized JSON, sequence and hashes). Verify checkpoint hashes and reconstructed content in an isolated environment before any operational change. Snapshot compaction alone may not reduce history. B2 provides a lossless export, never reseeding or epoch-changing recovery; resuming over-limit editing requires an explicitly reviewed E3 migration/recovery procedure. Restore only the same document identity and complete history/receipts, then verify replay and client queues. Do not delete update logs or receipts to clear this state.

Database/authority outage sends no optimistic ack. LISTEN disconnection or ACL notification closes affected sessions for fresh admission. Backpressure closes the slow socket with resync_required; durable state survives. Production deployment, backup retention, proxy controls and disaster-recovery rehearsal remain E-owned and unverified here.

Before enabling promotions, inventory legacy files. Each persisted file ID must have active scene_assets metadata in the same workspace and scene. Promotion strips renderer-only app state and dataURL fields after resolving asset UUIDs; original JSONB and contentVersion remain in the document checkpoint. PROMOTION_REQUIRED means repair the legacy scene/asset mapping first; writer_mode stays v1. Turning COLLAB_V2_ENABLED off never re-enables HTTP content writes for existing v2 scenes. Migration down refuses any existing collaborative document and requires an explicit checkpoint/drain rollback procedure.

`exportRecovery(ref)` is an operator-library API; it is not exposed on either HTTP listener. Instantiate Persistence with an operator PostgreSQL pool and call it using canonical workspace/scene UUIDs. Save the returned JSON with owner-only filesystem permissions. It includes promotionCheckpoint, documentId, sequence, snapshotSequence, base64 snapshot and SHA256, each remaining ordered update and SHA256, permanent receipts/audits, and materialized content. Tests reconstruct a valid >64 MiB retained history export and verify identity/receipt preservation. Automated history reduction or restore is deliberately not implemented; E3 must approve and rehearse any migration before resuming a blocked document.

Local verification uses COLLAB_TEST_DATABASE_URL (default the documented loopback devtool_a1_test fixture) and creates a random b2_ schema per test. The URL guard rejects remote/non-*_test databases. No database skip counts as a pass. Run `npm test --workspace ./collab` after a clean root npm ci, and the backend suite with ACCESS_TEST_DATABASE_URL set to the isolated fixture database. Runtime tests launch built dist/server.js and its real dist validation worker; deployment/proxy smoke remains E-owned.

## B2 review correction: ownership and maintenance (deployed; restart proof passed)

The primary exclusive session advisory key remains `(hashtext(current_database()), hashtext(current_schema() || ':collab-writer'))`. A separate handoff key is `(hashtext(current_database()), hashtext(current_schema() || ':collab-handoff'))`. Runtime transactions take a shared transaction-level handoff lock before workspace then scene/document row locks. They verify the exact primary owner PID and backend_start against pg_locks/pg_stat_activity before work and before commit. Local generation and the database owner are checked again after commit before publication/acknowledgement. A new owner takes the primary lock then an exclusive handoff barrier and becomes ready only after old transactions have drained. If ownership disappears in the final-check/commit gap, the durable receipt survives, the old runtime does not publish/ack, and the new owner replays that receipt. Lock/statement/pool acquisition and shutdown waits are bounded. No schema or document-identity change is required.

Compaction ignores inactive scenes/workspaces and rechecks activity under locks to handle trash racing a due scan. A skipped trashed document retains snapshot, updates, identity and receipts for restore. One scene's maintenance failure does not disconnect unrelated rooms or stop later eligible jobs. Periodic scans are single-flight; timer/count/scan requests share a bounded deduplicated queue with one active scene job. Already compacted documents are skipped under the row lock. Durable scanning rediscovers work deferred by the in-memory bound; shutdown stops intake and drains active work.

The primary owner PID is paired with its exact PostgreSQL backend_start timestamp, not PID alone. A lost generation never becomes usable again when LISTEN reconnects. New ownership remains unready until old shared-handoff transactions drain. The final precommit-check/COMMIT window can produce a committed receipt with a 503 response; resend the same request ID/bytes through the current owner. The old runtime publishes no committed event in that case. This is tested with an explicit pause immediately before an actual PostgreSQL COMMIT, actual listener termination and a competing owner lock.

Runtime pools cap connection acquisition at 3 seconds. Ownership barrier acquisition and due-query statements use 5-second database deadlines; transaction row/handoff lock waits use 3 seconds and statements 5 seconds. Reconnect attempts are single-flight, check shutdown after acquiring a pooled connection, and cannot reacquire ownership after close. Shutdown invalidates ownership first and drains active maintenance/socket/reconnect work; disconnected private requests remain protected by transaction fencing and durable retry receipts. There is no automatic deletion of pending durable work during shutdown.

Compaction uses one active scene job, one active scan, at most 64 pending jobs and 64 timer entries, and 64-row due pages. Excess event hints are recoverable from durable rows. Trash is skipped rather than reported as a storage outage. A genuine scene snapshot failure disconnects only that scene, with its log retained for repair; it does not disconnect unrelated live scenes. Scans proceed around failing rows using the stable pagination cursor. Restoring a trashed scene/workspace re-enables compaction with the original documentId and receipts.

Listener shutdown drains the checked-out PostgreSQL connection before returning it to the pool, including a concurrent backend termination. This prevents a late termination error from escaping through the pool idle-client error path.

Startup maintenance starts only after successful primary ownership, exclusive handoff drainage, LISTEN and persistence-fence binding. Failed acquisition closes safely without a scheduler. A real shared-handoff regression verifies that a due dirty document retains its snapshot cutoff and log while startup waits; compaction resumes after release.

## Browser integration policy (B3 implementation)

Pending IndexedDB updates are scoped by principal/workspace/scene/documentId. Revocation or history limits retain recoverable data; account changes never reassign queues. A v2 scene never falls back to HTTP content saves. Explicit COLLAB_DISABLED/PROMOTION_REQUIRED may permit the legacy authenticated canvas only after a fresh v1 writerMode check; a concurrent promotion409 stops this path and retains local recovery. Public legacy scenes remain read-only without an authorized scoped writer. Local B3 verification includes unit/build checks and a native browser drawing/offline/reload flow through actual Go authority, Node and isolated PostgreSQL. Public same-tab session recovery preserves the guest principal; definitive authorization rejection clears session credentials while retaining recovery data and private login. Recovery exports include lossless base64 Blob bytes. The `legacy-v1` namespace retains drafts/assets until confirmed HTTP save, including a concurrent promotion409. Full signed-storage and deployed public-path acceptance remain release gates.

## Bounded candidate workers and committed fanout

Candidate validation uses at most two reusable worker threads and 32 waiting jobs per runtime. Every job creates and destroys its own Y.Doc. The two-second deadline includes queue time; worker heap limits remain 256 MiB old generation, 16 MiB young generation and 4 MiB stack. A timed-out, crashed or exhausted worker is terminated and replaced. Idle workers do not keep shutdown alive. Receipt insertion, commit-before-ACK and ownership fencing are unchanged.

One task per room coalesces committed updates for 100 ms. Pending batches hold at most 4 MiB or 1,024 updates; one active batch can hold another 4 MiB. Frames retain the 1 MiB transport ceiling. Overflow closes that room for BACKPRESSURE resync from durable storage. Recipient authority is checked freshly for each batch. Temporary authority/read failure closes for UNAVAILABLE retry and never serves stale content; definitive denial uses AUTHORITY_CHANGED. These bounds are workload safeguards, not a product-level data-participant cap. See [real-stack evidence](../work/verification/collaboration-v2-real-stack.md) for measured host, workload, latency and limits of local proof.

## Live presentation operations

Live presentation uses the existing public `/collab/` listener, origin policy,
heartbeat and authority service. It adds no listener, RelayHub dependency,
database-resident participant roster or media port. Presentation links remain in
PostgreSQL; active host, waiting, admitted, frame, spotlight, lock and reaction
state live only in the owning collaboration process. A process restart therefore
ends active sessions by design and hosts must start a new session.

Monitor `PRESENTATION_AUTHORITY_UNAVAILABLE`, `PRESENTATION_CONFLICT`,
`PRESENTATION_REACTION_RATE`, waiting/audience capacity and abnormal host socket
churn. A host disconnect terminates its session. An admitted audience disconnect
must perform a fresh invite join and admission; the runtime never reuses cached
audience UI authority. Revoking an invite blocks new joins but does not remove
already admitted viewers. Feature or host-authority revocation ends current
audience sockets.

Host startup rechecks that the exact initiating socket remains attached after
each authority/token await. State fanout rechecks the current session and
revision after recipient authority refresh and immediately before every send.
These checks prevent a disconnected host or delayed broadcast from recreating
an ended session. Link create/revoke similarly checks the same runtime host
revision before and inside persistence.

Pending-queue fanout additionally rechecks the current host's room admission and
`presentation:host` feature before revealing guest identity. A failed check ends
the session and emits `AUTHORITY_CHANGED`. Host start acknowledgement has a
ten-second client deadline; expiry closes that exact socket, allowing the normal
detach path to terminate any server-side session before a reconnect and retry.
Watch repeated `START_TIMEOUT` plus host socket churn as an authority or network
incident rather than retrying on the same connection.

The runtime caps pending guests, admitted guests and retained live reactions at
100 each. Pending sockets receive no ready, sync, roster, asset or media data.
During incident review, never log invite, pending, host or audience tokens. To
verify a release, run the focused presentation suites, full frontend suite and
production build, then the guarded real-stack `enterprise-live.spec.ts` against
the disposable test database. That browser test covers manual invite copy,
mobile waiting isolation, admit, reaction, remove, deny and current-editor host
transfer, plus explicit waiting Leave, keyboard Escape, permission denial and
ended pending-auth recovery.
