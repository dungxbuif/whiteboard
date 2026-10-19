# E1 real-stack acceptance

Status: local_acceptance_passed; VM100 rollout verification pending. Approved by the [programme](../../superpowers/plans/2026-10-16-collaboration-v2.md), [release plan](../../superpowers/plans/2026-10-16-collaboration-release.md) and controller E1 brief. Links: [phase](../phases/PHASE-COLLAB-V2.md), [validation matrix](../VALIDATION_MATRIX.md), [central evidence](collaboration-v2.md).

## Intended behavior and test decision

The dedicated Playwright configuration runs only e2e/real-stack. Production Go server/userctl, real Node runtime/MCP, Vite browser bundle, PostgreSQL and SeaweedFS S3 execute with per-run generated distinct secrets. No API route mocks or fake signed authority. A loopback database whose name ends in _test is required; all migrations and destructive cleanup are confined to a random e1_ schema. Exact QA Docker container and temporary directory names are retained for cleanup. Production environment files are never loaded. BASE_URL can select a loopback frontend listener; this is not a production test client.

Tests cover real password login, invitations/roles, owner-only controls and server denials, public guest/password identity, UUID/alias reload, actual Blob upload/read and retained failures, concurrent/offline edits and revocation, MCP native IDs/browser edits/current-authority exports, commit-before-ACK restart replay, and 50 independent clients at one small update per second with durable receipts. Ten/eleventh voice admission follows B4 stable handoff. Actual cross-network media is separate E2 evidence. Load records p50/p95, CPU/RSS, platform/CPU/memory and checks no lost acknowledged updates; local target p95 <1000ms.

## Verification plan and documentation scope

Run `npm run test:e2e:collab --workspace ./frontend` from src with the recorded runtime PATH and rtk. New assertions first detect absent/incorrect product behavior; infrastructure errors are recorded separately and do not count as RED. Product defects go to their owners. Dedicated CI entrypoint builds prerequisites and starts the same guarded harness; no tracked CI workflow currently exists. Internal docs are this reproducible acceptance contract plus the E1 validation rows. Public docs do not change for test infrastructure alone; root reconciles any resulting product fixes and release guides. No schema/API contract changes are intended.

Ruling: use pinned SeaweedFS S3 3.97 because both Docker Hub and Quay MinIO release pulls were denied. It remains real S3 signature/object-byte storage; provider-specific MinIO behavior is not claimed.

## Results

Local acceptance passed on 2026-10-19. The unchanged dedicated command completed 17/17 browser cases in 1.4 minutes against the production Go server, Node collaboration runtime and MCP, PostgreSQL, SeaweedFS S3, and the Vite browser bundle. No API route or authority mocks were used. This ARM64 result is the pre-deployment gate; VM100 AMD64 and public-origin checks remain rollout evidence.

### Consolidated evidence

- Access: password login, invitations, effective roles, owner/editor/viewer permissions, owner settings, UUID routes, 21-character aliases, public guest/password identities, revocation, and continued authorized editing passed.
- Recovery: all four cases passed, including simultaneous edits, offline reload, downgrade with pending export, a real runtime kill immediately after durable commit and before ACK, exact receipt replay, epoch replacement, and recovered image movement after reopen.
- Assets: native PNG paste, signed S3 PUT/GET, first-load and reload rendering, durable asset identity, failed PUT retention, grant renewal, and exported-byte equality passed. The initial-data repair excludes metadata-only files until a signed URL is available.
- MCP: real HTTP and stdio discovery, scoped keys, write-only read denial, native browser IDs, duplicate receipt handling, stale conflicts, human-edit preservation, current PNG export, and revoked download denial passed.
- Presence and voice: native cursors, follow/camera behavior, 30 cursor moves, generated audio received in both browsers, mute/leave cleanup, microphone denial, and continued document editing passed.
- Load: 50 distinct public principals produced 650/650 durable ACKs and converged on the exact final native state. ACK latency was p50 9.742 ms and p95 13.236 ms, below the 1 s target. Peak runtime process-group RSS was 399,097,856 bytes.

The load repair uses at most two reusable validators with a bounded queue, fresh Y.Doc state per job, and serialized/coalesced per-room fanout. Existing authority, ownership, transaction, receipt, and fail-closed behavior remain covered by the runtime suite. The earlier failed profile remains useful causal evidence: fresh workers and one full read per recipient created avoidable lock contention; the acceptance target itself was not weakened.

Host boundary: Darwin ARM64, Apple M4 Pro, 12 logical CPUs, 51,539,607,552 bytes memory. VM100 deployment must repeat health, migrations, custom WebSocket, S3, MCP, browser routing, authorization, and a public-origin collaboration smoke test before production sign-off.
