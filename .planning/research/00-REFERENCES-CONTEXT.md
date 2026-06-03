# References Context Map

This document clarifies the purpose and relationship of the projects in the `references/` directory. These are forked/existing projects used as a baseline for the Phase 1 rewrite.

## 1. Directory Overview

| Directory | Name | Purpose | Tech Stack | Phase 1 Action |
|-----------|------|---------|------------|----------------|
| `devtool/` | Dashboard | User portal, Workspace/File management | Next.js, Shadcn, Jotai | **Rewrite** to React Vite + Zustand |
| `server/` | Backend | Metadata API, Auth, S3 proxy | NestJS, TypeORM, Postgres | **Rewrite** to Go Clean Architecture |
| `excalidraw/` | Whiteboard | Core drawing engine (Forked) | React, Vite, PeerJS | **Embed** as a library (`@excalidraw/excalidraw`) |
| `ws-whiteboard/`| Collab Server | Real-time socket & Voice calls | Node.js, Socket.io | **Ignore** (Deferred to Phase 4) |
| `nginx/` | Proxy | Unified routing & SSL | Nginx | **Maintain** parity in the new setup |

## 2. Key Custom Logic (The "Self-host" Delta)

The user has modified these from their original upstream sources to support self-hosting.

### A. Authentication Flow
- **Old (`devtool` & `server`):** Google OAuth via `@react-oauth/google`.
- **Logic:** FE sends Google token to BE -> BE validates and returns a custom JWT session.
- **Critical:** The `AuthChecking.tsx` logic in `devtool` ensures users are redirected to login if the session is invalid.

### B. Storage & Syncing
- **Database:** PostgreSQL stores Workspaces, Collections (Folders), and Scenes (Metadata + Excalidraw JSON).
- **Binary Assets:** Images/Videos are synced to S3 (MinIO supported) via the Go backend.
- **Sync Mechanism:** Debounced updates from the Excalidraw app to the BE API (`sceneService`).

### C. Whiteboard Integration (`excalidraw` folder)
- **Scene Name:** A custom component added to the Excalidraw UI to allow renaming drawings which syncs back to the dashboard database.
- **Multi-tenancy:** The URL structure `/w/[id]/[collectionId]` indicates that the dashboard passes workspace context to the whiteboard.

## 3. Reference Files of Interest

- **Infrastructure:** `references/docker-compose.integrated.yml` (Shows how services talk to each other).
- **Frontend Assets:** `references/devtool/src/components/ui/` (Source for Shadcn UI components).
- **API Definition:** `references/devtool/src/shared/services/` (Defines the REST interface we must recreate in Go).
- **Data Model:** `references/server/src/modules/*/entities/` (Defines the Postgres schema for Workspaces/Scenes).

## 4. Why Rewrite?
- **Backend:** Move from NestJS to **Go** for performance, smaller footprint, and better concurrency handling for Phase 4 (Real-time).
- **Frontend:** Move from Next.js to **React Vite** to simplify the stack (no need for SSR overhead) and improve build/dev speeds.
- **Code Quality:** Apply **Clean Architecture** (Go) and **Zustand** (React) to replace the mixed patterns in the references.

