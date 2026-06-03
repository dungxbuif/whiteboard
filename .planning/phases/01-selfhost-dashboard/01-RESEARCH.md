# Phase 1: Self-hosted Dashboard Rewrite - Research

**Researched:** 2026-06-03
**Domain:** Go Backend, React Frontend, PostgreSQL (Multi-tenant), Materialized Path
**Confidence:** HIGH

## Summary

This research establishes the blueprint for rewriting the Excalidraw Dashboard and Backend. We move from a NestJS/Next.js setup to a Go Clean Architecture backend and a React Vite frontend. The system is designed for multi-tenancy from the ground up, using PostgreSQL as the primary data store and S3 for binary assets. A key architectural decision is the use of an application-level materialized path for folder hierarchies, ensuring database portability and efficient subtree queries.

**Primary recommendation:** Use `go-chi` for routing and `pgx` for database interaction in the backend, while leveraging `Zustand` for state management and `@excalidraw/excalidraw` for the whiteboard integration in the frontend.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **Application-Level Materialized Path:** Folder hierarchies will be managed using a materialized path string (e.g., `root/work/design`). Path updates (e.g., when moving a folder) will be handled by the Go application logic.
- **UUID for Primary Keys:** Use UUIDs for all public-facing and internal entity IDs.
- **Single Active Workspace per Session:** The system will assume one active workspace per user session.
- **Context-based Tenancy:** `WorkspaceID` will be extracted at the middleware level and passed through the Go `context.Context` to repositories for automatic filtering.
- **Zustand for Dashboard:** Pure Zustand for managing folder tree, file lists, and UI state.
- **Last-Write-Wins (LWW):** Simple conflict resolution for whiteboard sync. No complex optimistic locking.
- **Debounced Auto-sync:** Whiteboard data and S3 assets sync after a short delay of inactivity.
- **Go Clean Architecture Pattern:** Strictly follow `domain`, `usecase`, `repository`, `delivery` (HTTP) separation.

### the agent's Discretion
- Database driver/ORM choice (Recommended: `pgx` + `sqlc` or raw SQL for transparency).
- HTTP framework (Recommended: `go-chi`).
- Folder path format (Recommended: Use UUIDs in path for rename-resilience).

### Deferred Ideas (OUT OF SCOPE)
- Multi-user collaboration (Phase 4).
- AI features (Phase 5).
- Complex permission systems (deferred).
- Public sharing links (Phase 3).
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| REQ-01 | Go Clean Architecture | Verified project structure via `bxcodec/go-clean-arch` v4 patterns. |
| REQ-02 | Multi-tenant Database Schema | Designed schema with `workspace_id` and Context-based filtering. |
| REQ-03 | S3 Asset Storage | Identified `aws-sdk-go-v2` for integration and debounced sync pattern. |
| REQ-04 | Google OAuth Integration | Verified flow using `golang.org/x/oauth2` and JWT session management. |
| REQ-06 | Nested Folders | Defined Materialized Path implementation per D-01. |
| REQ-07 | Trash System (Soft Delete) | Implemented via `deleted_at` column and application-level filtering. |
| REQ-10 | Embedded Excalidraw | Verified `@excalidraw/excalidraw` (v0.18.1) compatibility for React embedding. |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| User Authentication | API (Go) | Browser (Client) | Google OAuth callback handled by BE; JWT stored in FE. |
| Workspace Context | API (Go) | Frontend (Vite) | BE extracts `WorkspaceID` from token/context; FE manages active state in Zustand. |
| Folder Hierarchy | API (Go) | Database | Materialized path logic in Go; indexing in Postgres. |
| Whiteboard Editor | Browser (Client) | — | Embedded React component handles rendering and local state. |
| Data Persistence | Database (PG) | API (Go) | Postgres stores metadata/JSON; Go orchestrates SQL queries. |
| Asset Storage | S3 | API (Go) | Go provides signed URLs; S3 stores images/blobs. |

## Standard Stack

### Core (Backend)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `go-chi/chi/v5` | 5.2.1 | Router | Lightweight, idiomatic, `context.Context` native. [VERIFIED: pkg.go.dev] |
| `jackc/pgx/v5` | 5.9.1 | Postgres Driver | Performance, type-safety, standard for Go/PG apps. [VERIFIED: pkg.go.dev] |
| `golang-jwt/jwt/v5`| 5.3.1 | Auth Tokens | Standard for JWT operations in Go. [VERIFIED: pkg.go.dev] |
| `aws-sdk-go-v2` | v1.x | S3 Storage | Official SDK for S3-compatible storage. [VERIFIED: pkg.go.dev] |

### Core (Frontend)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `React` | 18.x | UI Library | Project requirement. |
| `Vite` | 5.x | Build Tool | Standard for modern React dev. |
| `Zustand` | 5.0.14 | State Management | Low-boilerplate, easy to use outside React context. [VERIFIED: npm registry] |
| `@excalidraw/excalidraw` | 0.18.1 | Whiteboard Engine | Core whiteboard component. [VERIFIED: npm registry] |
| `Tailwind CSS` | 3.x | Styling | Required for Shadcn UI. |
| `Shadcn UI` | — | Component Lib | Ported from `references/devtool`. |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `pgx` | `GORM` | GORM is easier for simple CRUD but obscures complex multi-tenant filtering. |
| `go-chi` | `Gin` | Gin is faster but `chi` is more standard-library aligned. |
| Materialized Path | `ltree` extension | `ltree` is powerful but limits portability to non-PG databases. |

