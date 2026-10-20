# Enterprise V2 restore verification — 2026-10-20

Scope: semantic version diff, writer-fenced restore, pinned pre-restore recovery,
browser history controls, offline replay guidance and production API wiring.

## Intended contract

- Diff compares stable element IDs and safe app-state fields without exposing
  credentials or minted asset URLs.
- Restore binds `workspaceId`, `sceneId`, `versionId`, `requestId` and exact
  decimal `expectedSequence` into a `version:restore` decision.
- Node rechecks current policy and authority inside the workspace/scene writer
  transaction. A stale sequence commits no checkpoint, update, receipt or event.
- A successful restore keeps `documentId`, mutates existing element field maps,
  tombstones later additions, restores order/files/presentation order, validates
  all references, creates one pinned pre-restore checkpoint and advances the
  durable sequence once. Identical retry returns the original receipt/checkpoint.
- Browser confirmation explains that queued offline edits remain durable and may
  replay after restore as later edits. Acknowledgement clears collaboration-local
  and native canvas undo history. Uncertain retries retain their exact request ID
  and expected sequence pair. Only sequence `CONFLICT` waits for a newer sequence
  before creating a new request identity; other 409 codes are definitive.
- Authorized fanout sends committed update frames before their durable sequence.
  Tombstoned image file metadata survives restore so an offline edit can revive
  the stable element map without losing its referenced asset.
- Never-sent offline intent is causally rewritten after fresh sync, independent
  of Yjs client-ID ordering. Reconnect waits for local durability and atomically
  replaces the complete creation-ordered pending set, independent of IndexedDB
  key order. Ambiguous sent rows keep immutable request bytes.
- Restore uses trusted worker validation up to the 64 MiB snapshot ceiling.
  Ordinary inbound frames remain 1 MiB; oversized restore fanout sends authorized
  512 KiB durable sync chunks before publishing its committed sequence.

## Verification plan

1. Unit-test semantic diff and stable-map restore, including bindings, frames,
   images, omitted fields, incompatible types, later-added elements and an offline
   image revival after restore.
2. PostgreSQL integration-test stale rollback, idempotent retry, atomic recovery
   checkpoint/update/receipt/event and offline replay at the next sequence.
3. Test Go grant/hash binding, authenticated routes, request-ID collision,
   browser retry/conflict behavior, undo reset and remote sequence fanout; run
   guarded collab/backend/frontend suites, production build and real stack.

## Verification results

- Focused offline-replay race suite passed three consecutive runs. It covers a
  delayed local durability write crossing reconnect, multi-row replacement whose
  creation order differs from IndexedDB key order, and a 32-iteration Yjs
  client-ID loop for image revival. If atomic replacement loses a concurrent
  storage race, the transport degrades without sending raw bytes and preserves
  the pending rows for recovery.
- Focused collab restore/integration coverage passed inside the full guarded
  suite. It covers bindings/order/images, tombstone image revival, a valid restore
  update above the socket frame ceiling, atomic stale rejection, idempotency,
  request-ID collision and ordered/chunked sequence fanout.
- Real stack: 1 Playwright test passed against browser + Go API + private Node
  writer + PostgreSQL. It pastes a real PNG, captures it, observes a second
  actor's update and fresh sequence, denies viewer restore, queues an offline edit,
  restores the image version, verifies both undo stacks are unavailable, then
  reconnects and replays the edit while preserving the image, document UUID and
  exactly one pinned pre-restore checkpoint.
- Full collab: 26 files, 196 tests passed against guarded PostgreSQL.
- Full guarded workspace `npm test`: collab 196, MCP 43, collaboration
  contracts 22 and diagram kit 41 tests passed; dependent bundles built.
- Full frontend: 43 files, 169 tests passed; TypeScript/Vite production build
  completed successfully. The replay assertion waits for its observable outbound
  frame, avoiding a fixed-delay race exposed only by the parallel full suite.
- Full backend: guarded `go test ./...` passed, including the authenticated
  version routes and authority/hash binding used by restore.
- Documentation: `git diff --check` passed; the human guide and its public
  agent-readable API page describe retry identity, sequence fanout, image revival
  and undo-reset behavior. `llms.txt` already links the public version guide.

The real-stack run also caught and fixed a V1 integration omission: login version
handlers now attach the canonical membership grant ID before minting Node-bound
capabilities. Node therefore accepts only the same principal binding already used
by room credentials.
