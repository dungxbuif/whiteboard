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
  replay after restore as later edits; only local undo history is cleared after ACK.

## Verification plan

1. Unit-test semantic diff and stable-map restore, including bindings, frames,
   images, omitted fields, incompatible types and later-added elements.
2. PostgreSQL integration-test stale rollback, idempotent retry, atomic recovery
   checkpoint/update/receipt/event and offline replay at the next sequence.
3. Test Go grant/hash binding, authenticated routes and browser conflict/draft
   behavior; run guarded collab/backend/frontend suites and production build.

## Verification results

- Focused collab restore: 2 files, 5 tests passed against guarded PostgreSQL.
- Full collab: 26 files, 190 tests passed against guarded PostgreSQL.
- Full backend: `go test ./...` passed, including handler routes, capability/hash
  binding, migrations and repository cases.
- Full frontend: 38 files, 144 tests passed; TypeScript/Vite production build
  completed successfully.
- Real stack: browser + Go API + private Node writer + PostgreSQL restore passed.
  The test captures a version, changes the scene, checks semantic diff, restores
  through the canvas panel, confirms the document UUID is unchanged and exactly
  one pinned pre-restore checkpoint exists.

The real-stack run also caught and fixed a V1 integration omission: login version
handlers now attach the canonical membership grant ID before minting Node-bound
capabilities. Node therefore accepts only the same principal binding already used
by room credentials.
