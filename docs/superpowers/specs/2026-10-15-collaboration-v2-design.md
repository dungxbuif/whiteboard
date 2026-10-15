---
artifact_type: architectural_design
id: COLLAB-V2-DESIGN-2026-10-15
status: ready_for_review
owner: shared
human_fields:
  - design_approval
  - rollout_approval
ai_fields:
  - architecture
  - data_contracts
  - verification_plan
  - legacy_comparison
shared_fields:
  - scope
  - decisions
---

# Collaboration v2 Design

## Design status

- ID: `COLLAB-V2-DESIGN-2026-10-15`
- Scope: authenticated workspace collaboration, public view/edit links, Yjs
  document sync, offline replay, presence, follow mode, and voice signaling.
- Design state: approved in conversation; awaiting written-spec review.
- Implementation state: not started.

## Goal

Upgrade the current owner-only realtime primitives into a durable collaboration
system. Workspace owners can invite users with a default role, override access
per scene, and create independent public view/edit links. Editors can make
concurrent changes that survive reconnects and browser restarts. Public edit
sessions can write directly to the scene and upload assets within the scene
scope.

The service is self-hosted with the Whiteboard deployment. RelayHub is not used
for collaboration transport.

## Non-goals

- Member invitations by email delivery. The first version uses copyable invite
  links; an email provider is not required.
- Member/editor management by non-owners.
- Public links granting workspace membership.
- Unbounded voice mesh. Data collaboration can have many participants, but
  voice mesh is limited to ten active voice participants per room.
- Replacing the Excalidraw canvas visual language. Scene components keep their
  current/native style.

## Accepted decisions

1. Collaboration uses a custom Node/TypeScript WebSocket service in the
   Whiteboard deployment; RelayHub is not involved.
2. Yjs is the CRDT runtime. The service implements standard Yjs sync and
   awareness protocols inside a custom authenticated WebSocket endpoint.
3. Go remains the authority for users, workspaces, membership, scene ACL,
   public links, room-token exchange, and asset capabilities.
4. Workspace roles are `owner`, `editor`, and `viewer`. Scene overrides are
   `editor` or `viewer` and override the workspace default. The owner always
   retains owner access.
5. Workspace invitation links are copyable, single-use, email-bound, and expire
   after seven days.
6. Public scene links may be `view` or `edit`, may have an optional password,
   may be independently revoked, and are scoped to one scene. Public editors
   enter a display name and can persist scene changes and upload assets.
7. Only workspace owners can invite/remove members, change workspace roles,
   change scene overrides, and create/revoke public links.
8. Each scene has a durable Yjs snapshot plus an append-only update log. A
   client receives an update acknowledgement only after the update is durable.
9. Client offline changes are queued in IndexedDB and replayed after state-vector
   resynchronization.
10. The custom WebSocket carries document sync, awareness, presence, follow,
    and voice signaling on one connection. Voice signaling remains peer-to-peer
    WebRTC with a ten-participant mesh limit.
11. The dashboard visual source of truth is the old dashboard in
   `references/devtool`: its shadcn sidebar, workspace switcher, search,
   collection/folder navigation, user footer, cards, spacing, dialogs, and
   light/dark token structure are ported faithfully. Dashboard/admin
   collaboration controls may use the black/white `monochrome` variant within
   that system. Scene/Excalidraw components retain their current/native style.
12. Agents can discover workspaces and scenes, create scenes, and generate
   native Excalidraw diagrams through an authenticated MCP server. MCP writes
   use the same ACL, Yjs transaction, persistence, and audit path as browser
   editors.

## Legacy comparison

The old implementation is available in the checked-in references:

- `references/ws-whiteboard/src/index.ts` — Socket.IO room server with
  `join-room`, `server-broadcast`, volatile broadcast, follow rooms,
  `room-user-change`, participant voice state, and `permission-change`.
- `references/server/src/modules/public/public.controller.ts` and
  `public.service.ts` — scene permission endpoint using a single mutable
  `active/value` row without ownership checks.
- `references/devtool` — old frontend dependency set included `peerjs`, but the
  permission hook was commented and there was no complete integrated client
  collaboration flow.

Behavior retained from the old system:

- Room roster/presence semantics equivalent to `room-user-change`.
- Follow/unfollow as an explicit room-scoped awareness feature.
- Participant voice metadata and join/leave notifications.
- Permission changes as server-originated notifications.

Behavior deliberately replaced:

- Full encrypted `server-broadcast/client-broadcast` snapshots are replaced by
  Yjs binary updates and state-vector resync.
