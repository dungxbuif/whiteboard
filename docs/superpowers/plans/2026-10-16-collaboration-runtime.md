# B — Durable collaboration runtime Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace snapshot broadcasts with authorized durable Yjs editing and offline recovery.

**Architecture:** A single Node collab process owns live rooms and persists validated updates before broadcast. Browser provider adapts Excalidraw to the shared document; Go owns access and controls writer mode.

**Tech Stack:** Yjs, y-protocols, ws, pg, IndexedDB, React, WebRTC; TypeScript 7.

**Spec:** [Approved collaboration design](../specs/2026-10-15-collaboration-v2-design.md); approved in conversation by “xác nhận”; implementation authorized by the user on 2026-10-16.

## Global Constraints

- Read [programme contracts](2026-10-16-collaboration-v2.md) before executing; its interface types, limits, dependency order and documentation requirements apply to every task.
- Custom Node/TypeScript WebSocket service; RelayHub is not involved.
- Workspace roles are `owner`, `editor`, and `viewer`; scene overrides are `editor` or `viewer`; management is owner-only.
- Invitation links: single-use, email-bound, seven-day expiry. Room JWT: ten minutes.
- Update frame: 1 MiB; materialized document: 25 MiB; 120 updates/minute/principal.
- Snapshot: first of 100 updates or five seconds. Voice: ten active participants; no product-level data-participant cap.
- Dashboard follows `references/devtool` styling; reuse its original colors and tokens without a monochrome override. Scene components retain native library styling.
- Retain TypeScript 7 toolchain, password login, default workspace provisioning, and OpenAI-compatible provider boundary.
- Every task updates its affected docs before implementation and reconciles them with verified behavior before commit.

## Review Focus

- Malicious sync-step-2 or decompressed update bypasses view access/limits — B1/B2.
- Crash after commit before acknowledgement duplicates replay — B2.
- Two clients seed an existing document or reorder/delete simultaneously — B2/B3.
- Offline editor returns after downgrade/account switch — B3.
- Duplicate tabs/users consume incorrect voice identities or slot counts — B4.

---

### B1: Shared protocol and validated scene document

**Files:** Create `src/package.json`, `src/package-lock.json`, `src/packages/collab-contracts/{package.json,tsconfig.json,src/index.ts,src/protocol.ts,src/protocol.test.ts}`, `src/collab/{package.json,tsconfig.json,src/document.ts,src/document.test.ts}`; adjust frontend workspace lockfile installation.

**Interfaces:** Produce `encodeFrame(frame:Frame):Uint8Array`, `decodeFrame(bytes:Uint8Array):Frame`, `createSceneDoc(content:SceneContent):Y.Doc`, `materializeScene(doc:Y.Doc):SceneContent`, `validateCandidate(current:Y.Doc,update:Uint8Array):ValidatedUpdate`.

- [x] **Step 1 — Record intended behavior:** update `docs/architecture/collaboration-protocol.md` (create), `docs/decisions/0010-durable-yjs-writer.md` (create) with this task's API/data/flow decision and the cases below.
- [x] **Step 2 — Add failing proof:** `protocol.test.ts`: malformed version/length/trailing bytes fail; max 1 MiB accepted boundary, +1 rejected. `document.test.ts`: materialized 25 MiB ceiling, invalid element/reference/path rejected, files exclude dataURL; test updates that decode beyond limits in an isolated worker and time out without touching live state.
- [x] **Step 3 — Verify failure:** run `rtk npm test --workspace ./collab -- document.test.ts && rtk npm test --workspace ./packages/collab-contracts`; confirm the new assertion fails for missing behavior, not a broken fixture.
- [x] **Step 4 — Implement:** Define exact Y.Map/Y.Array shape in spec; map element fields individually, preserve tombstones and bindings, derive deterministic duplicate-free materialized order. Use a worker with 2-second job deadline and memory limit for untrusted candidate validation; accepted updates only may touch live doc. Pin dependencies and TypeScript 7.0.2-compatible configuration. Root workspace scripts implement these test commands before proof execution.
- [x] **Step 5 — Verify:** rerun `rtk npm test --workspace ./collab -- document.test.ts && rtk npm test --workspace ./packages/collab-contracts`; require all listed cases to pass. Record actual results in the phase verification record; skipped integration tests do not count as passes.
- [x] **Step 6 — Reconcile and commit:** reconcile `docs/architecture/collaboration-protocol.md` (create), `docs/decisions/0010-durable-yjs-writer.md` (create); stage only the named files and supporting lockfiles; commit `feat: define validated collaboration protocol`.

