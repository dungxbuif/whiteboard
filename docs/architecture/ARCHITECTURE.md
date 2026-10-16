---
artifact_type: architecture_doc
id: ARCH-MASTER
status: active
owner: shared
human_fields: [approved_boundaries, architectural_constraints, tradeoff_approval]
ai_fields: [overview, modules, diagrams, flows, dependencies, risks]
shared_fields: [status, linked_decisions]
---

# Architecture: Excalidraw Ecosystem

## Field Ownership

- Human owns approved boundaries, constraints, and tradeoff approvals.
- AI maintains implementation-grounded overview, modules, diagrams, flows, dependencies, and risks.

## Collaboration v2 branch status

Production currently runs v1. The overview and flows below describe that deployed baseline; the v2 implementation is tracked separately until release.

- Go authority now has current workspace roles, scene overrides, short public aliases, workspace row locks and durable ACL invalidation. [Access decision](../decisions/0009-collaboration-access.md).
- Invitation API creates single-use seven-day links and atomically provisions new accounts with a personal workspace and invited membership. A1–A4 and C2 access/invitation/public-link controls passed task review; full deployed role acceptance remains a release gate.
- `src/packages/collab-contracts` validates native scene content and binary frames; `src/collab` validates candidates in isolated Yjs workers. The single-owner durable WebSocket writer passed review; the browser transport has local native/offline/reload proof and awaits task review. [Protocol](collaboration-protocol.md).
- `src/packages/diagram-kit` provides 19 editable native presets. MCP key/audit transports passed review. Authorized live commands and isolated native rendering/export have implementation tests and are in independent review. [Catalog](../user/diagram-catalog.md).
- Dashboard styling follows the original reference colors and layout, scoped away from native Excalidraw. [Dashboard guide](../user/dashboard.md).

The [programme](../superpowers/plans/2026-10-16-collaboration-v2.md) defines the remaining contracts; [verification](../work/verification/collaboration-v2.md) distinguishes completed local checks from release gates.

## Overview

High-performance, self-hosted whiteboard ecosystem using Go Clean Architecture
and React Vite. Phase 1 rewrites the dashboard and backend into a SaaS-ready
single-product system with explicit workspace tenancy, PostgreSQL metadata, and
S3-compatible binary asset storage.

## Technical Stack

- **Backend:** Go 1.22+
- **Architecture:** Clean Architecture (Domain, Usecase, Repository, Delivery)
- **Frontend:** React + Vite, Zustand, Shadcn UI, embedded Excalidraw
- **Database:** PostgreSQL (Multi-tenant)
- **Storage:** S3-compatible (MinIO/S3)
- **Auth:** Email/password login with backend-issued JWTs; legacy Google identity columns remain for migration compatibility.

## System Boundaries

- **Dashboard:** React SPA for workspace, folder, search, trash, starred, and
  scene management. Zustand owns client UI state and active workspace selection.
- **Backend API:** RESTful Go API for auth, workspace metadata, folder metadata,
  scene JSON, asset metadata, and signed asset URL flows.
- **Domain Layer:** Pure Go entities and repository interfaces. No SQL,
  transport, OAuth SDK, or S3 SDK details belong here.
- **Usecase Layer:** Business rules for workspace access, folder path mutation,
  folder trash cascade, scene last-write-wins save policy, search orchestration,
  and asset upload orchestration.
- **Repository Layer:** PostgreSQL row mapping and transactions for metadata;
  S3-compatible object operations for binary assets.
- **Delivery Layer:** HTTP handlers and middleware. Middleware validates auth,
  parses the selected workspace, and verifies that the authenticated user owns
  it before workspace-scoped handlers run.
- **Whiteboard:** Embedded `@excalidraw/excalidraw` engine. Excalidraw scene
  content is persisted through the backend as JSONB, not by direct browser DB
  access.
- **Relay/Collab:** WebRTC + Socket.io provide ephemeral scene-room presence,
  scene sync, and voice signaling. Coturn is self-hosted through compose;
  persistence still goes through the scene API.
- **AI Gateway:** Authenticated `/api/v1/ai/*` handlers call an
  OpenAI-compatible chat-completions repository. DeepSeek is the default; the
  key remains server-side and model output is validated in the usecase layer.

## Modules

| Module | Responsibility | Key Files | Notes |
| --- | --- | --- | --- |
| Domain | Pure entities, repository interfaces | `src/backend/internal/domain/` | No infrastructure deps |
| Usecase | Business rules, orchestration | `src/backend/internal/usecase/` | Depends only on domain |
| Repository | PostgreSQL + S3 implementations | `src/backend/internal/repository/` | Implements domain interfaces |
| Delivery | HTTP handlers, middleware, router | `src/backend/internal/delivery/` | Chi router, auth middleware |
| Frontend | React SPA | `src/frontend/src/` | Vite + Zustand + Shadcn |

## Key Patterns

- **Multi-tenancy:** `WorkspaceID` is extracted at the delivery boundary and
  carried through Go context. Every workspace-owned repository query must filter
  by `workspace_id`.
- **Workspace Membership:** The current runtime authorizes workspace routes by
  owner ID. Header-based selection remains the transport mechanism, but the
  header is never trusted without the JWT ownership check.
- **Data Organization:** Folder hierarchy uses application-managed materialized
  paths. Folder move and trash operations update descendants in a single
  workspace-scoped transaction.
- **Scene Persistence:** Excalidraw scene content is stored in PostgreSQL JSONB.
  Phase 1 uses last-write-wins save semantics and may track `content_version`
  for observability, not optimistic locking.