- Free-form room IDs are replaced by an authenticated
  `{workspaceId}:{sceneId}` room scope.
- In-memory-only call state is replaced by durable scene document state; voice
  state remains ephemeral.
- The old public permission row is replaced by scoped, hashed, independently
  revocable links and owner-checked APIs.

The old dashboard is reused as a visual and interaction reference, not as an
unreviewed application dependency. The port keeps its layout language and
component contracts while using the new app's route, auth, workspace/folder,
scene, collaboration, and accessibility contracts. Old implementation bugs,
commented permission hooks, and obsolete Next.js data-loading assumptions are
not copied.

## Current Excalidraw+ / Enterprise parity research

Research date: 2026-10-15. The official product comparison currently lists
Excalidraw+ collaboration capabilities including link invitations, view-only
access, voice hangouts and screen sharing, comments, and live presentations. It
also lists workspace teams, user management, collections, REST API, embeddable
links, and live presentations. See
<https://plus.excalidraw.com/excalidraw-plus-vs-excalidraw>.

The current Excalidraw+ changelog adds several collaboration-adjacent behaviors
that are relevant to this design: collaborator following, presenter spotlight,
guest commenting, presentation waiting-room admission, activity/audit-oriented
API behavior, real-time metadata, and protection against redundant image
re-uploads. See <https://plus.excalidraw.com/changelog>.

The public Enterprise page currently describes Enterprise as coming soon, while
the roadmap lists SSO, self-hosting, shared libraries, versioning, and improved
scene filtering as roadmap items. We treat these as parity candidates rather
than shipped Enterprise contracts. See
<https://plus.excalidraw.com/security-and-compliance> and
<https://plus.excalidraw.com/roadmap>.

The Excalidraw developer FAQ states that the npm package does not include
collaboration and that the host application must implement it through the
exposed APIs. The development documentation separately says a self-hosted
instance does not currently receive sharing/collaboration automatically. See
<https://docs.excalidraw.com/docs/@excalidraw/excalidraw/faq> and
<https://docs.excalidraw.com/docs/introduction/development>.

### Parity decisions

The current collab v2 scope covers the foundations required by these features:

- invite links, view/edit access, workspace roles, scene overrides, and public
  guest edit sessions;
- durable concurrent editing, presence/cursors, follow mode, voice signaling,
  and offline reconnect;
- asset upload capability, audit events, revoke/kick, and room admission.

The following capabilities should be added to the roadmap rather than hidden
inside the first CRDT implementation:

| Priority | Capability | Reason and relationship to this design |
| --- | --- | --- |
| P1 | Comments and guest comments | Needs durable comment threads, anchors to element IDs, moderation, and notification UI; it can reuse room ACL and awareness but is not a Yjs scene element. |
| P1 | Screen sharing | Needs a WebRTC media track/permission flow separate from scene voice mesh; can reuse the ten-participant admission guard. |
| P1 | Live presentation, spotlight, and waiting room | Follow mode exists in v2; presenter leadership, audience admission, reactions, and view locking need explicit session state. |
| P1 | Activity feed | The v2 audit event stream supplies the backend source; dashboard rendering and retention policy remain separate. |
| P2 | Scene versioning and restore | Snapshot/compaction provides technical checkpoints, but user-facing history, labels, diff, and restore need a product layer. |
| P2 | Workspace libraries and collections | Existing workspace/folder structure can host these, but library sharing and search are separate content contracts. |
| P2 | Domain capture, SSO/SCIM, and enterprise retention | The public roadmap treats SSO/self-hosting as roadmap items; add only after the email/password ACL model is stable. |

Do not claim these P1/P2 items as delivered by collab v2. The implementation
plan must keep them as separately testable work packages.

## Agent interaction and diagram catalog

The app exposes a native MCP server for agents that need to inspect a
workspace, create scenes, or draw diagrams. The host application owns the MCP
adapter; community servers are implementation references and are not runtime
dependencies. This keeps workspace ACL, scene permissions, audit, and Yjs
persistence under the same authority as browser clients.

The primary transport is Streamable HTTP at:

```text
https://whiteboard.dungxbuif.com/mcp
```

A stdio adapter is also provided for local CLI and desktop MCP clients. Both
transports call the same service layer. MCP never reads PostgreSQL directly,
returns raw public-share tokens, or bypasses Go authorization.

### MCP capabilities and tools

Agent API keys are created, rotated, and revoked by workspace owners. Each key
is scoped to one or more workspaces/scenes and receives explicit capabilities:

```text
workspace:read
scene:read
scene:create
scene:write
scene:move
scene:trash
scene:export
```

