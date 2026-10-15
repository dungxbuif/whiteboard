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
- **Auth:** Google OAuth 2.0

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
- **Delivery Layer:** HTTP handlers and middleware. Middleware validates auth
  and injects user/workspace context for usecase and repository calls.
- **Whiteboard:** Embedded `@excalidraw/excalidraw` engine. Excalidraw scene
  content is persisted through the backend as JSONB, not by direct browser DB
  access.
- **Relay/Collab:** WebRTC + Socket.io for real-time collaboration in a future
  phase. Phase 1 must not introduce collaboration persistence tables.

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
- **Workspace Membership:** Workspace access is checked against
  `workspace_members` after authenticated sessions exist. Header-based
  workspace selection is only a temporary pre-auth implementation bridge.
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
- **Sync:** Frontend saves scene changes through debounced API calls. Asset
  uploads go through backend S3 orchestration before scene JSON references the
  stored asset.
- **Editor loading:** Excalidraw JavaScript is dynamically imported only after
  a scene opens. Zustand owns scene metadata and save status while Excalidraw
  owns high-frequency live canvas state.
- **Save ordering:** A client generation token prevents stale request
  completions from replacing newer save status. The backend remains
  last-write-wins and `content_version` is observability metadata.

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
```

## Runtime Flow

```text
User Login --> Google OAuth --> Session/JWT --> Workspace Selection
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

- **Google OAuth:** External identity provider.
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

Tables for public sharing, collaboration presence, voice calls, AI prompts, and
AI outputs are deferred to later phases and should not be added during Phase 1
unless the product scope changes through a new decision record.

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

## Risks And Tradeoffs

- Last-write-wins may cause data loss in future multi-user scenarios (addressed in Phase 4).
- Materialized path updates require application-level transaction management.
- S3 signed URL expiration needs careful tuning for UX.

## Linked Decisions

- [0001-harness-first-development.md](../decisions/0001-harness-first-development.md)
- [0002-post-spec-product-lifecycle.md](../decisions/0002-post-spec-product-lifecycle.md)
- [0004-sqlite-durable-layer.md](../decisions/0004-sqlite-durable-layer.md)
- [0007-excalidraw-runtime-dependency.md](../decisions/0007-excalidraw-runtime-dependency.md)