### B2: Durable WebSocket runtime, snapshots and writer ownership

Required assertions after commit-before-ack fault injection and retry:

```ts
expect(retriedReceipt.sequence).toBe(firstCommittedSequence);
expect(await countAuditEvents(requestId)).toBe(1);
expect(await materializeAfterRestart()).toEqual(committedContent);
expect(viewerSyncStep2Write.status).toBe("forbidden");
```

**Files:** Create migration `000009_collab_documents.{up,down}.sql`; create `src/collab/src/{server,auth,rooms,persistence,compaction,commands}.ts` and `.test.ts`; create `src/collab/tests/durability.integration.test.ts`; modify Go scene usecase/repository to fence HTTP content writes.

**Interfaces:** Consume A4 grants and B1 codec; produce `appendUpdate(ref,principal,requestId,bytes):Promise<MutationReceipt>`, `applyCommands(grant,mutation):Promise<MutationReceipt>`, `compactScene(ref):Promise<void>`. Serve `/collab`, private `/internal/scenes/:id/commands`, `/healthz`, `/readyz`.

- [x] **Step 1 — Record intended behavior:** update `docs/architecture/collaboration-protocol.md`, `docs/operations/collaboration.md` (create), ADR-0010 with this task's API/data/flow decision and the cases below.
- [x] **Step 2 — Add failing proof:** `durability.integration.test.ts`: DB loss yields no ack/broadcast; crash after commit retry returns same receipt; duplicate ID with different bytes/principal fails; compaction retains replay receipts; snapshot failure deletes no log; double-seed produces one document. `auth.test.ts`: view sync-step-2 write denied, revoked/expired token denied. Go `TestV2SceneRejectsLegacyContentPatch`: 409 while metadata succeeds.
- [x] **Step 3 — Verify failure:** run `rtk npm test --workspace ./collab -- durability.integration.test.ts auth.test.ts`; confirm the new assertion fails for missing behavior, not a broken fixture.
- [x] **Step 4 — Implement:** Allocate monotonic bigint sequence under row lock; validate authority/version in same lock protocol as A4; append+receipt+audit atomically then apply/broadcast/ack. Persist snapshots at 100 updates or five seconds, record cutoff, atomically materialize JSON and prune only covered rows. Seed under writer-mode lock. Per-principal/IP limits, 120 updates/minute default; coalesce browser bursts. Backpressure sends resync_required/disconnect instead of losing acknowledged updates. NOTIFY loss forces ACL resync. Initial production replica count is one.
- [x] **Step 5 — Verify:** rerun `rtk npm test --workspace ./collab -- durability.integration.test.ts auth.test.ts`; require all listed cases to pass. Record actual results in the phase verification record; skipped integration tests do not count as passes.
- [x] **Step 6 — Reconcile and commit:** reconcile `docs/architecture/collaboration-protocol.md`, `docs/operations/collaboration.md` (create), ADR-0010; stage only the named files and supporting lockfiles; commit `feat: persist and authorize collaborative scene updates`.

### B3: Excalidraw adapter, offline queue and all content writers

**Files:** Create `src/frontend/src/collab/{provider.tsx,transport.ts,offline-store.ts,excalidraw-adapter.ts,undo.ts}` and matching tests; modify whiteboard wrapper, scene/presentation/AI stores, public-view, scene-content and realtime-store.

**Interfaces:** Produce `CollabProvider({sceneRef,principal,children})`, `applyLocalScene(content:SceneContent):void`, `applyAgentElements(elements:NativeElement[]):Promise<void>`, `enqueuePending(scope,requestId,update):Promise<void>` and `ackPending(scope,requestId):Promise<void>`; `scope` includes principal ID + workspace + scene + doc epoch.

