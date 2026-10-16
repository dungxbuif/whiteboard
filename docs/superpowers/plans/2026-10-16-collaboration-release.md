# E — Verification and VM100 rollout Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deploy verified collaboration and MCP with recoverable history and accurate public documentation.

**Architecture:** Test disposable real services, then stage identical images on VM100. Keep shared Coturn separate and fence old/v2 writers during canary and rollback.

**Tech Stack:** Go, PostgreSQL, TypeScript 7, MCP SDK, Excalidraw, Yjs, Docker, Playwright.

**Spec:** [Approved design](../specs/2026-10-15-collaboration-v2-design.md), approved by “xác nhận”; not implemented.

## Global Constraints

- Read [programme contracts](2026-10-16-collaboration-v2.md); exact interfaces, Go policy and documentation requirements apply.
- Custom Node/TypeScript WebSocket service; RelayHub is not involved. Go owns authorization.
- Roles: owner/editor/viewer; management owner-only. Invite TTL seven days, single-use/email-bound. Room JWT ten minutes.
- Update frame 1 MiB; document 25 MiB; 120 updates/minute/principal; snapshot at 100 updates or five seconds.
- Ten voice participants; no product-level data cap. Retain TypeScript 7 and password/default-workspace provisioning.
- Dashboard copies old shadcn styling with its original colors and tokens. Scene components retain native style.

## Review Focus

- Mocks pass but real DB/socket/browser integration fails — E1.
- TURN conflicts or NetBird routing is guessed — E2.
- Old client writes after v2 promotion/rollback — E3.
- Crash/recovery loses acknowledged updates — E1/E3.
- Docs claim parity or verification without evidence — E3.

---

### E1: Real-stack browser, recovery and load proof

**Files:** Create `docker-compose.collab-test.yml`, `src/frontend/playwright.collab.config.ts`, `src/frontend/e2e/{collab-live,mcp-live,offline-live,collab-assets}.spec.ts`, `src/collab/tests/load.ts`, `docs/work/verification/collaboration-v2.md`; modify actual tracked CI workflow after locating its filename.

**Interfaces:** Disposable Go/Node/MCP/frontend/Postgres/object-store harness with generated secrets. Frontend `test:e2e:collab` script invokes dedicated Playwright config supporting `BASE_URL`; no silently mocked APIs.

- [ ] **Step 1 — Record behavior:** update `docs/work/verification/collaboration-v2.md`, `docs/work/VALIDATION_MATRIX.md` with contracts and verification cases.
- [ ] **Step 2 — Add failing proof:** Browser cases: owner/editor/viewer/public guest, simultaneous/offline edits, offline revoke/downgrade, image upload/reload, frames, live MCP generation/export. Kill collab after commit-before-ack and confirm replay no duplicate. Load 50 data clients and 10 voice admissions; 11th voice fails while data continues; rate-limit/slow-client checks.
- [ ] **Step 3 — Verify failure:** run `rtk npm run test:e2e:collab --workspace ./frontend`; confirm missing behavior causes failure, not broken infrastructure.
- [ ] **Step 4 — Implement:** Fault hooks only in test runtime. Refuse production DB in destructive fixtures. Compare normalized scene content and await durable receipts. Record p50/p95 latency and CPU/memory/hardware; target local p95 <1s at 50 clients each sending one small update/sec, no lost acked updates. This load target is not a product cap. Real cross-network audio/TURN proof is separate from simulated admission.
- [ ] **Step 5 — Verify:** rerun `rtk npm run test:e2e:collab --workspace ./frontend`; require listed cases to pass and record actual results.
- [ ] **Step 6 — Reconcile and commit:** reconcile `docs/work/verification/collaboration-v2.md`, `docs/work/VALIDATION_MATRIX.md`, stage named files and lockfiles only, commit `test: verify collaboration and MCP on real services`.

### E2: Container routing and homelab preflight

**Files:** modify `docker-compose.prod.yml`, `src/frontend/{Dockerfile,nginx.conf}`, `src/.env.example`; create `src/{collab,mcp}/Dockerfile`. Homelab targets under `/Users/dungxbuif/Documents/ChatGPT/Homelab`: `deploy/whiteboard/compose.vm100.yml`, `docs/architecture/whiteboard-vm100.md`, `docs/operations/{whiteboard,coturn}.md`.

