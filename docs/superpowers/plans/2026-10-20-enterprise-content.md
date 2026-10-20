# Versions, Libraries and Collections Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver user-visible immutable checkpoints/diff/restore, native libraries and workspace/private collections.

**Architecture:** Node captures/restores canonical documents under its writer fence; Go owns authorization/metadata. Asset pin/reference tables protect version and library content, while collections organize accessible scenes without granting permissions.

**Tech Stack:** Existing Go/PostgreSQL, Node/Yjs validation, native Excalidraw and private S3 storage.

**Spec:** [Detail design](../specs/2026-10-20-enterprise-parity-design.md); [programme](2026-10-20-enterprise-parity.md).

## Global Constraints

Programme constraints apply; approved for implementation. Depends on F1–F3/UI1–UI2.
No epoch reset, map replacement, receipt/history deletion or HTTP content writes.
Versions <=25 MiB; auto checkpoint dirty5min, max1/minute, max100/scene.
Manual/pinned versions default indefinite. Library import <=500 items/25 MiB.
All new scene panels overlay the full canvas; dashboard follows legacy style.

## Review Focus

- Concurrent edits/checkpoint and compaction: V1 coherent sequence+asset capture.
- Restore ID/type/bindings and pending offline updates: V2 worker-valid replay tests.
- Hidden scenes exposed by private collections/counts: L1 current-access tests.
- Library ID collisions/file references/import bombs: L2 remapping and limits.
- Retention racing a reference insertion: V1/L2 pin locks consumed by R2.

## Task 1: V1 — Immutable user checkpoints and protected assets

**Files:** Create `src/collab/src/versions.ts`, `versions.integration.test.ts`,
backend `internal/domain/scene_version.go`, `internal/usecase/version_ucase.go`,
`internal/repository/postgres/version_repo.go`, `handler/version_handler.go`
under delivery/http and tests; create migrations for versions/version_assets;
modify collab Persistence/server and private Go bridge wiring.

**Interfaces:** Node `captureVersion(ref,grant,input:{requestId,expectedSequence,
label?:string,pinned:boolean}):Promise<VersionSummary>`;
Go `VersionService.Create/List/Get/SetMetadata` consume F1 version rights.
`VersionSummary={id,documentId,sequence,hash,label,pinned,createdAt,creator}`;
snapshot response contains canonical content and fresh authorized image URLs.
Captures and version asset pins commit in the same Node transaction.

- [x] Add `checkpointMatchesExactSequence`, `checkpointRetryIsImmutable`,
  `checkpointPinsImageBeforeRetention`, `versionSurvivesCompaction`,
  `TestGuestHistoryDenied`: 409 on stale sequence; digest/source sequence
  correspond exactly; no version appears if asset pin/capture fails.
- [x] Run `rtk npm test --workspace ./collab -- versions` and backend version
  tests (RED), implement immutable rows/indexes and signed version read/create
  private actions with purpose/action-specific verification; no new public Node route.
- [x] Register dirty checkpoint scheduling in Node maintenance under its existing
  ownership fence. At five minutes of unsnapshotted change create one automatic
  checkpoint, preserve manual/pinned entries, let R1 expire excess automatic rows.
  Return explicit unsupported/corrupt asset errors rather than silently omit images.
- [x] Run real DB concurrency/compaction/migration cases; add history API/ERD
  and reference ownership ADR; commit `feat(history): capture immutable scene versions`.

## Task 2: V2 — Semantic diff and durable restore with recovery checkpoint

**Files:** Create `src/collab/src/version-restore.ts`, `version-restore.test.ts`,
`version-restore.integration.test.ts`; modify `document.ts` only if invariant
coverage requires it, Persistence/authority bridge; create frontend
`src/components/history/{history-panel,version-diff,restore-dialog}.tsx` and
`src/lib/version-api.ts`, tests and real-stack version case.

**Interfaces:** `diffVersion(before:SceneContent,after:SceneContent):SceneDiff`
returns `{addedIds,removedIds,changedIds,changedAppStateFields}`;
`restoreVersion(ref,grant,{versionId,requestId,expectedSequence}):Promise<MutationReceipt &
{preRestoreVersionId:string}>`. Mutation hash binds all fields and scene/version.
Restore is a separate version:restore action, not a generic agent patch tool.

- [x] Add `restoreRetainsDocumentAndElementMaps`, `restoreTombstonesLaterAdditions`,
  `restoreRestoresBindingsOrderAndImages`, `restoreRetryCreatesOneCheckpoint`,
  `offlineReplayAfterRestoreIsALaterEdit`, `staleRestoreCommitsNothing`.
  Assert documentId unchanged, sequence increments, receipts remain, 409 before
  any pre-restore checkpoint on stale input, and validation worker accepts output.