- **Asset Persistence:** PostgreSQL stores asset metadata; S3-compatible storage
  stores binary bytes. Signed URL generation must verify workspace scope before
  returning asset access.
- **Lifecycle Flags:** Trash is represented by nullable `deleted_at`; starred
  views use `is_starred`. Default list and search queries exclude trashed rows.
- **Search:** Phase 1 search runs over folder names and scene titles inside the
  active workspace. The workspace-scoped search handler also powers the
  `all`, `starred`, and `trash` dashboard views; PostgreSQL trigram indexes
  are optional optimization.
- **Presentation:** Frame element IDs are stored as `presentationOrder` inside
  scene JSON. Presentation mode is viewer-local and uses Excalidraw's
  `viewModeEnabled` plus animated `scrollToContent` navigation.
- **Sharing:** Owners manage one workspace-scoped permission per scene. Public
  access resolves an opaque token, checks active state and an optional bcrypt
  hash, then returns scene JSON to a read-only viewer without JWT access.
- **Scene URLs:** Authenticated editors use `/w/{workspaceId}/scenes/{sceneId}`;
  the SPA loads the scene through the protected API and displays an access error
  when the workspace or scene is not authorized.
- **Sync:** Frontend saves scene changes through debounced API calls. Asset
  uploads go through backend S3 orchestration before scene JSON references the
  stored asset.
- **Editor loading:** Excalidraw JavaScript is dynamically imported only after
  a scene opens. Zustand owns scene metadata and save status while Excalidraw
  owns high-frequency live canvas state.
- **Save ordering:** A client generation token prevents stale request
  completions from replacing newer save status. The backend remains
  last-write-wins and `content_version` is observability metadata.
- **Realtime rooms:** A room key is `{workspaceId}:{sceneId}`. The server
  validates the JWT during the Socket.IO handshake, then broadcasts cursor,
  presence, scene update, and WebRTC signaling events only within that room.

## Data Flow

```text
Browser (React SPA)
    |
    v
Go HTTP Delivery (Chi Router + Middleware)
    |
    v
Usecase Layer (Business Logic)
    |
    +---> PostgreSQL Repository (Metadata, Scene JSONB)
    |
    +---> S3 Repository (Binary Assets)
    |
    +---> Socket.IO Room Manager (Ephemeral collaboration)
    |
    +---> OpenAI-Compatible Repository (DeepSeek default)
```

## Runtime Flow

```text
User Login --> Password Check --> Session/JWT --> Workspace Selection
    |
    v
Dashboard (Folders, Scenes, Search, Starred, Trash)
    |
    v
Open Scene --> Embedded Excalidraw --> Debounced Auto-Save --> Backend API
    |
    +---> Scene JSON --> PostgreSQL JSONB
    |
    +---> Assets --> S3 via Backend Orchestration
```

## Integration Points

- **Password auth:** bcrypt password hashes are checked by the backend; the
  frontend stores only the short-lived JWT session token.
- **S3:** Blob storage for images and videos.
- **Excalidraw Library:** Core drawing primitive.

## Phase 1 Persistence Contract

The canonical high-level database contract is
[Phase 1 Database Design](phase-01-database-design.md). Architecture
and implementation work should treat that document as the shared source for
Phase 1 table boundaries, migration order, tenant isolation rules, and
verification expectations.

Phase 1 persistence tables:

- `users`
- `auth_sessions`
- `workspaces`
- `workspace_members`
- `folders`
- `scenes`
- `scene_assets`

Collaboration presence, voice calls, AI prompts, and AI outputs remain
ephemeral; they are intentionally not stored in PostgreSQL. Public sharing is
stored in `scene_permissions` as documented by the Phase 3 design.

## Migration Strategy

Migrations should be incremental and reversible in dependency order:

1. PostgreSQL extensions.
2. Identity and sessions.
3. Workspaces and membership.
4. Folders and materialized path indexes.
5. Scenes and JSONB lifecycle indexes.
6. Scene asset metadata.
7. Optional search indexes.

Repository integration tests must prove that workspace-scoped data from one
workspace cannot be read or mutated through another workspace context.

## Dependencies

- Go 1.22+
- Chi router (`go-chi/chi/v5`)
- GORM
- Casbin v3
- PostgreSQL 16
- MinIO / S3-compatible storage
- React 18+, Vite, Zustand
- Shadcn UI
- `@excalidraw/excalidraw`
- `socket.io-client` 2.x-compatible realtime transport
- OpenAI-compatible AI API (DeepSeek default; optional local secret, required for production AI)

## Risks And Tradeoffs

- Last-write-wins may cause data loss in future multi-user scenarios (addressed in Phase 4).
- Materialized path updates require application-level transaction management.
- S3 signed URL expiration needs careful tuning for UX.

## Linked Decisions

- [0001-harness-first-development.md](../decisions/0001-harness-first-development.md)
- [0002-post-spec-product-lifecycle.md](../decisions/0002-post-spec-product-lifecycle.md)
- [0004-sqlite-durable-layer.md](../decisions/0004-sqlite-durable-layer.md)
- [0007-excalidraw-runtime-dependency.md](../decisions/0007-excalidraw-runtime-dependency.md)

## B3 browser integration (implemented; deployment pending)

The approved browser adapter uses durable scoped offline updates, fresh room admission and one Yjs content writer. AI additions and presentation order share that writer. See [browser design](BROWSER_COLLABORATION.md) for scope, recovery, legacy fallback and verification.
