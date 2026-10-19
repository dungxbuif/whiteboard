# Collaboration v2 programme Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver the approved legacy dashboard, durable collaboration and native agent diagram workflows on VM100.

**Architecture:** Go owns identity and permissions. Node owns Yjs and accepts authorized commands; MCP is a separate Node adapter using Go and the same collab command path. Five bounded plans share the contracts below.

**Tech Stack:** Go/GORM/Casbin, PostgreSQL, React/Vite/TypeScript 7, Yjs, WebSocket, MCP SDK, Excalidraw.

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

- Revoked or downgraded users replay old edits: deny persistence without silently discarding their local queue (A2, B2, B3).
- Legacy HTTP saves race with CRDT updates: one writer mode per scene, with a tested rollback checkpoint (B2, E3).
- Untrusted binary or diagram input expands excessively: validate before live mutation with bounded CPU/memory (B1, D2).
- Agent retry or stale patch overwrites human work: request idempotency plus revision precondition (D3).
- Long names and mobile navigation break copied dashboard: real browser comparison with legacy fixtures (C1, C2).

---

## Status and execution order

Status: **in_progress**. Design and implementation authorized by the user. No runtime change is shipped by this document.

| Order | Plan | Deliverable | Depends on |
| --- | --- | --- | --- |
| 1 | [A — Access](2026-10-16-collaboration-access.md) | roles, invite/share, asset and room authority | existing app |
| 2 | [B — Collaboration](2026-10-16-collaboration-runtime.md) | durable Yjs, browser convergence, presence/voice | A |
| 3 | [C — Dashboard](2026-10-16-collaboration-dashboard.md) | legacy style and management flows | C1 can run first; C2 needs A |
| 4 | [D — MCP and diagrams](2026-10-16-collaboration-agents.md) | native catalog and live agent writes | A + B; D2 compiler independently testable |
| 5 | [E — Release](2026-10-16-collaboration-release.md) | real-stack tests, VM100 canary, docs/rollback | A–D |

Suggested execution: C1 first for a visible style result, then A, B, C2, D, E. These are dependency slices, not authorization to spawn agents. Integrate shared edits to router/main/App serially.

## File ownership and boundaries

All code paths below are relative to `/Users/dungxbuif/workspace/devtool`.
Only create an isolated implementation worktree at execution time; preserve unrelated changes.

| Owner | Paths and responsibility |
| --- | --- |
| A | Go domain/usecase/repository/handler access modules, migrations 000007–000008, router/main wiring, asset authorization |
| B | `src/collab/`, `src/packages/collab-contracts/`, frontend `collab/`, whiteboard wrapper/realtime store, migration 000009 |
| C | frontend dashboard components, scoped styles, dashboard routes, membership/share UI |
| D | `src/mcp/`, `src/packages/diagram-kit/`, agent-key Go modules, migration 000010, agent settings component |
| E | Compose/nginx/CI/e2e and Homelab deployment/runbooks |

## Shared implementation contracts

These contracts guide implementation of the approved design.

### Access and tenancy

- Add `domain.Principal {Kind user|public|agent; ID string; UserID *uuid.UUID; GrantID string}`, `SceneRef {WorkspaceID, SceneID uuid.UUID}`, `AccessDecision {Access view|edit; Owner bool; ACLVersion int64}`.
- Go `AccessService.Resolve(ctx, principal, sceneRef) (AccessDecision,error)` validates scene/workspace relationship, soft-delete status and effective ACL. `RequireOwner(ctx,userID,workspaceID) error` guards management. Casbin evaluates actions from current data; a header is never evidence of membership.
- Scene overrides affect scene content/read, not workspace management. An override requires current workspace membership; removal deletes overrides. Search/list/asset endpoints filter through the same policy.
- Agent effective access is the intersection of the creating owner's current access, key scope, target and current scene status. A key never grants workspace creation or permission management implicitly.
- Public guest scope remains one scene. Public password exchange is POST body; never introduce password query parameters.
- Invitation-bound registration only: possession of a valid owner-issued invite can create a new password user and default personal workspace atomically. Existing email accounts must log in; accepting an invite never resets their password. No claim of email verification without an email-delivery mechanism.

### HTTP contracts

All authenticated requests retain bearer login JWT and workspace validation.

