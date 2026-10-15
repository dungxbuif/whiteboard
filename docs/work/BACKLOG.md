---
artifact_type: backlog
id: BACKLOG
status: active
owner: shared
human_fields:
  - priority_override
  - rank
  - blocker_decisions
ai_fields:
  - risk_flags
  - lane_recommendation
  - next_artifact
  - notes
shared_fields:
  - queue_items
  - status
updated: 2026-06-11
---

# Backlog

## Field Ownership

- Human owns priority overrides, rank changes, and blocker decisions.
- AI may recommend lane, risk flags, next artifact, and notes.
- Shared fields include queue rows and item status.

Use this file as the runtime work queue.

The backlog decides what should be worked on next. It does not replace tickets, bugs, requirements, phases, detail designs, or verification artifacts.

## Queue Rules

- Backlog order MAY change at runtime based on priority, severity, dependency, or new information.
- Bugs MAY preempt feature tickets when severity or user impact is higher.
- New requirements MAY enter the backlog before they become requirements docs, phases, or tickets.
- Maintenance and framework work MAY appear in the same queue as product work.
- The active queue focus MUST be reflected in `docs/CONTEXT.md`.
- Do not process all tickets before bugs by default; process the highest-priority queue item that is ready and appropriately scoped.

## Intake Rules

- Keep each item short and actionable.
- Link to source context, requirement, phase, ticket, bug, or decision when available.
- Classify each item by lane: `tiny`, `normal`, `high-risk`, or `blocked`.
- Promote non-tiny backlog items into a requirement slice, phase, ticket, or bug before execution.
- Do not execute directly from backlog unless the item is clearly tiny and records a small-task exemption.
- High-risk items require detail design approval before implementation.
- Blocked items must record the missing decision, dependency, or input.

## Lane Guide

| Lane | Use When | Required Next Artifact |
| --- | --- | --- |
| tiny | Low-risk docs/copy/naming/narrow edits with no contract or runtime impact | Backlog row may be enough with small-task exemption |
| normal | Bounded story-sized work, bug fix, or maintenance task | Ticket or bug |
| high-risk | Auth, authorization, data, security, public contract, external provider, migration, major dependency, or multi-domain impact | Ticket/bug plus detail design and approval |
| blocked | Work cannot proceed because input, decision, dependency, or environment is missing | Blocker note and owner |

## Risk Flags

Mark risk flags in the `Risk Flags` column when relevant:

- Auth
- Authorization
- Data model
- Migration/data loss
- Audit/security/privacy
- External system/provider
- Public contract/API
- Existing behavior
- Weak proof
- Multi-domain
- Deployment/runtime
- Standards change

## Candidate Epics

| Epic | Title | Description | Status |
| --- | --- | --- | --- |
| **E01** | Self-hosted Dashboard | Core backend & frontend rewrite | in_progress |
| **E02** | Presentation Mode | Slide navigation and frame tools | planned |
| **E03** | Shareable Production | Access control and public link sharing | planned |
| **E04** | Real-time Collaboration | Real-time presence, WebRTC sync & voice calls | planned |
| **E05** | AI Features | Gemini API text-to-diagram and visual reasoning | planned |

## Items

### Epic E01: Self-hosted Dashboard

