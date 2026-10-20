# Screen Sharing and Live Presentation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add screen sharing, hosted presentation, spotlight, waiting-room admission, reactions and viewport locking.

**Architecture:** Existing custom socket carries bounded typed controls; Node owns ephemeral session state while Go authorizes every capability. Screen peers have their own lifecycle independent of microphone voice.

**Tech Stack:** Existing Node/ws, Go authority, browser WebRTC/getDisplayMedia, TURN and React native editor slots.

**Spec:** [Detail design](../specs/2026-10-20-enterprise-parity-design.md); [programme](2026-10-20-enterprise-parity.md).

## Global Constraints

Programme constraints apply; draft pending review. Depends on F1/UI1/UI2.
One publisher/room, ten active media connections total, 720p/15fps target,
no recording/system audio. Audience waiting must receive no scene/assets/roster
until admitted. Host disconnect ends session; view lock never alters edit ACL.
All new controls are overlays/native slots; drawing survives capture errors.

## Review Focus

- Cancellation/late capture completion after leave: M1 track cleanup tests.
- Voice/screen union capacity and cross-room signal spoof: M1 runtime tests.
- Pending audience invokes sync/asset/room APIs: M2 authority-negative tests.
- Concurrent host transfer/spotlight and stale revisions: M3 session tests.
- Public revoke/refresh during media and mobile autoplay: M3 browser tests.

## Task M1: Independent admitted screen-media lifecycle

**Files:** Create `src/collab/src/screen-share.ts`, `screen-share.test.ts`,
`src/frontend/src/collab/screen-share.ts`, `screen-share.test.ts`,
`src/frontend/src/components/whiteboard/screen-share-panel.tsx`; modify server
`rooms.ts`, `voice.ts`, `awareness.ts`, frontend provider/toolbar and
`src/packages/collab-contracts/src/protocol.ts`.

**Interfaces:** Server `MediaAdmission` owns union slots via
`join(room,connectionId,kind:'voice'|'screen'):Admission`, `leave(...)`,
`canSignal(room,from,to,kind):boolean`; existing Voice delegates slot accounting
without changing voice controls. `ScreenShare.join():Promise<void>`, `leave():void`,
`subscribe/getSnapshot` produces `{status,publisher,receiving,error}`.
Controls `screen_request`, `screen_admission`, `screen_stop`, `screen_signal`
carry connection/session IDs and use current sender/recipient authority.

- [ ] Write `screenIndependentOfMic`, `voiceScreenCountOnceAtTen`,
  `eleventhMediaDeniedButDrawingWorks`, `lateCaptureStopsAllTracks`,
  `publisherLeaseExpires`, `signalCannotCrossRoom`: union capacity <=10, one
  publisher, reservation expires in30s, permission only after admission and
  explicit click, leave stops late tracks and revokes sender routing.
- [ ] Run `rtk npm test --workspace ./collab -- screen-share voice` and
  `rtk npm run test:unit --workspace ./frontend -- screen-share` in `src`;
  expect RED, then implement bounded control parsing and separate peer objects.
- [ ] After asynchronous admission show Start sharing; invoke getDisplayMedia
  synchronously from that second trusted user click, capturing video only.
  apply 1280x720/15fps constraints, handle unsupported browser without losing
  voice/drawing, replace no microphone tracks. Track-ended/authority loss closes
  senders/receivers and clears video elements. Overflow uses existing budgets.
- [ ] Prove generated video/frame delivery locally and two-network relay path
  using `tests/packaging/enterprise-media.mjs` (E1), plus voice regression.
  Reconcile protocol/media guide and commit `feat(media): add admitted screen sharing`.

## Task M2: Separate presentation admission and waiting credentials

**Files:** Create backend `internal/domain/presentation.go`,
`internal/usecase/presentation_ucase.go`,
`internal/repository/postgres/presentation_repo.go`,
`internal/delivery/http/handler/presentation_handler.go` and tests; add
presentation-link migration, purpose-specific token claims; modify room/capability
authority and router. Create `src/collab/src/presentation.ts` and tests; modify Rooms.

