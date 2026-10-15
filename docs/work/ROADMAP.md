---
artifact_type: roadmap
id: ROADMAP
status: active
owner: human
human_fields:
  - milestones
  - priority
  - phase_order
ai_fields:
  - phase_links
  - status_summaries
shared_fields:
  - milestone_status
updated: 2026-06-11
---

# Roadmap

## Field Ownership

- Human owns milestones, priority, and phase order.
- AI maintains phase links and status summaries.

Use this file to group work into milestones or major phases.

## Roadmap Rules

- A roadmap item becomes executable only after it has a phase file in `docs/work/phases/`.
- Each phase should contain one or more tickets or bugs.
- Completed phases should link to release notes or changelog entries when relevant.

## Vision

A high-performance, self-hosted whiteboard ecosystem with Google Drive-like management, advanced presentation tools, and AI integration, built on Go (Clean Architecture) and React Vite.

## Milestones

| Milestone | Goal | Status | Phase Files | Plans |
| --- | --- | --- | --- | --- |
| M1 | Phase 1: Self-hosted Dashboard Rewrite | in_progress | [PHASE-1.md](phases/PHASE-1.md) | 11 plans (01-01 to 01-11) |
| M2 | Phase 2: Presentation Mode | done | [PHASE-2-PRODUCT.md](phases/PHASE-2-PRODUCT.md) | 3 plans (02-01 to 02-03) |
| M3 | Phase 3: Shareable Production | done | [PHASE-3.md](phases/PHASE-3.md) | 3 plans (03-01 to 03-03) |
| M4 | Phase 4: Real-time Collaboration & Calling | planned | [PHASE-4.md](phases/PHASE-4.md) | 3 plans (04-01 to 04-03) |
| M5 | Phase 5: AI Features (Gemini API) | planned | [PHASE-5.md](phases/PHASE-5.md) | 3 plans (05-01 to 05-03) |
| MH | Harness: Develop Harness CLI (Hard Enforcement) | ready | [PHASE-2.md](phases/PHASE-2.md) | TBD |

## Phase 1 Plans

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

## Phase 2 Plans

- [x] 02-01-PLAN.md — Frame Persistence & Management
- [x] 02-02-PLAN.md — Presentation Engine & Navigation
- [x] 02-03-PLAN.md — Navigator & Final Polish

## Phase 3 Plans

- [x] 03-01-PLAN.md — Sharing API and DB Schema
- [x] 03-02-PLAN.md — Public UI and Access Control
- [x] 03-03-PLAN.md — Legacy Parity and Testing

## Phase 4 Plans

- [ ] 04-01-PLAN.md — Real-time Infrastructure (Coturn & Socket.io)
- [ ] 04-02-PLAN.md — Collaborative Presence & Scene Sync
- [ ] 04-03-PLAN.md — Real-time Voice Calling

## Phase 5 Plans

- [ ] 05-01-PLAN.md — Foundation & Gemini Proxy
- [ ] 05-02-PLAN.md — Text & Code Transformations
- [ ] 05-03-PLAN.md — Multimodal & Visual Chat
