# System Architecture: Excalidraw Ecosystem

## 1. Overview
High-performance, self-hosted whiteboard ecosystem using Go (Clean Architecture) and React Vite.

## 2. Technical Stack
- **Backend:** Go 1.22+
- **Architecture:** Clean Architecture (Domain, Usecase, Repository, Delivery)
- **Frontend:** React + Vite, Zustand, Shadcn UI
- **Database:** PostgreSQL (Multi-tenant)
- **Storage:** S3-compatible (MinIO/S3)
- **Auth:** Google OAuth 2.0

## 3. System Boundaries
- **Dashboard:** React SPA for workspace and file management.
- **Backend API:** RESTful API for metadata and asset management.
- **Whiteboard:** Embedded `@excalidraw/excalidraw` engine.
- **Relay/Collab:** WebRTC + Socket.io for real-time (Future Phase).

## 4. Key Patterns
- **Multi-tenancy:** Enforced via `WorkspaceID` in Go Context.
- **Data Org:** Materialized Path for folder hierarchies.
- **Sync:** Debounced auto-save to Backend/S3.

## 5. Integration Points
- **Google OAuth:** External identity provider.
- **S3:** Blob storage for images and videos.
- **Excalidraw Library:** Core drawing primitive.
