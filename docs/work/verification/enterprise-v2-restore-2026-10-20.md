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
  and native canvas undo history. Uncertain retries retain their request ID;
  conflicts wait for a newer sequence before creating a new request identity.
- Authorized fanout sends committed update frames before their durable sequence.
  Tombstoned image file metadata survives restore so an offline edit can revive
  the stable element map without losing its referenced asset.

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

- Focused collab review suite: 3 files, 13 tests passed against guarded
  PostgreSQL. It covers bindings/order/images, tombstone image revival, atomic
  stale rejection, idempotency, request-ID collision and ordered sequence fanout.
- Focused frontend review suite: 4 files, 23 tests passed, followed by a
  successful TypeScript/Vite production build. It covers actual local undo reset,
  transport sequence advancement, uncertain retry identity and conflict rebase.
- Real stack: 1 Playwright test passed against browser + Go API + private Node
  writer + PostgreSQL. It pastes a real PNG, captures it, observes a second
  actor's update and fresh sequence, denies viewer restore, queues an offline edit,
  restores the image version, verifies both undo stacks are unavailable, then
  reconnects and replays the edit while preserving the image, document UUID and
  exactly one pinned pre-restore checkpoint.
- Full collab: 26 files, 193 tests passed against guarded PostgreSQL.
- Full workspace `npm test`: collab 193, MCP 43, collaboration contracts 22
  and diagram kit 41 tests passed; dependent TypeScript bundles built.
- Full frontend: 40 files, 151 tests passed; TypeScript/Vite production build
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