## Package Legitimacy Audit

| Package | Registry | Age | Downloads | Source Repo | slopcheck | Disposition |
|---------|----------|-----|-----------|-------------|-----------|-------------|
| zustand | npm | 5 yrs | 5M/wk | pmndrs/zustand | [OK] | Approved |
| lucide-react | npm | 3 yrs | 2M/wk | lucide-icons/lucide | [OK] | Approved |
| clsx | npm | 6 yrs | 20M/wk | lukeed/clsx | [OK] | Approved |
| tailwind-merge | npm | 3 yrs | 10M/wk | dcastil/tailwind-merge | [OK] | Approved |
| @excalidraw/excalidraw | npm | 4 yrs | 100k/wk | excalidraw/excalidraw | [OK] | Approved |
| axios | npm | 10 yrs | 40M/wk | axios/axios | [OK] | Approved |

## Architecture Patterns

### Recommended Project Structure (Backend)
```
internal/
├── domain/             # Models & Repository/Usecase Interfaces
│   ├── workspace.go
│   ├── folder.go
│   ├── scene.go
│   └── user.go
├── usecase/            # Business Logic
│   ├── workspace_ucase.go
│   ├── folder_ucase.go
│   ├── scene_ucase.go
│   └── auth_ucase.go
├── repository/         # Data Access Implementations
│   ├── postgres/
│   │   ├── folder_repo.go
│   │   └── ...
│   └── s3/
│       └── s3_repo.go
└── delivery/           # Transport Layer (HTTP)
    └── http/
        ├── handler/
        └── middleware/
```

### Pattern 1: Multi-tenant Middleware
**What:** Extract `workspace_id` from the request (header or token) and inject into `context.Context`.
**When to use:** All protected routes.
**Example:**
```go
func TenantMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        workspaceID := r.Header.Get("X-Workspace-ID")
        ctx := context.WithValue(r.Context(), "workspaceID", workspaceID)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

### Pattern 2: Materialized Path Update
**What:** When a folder is moved, update the path of all descendants.
**When to use:** `FolderUsecase.Update` where `parent_id` changes.
**Strategy:**
1. Fetch old path.
2. Calculate new path.
3. Update descendant paths using `UPDATE folders SET path = REPLACE(path, 'old/path', 'new/path') WHERE path LIKE 'old/path/%'`.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Google OAuth Exchange | Custom HTTP client | `golang.org/x/oauth2` | Security, edge case handling. |
| JWT Verification | Custom parser | `golang-jwt/jwt/v5` | Complex claim validation. |
| Database Migrations | Manual SQL scripts | `golang-migrate/migrate` | Version control, rollbacks. |
| UI Components | Custom buttons/inputs | `Shadcn UI` | Accessibility (Radix) and consistency. |

## Common Pitfalls

### Pitfall 1: Materialized Path Rename
**What goes wrong:** Renaming a folder name that is part of the path requires updating all descendants.
**How to avoid:** Use **IDs** in the materialized path (e.g., `workspace_id/uuid/uuid`) instead of human-readable names. Only change the path when moving (changing `parent_id`), not when renaming.

### Pitfall 2: Context Leakage
**What goes wrong:** Forgetting to pass `context.Context` to the repository leads to missing `workspace_id` filtering.
**How to avoid:** Enforce `ctx` as the first argument in all repository/usecase methods.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go `testing`, Vitest |
| Config file | `vitest.config.ts`, `Makefile` for Go tests |
| Quick run command | `go test ./...`, `npm test` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| REQ-01 | Layers separation | Unit | `go test ./internal/usecase/...` | ❌ Wave 0 |
| REQ-02 | Tenancy filtering | Integration | `go test ./internal/repository/postgres/...` | ❌ Wave 0 |
| REQ-04 | JWT Validation | Unit | `go test ./internal/delivery/http/middleware_test.go` | ❌ Wave 0 |
| REQ-06 | Move Folder | Integration | `go test ./internal/usecase/folder_test.go` | ❌ Wave 0 |

## Security Domain

### Applicable ASVS Categories
| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | yes | Google OAuth2 via official SDK |
| V3 Session Management | yes | JWT with HTTP-only cookies or Bearer tokens |
| V5 Input Validation | yes | Go `validator/v10` or Pydantic-like structs |
| V12 File Upload | yes | S3 Content-Type validation & Size limits |

### Known Threat Patterns for Go/React
| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Cross-tenant Data Leak | Information Disclosure | Mandatory `workspace_id` filtering in Repository layer via Context. |
| Broken Access Control | Elevation of Privilege | Middleware verifies token ownership of the target workspace. |

## Sources

### Primary (HIGH confidence)
- `bxcodec/go-clean-arch` v4 - Project structure baseline.
- `official pkg.go.dev` - Package versions for `pgx`, `chi`, `jwt`.
- `npm registry` - Versions for `zustand`, `@excalidraw/excalidraw`.
- `01-SPEC.md` & `01-CONTEXT.md` - Phase constraints.

### Secondary (MEDIUM confidence)
- Web search for "PostgreSQL materialized path vs ltree".

## Metadata
**Research date:** 2026-06-03
**Valid until:** 2026-07-03