Every call records the agent principal, key, tool, workspace, scene, result,
latency, and failure class in the audit stream. Key material is shown once,
stored hashed, and never included in agent-readable scene output.

The P0 tool surface is:

```text
list_workspaces
list_scenes
get_scene
create_scene
validate_diagram
generate_diagram
patch_elements
delete_elements
export_scene
get_scene_activity
```

`get_scene` returns a bounded projection with element IDs, types, geometry,
labels, groups, revision, and warnings. `patch_elements` and
`delete_elements` address element IDs and are preferred over whole-document
replacement. `generate_diagram` supports a typed preset DSL or Mermaid. It
returns `sceneId`, created element IDs, revision, and validation warnings.
Preview/validate is available before a write. Publishing a share link,
changing ACL, managing credentials, and hard deletion are not granted by
default.

### Diagram presets

The catalog provides deterministic JSON schemas and layout rules for common
agent-generated diagrams:

```text
flowchart             sequence
architecture          deployment / infrastructure
c4-context            c4-container / c4-component
erd                   class / UML
state                 mindmap
timeline              user-journey
swimlane              org-chart
network               data-flow
agent-workflow
```

Preset output is native Excalidraw elements with deterministic IDs, groups,
connectors, labels, and optional whitelisted library shapes. It remains
editable, undoable, and synchronizable rather than being flattened into one
SVG or image. Mermaid is a second input path for flowchart, sequence, ERD,
state, mindmap, timeline, and C4 forms. The implementation uses the official
`@excalidraw/mermaid-to-excalidraw` parser where its supported syntax matches
the preset contract, then converts the result into fully qualified
Excalidraw elements.

`generate_diagram` applies one Yjs transaction. Open browser sessions receive
the change on the existing collaboration socket, and the update is persisted
and acknowledged using the same sequence and compaction rules as human edits.

### Research and reuse boundary

The official `excalidraw-mcp` repository is the reference for MCP Apps,
viewport control, and streamed previews. `yctimlin/mcp_excalidraw` is a
reference for element CRUD, batch layout, screenshot/describe, and Mermaid.
`debu-sinha/excalidraw-mcp-server` is a reference for capability checks,
WebSocket authentication, rate limits, validation, and audit. `excaliwow/mcp`
is a reference for a higher-level diagram DSL and bounded generate/read/edit
tools. `hangarbay/excalidraw.mcp` is a reference for a Go Streamable HTTP
process boundary and revisioned scenes. The official Excalidraw libraries
collection is a candidate source for whitelisted architecture/network shapes.

Before copying any code or asset, pin its version, verify license and
provenance, run dependency/security checks, and record the decision in the
implementation plan. The native server remains the maintained production
implementation.

## System boundary

```text
Browser
  | HTTPS: auth, ACL, invite, share, room-token, asset capability
  v
Go API -------------------- PostgreSQL
  |                          |
  | room token + ACL events  | snapshots, update log, memberships
  v                          |
Node Collab WebSocket <------+
  | Yjs document, awareness, presence, follow, voice signaling
  v
WebRTC peers (voice mesh; max 10 active voice participants)
```

The public origin routes `wss://whiteboard.dungxbuif.com/collab` to the Node
service. Existing HTTP routes remain under the Go API. The existing Socket.IO
endpoint is retired for scene collaboration after v2 reaches the rollout exit
criteria.

## Data model and ACL

The existing `workspace_members` table is migrated from `owner|member` to
`owner|editor|viewer`. New tables are:

Existing `member` rows migrate to `editor` to preserve the current ability to
edit scenes. Existing active `scene_permissions` rows migrate to `view` links;
their existing token is hashed during migration and their active/password state
is preserved.

```text
scene_members
  scene_id, user_id, role(editor|viewer), created_at, updated_at
  unique(scene_id, user_id)

workspace_invitations
  id, workspace_id, email, role(editor|viewer), token_hash,
  expires_at, accepted_at, invited_by, created_at

scene_share_links
  id, scene_id, token_hash, access(view|edit), password_hash,
  active, expires_at, created_by, created_at, updated_at, revoked_at

scene_collab_documents
  scene_id primary key, snapshot_bytes, state_vector,
  snapshot_version, updated_at, compacted_at

scene_collab_updates
  scene_id, sequence, update_bytes, principal_id, client_update_id,
  created_at
  primary key(scene_id, sequence)
  unique(scene_id, client_update_id)
```

The per-scene sequence is allocated inside the same transaction as the update
append while holding the scene document row lock. This makes update ordering
deterministic across concurrent Node instances.