| Method/path | Input/output |
| --- | --- |
| GET/POST `/api/v1/workspaces/{id}/invitations` | owner lists pending metadata / creates `{email,role}`; returns raw invite URL once |
| DELETE `/api/v1/workspaces/{id}/invitations/{inviteId}` | owner revokes pending invite |
| GET/PATCH/DELETE `/api/v1/workspaces/{id}/members[/{userId}]` | list / set editor or viewer / remove; owner cannot be removed or demoted |
| PUT/DELETE `/api/v1/scenes/{id}/members/{userId}` | set/remove scene override |
| POST `/api/v1/invitations/accept` | authenticated `{token}`; email must match |
| POST `/api/v1/invitations/register` | `{token,email,password,name}`; create account + membership + personal workspace transactionally |
| GET/POST/DELETE `/api/v1/scenes/{id}/share-links[/{linkId}]` | owner list metadata / create `{access,password?,expiresAt?}` / revoke |
| POST `/api/v1/public/scene-session` | `{token,password?,displayName?}` -> short-lived scene session; display name required for edit |
| POST `/api/v1/scenes/{id}/room-token` | authenticated or scoped scene session -> room JWT |
| POST `/api/v1/scenes/{id}/assets/upload-capability` | verified edit access + asset metadata -> upload grant |
| GET `/api/v1/scenes/{id}/activity` | authorized bounded scene activity, no keys or raw documents |

Error contract: `{code,message,requestId}`; 401 unauthenticated, 403 prohibited operation, 404 inaccessible scene, 409 stale revision/idempotency conflict, 410 expired invite, 413 limits, 429 throttling, 503 unavailable authority/storage. Avoid account enumeration in unauthenticated registration errors.

### Node packages and protocol

Create npm workspace root `src/package.json` spanning `frontend`, `collab`, `mcp`, `packages/collab-contracts`, `packages/diagram-kit`. Keep TypeScript 7; update Docker contexts for workspace installs in E2. Pin actual tested package versions and lock them at execution; do not guess a current version in this plan.

In `src/packages/collab-contracts/src/index.ts`, define:

```ts
type SceneRef = { workspaceId: string; sceneId: string };
type Revision = string; // decimal PostgreSQL bigint; never JS floating point
type MutationReceipt = { requestId: string; sequence: Revision; stateVector: string };
type SceneMutation = SceneRef & {
  requestId: string; expectedSequence: Revision;
  commands: ElementCommand[];
};
type ElementCommand =
  | { op: "add"; elements: NativeElement[] }
  | { op: "patch"; patches: { id: string; fields: Record<string, unknown> }[] }
  | { op: "delete"; ids: string[] };
```

`NativeElement` and `SceneContent` are defined in collab-contracts from the installed Excalidraw types with an explicit serializable field schema; diagram-kit consumes and re-exports them. This prevents a runtime-to-diagram-compiler dependency cycle. Reject immutable ID/type changes, executable content and unknown fields. Plain text labels are data, never agent instructions.

- JSON control frames: `{type,requestId,payload}`; room is bound at authentication.
- Binary frames: version byte, channel byte, length-prefixed update ID, Yjs protocol payload. Reject trailing bytes and wrong version. Server never trusts a client-supplied room inside a frame.
- Room token: HS256 only, issuer `whiteboard-api`, audience `whiteboard-collab`, ten-minute expiry, scene scope, principal/grant ID and ACL version. Public/agent IDs are not user UUIDs.
- Authenticate in first frame within five seconds. Refresh before expiry via Go, replace token on same connection after scope validation; expiry terminates write access.
- Treat Yjs sync-step-2 as a possible write: every mutating frame requires edit access, validation, durable append and acknowledgement.
- Revision is the committed update sequence. Request IDs bind to principal + payload hash. Retry returns the original receipt; changed payload or principal with the same ID is rejected.
- Go signs single-operation grants for MCP commands. `POST /internal/scenes/{sceneId}/commands` on private Node ingress accepts command plus grant; Node verifies target, expected sequence, request ID, payload hash and current ACL version, then calls the same append path as browser writes.
- ACL changes increment a durable version. NOTIFY prompts disconnect; LISTEN reconnection requires resync. Writes and ACL mutations serialize on the relevant authority rows so a revoke cannot race a successful later write. Go defines policy; Node validates version/grant rather than inventing roles.

