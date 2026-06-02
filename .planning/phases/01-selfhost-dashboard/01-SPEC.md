# Phase 1 Spec: Self-hosted Whiteboard Dashboard (Rewrite)

## Ambiguity Report
- **Goal Clarity:** 0.90 (min 0.75) ✓
- **Boundary Clarity:** 0.85 (min 0.70) ✓
- **Constraint Clarity:** 0.90 (min 0.65) ✓
- **Acceptance Criteria:** 0.75 (min 0.70) ✓
- **Final Ambiguity Score:** 0.14 (Gate: ≤ 0.20)

## 1. Goal
Rewrite the existing Excalidraw dashboard and hosting logic into a clean, modern SaaS-ready architecture (multi-tenant) that functions like Google Drive for Excalidraw drawings.

## 2. Requirements

### 2.1 Core Infrastructure & Architecture
- **REQ-01: Go Clean Architecture**
  - **Current State:** Existing reference uses NestJS or vanilla Express.
  - **Target State:** Backend implemented in Go using `bxcodec/go-clean-arch` pattern (Domain, Usecase, Repository, Delivery layers).
  - **Acceptance:** Code structure follows the specified layers; no logic leaks between layers.
- **REQ-02: Multi-tenant Database Schema**
  - **Current State:** Simple relational schema.
  - **Target State:** PostgreSQL schema designed for multi-tenancy (every table related to data has a `workspace_id` and/or `tenant_id`).
  - **Acceptance:** SQL migrations exist; queries always filter by workspace context.
- **REQ-03: S3 Asset Storage**
  - **Current State:** Local file storage or unknown.
  - **Target State:** Integration with S3 (or S3-compatible) for storing images, videos, and large assets.
  - **Acceptance:** Uploaded files are successfully stored in S3; UI retrieves them via signed URLs or public paths.

### 2.2 Authentication & Access
- **REQ-04: Google OAuth Integration**
  - **Current State:** Custom auth or none.
  - **Target State:** Users can log in via Google; sessions managed by the Go backend (JWT or Secure Cookies).
  - **Acceptance:** User can log in, see their profile, and session persists.

### 2.3 Workspace & File Management (Google Drive-like)
- **REQ-05: Multi-tenant Workspaces**
  - **Current State:** Single space.
  - **Target State:** Support for creating and switching between multiple Workspaces (even if Phase 1 uses one account).
  - **Acceptance:** UI allows workspace creation; data is isolated per workspace.
- **REQ-06: Nested Folders**
  - **Current State:** Flat list of files.
  - **Target State:** Users can create folders and subfolders to organize drawings.
  - **Acceptance:** Folder creation, navigation, and file moving work as expected.
- **REQ-07: Trash System (Soft Delete)**
  - **Current State:** Immediate deletion.
  - **Target State:** Files/Folders moved to "Trash" instead of being permanently deleted; can be restored.
  - **Acceptance:** "Move to Trash" and "Restore" actions work; trash is excluded from main view.
- **REQ-08: Starring & Favorites**
  - **Current State:** None.
  - **Target State:** Users can "Star" files or folders for quick access.
  - **Acceptance:** Starred items appear in a dedicated "Starred" view.
- **REQ-09: Search Functionality**
  - **Current State:** Manual browsing.
  - **Target State:** Search bar to filter files/folders by name within the current workspace.
  - **Acceptance:** Real-time or submit-based search returns correct filtered results.

### 2.4 Whiteboard Integration
- **REQ-10: Embedded Excalidraw Library**
  - **Current State:** Standalone Excalidraw app proxied.
  - **Target State:** Excalidraw embedded as a React component using `@excalidraw/excalidraw`.
  - **Acceptance:** Drawing interface opens within the dashboard application; state is saved to the PostgreSQL database via Go API.

## 3. Boundaries

### 3.1 In Scope
- Full rewrite of Frontend in React Vite + Shadcn (reusing UI components from `./devtool`).
- Full rewrite of Backend in Go (Clean Architecture).
- Google OAuth login.
- Multi-tenant database design.
- Folder/File CRUD with Trash, Star, and Search.
- S3 integration for media.

### 3.2 Out of Scope
- Multi-user collaboration (Phase 4).
- AI features (Phase 5).
- Complex permission systems (e.g., inviting other users to workspaces) - deferred for later phases.
- Public sharing links (Phase 3).

## 4. Acceptance Criteria
- [ ] Backend code passes "Clean Architecture" review (separation of concerns).
- [ ] Database schema verified for multi-tenancy support.
- [ ] User can log in with Google.
- [ ] User can create multiple Workspaces.
- [ ] User can create, rename, and move drawings into nested folders.
- [ ] User can "Star" a drawing and see it in the Starred section.
- [ ] User can move a drawing to Trash and restore it.
- [ ] Searching for a file name returns correct results.
- [ ] Drawing state (JSON) is saved to DB and assets (images) are saved to S3.
- [ ] Excalidraw UI is fully functional within the React app.