**Interfaces:** public /collab WebSocket -> collab; /mcp streaming HTTP -> MCP; /api -> Go. Internal command/authority endpoints private. One active collab instance.

- [x] **Step 1 — Record topology:** describe route map, service identity and rollback target in deployment docs. Homelab files must be readable before editing; never replace timed-out placeholders.
- [x] **Step 2 — Read-only inventory:** use verified saved VM100/gateway connection metadata to inspect listeners, Docker services/networks, reverse-proxy routes, NetBird relay configuration and TURN allocation range. Record actual hosts/ports; do not assume NetBird relay serves browser TURN. Missing access blocks only the corresponding production check.
- [x] **Step 3 — Package:** workspace-aware locked multi-stage builds, non-root processes, bounded renderer, health/readiness, WebSocket upgrade, unbuffered streaming HTTP. Require dedicated collab JWT/service secrets and fail closed on bad config.
- [x] **Step 4 — Verify locally:** run `rtk docker compose --env-file .env.prod -f docker-compose.prod.yml config --quiet` using securely provided local values, then build. Exercise /collab upgrade and /mcp lifecycle behind nginx; assert /internal is unreachable publicly. Do not print expanded secrets.
- [x] **Step 5 — Verify TURN:** actual ICE relay allocation/audio from two networks; record selected relay candidate and cleanup. Existing Coturn remains; data collaboration needs no extra TURN listener or new media port allocation.
- [x] **Step 6 — Reconcile and commit:** record actual topology/env/health/routing evidence. App commit `build: package collaboration and MCP services`; Homelab deployment/docs commit separately preserving unrelated changes.

### E3: Canary, rollback and public documentation

**Files:** update `docs/operations/{DEPLOYMENT,ENVIRONMENTS,RELEASE_CHECKLIST}.md`, `docs/architecture/{ARCHITECTURE,API,ERD}.md`, `docs/requirements/SPEC.md`, release changelog/context/backlog/validation; create release note and `src/frontend/public/{llms.txt,docs/collaboration.md,docs/mcp.md,docs/diagram-catalog.md}`. Update Homelab user/developer/public-docs surfaces.

**Interfaces:** B2 writer_mode and snapshot sequence; `COLLAB_V2_ENABLED` controls new promotions, not ownership of existing v2 documents.

- [ ] **Step 1 — Prepare evidence:** record image digests/migration versions/restore point/public origin. Docs include placeholder-credential examples, schemas, API errors, scopes, limits, supported native diagram matrix and troubleshooting. Mark unshipped behavior planned.
- [ ] **Step 2 — Rollback drill:** on disposable staging pause writes, drain, persist and verify latest snapshot+log, materialize JSON, record epoch/checkpoint. Switch fixture to v1 only after comparing latest durable content. Old v2 clients receive resync/read-only and cannot replay into new epoch. Image rollback alone is unsafe; choose read-only recovery if checkpoint fails.
- [ ] **Step 3 — Canary:** use established VM100 deployment authorization, back up, apply non-destructive migrations, keep prior image. Promote test scene; test two-browser edit, guest share, agent diagram, restart/restore and deny probes. Never use down migrations for image rollback.
- [ ] **Step 4 — Browser verification:** use available browser automation on production, securely sign in, test dashboard desktop/mobile/keyboard and native canvas, deep-link refresh, invite, edit/view, live agent write and export. Capture screenshots without secrets; one session is not proof for every role.
- [ ] **Step 5 — Expand:** health/readiness/error/ack/compaction metrics, load evidence and snapshot equality must pass. Retire v1 only after exit criteria. On failure disable new promotions and follow recovery drill; record outstanding UAT accurately.
- [ ] **Step 6 — Reconcile and commit:** human and llms docs reflect deployed behavior, source/license notices and P1/P2 backlog. Record exact pass/fail/skipped results. Commit `docs: publish collaboration v2 release and operations`; follow prior merge/release authorization without unrelated changes.

## Later milestones

Comments/guest comments, screen sharing, presenter spotlight/waiting room, activity-feed UI, user version history, shared libraries/collections, SSO/SCIM/retention remain parity backlog. Snapshots/audit alone do not deliver those products. The user confirmed that org means workspace; no organization hierarchy is required.
