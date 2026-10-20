# Comments and Activity Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver anchored member/guest comments, moderation, in-app notifications and an authorized activity feed.

**Architecture:** Go metadata transactions emit F2 events; comments are separate from Yjs canvas content. UI uses UI2 overlay panels; all pages and notifications enforce current scene authority.

**Tech Stack:** Existing Go/PostgreSQL, React/Excalidraw and custom transport invalidations.

**Spec:** [Detail design](../specs/2026-10-20-enterprise-parity-design.md); [programme](2026-10-20-enterprise-parity.md).

## Global Constraints

Programme constraints apply; draft pending review. F1–F3 and UI1–UI2 are
prerequisites. Guest comments default off independently of view/edit access;
plain text 1–4,000 code points, pages50, writes30/minute/principal/scene.
No raw content in events, no email delivery, no comments embedded as elements.

## Review Focus

- A viewer's comment permission must not enable drawing: C1 role tests.
- Revoked guest or deleted scene during a write: C1 lock/race tests.
- Moving/deleting/restoring anchor elements: C2 native browser tests.
- Retry after notification delivery and recipient revoke: C2 job tests.
- Activity counts/cursors on private scenes: C3 SQL/API negative tests.

## Task C1: Durable comment service and guest authorization

**Files:** Create `src/backend/internal/domain/comment.go`,
`internal/usecase/comment_ucase.go`, `internal/repository/postgres/comment_repo.go`,
`internal/delivery/http/handler/comment_handler.go`, sibling tests and migration
pairs for threads/messages/subscriptions; modify router/main.

**Interfaces:** `CommentService.CreateThread(ctx,p,ref,CreateThreadInput)
(CommentThread,error)`, `Reply(ctx,p,ref,threadID,ReplyInput) (Comment,error)`,
`UpdateMessage(ctx,p,ref,threadID,messageID,CommentEditInput) (Comment,error)`,
`DeleteMessage(ctx,p,ref,threadID,messageID,RevisionInput) error`,
`UpdateThread(ctx,p,ref,threadID,ThreadEditInput) (CommentThread,error)` and
`List(ctx,p,ref,cursor,limit) (CommentPage,error)`; input mutations include
requestId and edits include expectedRevision. Anchor is `{elementId?:string,x:number,y:number}`.
Produces the six comment routes in the spec; DELETE uses a strict JSON body.

- [ ] Write `TestGuestViewCommentWithoutEdit`, `TestThreadScopeAndOwnerModeration`,
  `TestCommentUnicodeLimit`, `TestCommentRevokeRace`, `TestCommentIdempotency`:
  false guest policy403, foreign IDs404, body4001 rejected, changed edit409,
  transaction rollback leaves no message/event/notification.
- [ ] Run `rtk go test ./internal/usecase ./internal/repository/postgres
  ./internal/delivery/http/handler -run Comment -count=1` against guarded DB;
  expect RED. Implement handlers/services/repository using F1 locked writes,
  composite FKs and F2 events; no author identity accepted from request body.
- [ ] Verify authors edit/delete own messages, owner redaction, editor resolve,
  revoked principal denial and member/guest page visibility; run migrations on
  fresh DB and migration13 upgrade.
- [ ] Reconcile API/ERD/public-link policy and commit
  `feat(comments): persist authorized anchored discussions`.

## Task C2: Canvas threads, moderation and notification inbox

**Files:** Create frontend `src/components/comments/{comment-panel,comment-anchor,
notification-inbox}.tsx`, `src/lib/comment-api.ts` and tests under
`src/frontend/src`; create Go `internal/usecase/notification_ucase.go`,
`internal/repository/postgres/notification_repo.go`, `handler/notification_handler.go`
under existing backend layer directories and their tests; modify wrapper,
dashboard sidebar and route wiring; add notification migration.

**Interfaces:** `CommentPanel({sceneRef,principal})`, `CommentAnchor({anchor,api})`,
`NotificationInbox()` consume C1/F2/UI2. Backend
`NotificationService.DeliverEvent(ctx,eventID) error`,
`List(ctx,userID,cursor,limit) (NotificationPage,error)`,
`MarkRead(ctx,userID,id,requestID) error`. Deliver only currently authorized
subscribers, unique `(user_id,event_id)`; no guest private inbox.

- [ ] Add `anchorTracksMoveAndDeletedElement`, `guestReplyAndOwnerRedaction`,
  `commentPanelKeepsCanvasFull`, `TestNotificationRetryAndRevocation`: moves
  follow native geometry, deleted anchor falls back with unavailable label,
  duplicate job produces one inbox row, revoked recipient cannot read it.
- [ ] Run `rtk npm run test:unit --workspace ./frontend -- comments` in `src`
  and backend notification tests; expect RED. Implement plain-text rendering,
  escaped names, revision conflict UI with draft preserved, subscribe/read and
  actionable deeplink `/w/{workspace}/scenes/{scene}?thread={thread}`.
- [ ] Add `src/frontend/e2e/real-stack/enterprise-comments.spec.ts`: member and
  guest thread flows, desktop/mobile keyboard, Escape/focus, malicious text,
  resolve/reopen, deleted element, revocation while composer has a draft.
- [ ] Run focused unit/Go/browser tests and frontend build; add human/public
  comment and inbox guides plus llms links; commit
  `feat(comments): add canvas discussion and notifications`.

## Task C3: Workspace/scene activity browsing

**Files:** Create backend `internal/usecase/activity_ucase.go`,
`internal/repository/postgres/activity_repo.go`, `internal/delivery/http/handler/activity_handler.go`
and sibling tests; create frontend `src/components/activity/activity-panel.tsx`,
`src/lib/activity-api.ts`, tests; modify App/dashboard/sidebar/UI2 registration.

**Interfaces:** `ActivityService.List(ctx,p,workspaceID,sceneID *uuid.UUID,
cursor string,limit int) (ActivityPage,error)`, `ActivityPanel({workspaceId,sceneId?})`.
Produces GET workspace/scene activity routes and `/w/{workspace}/activity`.
Response includes `items,nextCursor,retainedSince`; safe display names only after
current permission check; security/admin event kinds owner-only.

- [ ] Add `TestActivityFiltersBeforePagination`, `TestActivityMixedSourcesStableCursor`,
  `TestActivityRetainedSince`, `activityMobileAndKeyboard`; assert restricted
  scenes absent from items/counts/cursor, equal timestamps stable, source replay
  deduplicated, secrets/body never displayed. Keep existing MCP API unchanged.
- [ ] Run focused activity tests (RED), implement index-backed keyset paging
  over F2 projections, filtered before LIMIT; fetch current subject visibility
  rather than trusting actor/event historical membership.
- [ ] Run real-stack events from edit/comment/share/agent, duplicate projection,
  revoke while paging and retention cutoff; expose authorized empty/error states.
- [ ] Reconcile activity API/guide/retention linkage and commit
  `feat(activity): expose authorized workspace timeline`.
