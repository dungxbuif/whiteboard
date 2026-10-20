# Enterprise Foundation and Canvas Controls Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship full-canvas controls and the policy/event/job contracts required by all enterprise modules.

**Architecture:** UI integration stays inside the native editor; Go owns metadata policy and durable events; bounded jobs project existing audits without changing Yjs receipt semantics.

**Tech Stack:** Existing React/Excalidraw, Go/GORM/Casbin, PostgreSQL and Node.

**Spec:** [Enterprise detail design](../specs/2026-10-20-enterprise-parity-design.md); [programme/global constraints](2026-10-20-enterprise-parity.md).

## Global Constraints

All programme constraints apply. Status approved for implementation on 2026-10-20.
Canvas is `100dvh`; no separate control header. Preserve native editor tools,
legacy dashboard tokens, authority, document identities and permanent receipts.
New mutations use principal-bound UUID requestIds and decimal revisions.

## Review Focus

- Native toolbar/library overlap at 390x844 and 1440x900: UI1 bounding-box tests.
- Dialog/textarea keyboard handling versus canvas undo: UI2 focus/shortcut tests.
- Mid-transaction role/policy revoke: F1 locked authority test.
- Duplicate/out-of-order events and restricted-scene pages: F2 replay/filter tests.
- Worker crash after side effect before acknowledgement: F3 lease/idempotency test.

## Task 1: UI1 — Move existing scene controls into Excalidraw

**Files:** Modify `src/frontend/src/components/whiteboard/excalidraw-wrapper.tsx`,
`collaboration-toolbar.tsx`, `src/frontend/src/index.css`; create
`src/frontend/src/components/whiteboard/scene-controls.tsx`,
`src/frontend/src/test/scene-controls.test.tsx`,
`src/frontend/e2e/real-stack/canvas-controls.spec.ts`.

**Interfaces:** Consumes current wrapper title/onBack/state/undo/presence/voice.
Produces `SceneControls({title,onBack,status,canUndo,canRedo,onUndo,onRedo,
onPresent,onExportRecovery,onRetry,children}: SceneControlsProps)` and native
`renderTopRightUI`/MainMenu integration. `status` carries text, severity and
recovery actions; preserve existing accessible button names.

- [ ] Write tests `fillsViewportWithoutHeader`, `controlsAvoidNativeTools`,
  `keepsSavedRecoveryAndBackAccessible`: canvas bounds equal viewport (1px
  tolerance), zero reserved header height, title/Back/People/Join voice reachable,
  no idle `Voice 0/10`, native tool hitboxes unobstructed on both viewport sizes.
- [ ] Run `rtk npm run test:unit --workspace ./frontend -- scene-controls` in
  `src`; expect RED for missing component. Run
  `rtk npm run test:e2e:collab --workspace ./frontend -- canvas-controls` after
  the real-stack fixture is ready; expect viewport/control RED before the fix.
- [ ] Implement a single `100dvh` canvas root; use native `renderTopRightUI` for
  People/voice/Present and MainMenu for Back/secondary undo/redo/recovery.
  Float title/status within the canvas without changing native editor geometry.
  Safe-area aware overlays ignore pointer input outside their buttons.
- [ ] Run focused tests, the real-stack canvas-controls case and frontend build;
  verify light/dark, keyboard, mobile and public view-only state with screenshots.
- [ ] Reconcile `docs/architecture/BROWSER_COLLABORATION.md`, dashboard/scene user
  guide and public counterpart; commit `feat(ui): integrate controls into canvas`.

## Task 2: UI2 — Shared overlay panel and focus ownership

**Files:** Create `src/frontend/src/components/whiteboard/scene-panels.tsx`,
`scene-panels.test.tsx`; modify `whiteboard/ai-command-panel.tsx`,
`presentation-controls.tsx`, `slide-navigator.tsx`, `excalidraw-wrapper.tsx`.

**Interfaces:** Produces `ScenePanelId = 'comments'|'activity'|'history'|'library'|'ai'`
and `ScenePanelsProvider`, `useScenePanels(): {active:ScenePanelId|null,
open(id:ScenePanelId):void,close():void}`. Only implemented panels are registered;
do not expose dead feature buttons. Consumes UI1 native control surface.

- [ ] Write `onlyOnePanelOpen`, `escapeReturnsFocus`, `textareaUndoIsNative`,
  `panelDoesNotResizeCanvas`, `mobilePanelHas44pxControls`; assert replacing a
  panel restores/retargets focus and never changes scene elements or viewport.
- [ ] Run `rtk npm run test:unit --workspace ./frontend -- scene-panels`; expect
  missing provider RED, then implement named panel registration and overlay focus
  management. Use non-docked native Sidebar only if its installed behavior
  preserves canvas bounds; otherwise use the existing accessible dialog primitive.
- [ ] Integrate AI and floating presentation navigation; test no collision with
  native Library/zoom tools and no global shortcut interception in inputs.
- [ ] Run focused tests and UI1 real-stack cases; document panel behavior in
  the same internal/public guides and commit `feat(ui): coordinate canvas panels`.

## Task 3: F1 — Feature authority, policy and idempotent metadata writes

**Files:** Create `src/backend/internal/domain/enterprise.go`,
`internal/usecase/feature_ucase.go`, `internal/repository/postgres/feature_repo.go`,
`internal/delivery/http/handler/feature_handler.go` and sibling tests; modify
`internal/usecase/access_ucase.go`, router/main wiring and new migration pairs.

