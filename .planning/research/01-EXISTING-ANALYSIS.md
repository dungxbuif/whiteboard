# Analysis of Existing Excalidraw Fork & Self-hosting Logic

## 1. Project Overview
This project is a rewrite of a custom Excalidraw ecosystem previously built using NestJS (Backend), Next.js (Dashboard), and a forked Excalidraw frontend.

## 2. High-Level Changes Identified (Reference Review)

### A. Frontend (Dashboard & Excalidraw App)
- **Multi-layered Organization:** Workspace > Collection (Folder) > Scene (Drawing).
- **Authentication:** Shifted from Firebase to Google OAuth (`@react-oauth/google`).
- **State Management:** Extensive use of `Jotai` to sync state (e.g., `sceneAtom`, `workspaceAtom`).
- **UI Enhancements:** 
  - Integrated `SceneName` component into the Excalidraw toolbar for real-time renaming.
  - Custom `Sidebar` with Workspace switching and Folder navigation.
  - Custom hooks for auth checking (`useCheckValid`) and permission management.
- **Routing:** Complex routing handled via Nginx proxying (`/` for dashboard, `/dashboard/whiteboard` for Excalidraw).

### B. Backend (NestJS Server)
- **Module Structure:** Auth, Workspace, Collection, Scene, Team, File (S3/MinIO).
- **Multi-tenancy:** Data isolation using `workspace_id` and `tenant_id` across entities.
- **Storage:** Support for Google Cloud Storage and MinIO (S3 compatible).
- **Collaboration:** Dedicated `ws-whiteboard` service for Socket.io collaboration and **Voice Call** support.

## 3. Critical Self-hosting Logic to Preserve
- **Nginx Integration:** Reverse proxy configuration that handles both static assets and WebSocket upgrades for collaboration.
- **Auto-save Logic:** Scenes are saved to the custom PostgreSQL backend via `sceneService`.
- **Environment Parity:** Heavy reliance on `VITE_HTTP_STORAGE_BACKEND_URL` and `VITE_APP_WS_SERVER_URL` for service discovery.

## 4. Simulation & Development Strategy
- **Blueprints:** Use `references/docker-compose.integrated.yml` to orchestrate the dev environment.
- **Database:** PostgreSQL is the source of truth for metadata; S3 for binary assets.
- **Clean Code Target:** Transition NestJS (`references/server`) to Go (`Clean Architecture`) and Next.js (`references/devtool`) to React Vite for better performance and smaller bundle size.