- [x] **Step 1 — Record intended behavior:** update `docs/user/collaboration.md`, `docs/architecture/ARCHITECTURE.md`, `docs/operations/collaboration.md` with this task's API/data/flow decision and the cases below.
- [x] **Step 2 — Add failing proof:** `excalidraw-adapter.test.ts`: remote update creates no echo; concurrent edits/delete/reorder converge, bindings preserved. `offline-store.test.ts`: only matching ack clears update; reload/reconnect replays missing changes after sync; revoked editor retains pending recovery but sends no writes; account switch cannot replay prior user's queue. `undo.test.ts`: local undo preserves another actor's edits. `collab-writers.test.ts`: AI apply and presentation reorder persist via Yjs.
- [x] **Step 3 — Verify failure:** run `rtk npm run test:unit --workspace ./frontend -- collab`; confirm the new assertion fails for missing behavior, not a broken fixture.
- [x] **Step 4 — Implement:** Replace full-scene Socket.IO publication and HTTP autosave for v2 with diff transactions and durable ack status. Refresh room grant before expiry; use exponential reconnect with jitter. Store pending updates before send. Incoming snapshot and pending state merge via state vectors, never overwrite queue. Use transaction-origin scoped undo (including agent operation inverse only when safe); do not promise browser undo magically owns remote transactions. UI exposes pending/readonly/revoked/degraded states. Upload bytes to object storage then commit references; retry failed uploads.
- [x] **Step 5 — Verify:** rerun `rtk npm run test:unit --workspace ./frontend -- collab`; require all listed cases to pass. Record actual results in the phase verification record; skipped integration tests do not count as passes.
- [x] **Step 6 — Reconcile and commit:** reconcile `docs/user/collaboration.md`, `docs/architecture/ARCHITECTURE.md`, `docs/operations/collaboration.md`; stage only the named files and supporting lockfiles; commit `feat: integrate offline-capable Excalidraw collaboration`.

### B4: Presence, follow and voice migration

**Files:** Create `src/collab/src/{awareness,voice}.ts` and tests; create `src/frontend/src/collab/{presence,follow,voice}.ts` and tests; modify collaboration-toolbar and realtime-store; reuse existing TURN credential service.

**Interfaces:** Produce `joinVoice(connectionId):VoiceAdmission`, `leaveVoice(connectionId):void`, `follow(targetConnectionId|null):void`; server assigns connection IDs independently of user IDs and overwrites claimed sender metadata.

- [ ] **Step 1 — Record intended behavior:** update `docs/user/collaboration.md`, `docs/operations/collaboration.md`, `docs/architecture/INTEGRATIONS.md` with this task's API/data/flow decision and the cases below.
- [ ] **Step 2 — Add failing proof:** `voice.test.ts`: 10 admitted, 11th rejected only for voice, data editing continues; repeated join same connection idempotent; disconnect frees slot; spoofed cross-room recipient denied; two tabs same user distinct connections. `follow.test.ts`: self/cyclic follows rejected, target exit clears follow; viewport never persisted. Browser media denial leaves editor usable.
- [ ] **Step 3 — Verify failure:** run `rtk npm test --workspace ./collab -- voice awareness && rtk npm run test:unit --workspace ./frontend -- collab`; confirm the new assertion fails for missing behavior, not a broken fixture.
- [ ] **Step 4 — Implement:** Use one authenticated socket for awareness, follow and signaling. Restore roster/voice metadata behavior from legacy ws-whiteboard. Bind awareness clients to connection; prevent modifying others' awareness. Request media only after voice admission, clean tracks on leave/revoke, buffer early ICE and use stable offer negotiation. TURN remains existing shared Coturn; data sync never goes through media channels.
- [ ] **Step 5 — Verify:** rerun `rtk npm test --workspace ./collab -- voice awareness && rtk npm run test:unit --workspace ./frontend -- collab`; require all listed cases to pass. Record actual results in the phase verification record; skipped integration tests do not count as passes.
- [ ] **Step 6 — Reconcile and commit:** reconcile `docs/user/collaboration.md`, `docs/operations/collaboration.md`, `docs/architecture/INTEGRATIONS.md`; stage only the named files and supporting lockfiles; commit `feat: migrate presence follow and voice to collab socket`.