| Rank | ID | Type | Lane | Title | Priority | Status | Links | Risk Flags | Next Artifact | Notes |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | EXC-001 | ticket | normal | Go Backend Scaffold | High | planned | [EXC-001](tickets/phase-01/EXC-001/EXC-001-go-backend-scaffold.md) | | Ticket | |
| 2 | EXC-002 | ticket | normal | Frontend Scaffold Vite+React | High | planned | [EXC-002](tickets/phase-01/EXC-002/EXC-002-frontend-scaffold-vite-react.md) | | Ticket | |
| 3 | EXC-003 | ticket | high-risk | DB Migrations & Multi-tenant Schema | High | planned | [EXC-003](tickets/phase-01/EXC-003/EXC-003-db-migrations-and-multi-tenant-schema.md) | Data model, Migration | Ticket + Detail Design | |
| 4 | EXC-004 | ticket | high-risk | Tenant Middleware & Workspace Repository | High | planned | [EXC-004](tickets/phase-01/EXC-004/EXC-004-tenant-middleware-and-workspace-repository.md) | Authorization | Ticket + Detail Design | |
| 5 | EXC-005 | ticket | normal | Shadcn UI & Tailwind Setup | Medium | planned | [EXC-005](tickets/phase-01/EXC-005/EXC-005-shadcn-ui-and-tailwind-setup.md) | | Ticket | |
| 6 | EXC-006 | ticket | high-risk | Google OAuth Backend | High | done | [EXC-006](tickets/phase-01/EXC-006/EXC-006-google-oauth-backend.md) | Auth | Ticket + Detail Design | Active OAuth implementation verified through EXC-034. |
| 7 | EXC-007 | ticket | normal | Frontend Auth Store | Medium | done | [EXC-007](tickets/phase-01/EXC-007/EXC-007-frontend-auth-store.md) | Auth | Ticket | OAuth callback token handoff verified through EXC-034. |
| 8 | EXC-008 | ticket | normal | Workspace CRUD Full Stack | Medium | planned | [EXC-008](tickets/phase-01/EXC-008/EXC-008-workspace-crud-full-stack.md) | Data model | Ticket | |
| 9 | EXC-009 | ticket | normal | Folder Backend Materialized Path | Medium | completed | [EXC-009](tickets/phase-01/EXC-009/EXC-009-folder-backend-materialized-path.md) | Data model | Ticket | |
| 10 | EXC-010 | ticket | normal | Folder Frontend Tree+Nav | Medium | completed | [EXC-010](tickets/phase-01/EXC-010/EXC-010-folder-frontend-tree-nav.md) | | Ticket | |
| 11 | EXC-011 | ticket | normal | Scene Backend | Medium | completed | [EXC-011](tickets/phase-01/EXC-011/EXC-011-scene-backend.md) | Data model | Ticket | |
| 12 | EXC-012 | ticket | normal | Excalidraw Integration | Medium | done | [EXC-012](tickets/phase-01/EXC-012/EXC-012-excalidraw-integration.md) | Upstream transitive advisories recorded in ADR-0007 | [Verified Detail Design](tickets/phase-01/EXC-012/EXC-012-design.md) | |
| 13 | EXC-013 | ticket | normal | S3 Assets | Medium | done | [EXC-013](tickets/phase-01/EXC-013/EXC-013-s3-assets.md) | External system | [Detail Design](tickets/phase-01/EXC-013/EXC-013-design.md) | |
| 14 | EXC-014 | ticket | normal | Search + Starred/Trash | Low | done | [EXC-014](tickets/phase-01/EXC-014/EXC-014-search-starred-trash.md) | | [Detail Design](tickets/phase-01/EXC-014/EXC-014-design.md) | Workspace-scoped search plus Starred and Trash dashboard views implemented. |
| 34 | EXC-034 | ticket | high-risk | Replace Temporary Admin Auth With Google OAuth | High | done | [EXC-034](tickets/phase-01/EXC-034/EXC-034-replace-temporary-admin-auth-with-google-oauth.md) | Auth, Audit/security/privacy, External system/provider | [Detail Design](tickets/phase-01/EXC-034/EXC-034-design.md) | Code and automated verification complete; rotate deployment credentials before release. |

### Epic E02: Presentation Mode

| Rank | ID | Type | Lane | Title | Priority | Status | Links | Risk Flags | Next Artifact | Notes |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 15 | EXC-015 | ticket | normal | Frame Persistence & Scene Store Extension | Medium | done | [EXC-015](tickets/phase-01/EXC-015/EXC-015-frame-persistence-and-scene-store-extension.md) | Data model | [Detail Design](tickets/phase-01/EXC-015/EXC-015-design.md) | Frame order is persisted inside scene JSON. |
| 16 | EXC-016 | ticket | normal | Frame Tool Activation | Medium | done | [EXC-016](tickets/phase-01/EXC-016/EXC-016-frame-tool-activation.md) | | [Detail Design](tickets/phase-01/EXC-016/EXC-016-design.md) | Native frame tool changes sync automatically. |
| 17 | EXC-017 | ticket | normal | Presentation Engine & Controls | Medium | done | [EXC-017](tickets/phase-01/EXC-017/EXC-017-presentation-engine-and-controls.md) | | [Detail Design](tickets/phase-01/EXC-017/EXC-017-design.md) | Local presentation mode with animated frame navigation. |
| 18 | EXC-018 | ticket | normal | Slide Navigator & Keyboard Support | Medium | done | [EXC-018](tickets/phase-01/EXC-018/EXC-018-slide-navigator-and-keyboard-support.md) | | [Detail Design](tickets/phase-01/EXC-018/EXC-018-design.md) | Navigator reorder controls and keyboard navigation implemented. |

### Epic E03: Shareable Production

