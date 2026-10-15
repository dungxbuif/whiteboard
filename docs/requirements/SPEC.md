---
artifact_type: requirement_spec
id: REQ-MASTER
status: active
owner: human
human_fields: [product_summary, goals, non_goals, users_and_stakeholders, acceptance_criteria]
ai_fields: [functional_requirements, non_functional_requirements, constraints, linked_decisions]
shared_fields: [status, trace]
---

# Project Specification: Excalidraw Ecosystem Rewrite

## Field Ownership

- Human owns product intent, goals, non-goals, users, and acceptance criteria.
- AI may draft requirements, constraints, and linked decisions for review.

## Product Summary

Rewrite the existing Excalidraw-based whiteboard and dashboard ecosystem (currently NestJS/Next.js) into a high-performance, clean, and SaaS-ready architecture using Go (Clean Architecture) and React Vite + Zustand.

## Goals

- High-performance, self-hosted whiteboard ecosystem with Google Drive-like management.
- Advanced presentation tools.
- AI integration.
- Built on Go (Clean Architecture) and React Vite.

## Non-Goals

- Multi-user collaboration (deferred to Phase 4).
- AI features (deferred to Phase 5).
- Complex permission systems (deferred to later phases).
- Public sharing links (implemented in Phase 3).

## Users And Stakeholders

- Self-hosted whiteboard users.
- Teams requiring collaborative drawing tools.

## Core Architecture

- **Backend:** Go with Clean Architecture (`src/backend/internal/domain`, `src/backend/internal/usecase`, `src/backend/internal/repository`, `src/backend/internal/delivery`).
- **Frontend:** React Vite, Shadcn UI, Zustand for state management.
- **Database:** PostgreSQL with Multi-tenancy support (UUID-based identifiers).
- **Storage:** S3-compatible (MinIO/AWS/GCS) for binary assets.
- **Auth:** Google OAuth integration.

## Phase Breakdown

### Phase 1: Self-hosted Dashboard & Core Infrastructure
- **Status:** Planned.
- **Goal:** Core rewrite of Dashboard and Backend.
- **Requirements:**
  - Google Drive-like file management (Nested Folders, Trash, Starring).
  - Materialized Path for folder hierarchies.
  - Multi-tenant isolation at DB and Middleware levels.
  - Debounced auto-sync for drawings.

### Phase 2: Presentation Mode
- **Goal:** Mirror Excalidraw Plus presentation features.
- **Requirements:**
  - Frame-based slide management.
  - Presentation view with navigation.
  - Focus mode for distraction-free viewing.

### Phase 3: Shareable Production
- **Goal:** Parity with original Excalidraw sharing.
- **Requirements:**
  - Public read-only sharing links.
  - Link permissions and scene-level access control.
  - Legacy parity with existing `references/excalidraw` sharing logic.

### Phase 4: Real-time Collaboration & Voice Calling
- **Goal:** High-fidelity P2P and Relay-based collaboration.
- **Requirements:**
  - WebRTC/Socket.io integration.
  - Voice call support within the whiteboard.
  - Unified TURN/STUN relay solution (Single port/endpoint target).

### Phase 5: AI Integration (OpenAI-Compatible API)
- **Goal:** Intelligent diagramming and code generation.
- **Requirements:**
  - Diagram-to-code (React/HTML generation).
  - Text-to-diagram (Natural language prompts).
  - Image-to-diagram (Visual reasoning).

## Design Guidelines

- **Clean Code:** Strictly separate domain logic from infrastructure.
- **Performance:** Minimal bundle size, optimized SQL queries.
- **User Experience:** Google Drive-like familiarity for management; Excalidraw-native feel for drawing.

---

## Phase 1 Detailed Requirements

### Ambiguity Report
- **Goal Clarity:** 0.90 (min 0.75) ✓
- **Boundary Clarity:** 0.85 (min 0.70) ✓
- **Constraint Clarity:** 0.90 (min 0.65) ✓
- **Acceptance Criteria:** 0.75 (min 0.70) ✓
- **Final Ambiguity Score:** 0.14 (Gate: ≤ 0.20)

### Goal
Rewrite the existing Excalidraw dashboard and hosting logic into a clean, modern SaaS-ready architecture (multi-tenant) that functions like Google Drive for Excalidraw drawings.

### Functional Requirements

#### Core Infrastructure & Architecture
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

#### Authentication & Access
- **REQ-04: Google OAuth Integration**
  - **Current State:** Google OAuth authorization-code flow with backend-issued JWT sessions.
  - **Target State:** Users can log in via Google; sessions managed by the Go backend (JWT or Secure Cookies).
  - **Acceptance:** User can log in, see their profile, and session persists.

#### Workspace & File Management (Google Drive-like)
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

#### Whiteboard Integration
- **REQ-10: Embedded Excalidraw Library**
  - **Current State:** Standalone Excalidraw app proxied.
  - **Target State:** Excalidraw embedded as a React component using `@excalidraw/excalidraw`.
  - **Acceptance:** Drawing interface opens within the dashboard application; state is saved to the PostgreSQL database via Go API.