**Interfaces:** Go `PresentationService.Create(ctx,p,ref,requestID) (SessionGrant,error)`,
`CreateLink(ctx,p,ref,sessionID,requestID) (OneTimeShownLink,error)`,
`Join(ctx,token,displayName) (PendingPresentationGrant,error)`,
`Admit(ctx,host,sessionID,connectionID,revision) (SessionGrant,error)`,
`ResolveAudience(ctx,p,ref,sessionID) error`. Pending token purpose
`presentation-pending`, audience `whiteboard-presentation`; admitted claims
bind session+principal+scene+expiry and remain independent of ordinary links.
Node `PresentationSessions.start/stop/admit/remove/transfer` owns current
connection state; Go checks a signed private host/session decision before grant.

- [ ] Write `TestPendingTokenCannotMintRoomOrReadAssets`,
  `TestPresentationLinkRevokeDoesNotRevokeMembership`,
  `waitingClientReceivesOnlyOwnStatus`, `hostDisconnectEndsSession`.
  Pending auth sends neither ready/sync_chunk nor roster and rejects binary
  data/voice/screen. Admission requires active matching session and current host.
- [ ] Run focused Go Presentation tests and Node `presentation` tests; expect
  RED. Implement grant-bound links (hashed, expiry/revision, explicit revoke),
  private session validation and `presentation_auth` distinct from room auth.
- [ ] Serialize active session changes on existing single runtime owner; cap
  waiting queue at100, session invites24h, pending grants5min; attempts rate60/min/IP.
  Store links, not ephemeral participants, in Postgres. Restart invalidates sessions.
- [ ] Run member/guest/revoke/expiry/cross-scene negative tests, migration
  upgrade and room/asset regressions; reconcile API/ERD/token ADR and commit
  `feat(presentation): enforce audience waiting-room admission`.

## Task M3: Host controls, spotlight/reactions/view lock and audience UI

**Files:** Create frontend `src/collab/live-presentation.ts`, tests,
`src/components/presentation/{host-controls,audience-view,waiting-room}.tsx`
under `src/frontend`; create `src/frontend/e2e/real-stack/enterprise-live.spec.ts`;
modify App, presentation store, native presence mapping and wrapper;
extend Node presentation tests/protocol from M2.

**Interfaces:** `LivePresentation` subscribes to existing transport and produces
`{sessionId,revision,hostConnectionId,frameId,spotlight,locked,status,reactions}`;
actions `setFrame`, `spotlight`, `setLocked`, `admit`, `remove`, `transfer`,
`react`, `leave` send expected session revision (except ephemeral reactions).
`/present/{token}` loads pending/admitted UI with refresh handling; host links
are copied manually, never emailed automatically.

- [ ] Add `staleHostRevisionRejected`, `transferOnlyToEditor`,
  `spotlightClearsOnTargetLeave`, `reactionAllowlistRateAndExpiry`,
  `lockedViewportStillAllowsLeave`, `pendingAudienceNoSceneLeak`.
  Concurrent frame/host changes must have one current authoritative revision;
  reaction expiry3s/rate1s, max100 queued guests (M2), offline host ends session.
- [ ] Run Node/frontend focused tests (RED), implement native frame/camera
  follow using IDs, not slide array index; removed frames produce a fallback
  message without jumping to a different frame silently. Lock changes viewport
  only; authorization remains F1/current scene policy.
- [ ] Run real-stack two guests+host: admit/deny/remove/transfer, mobile waiting
  dialog, screen/voice permission denial, autoplay prompt, room refresh/revoke,
  keyboard Escape and controls remain floating over full canvas.
- [ ] Reconcile presentation/media API/user/public docs and llms links; commit
  `feat(presentation): add live host and audience controls`.