| Rank | ID | Type | Lane | Title | Priority | Status | Links | Risk Flags | Next Artifact | Notes |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 19 | EXC-019 | ticket | high-risk | Sharing DB Schema & Domain | Medium | done | [EXC-019](tickets/phase-01/EXC-019/EXC-019-sharing-db-schema-and-domain.md) | Data model, Public contract | [Detail Design](tickets/phase-01/EXC-019/EXC-019-design.md) | Permission migration and domain contract implemented. |
| 20 | EXC-020 | ticket | high-risk | Sharing Backend | Medium | done | [EXC-020](tickets/phase-01/EXC-020/EXC-020-sharing-backend.md) | Public contract, Authorization | [Detail Design](tickets/phase-01/EXC-020/EXC-020-design.md) | Authenticated share/revoke and public access usecases implemented. |
| 21 | EXC-021 | ticket | normal | Public API & Share Dialog UI | Medium | done | [EXC-021](tickets/phase-01/EXC-021/EXC-021-public-api-and-share-dialog-ui.md) | Public contract | [Detail Design](tickets/phase-01/EXC-021/EXC-021-design.md) | Public API, share dialog, and read-only viewer implemented. |
| 22 | EXC-022 | ticket | high-risk | Password Protection | Medium | done | [EXC-022](tickets/phase-01/EXC-022/EXC-022-password-protection.md) | Auth, Audit/security | [Detail Design](tickets/phase-01/EXC-022/EXC-022-design.md) | bcrypt password challenge implemented. |
| 23 | EXC-023 | ticket | normal | Sharing Integration Tests | Low | done | [EXC-023](tickets/phase-01/EXC-023/EXC-023-sharing-integration-tests.md) | | [Detail Design](tickets/phase-01/EXC-023/EXC-023-design.md) | Backend and browser sharing verification implemented. |

### Epic E04: Real-time Collaboration

| Rank | ID | Type | Lane | Title | Priority | Status | Links | Risk Flags | Next Artifact | Notes |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 24 | EXC-024 | ticket | high-risk | Coturn & Docker Setup | Medium | planned | [EXC-024](tickets/phase-01/EXC-024/EXC-024-coturn-and-docker-setup.md) | Deployment/runtime, External system | Ticket + Detail Design | |
| 25 | EXC-025 | ticket | high-risk | Socket.io Backend + Frontend Client | Medium | planned | [EXC-025](tickets/phase-01/EXC-025/EXC-025-socket.io-backend---frontend-client.md) | Public contract | Ticket + Detail Design | |
| 26 | EXC-026 | ticket | normal | Collaborative Cursors & Presence | Medium | planned | [EXC-026](tickets/phase-01/EXC-026/EXC-026-collaborative-cursors-and-presence.md) | | Ticket | |
| 27 | EXC-027 | ticket | high-risk | WebRTC DataChannel + Scene Sync | Medium | planned | [EXC-027](tickets/phase-01/EXC-027/EXC-027-webrtc-datachannel---scene-sync.md) | External system | Ticket + Detail Design | |
| 28 | EXC-028 | ticket | normal | Voice Calling | Medium | planned | [EXC-028](tickets/phase-01/EXC-028/EXC-028-voice-calling.md) | External system | Ticket | |

### Epic E05: AI Features

| Rank | ID | Type | Lane | Title | Priority | Status | Links | Risk Flags | Next Artifact | Notes |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 29 | EXC-029 | ticket | normal | AI Domain & Gemini Repository | Medium | planned | [EXC-029](tickets/phase-01/EXC-029/EXC-029-ai-domain-and-gemini-repository.md) | External system | Ticket | |
| 30 | EXC-030 | ticket | normal | AI Proxy Endpoint & FE Client | Medium | planned | [EXC-030](tickets/phase-01/EXC-030/EXC-030-ai-proxy-endpoint-and-fe-client.md) | Public contract | Ticket | |
| 31 | EXC-031 | ticket | normal | Text-to-Diagram & Diagram-to-Code (Backend) | Medium | planned | [EXC-031](tickets/phase-01/EXC-031/EXC-031-text-to-diagram-and-diagram-to-code-backend.md) | External system | Ticket | |
| 32 | EXC-032 | ticket | normal | AI Command Panel & Code Preview (Frontend) | Medium | planned | [EXC-032](tickets/phase-01/EXC-032/EXC-032-ai-command-panel-and-code-preview-frontend.md) | | Ticket | |
| 33 | EXC-033 | ticket | normal | Multimodal: Image-to-Diagram & Visual Chat | Medium | planned | [EXC-033](tickets/phase-01/EXC-033/EXC-033-multimodal-image-to-diagram-and-visual-chat.md) | External system | Ticket | |

### Harness Framework

| Rank | ID | Type | Lane | Title | Priority | Status | Links | Risk Flags | Next Artifact | Notes |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| - | BL-001 | framework | normal | Finalize first project-specific standards | Medium | open | `docs/standards/` | Standards change | TBD | Add human-maintained rules in `docs/standards/`. |
