# E1 real-stack acceptance

Status: in_progress. Approved by the [programme](../../superpowers/plans/2026-10-16-collaboration-v2.md), [release plan](../../superpowers/plans/2026-10-16-collaboration-release.md) and controller E1 brief. Links: [phase](../phases/PHASE-COLLAB-V2.md), [validation matrix](../VALIDATION_MATRIX.md), [central evidence](collaboration-v2.md).

## Intended behavior and test decision

The dedicated Playwright configuration runs only e2e/real-stack. Production Go server/userctl, real Node runtime/MCP, Vite browser bundle, PostgreSQL and SeaweedFS S3 execute with per-run generated distinct secrets. No API route mocks or fake signed authority. A loopback database whose name ends in _test is required; all migrations and destructive cleanup are confined to a random e1_ schema. Exact QA Docker container and temporary directory names are retained for cleanup. Production environment files are never loaded. BASE_URL can select a loopback frontend listener; this is not a production test client.

Tests cover real password login, invitations/roles, owner-only controls and server denials, public guest/password identity, UUID/alias reload, actual Blob upload/read and retained failures, concurrent/offline edits and revocation, MCP native IDs/browser edits/current-authority exports, commit-before-ACK restart replay, and 50 independent clients at one small update per second with durable receipts. Ten/eleventh voice admission follows B4 stable handoff. Actual cross-network media is separate E2 evidence. Load records p50/p95, CPU/RSS, platform/CPU/memory and checks no lost acknowledged updates; local target p95 <1000ms.

## Verification plan and documentation scope

Run `npm run test:e2e:collab --workspace ./frontend` from src with the recorded runtime PATH and rtk. New assertions first detect absent/incorrect product behavior; infrastructure errors are recorded separately and do not count as RED. Product defects go to their owners. Dedicated CI entrypoint builds prerequisites and starts the same guarded harness; no tracked CI workflow currently exists. Internal docs are this reproducible acceptance contract plus the E1 validation rows. Public docs do not change for test infrastructure alone; root reconciles any resulting product fixes and release guides. No schema/API contract changes are intended.

Ruling: use pinned SeaweedFS S3 3.97 because both Docker Hub and Quay MinIO release pulls were denied. It remains real S3 signature/object-byte storage; provider-specific MinIO behavior is not claimed.

## Results

Pending execution. Local ARM64 evidence must not be represented as the VM100 AMD64 release gate.

### Interim evidence

- Real password/invitation/effective role API case passed. Owner/editor/viewer browser permissions, native editor write convergence, owner settings, UUID and 21-character alias reload passed.
- Recovery suite 3/3 passed (37.3s including setup): simultaneous native edits, offline reload, offline downgrade with exported queue, real SIGKILL immediately before ACK after durable commit then exact receipt replay, and epoch mismatch retaining pending work.
- MCP HTTP/stdio full case passed (34.8s including setup): real keys, write-only read denial, native browser IDs, duplicate receipt, stale conflict, preserved human edit, current PNG export and revoked download.
- Real failed signed PUT retains IndexedDB Blob and compatible grant identity; subsequent signed PUT/GET byte equality passed. Native Excalidraw re-encodes the source PNG; final assertion compares the actual PUT body against recorded size and downloaded bytes, not the original clipboard encoding.
- Infrastructure corrections (not product RED): fixture relative path, valid generated email domain, denied MinIO image pulls, explicit S3 bucket CORS, native canvas focus instead of intercepted static-canvas locator click. Expected API codes corrected to actual documented inaccessible404 and staleCONFLICT. No product changes were made to force these tests green.

### Load failure and bounded investigation

The target is unchanged: 50 distinct public principals, one native-element x update each second, 3 warmup seconds plus 10 measured seconds, 650 total writes and 500 measured attempts. All 50 replicas must end with every native element at x=13; every ACK must match a durable receipt.

- First valid quiet-window attempt failed ACK settlement. Diagnostic run after releasing the quiet window recorded 203/650 ACKs, observed-ACK p50 8507ms/p95 10112ms, peak runtime process-group RSS748240896B and sampledCPU77.4%. This fails acceptance; its latency distribution excludes unacknowledged writes and cannot be presented as healthy throughput.
- Profiled diagnostic: 178 ACKs/650 attempted, 219 durable receipts; all178 ACK request IDs/sequences matched PostgreSQL (zero lost acknowledged writes). All50 connections closed1013 AUTHORITY_CHANGED. Observed-ACK p95 8557ms, peak runtime-group RSS690372608B. No successful convergence claim.
- Actual host: Darwin ARM64, Apple M4 Pro,12logicalCPUs,51539607552bytes memory. This is not VM100 AMD64 production proof. `ps` samples aggregate runtime process-group RSS/%CPU; client CPU/RSS and raw latencies are separately recorded.
- Test-only profiler:219validationWorkers, median job54.45ms/max106.41ms, worker-online median10.84ms;2200workspace-lock queries with269355ms cumulative wait/max833ms;2199document reconstructions;6600ownership checks; no PostgreSQL query error codes. Fresh workers execute while persistence holds the exclusive workspace/scene transaction. Per-commit fanout starts one fresh full read per recipient, multiplying contention.
- Proposed repair sent to controller: bounded reusable validation worker with fresh document per job and unchanged kill/resource limits; bounded serialized/coalesced room fanout retaining Go authority and current ACL/ownership checks before recipient delivery. No product repair is included in E1 yet.

Raw diagnostic JSON is retained in the task workspace as e1-load-diagnostic-first.json and e1-{load-results,runtime-profile,durable-receipts}.json. Results intentionally fail rather than weakening the throughput or durability target.

### Authorized bounded runtime repair (before implementation)

Root authorized `validation-job.ts`, `validation-worker.ts`, and `rooms.ts` after the measured failure. Use at most two reusable workers with at most 32 queued jobs; each candidate starts a fresh Y.Doc and retains the 2 s submission deadline and existing worker memory/stack limits. Deadline/error/exit terminates and replaces the worker. Idle workers are unreferenced. No transaction, durable receipt, ACK, ownership fence, or authority token contract changes.

Committed fanout becomes one serialized task per room with a 100 ms coalescing window, a bounded pending byte budget, and merged valid Yjs updates split into protocol-sized batches. Every recipient still performs a fresh authority-and-ownership-checked store.read immediately before delivery. Transient read/authority failure sends no data and closes for retry with UNAVAILABLE; actual denial uses AUTHORITY_CHANGED. Backlog overflow forces resync; durable updates remain recoverable from storage. Tests must prove worker reuse/limits/replacement, batch convergence, denied recipient exclusion, transient fail-closed classification, and existing B2 ACL/ownership/receipt lifecycle before repeating unchanged 50-client acceptance.
