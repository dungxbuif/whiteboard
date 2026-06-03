# Project Roadmap: Excalidraw Ecosystem Rewrite (SaaS-Ready)

## Vision
A high-performance, self-hosted whiteboard ecosystem with Google Drive-like management, advanced presentation tools, and AI integration, built on Go (Clean Architecture) and React Vite.

## Phase 1: Self-hosted Dashboard Rewrite (Current)
- **Goal:** Replace NestJS/Next.js stack with Go/React Vite.
- **Key Features:** Google OAuth, Multi-tenant DB (UUIDs), Materialized Path folders, S3 sync.
- **Status:** Planning complete. 11 plans created across 9 waves.
- **Plans:** 11 plans
- [ ] 01-01-PLAN.md — Backend & Infrastructure Scaffold
- [ ] 01-02-PLAN.md — Multi-tenancy & Database
- [ ] 01-03-PLAN.md — Frontend Core & Shadcn Port
- [ ] 01-04-PLAN.md — Identity & Session Management
- [ ] 01-05-PLAN.md — Workspace Management
- [ ] 01-06-PLAN.md — Folder Hierarchy (Backend)
- [ ] 01-07-PLAN.md — Folder Tree & Navigation (Frontend)
- [ ] 01-08-PLAN.md — Scene Persistence & Features (BE)
- [ ] 01-09-PLAN.md — Excalidraw Integration (Frontend)
- [ ] 01-10-PLAN.md — S3 Asset Management
- [ ] 01-11-PLAN.md — Search & Refinement

## Phase 2: Presentation Mode
- **Goal:** Mirror Excalidraw Plus presentation features.
- **Reference:** [Excalidraw Plus Presentations](https://plus.excalidraw.com/use-cases/presentations)
- **Key Features:** Frame-based slides, focus mode, presenter navigation.
- **Plans:** 3 plans
- [ ] 02-01-PLAN.md — Frame Persistence & Management
- [ ] 02-02-PLAN.md — Presentation Engine & Navigation
- [ ] 02-03-PLAN.md — Navigator & Final Polish

## Phase 3: Shareable Production
- **Goal:** Enable public sharing of drawings with fine-grained access control.
- **Key Features:** Public read-only links, link management, password protection.
- **Plans:** 3 plans
- [ ] 03-01-PLAN.md — Sharing API and DB Schema
- [ ] 03-02-PLAN.md — Public UI and Access Control
- [ ] 03-03-PLAN.md — Legacy Parity and Testing

## Phase 4: Real-time Collaboration & Calling
- **Goal:** High-performance, self-hosted collaboration.
- **Key Features:** WebRTC/Socket.io integration, Voice calls.
- **Constraint:** Must use a unified relay/TURN/STUN solution to minimize port exposure (1 endpoint target).
- **Plans:** 3 plans
- [ ] 04-01-PLAN.md — Real-time Infrastructure (Coturn & Socket.io)
- [ ] 04-02-PLAN.md — Collaborative Presence & Scene Sync
- [ ] 04-03-PLAN.md — Real-time Voice Calling

## Phase 5: AI Features (Gemini API)
- **Goal:** Diagram-to-code, text-to-diagram, and image-to-diagram workflows.
- **Key Features:** Gemini 1.5/2.0 integration, visual reasoning for UI generation.
- **Plans:** 3 plans
- [ ] 05-01-PLAN.md — Foundation & Gemini Proxy
- [ ] 05-02-PLAN.md — Text & Code Transformations
- [ ] 05-03-PLAN.md — Multimodal & Visual Chat