### Persistence correctness

Initial release runs one active collab process on VM100. Multi-replica editing is not enabled merely because sequence allocation uses row locks; cross-process doc fanout and awareness need a separate tested design.

Columns added alongside spec tables: document `last_sequence`, `snapshot_sequence`, `writer_mode(v1|v2)`; update `payload_hash`; durable principal-scoped dedup receipts survive compaction. Lock scene and relevant ACL rows consistently. Validate an incoming update against a temporary document, including resulting shape/size, before durable append and live broadcast.

Seed Yjs once from `scenes.content` under a row lock and change writer mode atomically. Reject old content PATCH with `409 COLLAB_REQUIRED` for v2 scenes; metadata PATCH remains allowed. Browser, AI apply, presentation reorder and MCP all use Yjs for v2 content.

Persist files as references; exclude data URLs, selection, camera and credentials. Compare normalized scene data, not transient renderer versions, in convergence tests. Keep pending updates on failures. Ack receipts survive compaction so replay never fabricates a second audit/mutation. Do not clear another user's IndexedDB namespace after an account switch.

### Research coverage and unresolved product boundary

| Request | Plan treatment |
| --- | --- |
| Legacy dashboard | C1 ports exact reference tokens/components; C2 extends them |
| Short IDs / legacy reuse | A1 records NanoID source and adds public aliases; UUID database keys retained |
| Excalidraw+ / Enterprise | approved spec parity matrix retained; comments/screen sharing/presentation session/activity UI are P1; versioning/shared libraries/SSO are P2, not silently included in v2 |
| Community MCP/libs | D2 pins/audits official parser, ELK layout and curated libraries; D1/D3 use community contracts as references |
| Fancy diagrams | D2 test matrix covers every requested preset and discoverable schemas |
| TURN/NetBird | E1 requires actual topology and listener inventory before production changes |
| Workspace interaction | User confirmed that “org” meant workspace; implement workspace tools without adding an organization hierarchy |

On 2026-10-16 the user resolved the terminology: org means workspace. There is no separate organization scope.

## Documentation and traceability

Entry point: [phase](../../work/phases/PHASE-COLLAB-V2.md), [backlog](../../work/BACKLOG.md), [validation matrix](../../work/VALIDATION_MATRIX.md).
Before code, create ADRs for ACL, Yjs writer ownership, scoped MCP, and legacy dashboard reuse.
Reconcile `docs/architecture/{ARCHITECTURE,API,ERD}.md`, `docs/requirements/SPEC.md`, `docs/operations/{DEPLOYMENT,ENVIRONMENTS,RELEASE_CHECKLIST}.md`, and release notes.
Publish human guides plus `src/frontend/public/docs/*.md`, `src/frontend/public/llms.txt`, and schemas/endpoint catalog for agents. Documentation describes v2 as planned until shipped.
Homelab targets: `docs/architecture/whiteboard-vm100.md`, `docs/operations/whiteboard.md`, `docs/operations/coturn.md`, `docs/deployments/` and matching public-docs surfaces. Previous filesystem reads timed out; recheck availability without overwriting placeholders or claiming live verification.

## Verification commands and completion rule

Run commands from stated directories with the required `rtk` prefix.
Backend: `rtk go test ./...` in `src/backend`.
All npm workspace commands below run from `src`; backend commands run from `src/backend`.
From `src`: `rtk npm run test:unit --workspace ./frontend`, `rtk npm run build --workspace ./frontend`, `rtk npm test --workspace ./collab`, `rtk npm test --workspace ./mcp`, `rtk npm test --workspace ./packages/diagram-kit`.
Run database integration against a disposable PostgreSQL database; fixture guards refuse production URLs. Browser end-to-end must include real API/collab/DB, not only route mocks.
No claim of passing tests is made by this plan.

## Self-review

- Every accepted design section maps to A–E; workspace terminology is confirmed by the user.
- Browser/MCP write paths share grants, schema and durable sequence; shadow HTTP writer removed.
- Dashboard theme scope and legacy style priority preserved.
- Unsupported Mermaid native conversion must report an explicit capability error; no silent image fallback.
- Production canary, migration rollback boundaries, offline revocation and client compatibility are owned by E.
- Review-focus cases appear in task proof steps. Runtime verification remains pending execution.