- [x] Run focused tests (RED). Implement current-doc clone/semantic field writes:
  clear optional fields absent in target; revive/tombstone stable maps; preserve
  immutable types; restore frame/order/files with reference validation. Reject
  incompatible/missing IDs/types/assets atomically. Capture pre-restore pinned
  version and restore update+receipt+event in one transaction before ACK.
- [x] Build overlay history browser UI with label/pin/filter, diff preview,
  explicit restore confirmation and conflict retry; warn pending local/offline
  edits can replay later. Clear only local undo entries after acknowledged restore,
  preserving queued durable edits and exportable recovery.
- [x] Run real-stack image/bindings/multiple actors/offline/revoke/restore, then
  frontend build; reconcile recovery/API/user/public guides and commit
  `feat(history): restore versions through the durable writer`.

## Task 3: L1 — Legacy-style workspace/private collections

**Files:** Create backend `internal/domain/collection.go`,
`internal/usecase/collection_ucase.go`, `internal/repository/postgres/collection_repo.go`,
`internal/delivery/http/handler/collection_handler.go` and tests/migrations;
create frontend `src/components/collections/{collection-list,collection-view,
collection-dialog}.tsx`, `src/lib/collection-api.ts`, tests; modify App/sidebar.

**Interfaces:** `CollectionService.Create/Update/Delete/List/AddScene/RemoveScene/Reorder`
with F1 requestId and expectedRevision; `CollectionPage={items,nextCursor}`;
`CollectionView({workspaceId,collectionId})`. Canonical route
`/w/{workspace}/collections/{id}`. Composite collection/scene workspace FKs,
explicit private creator and visibility, stable position order.

- [ ] Add `TestPrivateCollectionDeniedToOtherMembers`,
  `TestCollectionNeverGrantsSceneAccess`, `TestCollectionForeignWorkspaceRejected`,
  `TestCollectionOrderConflict`, `legacyCollectionSidebarAndReload`;
  owners can audited-delete inaccessible private containers without reading them.
- [ ] Run focused Go/frontend tests (RED). Port legacy CollectionList/CollectionItem
  visual markup and focus states, replace Jotai/Next URLs with current router/store;
  implement all displayed menu actions. Keep folders and collections independent.
- [ ] Filter list contents/counts before pagination using scene ACL; deletion
  removes membership rows, never scenes. Test search, keyboard, mobile, deep-link
  reload, private denial and no scene relocation in real stack.
- [ ] Update collection API/ERD/dashboard/public docs; commit
  `feat(collections): organize scenes with legacy dashboard controls`.

## Task 4: L2 — Reusable native library with safe insert/export

**Files:** Create backend `internal/domain/library.go`,
`internal/usecase/library_ucase.go`, `internal/repository/postgres/library_repo.go`,
`internal/delivery/http/handler/library_handler.go` and tests/migrations;
create `src/packages/diagram-kit/src/library.ts`, tests, frontend
`src/components/library/library-panel.tsx`, `src/lib/library-api.ts`, tests;
modify Node command/asset capability bridge and installed Excalidraw callbacks.

**Interfaces:** `validateLibrary(input:unknown):NativeLibrary`,
`remapLibraryItem(item:LibraryItem,ids:IdAllocator):SceneContent` in diagram-kit;
Go `LibraryService.Import/Publish/List/GetRevision/Export/Insert` with owner-only
writes, membership reads and target edit checks. `Insert(ref,itemId,revision,
expectedSequence,requestId)` returns native element IDs and a durable receipt.

- [ ] Add `libraryRemapsEveryReference`, `libraryRejectsUnsupportedOrOversize`,
  `TestLibraryForeignAssetDenied`, `TestLibraryAssetPinRace`,
  `libraryInsertUndoAndExport`: fresh element/group/file IDs, valid arrow/text/frame
  links, native library v2 import/export, no credential-bearing scene reference.
- [ ] Run compiler/Go/frontend focused tests (RED), implement strict imported
  JSON/native/raster validation; store immutable library revisions and asset pins.
  Scope target asset metadata before a single expected-sequence writer insertion;
  stage copies idempotently and clean unused staged objects via R2 after grace.
- [ ] Integrate native onLibraryChange/updateLibrary APIs as an authorized view
  of workspace items; do not publish local personal library automatically. Use
  existing panel coordinator for management/search and `/w/{workspace}/libraries`.
- [ ] Run native image import -> insert -> reload -> version capture -> export,
  cross-workspace denial and source/target revoke tests; reconcile library/user/
  public docs and commit `feat(libraries): reuse native diagram bundles safely`.