ACL precedence is deterministic:

1. Workspace owner — full access.
2. Scene override — `viewer` or `editor` if present.
3. Workspace role — `editor` or `viewer`.
4. Public link — access only to its own scene and only for the link's `view` or
   `edit` mode.
5. No matching rule — deny.

An owner-only API manages membership, scene overrides, pending invitations, and
public share links. A scene editor can edit content but cannot escalate access.

## Invite and public-share flows

### Invite link

1. Owner submits workspace, recipient email, and `viewer`/`editor` role.
2. Go stores a SHA-256 token hash, expiry seven days from creation, and a
   pending invitation row.
3. UI copies `/invite/{token}`; the raw token is never stored in the database
   or logs.
4. The recipient authenticates or registers with email/password.
5. Accept requires the authenticated email to match the invitation email,
   verifies expiry and single-use state, then creates/updates membership.
6. The invitation is marked accepted in one transaction.

### Public view/edit link

1. Owner creates one or more links for a scene with `view` or `edit` access,
   optional password, and optional expiry.
2. Go validates the link and exchanges it for a short-lived room token.
3. The browser opens read-only or editable scene UI based on the room token.
4. Public editors must provide a display name; the name is included in
   awareness and audit metadata, never treated as an authenticated user ID.
5. Revoke or expiry prevents new room-token exchange and publishes an ACL event
   to disconnect active sessions.

## Room token and WebSocket protocol

Go signs a dedicated short-lived room JWT with `COLLAB_JWT_SECRET`. The token
contains `sub`, `workspaceId`, `sceneId`, `principalType` (`user` or `public`),
`access` (`view` or `edit`), `principalId`, `displayName`, and `exp`. The Node
service verifies the signature and refuses a room change after handshake.

The endpoint is:

```text
wss://whiteboard.dungxbuif.com/collab
```

The first control frame authenticates and joins the canonical room key
`{workspaceId}:{sceneId}`. Subsequent frames use a typed envelope:

```text
control JSON: { type, requestId, room, payload }
binary Yjs frame: { messageType, room, bytes }
```

Supported channel semantics:

- `document`: standard Yjs sync step 1/2, update, and state-vector resync.
- `awareness`: cursor, display name, selection, and online state.
- `presence`: roster join/leave/update with role and principal type.
- `follow`: follow/unfollow and followed-by notifications, retained from the old
  implementation.
- `voice`: offer/answer/ICE signaling, with a server-enforced limit of ten
  active voice participants.
- `system`: `acl:changed`, `snapshot:committed`, `resync_required`,
  `rate_limited`, `degraded`, and structured errors.

The server verifies the room scope on every frame, enforces maximum frame and
document sizes, applies per-principal/IP rate limits, and uses backpressure for
slow clients. Public edit and member editor sessions share the same document
path and persistence guarantees.

## Persistence, acknowledgement, and compaction

For an accepted Yjs update, Node performs the following sequence:

1. Validate edit access and client update ID.
2. Append the binary update to `scene_collab_updates` in a PostgreSQL
   transaction.
3. Commit successfully.
4. Broadcast the update and send `document:ack`.

The client only removes an IndexedDB pending update after its acknowledgement.
Snapshot creation runs at the first of 100 new updates or five seconds since
the previous snapshot. A compaction lock is held per scene. The worker writes a
new snapshot and state vector, materializes the Excalidraw-compatible scene JSON
to `scenes.content`, commits, and only then deletes updates covered by the
snapshot. Audit events remain separate from document updates.

On reconnect, the client sends its state vector. Node replies with missing
updates, or a current snapshot followed by newer updates when the vector is too
old. Local pending updates replay after the server state is applied. Duplicate
client update IDs are acknowledged idempotently.

If PostgreSQL is unavailable, the room enters `degraded`: viewers may continue
reading the last snapshot, editors receive no successful update acknowledgement,
and clients retain pending changes for retry.

## Frontend and Excalidraw integration

`CollabProvider` owns `Y.Doc`, custom WebSocket lifecycle, IndexedDB
persistence, and awareness. The existing realtime store is reduced to connection
status and voice state; it no longer owns scene snapshots.

The Yjs document shape is:

```text
scene.elementsById       Y.Map<elementId, element-map>
scene.elementOrder       Y.Array<elementId>
scene.files              Y.Map<fileId, file-metadata>
scene.presentationOrder  Y.Array<elementId>
scene.meta               Y.Map<metadata>
```

Excalidraw `onChange` computes a diff and applies one Yjs transaction. Remote
transactions call `updateScene()` under a loop guard. Camera, selection, and
collaborator maps are awareness state, not durable scene JSON.

