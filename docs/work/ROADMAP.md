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
updated: 2026-10-19
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
| M1 | Phase 1: Self-hosted Dashboard Rewrite | done | [PHASE-1.md](phases/PHASE-1.md) | 11 plans (01-01 to 01-11) |
| M2 | Phase 2: Presentation Mode | done | [PHASE-2-PRODUCT.md](phases/PHASE-2-PRODUCT.md) | 3 plans (02-01 to 02-03) |
| M3 | Phase 3: Shareable Production | done | [PHASE-3.md](phases/PHASE-3.md) | 3 plans (03-01 to 03-03) |
| M4 | Phase 4: Real-time Collaboration & Calling | in_review | [PHASE-COLLAB-V2.md](phases/PHASE-COLLAB-V2.md) | Local custom WebSocket/Yjs/MCP acceptance passed; VM100 release pending |
| M5 | Phase 5: AI Features (OpenAI-Compatible API) | in_review | [PHASE-5.md](phases/PHASE-5.md) | Implementation verified; public live-provider UAT blocked on key |
| ME | Enterprise parity | ready | [Requirements](../requirements/REQUIREMENTS.md#approved-product-backlog) | Comments, screen sharing, live presentation/activity, versioning, libraries, domain/SSO/SCIM/retention need tickets |
| MH | Harness: Develop Harness CLI (Hard Enforcement) | ready | [PHASE-2.md](phases/PHASE-2.md) | Create implementation tickets or record an owner decision removing the milestone |

## Phase 1 Plans

The approved [Collaboration v2 phase](phases/PHASE-COLLAB-V2.md) and
[implementation task index](../superpowers/plans/2026-10-16-collaboration-v2.md)
now supersede the historical Phase 4 transport plans. Local implementation and
real-stack acceptance pass; production remains v1 until the release gate closes.
No milestone status implies enterprise feature parity.

- [x] 01-01-PLAN.md — Backend & Infrastructure Scaffold
- [x] 01-02-PLAN.md — Multi-tenancy & Database
- [x] 01-03-PLAN.md — Frontend Core & Shadcn Port
- [x] 01-04-PLAN.md — Identity & Session Management (active runtime is email/password)
- [x] 01-05-PLAN.md — Workspace Management
- [x] 01-06-PLAN.md — Folder Hierarchy (Backend)
- [x] 01-07-PLAN.md — Folder Tree & Navigation (Frontend)
- [x] 01-08-PLAN.md — Scene Persistence & Features (BE)
- [x] 01-09-PLAN.md — Excalidraw Integration (Frontend)
- [x] 01-10-PLAN.md — S3 Asset Management
- [x] 01-11-PLAN.md — Search & Refinement

## Phase 2 Plans

- [x] 02-01-PLAN.md — Frame Persistence & Management
- [x] 02-02-PLAN.md — Presentation Engine & Navigation
- [x] 02-03-PLAN.md — Navigator & Final Polish

## Phase 3 Plans

- [x] 03-01-PLAN.md — Sharing API and DB Schema
- [x] 03-02-PLAN.md — Public UI and Access Control
- [x] 03-03-PLAN.md — Legacy Parity and Testing

## Phase 4 Plans

- [x] 04-01-PLAN.md — Historical Coturn/Socket.IO infrastructure; Socket.IO is retired
- [x] 04-02-PLAN.md — Historical collaboration baseline; superseded by custom /collab/ and durable Yjs
- [x] 04-03-PLAN.md — Voice calling baseline; current signaling shares /collab/
- [x] Collaboration v2 local implementation and 17/17 real-stack acceptance
- [ ] VM100 cutover and public-origin release acceptance

## Phase 5 Plans

- [x] 05-01-PLAN.md — Foundation & OpenAI-Compatible Proxy
- [x] 05-02-PLAN.md — Text & Code Transformations
- [x] 05-03-PLAN.md — Multimodal & Visual Chat
- [ ] Public-origin live-provider UAT with a user-supplied backend-only key