### Phase 1 Boundaries

#### In Scope
- Full rewrite of Frontend in React Vite + Shadcn (reusing UI components from `./devtool`).
- Full rewrite of Backend in Go (Clean Architecture).
- Google OAuth login.
- Multi-tenant database design.
- Folder/File CRUD with Trash, Star, and Search.
- S3 integration for media.

#### Out of Scope
- Multi-user collaboration (Phase 4).
- AI features (Phase 5).
- Complex permission systems (e.g., inviting other users to workspaces) - deferred for later phases.
- Public sharing links (Phase 3).

### Phase 1 Acceptance Criteria
- [ ] Backend code passes "Clean Architecture" review (separation of concerns).
- [ ] Database schema verified for multi-tenancy support.
- [x] User can log in with Google.
- [ ] User can create multiple Workspaces.
- [ ] User can create, rename, and move drawings into nested folders.
- [ ] User can "Star" a drawing and see it in the Starred section.
- [ ] User can move a drawing to Trash and restore it.
- [ ] Searching for a file name returns correct results.
- [ ] Drawing state (JSON) is saved to DB and assets (images) are saved to S3.
- [ ] Excalidraw UI is fully functional within the React app.

---

## Phase 2 Requirements: Presentation Mode

### Goal
Implement a high-fidelity presentation mode that mirrors Excalidraw Plus functionality, allowing users to define frames as slides and navigate through them.

### Requirements
- **REQ-01: Frame Management:** Users can create, resize, and label "Frames" in Excalidraw that act as slide boundaries.
- **REQ-02: Presentation UI:** A dedicated mode that hides all editing tools and focuses on the current frame.
- **REQ-03: Navigation:** Support for Next/Prev slide via keyboard (Arrows/Space) and UI controls.
- **REQ-04: Slide List:** A sidebar or overlay showing all frames in order for quick jumping.
- **REQ-05: Persistence:** Frame positions and order must be saved in the Scene JSON.

### Acceptance Criteria
- [x] Frames can be created and ordered.
- [x] Presentation mode activates/deactivates cleanly.
- [x] Keyboard navigation works between frames.

---

## Phase 3 Requirements: Shareable Production

### Goal
Enable public sharing of drawings with fine-grained access control, replicating the original Excalidraw production sharing logic.

### Requirements
- **REQ-01: Public Links:** Generate obfuscated public URLs for read-only access.
- **REQ-02: Link Management:** Users can enable/disable public sharing for any scene.
- **REQ-03: Access Control:** Support for basic password protection on shared links.
- **REQ-04: Parity:** Ensure the sharing logic matches the reference server's public module.

### Acceptance Criteria
- [x] Public links are accessible without login.
- [x] Revoking a link immediately blocks access.

---

## Phase 4 Requirements: Real-time Collaboration & Calling

### Goal
Implement seamless multi-user collaboration and voice calling with a focus on self-hosted infrastructure and minimal port exposure.

### Requirements
- **REQ-01: Real-time Sync:** Multi-user editing with cursor visibility and presence.
- **REQ-02: Voice Calling:** Integrated P2P voice calls using WebRTC.
- **REQ-03: Relay Infrastructure:** Deployment of a unified TURN/STUN relay (e.g., Coturn) on a single port.
- **REQ-04: Performance:** Low-latency synchronization for up to 10 concurrent users per scene.

### Acceptance Criteria
- [ ] Multiple users can edit the same scene simultaneously.
- [ ] Voice calls are clear and stable.
- [ ] Infrastructure runs behind a single endpoint.

---

## Phase 5 Requirements: AI Features (OpenAI-Compatible API)

### Goal
Integrate an OpenAI-compatible provider (DeepSeek by default) to provide intelligent drawing assistance, automation, and code generation.

### Requirements
- **REQ-01: Text-to-Diagram:** Generate Excalidraw elements from natural language prompts.
- **REQ-02: Diagram-to-Code:** Convert visual UI components into React/HTML code.
- **REQ-03: Image-to-Diagram:** Convert uploaded images into editable Excalidraw elements.
- **REQ-04: Visual Chat:** Chat with the configured provider about the current whiteboard content.

### Acceptance Criteria
- [ ] Prompt results in valid Excalidraw JSON.
- [ ] Code generation reflects the visual layout accurately.

## Constraints

- Go 1.22+ required for backend.
- PostgreSQL for persistence.
- S3-compatible storage for binary assets.
- Google OAuth 2.0 for authentication.

## Linked Decisions

- [0001-harness-first-development.md](../decisions/0001-harness-first-development.md)
- [0002-post-spec-product-lifecycle.md](../decisions/0002-post-spec-product-lifecycle.md)
- [0003-generic-spec-intake-harness.md](../decisions/0003-generic-spec-intake-harness.md)
