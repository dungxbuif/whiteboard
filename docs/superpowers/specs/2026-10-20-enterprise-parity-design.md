# Enterprise parity — proposed detail design

Status: draft for review, 2026-10-20. Scope direction is approved by
ENT-P1-01–04 and ENT-P2-01–03; the new contracts and defaults below are proposals.
Implementation approval has not been recorded. This document is the enterprise
`DETAIL_DESIGN` artifact; no runtime change is authorized by its draft status.

## Goal and trace

Deliver comments/guest comments, screen sharing, live presentation/spotlight/
waiting room/reactions/view locking, activity, version history/diff/restore,
workspace libraries/collections, domain capture, SSO/SCIM and retention.
Org means workspace; no new organization hierarchy is introduced.

- [Requirements](../../requirements/REQUIREMENTS.md#approved-product-backlog)
- [Programme/task index](../plans/2026-10-20-enterprise-parity.md)
- [Phase](../../work/phases/PHASE-ENTERPRISE.md)
- [Backlog](../../work/BACKLOG.md), [roadmap](../../work/ROADMAP.md)
- [Planning verification](../../work/verification/enterprise-planning-2026-10-20.md)
- [Validation matrix](../../work/VALIDATION_MATRIX.md)
- [Existing accepted access design](2026-10-15-collaboration-v2-design.md)
- [Dashboard decision](../../decisions/0012-legacy-dashboard-presentation.md)

## Evidence and legacy comparison

Baseline: application `1d2af8a`, migrations 1–13, Go/Chi/GORM/Casbin,
React/Vite/Zustand, native Excalidraw, Node/Yjs custom `/collab/`, PostgreSQL,
private S3-compatible `whiteboard` storage and native MCP.

| Source inspected | Reuse | Missing or unsuitable behavior |
| --- | --- | --- |
| `src/backend/internal/domain/access.go`, `repository/postgres/access_repo.go` | Principal/SceneRef, current ACL, workspace-first lock order | No enterprise action policy or SSO assurance |
| `src/collab/src/persistence.ts`, `commands.ts`, `document.ts` | Exclusive writer, expected sequence, validation worker, permanent receipts, tombstones | No user versions or restore command; compaction is not product history |
| migrations `000007`, `000009`, `000013` | ACL audit, durable mutation audit, agent operation audit | No unified human activity feed or notifications |
| `src/collab/src/rooms.ts`, `voice.ts`, frontend `collab/voice.ts` | Authenticated presence, per-connection identity, bounded signaling, cleanup | No screen/session channels or audience admission |
| frontend `store/presentation-store.ts`, `whiteboard/presentation-controls.tsx` | Local frame navigation, keyboard controls | No server-owned presenter leadership |
| `references/devtool/src/components/Sidebar/CollectionList/{index,CollectionItem}.tsx` | Sidebar spacing, icons, selection states, private collection visual affordance | Menu actions have no implemented handlers; Next/Jotai wiring must not be copied |
| `references/server/src/modules/collection/{collection.service,collection.controller}.ts`, `entities/collection.entity.ts` | Workspace collection naming and private-collection concept | CRUD is incomplete; `isPrivate` alone is not authorization |
| `references/ws-whiteboard/src/index.ts` | Follow and participant lifecycle concepts | Free-form Socket.IO rooms cannot replace current authenticated transport |
| frontend `styles/dashboard.css`, `index.css` | Legacy dashboard appearance and scene Excalidraw accent | Do not introduce another theme or redesign |

Targeted search found no complete comments, product versions, screen sharing,
OIDC/SCIM, or enterprise retention module in these inspected modules. Reuse the
working primitives above; do not treat stubs as feature completion.

## Architecture and invariant ownership

```text
Browser / MCP -> Go authority + feature services -> PostgreSQL metadata/outbox
     |                     |                              |
     |                     +-> signed private commands    +-> feature job worker
     v                              v                              |
Custom /collab/ <---------- Node/Yjs single writer              S3 objects
     |
     +-> ephemeral session/media signaling -> browser WebRTC + existing TURN
Go OIDC adapter <-> workspace IdP; SCIM clients -> Go SCIM API
```

1. Go owns metadata, ACL, feature policy, domain verification and identity.
   Node owns canonical Yjs changes. Metadata cannot write `scenes.content` for
   promoted documents. No second socket, RelayHub or new scene writer.
2. All tenant tables carry `workspace_id`; scene children use composite foreign
   keys `(workspace_id,scene_id)`. Authorization runs before every read/write,
   export, fanout and asynchronous job, under current workspace policy.
3. Scene versions preserve `documentId`, existing element IDs/types and nested
   Y.Map identity. No epoch reset, receipt pruning or deletion of `gc:false`
   history is permitted by product restore or retention.
4. Native canvas style remains Excalidraw. Dashboard additions reuse legacy
   sidebar/cards/dialogs/light-dark tokens. Scene controls use the deployed
   Excalidraw primary. All new routes reload and respect UUID/alias resolution.
5. Every mutating HTTP feature operation accepts `requestId` (UUID), binds the
   canonical request hash and principal, and returns its prior response on an
   exact retry; changed bytes with the same ID return `409 REQUEST_ID_REUSED`.
   All revisions, sequences and cursor counters cross JSON as decimal strings.
6. Proposed ADR topics: feature policy/outbox, version restore, OIDC/SCIM
   assurance, retention/asset reference lifecycle. Create accepted ADRs at each
   approved implementation slice, not by declaring this draft accepted.

## Canvas controls — user decision 2026-10-20

The user explicitly requests controls inside/floating over the canvas, without
a separate header/control card consuming canvas height. This applies to the
current Back/title/People/voice controls and all new enterprise scene controls.

Replace `excalidraw-wrapper.tsx`'s two-row main/header layout with one
`100dvh` canvas surface (safe-area aware). Use the installed Excalidraw
`renderTopRightUI`, `MainMenu`, and non-docked Sidebar integration points:
Back and full scene title are available in MainMenu; the title also appears in
a compact top-left floating island where it fits next to the native menu.
People/avatar count, microphone state and Present live in the top-right native
slot, with one More menu for secondary actions. Hide `Voice 0/10` when idle;
show meaningful call count only during a call. Preserve accessible names and a
single h1 (visually hidden on compact/mobile layouts).

Comments/activity/history open an overlay panel using at most one active panel
at a time; library and AI controls share that panel coordinator. Presentation
navigation is a floating bottom island. Panels do not shrink the canvas.
Default save state is a compact status chip; errors/recovery appear in an
accessible floating notice with actionable export/retry, not a new document row.
Keep semantic undo/redo through existing keyboard handling; expose labeled
controls in More and native integration, without intercepting text-field undo.

Below 768px collapse title and secondary controls into the menu; primary touch
targets are at least 44px. At 390x844 and 1440x900 the canvas bounding rectangle
covers the entire viewport, toolbar controls never overlap native shape tools,
zoom/library buttons or safe areas, and transparent overlay containers use
`pointer-events:none` with interactive children `pointer-events:auto`. Escape,
focus return, keyboard traversal and screen-reader status remain functional.
No user drawing or viewport is changed by opening a control panel.

## Proposed common contracts and defaults

`FeatureAction` = `comment:read|comment:write|comment:moderate|activity:read|
version:read|version:create|version:restore|library:read|library:write|
collection:manage|presentation:host|screen:publish|enterprise:manage`.
`FeatureService.Require(ctx, principal, sceneRef, action)` resolves effective
scene role plus workspace policy. Authenticated owners/editors comment; viewers
can comment when `memberComments=true` (default true). Public view/edit grants
have independent `allowComments=false` by default; allowing comments never
turns a viewer into an editor. Agents have no new rights through old key scopes;
explicit future key scopes must be added with the corresponding tools.

Owners moderate comments, manage workspace libraries, sharing, domain/SSO/SCIM
and retention. Editors can create versions and restore accessible scenes;
viewers may inspect versions when `memberHistory=true` (default true). Public
guests cannot read history or workspace activity. Members see scene activity
only for currently accessible scenes; owners alone see security/admin events.
Public guest comments are visible to current scene readers with comments enabled.

Feature metadata uses Go transactions and an outbox written atomically.
`workspace_events(id BIGINT, workspace_id, scene_id nullable, actor_kind,
actor_id, kind, subject_id, occurred_at, safe_metadata)` contains no document
content, raw comments, credentials, SDP or ICE candidates. A separate
`workspace_event_outbox` fans out cursor-only invalidations; duplicate delivery
is normal. Browsers refetch authorized pages (limit 50, stable `(occurred_at,id)`
cursor); reconnect refetches. Notification inbox rows are unique by
`(user_id,event_id)` and marked read per user. Only in-app notifications ship;
email/push would require a separate delivery decision.

Proposed `FeatureJobs.RunOnce(ctx, now, limit)` processes at most 100 rows per
batch with leases, retries and `FOR UPDATE SKIP LOCKED`. Node maintenance projects
`collab_mutation_audit` into `workspace_events` using a persistent source cursor
and unique `(source,source_key)`. Go projects ACL/agent audit likewise. Workspace
listing never discloses a restricted scene through counts, actors or cursors.

## Comments and activity

Tables: `comment_threads` (scene, creator principal, optional elementId plus
fallback x/y, open/resolved state, revision), `comments` (thread, author
principal, plain text body, revision, edited/deleted timestamps),
`comment_subscriptions`, `user_notifications`. Body 1–4,000 Unicode code points,
no raw HTML, max 50 messages per page, 30 writes/minute/principal/scene.
Foreign thread IDs and stale edit revisions fail; authors edit/delete their own
messages, owners can moderate, editors can resolve. Moderation leaves a visible
redaction marker; privacy purge follows the retention policy.

Anchors track native element IDs through move/resize. Deleted elements show
“Element unavailable” with the stored fallback location; comments never create
fake Excalidraw elements. No automatic anchor retarget on restore. Replies and
thread creation subscribe their authenticated authors; inbox recipients are
subscribers reauthorized at delivery and read time. Guests have no private
workspace inbox and no implicit email identity.

Proposed API under `/api/v1`:
- `GET|POST /scenes/{scene}/comments`; `GET /scenes/{scene}/comments/{thread}`.
- `POST /scenes/{scene}/comments/{thread}/replies`;
  `PATCH|DELETE /scenes/{scene}/comments/{thread}/messages/{message}`.
- `PATCH /scenes/{scene}/comments/{thread}` for resolution/subscription.
- `GET /workspaces/{workspace}/activity`; `GET /scenes/{scene}/activity`.
- `GET /notifications`; `POST /notifications/{id}/read`.

Public callers use the existing scoped scene-session token with separately
checked comment policy. Routes never accept a display name as identity.
Activity adapters retain source event provenance and do not modify the existing
MCP `get_scene_activity` response shape in this release.

## Screen sharing and live presentation

Separate `ScreenShare` and `PresentationSession` controllers consume the current
transport; they do not depend on joining microphone voice. Room capacity is ten
active media connections total (voice or screen, counted once), one screen
publisher per room, default max 720p at 15 fps, no recording/system audio.
Screen publisher must be an authenticated owner/editor; admitted member/guest
viewers can receive. Admission occurs before the user-triggered capture prompt;
picker cancellation, late permission, disconnect, expiry or revocation releases
the reservation. Browser Stop sharing ends tracks and clears roster metadata.
Use two explicit user steps when admission is asynchronous: request a slot,
then click Start sharing after admission; call getDisplayMedia synchronously
from that second click to preserve browser transient activation. Reservations
expire after 30 seconds if capture never starts.
Feature-specific errors leave drawing usable. Presentation sessions may have
more than ten audience connections, but media admission is separately bounded.

Server-owned ephemeral session: `sessionId`, host connection/principal,
revision, frameId, spotlight target, view-lock flag, admitted/pending audience,
media slots and bounded reactions. Only authenticated owners/editors host;
explicit transfer is allowed to another authenticated editor. Host disconnect
ends the session immediately; reconnect requires an explicit new session.
Session state is not Yjs content. Reactions use a fixed allowlist
`thumbs_up|heart|clap|question`, one/second, expire after three seconds.
Waiting queue is capped at 100; presentation links expire after 24 hours and
pending grants after five minutes; attempts are limited to 60/minute/IP.

Waiting-room access is presentation-specific: a waiting guest receives title
and its own admission status only. No scene snapshot, assets, roster, media or
presentation viewport before admission. A normal existing scene link continues
to obey its independent view/edit permission. Kicking a presentation participant
revokes that session grant; it does not silently revoke unrelated memberships.
View locking controls the attendee viewport only, never grants or revokes edit
permission. Attendees always have a visible Leave presentation control.

API: owner/editor `POST /scenes/{scene}/presentations`; host creates/revokes
presentation invite links; `POST /presentations/{token}/join` obtains a pending
audience credential. New token purpose/audience is separate from room/login
tokens. `/collab/` accepts `presentation_auth` with metadata-only pending state;
on admission Go issues a current-authority scene/presentation credential before
sync. Add strict `screen_*` and `presentation_*` controls with server-derived
sender IDs, expected session revision, and current authority on both peers.

## User versions and restore

`scene_versions` stores immutable canonical SceneContent JSON, SHA-256,
documentId, source sequence, label, creator and pinned status; JSON <=25 MiB.
`version_assets` pins existing authorized asset metadata/objects. Manual create
is expected-sequence guarded; automatic checkpoints occur after five minutes of
dirty state, at most one/minute, with 100 automatic versions/scene by default.
Manual/pinned versions default to no automatic expiry. Version capture, asset
pinning and its event commit atomically under the writer transaction.

APIs: `GET|POST /scenes/{scene}/versions`, `GET /scenes/{scene}/versions/{id}`,
`PATCH /scenes/{scene}/versions/{id}` (label/pin),
`GET /scenes/{scene}/versions/{id}/diff?againstSequence=...`,
`POST /scenes/{scene}/versions/{id}/restore` with `expectedSequence,requestId`.
Diff reports added/removed/changed element IDs and changed safe appState fields;
asset URLs are minted afresh. Restore first captures a pinned pre-restore
checkpoint, validates asset availability, restores allowed fields/order and
tombstones later additions through one new durable update. It never rewinds the
sequence. Stale expectedSequence returns 409 without a partial write. Existing
offline edits can replay after restore and become later edits; UI explains this.
Legacy v1 scenes are promoted through the existing path before version writes.

## Libraries and collections

Collections are named, many-to-many scene lists, separate from folders; adding
a scene never moves it or grants its access. Tables `collections` (workspace,
creator, private/workspace visibility, revision), `collection_scenes` (position),
`library_items` (immutable native element bundle revisions, metadata),
`library_assets`. Private collections are creator-visible (owners can remove a
private collection via an audited management action without reading its content);
workspace collections obey membership plus each scene's current ACL.

Library item writes are owner-only; members can browse workspace items and
insert them into editable scenes. Guest import/export is excluded. `.excalidrawlib`
imports validate version 2 and native elements, <=500 items / 25 MiB; items may
contain images only after existing raster constraints, metadata and pinned asset
storage are verified. Insert generates fresh IDs for elements/groups/files and
rewrites frameId/containerId/boundElements/startBinding/endBinding/presentation
references. Scene-scoped asset capabilities are minted after both source-library
and target-scene authorization; no foreign scene asset ID is trusted.

API: `/workspaces/{workspace}/collections` CRUD, membership/order subroutes,
`/workspaces/{workspace}/libraries` list/create/import, item revision/export
subroutes, `POST /scenes/{scene}/library-insertions` with expectedSequence.
Routes `/w/{workspace}/collections/{id}` and `/w/{workspace}/libraries` reload;
legacy `Private` UI is reused only once its backend rule is implemented.

## Domain capture, SSO and SCIM

SSO proposal is OIDC Authorization Code with PKCE using `coreos/go-oidc/v3`
plus existing `golang.org/x/oauth2`; a pinned compatible release is selected and
audited at execution. SAML is not assumed by the current requirement; an IdP
needing SAML requires a separately reviewed adapter. Google consumer OAuth stays
retired. Password users/default workspace provisioning continue to work.

`workspace_domains` verifies DNS TXT `_whiteboard-verification.{domain}` using
a 32-byte random single-use challenge, 24h expiry and exact normalized IDNA
domain match; verification is rechecked before capture. Domain capture suggests
the workspace login and requires explicit user acceptance to join. It never
moves existing scenes, steals accounts, or merges identities by email alone.
`sso_connections`, `external_identities(issuer,subject)`, `auth_sessions` and
`workspace_identity_memberships` carry workspace-specific assurance and status.
Verified domain does not authorize an arbitrary issuer.

OIDC state/nonce/PKCE expire in ten minutes, are single-use and browser-bound.
Pin issuer/client/redirect, validate signature/audience/expiry/nonce and verified
email; enforce configured issuer-host egress rules and timeouts against SSRF.
Store provider secrets encrypted using a backend-only deployment key; secret
updates accept secret references/write-only values. Explicitly authenticated
account linking is required for an existing email. A one-time 60-second exchange
code returns browser session credentials; no bearer token goes into query logs.
SSO-enforced workspaces require a current matching connection/session assurance
at HTTP, room renewal, mutation, export and asset boundaries. Password login
still reaches other workspaces. Owner-managed local break-glass access is scoped,
audited, tested, and cannot be granted by SCIM.

SCIM 2.0: dedicated hashed workspace tokens; `/scim/v2/{workspace}/Users`,
`Groups`, `Schemas`, `ResourceTypes`, `ServiceProviderConfig`. Implement GET,
POST, PUT, PATCH, DELETE, pagination (startIndex 1; count max100), equality
filters for userName/externalId/id, ETags and If-Match. Unsupported filters/paths
return SCIM errors. Bulk is advertised false. Groups map to editor/viewer,
never owner. Group/manual-role conflicts use a recorded membership source;
SCIM updates only its managed membership. `active:false` and DELETE revoke
managed workspace access and sessions/room/media, without deleting the person's
other workspaces or authored scenes. Duplicate/reordered requests are idempotent.
Protect final owner, default My Workspace and break-glass identity.

## Retention and operations

Policies are per workspace, versioned and owner-managed; preview counts and a
seven-day pending period precede the first destructive application. Defaults:
activity 90 days; in-app notifications 30 days; automatic versions 100/scene;
manual/pinned versions and comment content kept until explicit policy; Trash
hard purge disabled. Supports hold on a scene/version, audited policy changes,
manual versions/comments retention by days, and optional Trash purge by days.
Legal/compliance certification is not claimed.

The job claims rows in bounded batches and rechecks the effective policy, holds,
membership and asset references immediately before deletion. Mark objects
pending-delete with a seven-day grace; exclude versions/library items/live scene
references and current backup protection sets; recheck under lock before S3
delete. Retry safely after DB/S3 failures. No S3 bucket lifecycle may delete
objects independently of these references. Dedicated operator disaster archives
and collab receipts/logs/tombstones remain outside automatic purge. Purging live
canonical Yjs history requires a separate future recovery/migration design.

Deploy Go feature worker as a separate Compose service with the existing source
image, health/readiness, single-job leases and least-privilege DB/S3 credentials.
Add encrypted-secret key only when identity is enabled. New feature flags are
off initially and cannot change existing access. Migration numbers start at 14
only if still free at execution; serialize allocation and record a manifest.
Do not remove or renumber shipped migrations. Each release verifies fresh DB
and migration13 upgrades, backup/restore of metadata+objects, rollback by flags
(not old writers), load/memory, and public denial/revoke/reconnect/media behavior.

## Alternatives and review decisions

Rejected: comments in Yjs element metadata (moderation/retention/ACL coupling),
product versions from compaction snapshots (not durable user history), restore
by replacing Y.Doc (offline and receipt corruption), private collection via UI
filter alone (leak), shared voice/screen peer state (permission coupling),
global account disable for SCIM (cross-workspace damage), automatic domain-email
account linking (takeover), and retention via bucket expiry (broken live assets).

Review these proposed product defaults together: member/guest comment policy,
ten media connections/one publisher, editor restore rights, private collection
visibility, OIDC-first SSO and membership-source precedence, and retention
durations/graces. The implementation plans expose these as named constants and
tests. Real IdP issuer/client secret, DNS access and real provider tenant UAT are
operator inputs at rollout; local integration uses isolated fixtures meanwhile.

## Verification and docs contract

Each task has RED -> GREEN unit/integration tests, role-negative cases, browser
keyboard/mobile/deep-link tests where visible, scoped review and an atomic commit.
Real PostgreSQL tests use a guarded loopback `*_test` DB; never production.
Public canaries use disposable resources and verify their cleanup.
Full-stack acceptance includes human/guest/agent concurrent edits, revoke while
pending work/media, version restore with image/bindings, duplicate events, and
SCIM deprovision during active rooms. Distinguish generated media/local IdP
proof from two-network TURN and real-provider proof.

Reconcile requirements, API, ERD, protocol, architecture, operational runbooks,
recovery and retention docs; add human guides plus `/docs/*.md`, `/llms.txt`,
OpenAPI and SCIM discovery schemas. This planning commit changes no public
runtime docs because these features are not shipped. Homelab deployment/TURN/
worker/secrets/backup docs change with their implementation releases.

## Primary references checked 2026-10-20

- [go-oidc official client](https://github.com/coreos/go-oidc): Go ID-token
  verification integrated with OAuth2; product session assurance is our layer.
- [SCIM protocol RFC 7644](https://www.rfc-editor.org/rfc/rfc7644): resource
  operations, filtering, PATCH, paging and discovery protocol baseline.
- [Excalidraw imperative API](https://docs.excalidraw.com/docs/@excalidraw/excalidraw/api/props/excalidraw-api): native scene/library integration points; confirm installed types at execution.
- [Display capture API](https://developer.mozilla.org/en-US/docs/Web/API/MediaDevices/getDisplayMedia): capture requires secure context, transient activation and user selection; permission cannot be silently persisted.
