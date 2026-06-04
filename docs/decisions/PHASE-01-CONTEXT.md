# Phase 1: Self-hosted Dashboard Rewrite - Context (REFINED)

**Gathered:** 2026-06-03
**Status:** Ready for planning

<domain>
## Phase Boundary
This phase delivers a complete rewrite of the Excalidraw Dashboard (Frontend) and Backend into a clean, modern architecture while preserving the core self-hosting logic from the existing fork.
</domain>

<spec_lock>
## Requirements (locked via SPEC.md)

**10 requirements are locked.** See `01-SPEC.md` for full requirements, boundaries, and acceptance criteria.

**In scope (from SPEC.md):**
- Full rewrite of Frontend in React Vite + Shadcn.
- Full rewrite of Backend in Go (Clean Architecture).
- Google OAuth login.
- Multi-tenant database design.
- Folder/File CRUD with Trash, Star, and Search.
- S3 integration for media.
</spec_lock>

<decisions>
## Implementation Decisions

### Data Organization (Folders & Files)
- **D-01: Application-Level Materialized Path:** Folder hierarchies will be managed using a materialized path string (e.g., `root/work/design`). Path updates (e.g., when moving a folder) will be handled by the Go application logic (finding all sub-items and updating their paths) to remain database-agnostic.
- **D-02: UUID for Primary Keys:** Use UUIDs for all public-facing and internal entity IDs.

### Multi-tenancy & Session
- **D-03: Single Active Workspace per Session:** The system will assume one active workspace per user session. Multi-tab workspace switching (different workspace in different tab) is NOT required for Phase 1.
- **D-04: Context-based Tenancy:** `WorkspaceID` will be extracted at the middleware level and passed through the Go `context.Context` to repositories for automatic filtering.

### State Management & Sync
- **D-05: Zustand for Dashboard:** Pure Zustand for managing folder tree, file lists, and UI state.
- **D-06: Last-Write-Wins (LWW):** Simple conflict resolution for whiteboard sync. No complex optimistic locking; the last successfully received drawing JSON overwrites the previous version.
- **D-07: Debounced Auto-sync:** Whiteboard data and S3 assets sync after a short delay of inactivity.

### Architecture
- **D-08: Go Clean Architecture Pattern:** Strictly follow `domain`, `usecase`, `repository`, `delivery` (HTTP) separation.
  - `domain/`: Models & Interfaces.
  - `usecase/`: Business logic (Path manipulation, S3 flow).
  - `repository/`: SQL implementation.
  - `delivery/`: HTTP handlers & Middleware.

</decisions>

<canonical_refs>
## Canonical References
- `.planning/phases/01-selfhost-src/frontend/01-SPEC.md`
- `.planning/research/01-EXISTING-ANALYSIS.md`
- `.planning/research/00-REFERENCES-CONTEXT.md`
- `https://github.com/bxcodec/go-clean-arch`
</canonical_refs>

<code_context>
## Existing Code Insights
- **UI Porting:** Components from `references/devtool/src/components/ui/` are the visual baseline.
- **Excalidraw Logic:** The forked app in `references/excalidraw` contains the `SceneName` and `authService` integration patterns to be replicated in the new React app.
</code_context>

---

*Phase: 01-selfhost-dashboard*
*Context gathered: 2026-06-03 (Refined)*