**Interfaces:** Produces `FeatureService.Require(ctx domain.Context,
p domain.Principal, ref domain.SceneRef, action FeatureAction) error`,
`Policy(ctx,p,workspaceID) (WorkspaceFeaturePolicy,error)` and
`FeatureRepository.WithMutation(ctx,p,ref,action,requestID,payloadHash,fn)
(StoredFeatureResponse,error)`. `fn` executes against the transaction-bound
context; returning an error rolls back feature state, event and receipt.
`WorkspaceFeaturePolicy` uses the spec's memberComments/memberHistory,
guest/link overrides and feature enable flags. Owner policy updates use
expectedRevision. Produces `GET|PATCH /workspaces/{id}/feature-policy`.
Update existing `internal/domain/share_link.go`, share-link service/handler and
frontend `src/lib/collaboration-api.ts`/share-link settings DTOs for
`allowComments`, preserving the existing view/edit contract.

- [ ] Add `TestFeaturePolicyMatrix`, `TestCommentDoesNotGrantEdit`,
  `TestFeatureRevokeDuringMutation`, `TestMetadataRetryBinding` with actual
  PostgreSQL; viewers cannot restore/manage, public comment false defaults deny,
  matching request returns prior JSON, changed request returns 409.
- [ ] Run `rtk go test ./internal/usecase ./internal/repository/postgres -run
  'Feature|MetadataRetry' -count=1` with guarded DB; expect missing implementation
  RED. Allocate free migration numbers >=14 for policy and feature receipts.
- [ ] Implement current-authority workspace-first lock and transaction-bound
  Casbin feature rules. Share-link `allow_comments` changes bump ACL/revision.
  Go handler validates strict DTO/body and never trusts workspace headers alone.
- [ ] Run negative and concurrency tests plus full Go tests; add API/ERD/access
  docs and a reviewed ADR for feature policy; commit
  `feat(access): authorize enterprise feature actions`.

## Task 4: F2 — Transactional events, projection and invalidations

**Files:** Create `src/backend/internal/domain/workspace_event.go`,
`internal/usecase/event_ucase.go`, `internal/repository/postgres/event_repo.go`,
`src/collab/src/activity-projector.ts` and sibling tests; modify Node maintenance
registration, `rooms.ts`, `src/packages/collab-contracts/src/protocol.ts`,
frontend transport and new migration pairs.

**Interfaces:** Produces `AppendEvent(ctx, EventInput) (Event,error)` within an
existing metadata transaction; `ProjectAuditBatch(ctx,limit int) (int,error)`
for ACL/agent sources and Node `projectMutationAuditBatch(client,limit):Promise<number>`
for collab audit. All sources use unique `(source,sourceKey)` and persistent
high-water cursors. `feature_changed {eventCursor:string}` contains no content
or subject names; recipients must be reauthorized before delivery.

- [ ] Add `eventRollbackIsAtomic`, `projectionCrashRetryDeduplicates`,
  `restrictedSceneDoesNotLeak`, `TestEventContainsNoSecrets`. Assert commit-only
  events, identical retry does not notify twice, stale cursor triggers refetch,
  and raw text/body/SDP/ICE/token fields are rejected from safe metadata.
- [ ] Run backend event tests and `rtk npm test --workspace ./collab --
  activity-projector` in `src`; expect RED, then implement outbox/projectors
  with limit 100, source-specific cursor locking and idempotent inserts.
- [ ] Wire invalidations on existing admitted `/collab/`; dashboard initially
  refetches on focus/route and manual refresh (no second socket needed). An
  invalidation can be dropped safely because pages are refetched after reconnect.
- [ ] Prove old MCP activity shape and mutation ACK/receipt regressions unchanged;
  reconcile protocol/API/event ERD and commit `feat(activity): project workspace events`.

## Task 5: F3 — Bounded feature jobs and readiness

**Files:** Create `src/backend/cmd/feature-worker/main.go`,
`internal/usecase/feature_jobs.go`, `internal/repository/postgres/job_repo.go`,
sibling tests and lease migration; modify Docker backend build target and
production Compose; create `docs/operations/enterprise-jobs.md`.

**Interfaces:** Produces `FeatureJobs.RunOnce(ctx context.Context, now time.Time,
limit int) (JobBatchResult,error)` and job handlers registered by event kind.
Claims max100 rows, 60-second lease, 30-second operation deadline, backoff
1/2/4/8/16/32/60 minutes and dead-letter after 10 failures. Handlers are
idempotent; destructive handlers additionally consume R1/R2 safeguards.

- [x] Add `TestTwoWorkersClaimOnce`, `TestExpiredLeaseResumes`,
  `TestCrashAfterEffectRetriesSafely`, `TestShutdownDrainsWithinDeadline`;
  queue outage must make worker unready without changing web mutation durability.
- [x] Run `rtk go test ./internal/usecase ./internal/repository/postgres -run
  'Job|Lease' -count=1`; expect RED. Implement leases via SKIP LOCKED with
  deterministic injectable clock and no unbounded in-memory queue.
- [x] Add health/readiness and Compose nonroot/resource/secrets configuration;
  verify process kill/restart in isolated packaging fixture before VM100.
- [x] Reconcile deployment/environment/Homelab worker docs during rollout;
  commit `feat(jobs): run bounded enterprise background work`.
