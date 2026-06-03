# Project Specification: Excalidraw Ecosystem Rewrite

## 1. Project Vision
Rewrite the existing Excalidraw-based whiteboard and dashboard ecosystem (currently NestJS/Next.js) into a high-performance, clean, and SaaS-ready architecture using Go (Clean Architecture) and React Vite + Zustand.

## 2. Core Architecture
- **Backend:** Go with Clean Architecture (`internal/domain`, `internal/usecase`, `internal/repository`, `internal/delivery`).
- **Frontend:** React Vite, Shadcn UI, Zustand for state management.
- **Database:** PostgreSQL with Multi-tenancy support (UUID-based identifiers).
- **Storage:** S3-compatible (MinIO/AWS/GCS) for binary assets.
- **Auth:** Google OAuth integration.

## 3. Phase Breakdown

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

### Phase 5: AI Integration (Gemini API)
- **Goal:** Intelligent diagramming and code generation.
- **Requirements:**
  - Diagram-to-code (React/HTML generation).
  - Text-to-diagram (Natural language prompts).
  - Image-to-diagram (Visual reasoning).

## 4. Design Guidelines
- **Clean Code:** Strictly separate domain logic from infrastructure.
- **Performance:** Minimal bundle size, optimized SQL queries.
- **User Experience:** Google Drive-like familiarity for management; Excalidraw-native feel for drawing.