New files upload through a Go-issued scene-scoped capability. Yjs stores only
file metadata/reference; `dataURL` and storage credentials never enter the
document update log. Public edit links receive the same capability policy as
member editors.

Dashboard/admin components use shadcn/ui and a black/white `monochrome`
variant, following the old dashboard's sidebar/workspace/search/card visual
language. Scene/Excalidraw tools retain their current/native style. State
labels and icons must remain understandable without color alone.

## Security and operational limits

- Room JWT default lifetime: ten minutes; the authenticated browser refreshes it
  through Go while the session remains authorized.
- Invite tokens: SHA-256 at rest, one use, seven-day expiry.
- Public share tokens: hashed at rest, independently active/revoked, optional
  bcrypt password and expiry.
- Maximum Yjs update frame: 1 MiB. Maximum materialized document: 25 MiB.
- Default update rate limit: 120 updates/minute/principal, with stricter limits
  for public tokens and IP circuit breaking.
- No product-level participant cap; resource exhaustion protection may reject a
  new room join or mark a room degraded.
- Audit invite/share/role changes, public edit sessions, ACL revocations,
  persistence failures, and compactions. Never log secrets, raw tokens, or raw
  document updates.
- ACL changes are published through PostgreSQL `NOTIFY` on a dedicated channel;
  Node disconnects affected principals without waiting for token expiry.

## Verification and acceptance

### Backend and database

- Migration tests cover role values, invitation uniqueness/expiry, scene
  overrides, multiple share links, and collab tables.
- Authorization tests cover owner/editor/viewer, scene override precedence,
  public view/edit scope, asset capability scope, and revoked sessions.
- Invitation tests cover email match, single use, seven-day expiry, and
  registration-before-accept.
- MCP authorization tests cover workspace/scene capability scopes, revoked
  keys, bounded projections, audit records, and denial of share/ACL/credential
  escalation.

### Collab service

- WebSocket tests cover JWT handshake, room isolation, malformed frames,
  idempotent update IDs, state-vector resync, awareness cleanup, ACL kick,
  rate-limit/backpressure, snapshot/compaction, and degraded persistence.
- Voice tests cover offer/answer/ICE routing and rejection after ten active
  voice participants.

### Frontend and browser

- Unit tests cover Yjs-to-Excalidraw diffing, remote loop guard, IndexedDB queue,
  replay ordering, public display name, permission UI, and monochrome states.
- Visual browser checks compare the dashboard port against the old reference
  for sidebar collapse, workspace switching, search, collection/folder
  navigation, scene cards, dialogs, user footer, responsive layout, and theme
  tokens. These checks do not apply the dashboard theme to the scene canvas.
- Playwright covers owner/member/public flows, invite accept, two tabs editing
  one scene, offline/reconnect replay, public edit asset upload, revoke while
  connected, and voice cap behavior.
- Load tests cover many data-only participants, slow clients, large documents,
  and update bursts.
- MCP integration tests cover scene creation, preset generation, Mermaid
  conversion, deterministic element IDs, one-transaction live updates, and
  export formats. A browser tab must observe an agent-generated diagram
  without a refresh.

### Rollout exit criteria

- New collab service health and metrics are green on VM100.
- Two-tab edits converge after reconnect with no lost acknowledged updates.
- Snapshot reload produces the same Excalidraw scene as the live Yjs document.
- Public view/edit links, revoke, password challenge, and asset capability pass
  on the production origin.
- Existing v1 realtime fallback remains available until these checks pass; then
  the feature flag switches new rooms to v2.
- Rollback restores the v1 image and leaves database snapshots/update logs
  available for recovery.

## Implementation boundaries

The implementation plan will split work into independently testable slices:

1. Schema, Go ACL/invitation/share APIs, and room-token exchange.
2. Node/TypeScript custom WebSocket server and Yjs persistence adapter.
3. Frontend Yjs provider, Excalidraw adapter, IndexedDB replay, and native-style
   scene integration.
4. Dashboard shadcn monochrome invite/member/share management UI.
5. Dashboard visual parity port from `references/devtool`, including sidebar,
   workspace switcher, search, collection/folder navigation, user footer,
   cards, dialogs, responsive collapse, and theme tokens.
6. Native MCP service, agent-key capabilities, audit, scene tools, diagram
   catalog, Mermaid conversion, exports, and stdio adapter.
7. Presence/follow/voice migration and the ten-participant voice guard.
8. E2E/load verification, VM100 deployment, canary, and v1 rollback.
