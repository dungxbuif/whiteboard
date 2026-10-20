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
updated: 2026-10-19
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
| **E01** | Self-hosted Dashboard | Core backend & frontend rewrite | done |
| **E02** | Presentation Mode | Slide navigation and frame tools | done |
| **E03** | Shareable Production | Access control and public link sharing | done |
| **E04** | Real-time Collaboration | Custom /collab/ Yjs sync, presence/follow, WebRTC voice signaling and MCP | in_review |
| **E05** | AI Features | OpenAI-compatible (DeepSeek default) text-to-diagram and visual reasoning | in_review |

## Items

### Collaboration v2 intake — 2026-10-16

| ID | Type | Lane | Status | Scope | Next artifact |
| --- | --- | --- | --- | --- | --- |
| COLLAB-V2 | phase | high-risk | in_review | Deployed on VM100: legacy dashboard style, ACL/invites/public edit, Yjs/custom socket, presence/voice, native MCP and diagram catalog. Final public-origin operational acceptance remains open. | [Programme plans](../superpowers/plans/2026-10-16-collaboration-v2.md), [phase](phases/PHASE-COLLAB-V2.md) |
| COLLAB-V2-RELEASE | release | high-risk | done | Backup/restore, secrets, Linux/amd64 images, migration 13, deploy, denial/log checks, Docker DNS restart/replay, authenticated TURN allocation, automatic reboot recovery, public roles/share/invite, native MCP write/export, mobile/keyboard, public two-peer browser voice/media and Homelab reconciliation pass. | [Release checklist](../operations/RELEASE_CHECKLIST.md), [release note](../releases/2026-10-19-collaboration-v2.md) |
| PROD-REBOOT-RECOVERY | bug | high-risk | done | Controlled VM reboot restored external networks, four healthy services, migration 13 and public health automatically through the recovery unit. | [Release note](../releases/2026-10-19-collaboration-v2.md) |
| RELEASE-SECURITY-REVIEW | release | high-risk | done | Whole-branch correctness/auth/deployment review passed; production audit is clean and all nine development/build findings are classified. | [Dependency review](verification/collaboration-dependencies.md) |
| PLANNING-RECONCILIATION | maintenance | normal | done | Epic/ticket/roadmap/validation/API status now distinguishes shipped local behavior, retired transports and open release gates. | [Requirements](../requirements/REQUIREMENTS.md), [validation](VALIDATION_MATRIX.md) |
| ENTERPRISE-PARITY | phase | high-risk | needs_tickets | Promote approved-design items: comments, screen sharing, live presentation controls, activity UI, version history, libraries, domain capture, SSO/SCIM and retention. | [Pause audit](verification/2026-10-19-release-pause-audit.md) |
| HARNESS-MILESTONE | phase | normal | needs_tickets | Implement the ready Harness CLI hard gates, loop guard and security standard, or record an owner decision removing the milestone. | [Pause audit](verification/2026-10-19-release-pause-audit.md) |
| REQUIREMENTS-TRACE | docs | normal | done | Requirements and user stories now trace approved scope, local proof, production boundaries, enterprise parity and Harness backlog. | [Requirements](../requirements/REQUIREMENTS.md), [user stories](../requirements/USER_STORIES.md) |
| AI-PROVIDER-UAT | release | blocked | blocked | Public-origin chat, text/image-to-diagram and to-code UAT requires a user-provided OpenAI-compatible key; keep it backend-only. | [Pause audit](verification/2026-10-19-release-pause-audit.md) |
| LEGACY-OAUTH-REVOCATION | security | blocked | blocked_p0 | Git history was scrubbed and the rewritten master was pushed without the retired Google OAuth values. Revoke the retired credential in Google Cloud. | [Pause audit](verification/2026-10-19-release-pause-audit.md) |

Design and implementation are complete. Local real-stack acceptance passes17/17, including native S3 image reload/recovery, browser presence/follow, generated two-browser audio, MCP mutations/exports and650/650 durable ACKs at p9513.236ms. The pending-image movement/reopen regressions and Chromium startup endpoint race have deterministic tests and fixes. The final merged-master sequential gate passes: root npm246 tests, frontend93 unit tests and production build, and all Go packages.

Collaboration v2 is running on VM100 from `772d3ea`. The fresh cutover dump was
restored and compared, four service secrets are installed, the frozen
Linux/amd64 images are recorded, migration 13 is clean, and public
health/deep-link/S3/MCP checks pass. A Compose tmpfs parsing defect found during
create is fixed and verified. Controlled reboot, Docker DNS restart/replay and
authenticated TURN allocation, public share/invite/roles, native MCP
write/export, mobile/keyboard and public two-peer browser voice/media canaries
pass. The release evidence is reconciled; AI-provider UAT and retired OAuth
revocation remain external provider actions.
Homelab reconciliation is committed locally as `9722490`; that
repository has no remote.

User confirmed org means workspace. Actual two-network TURN audio passes using the gatewayIP; local hostname DNS still needs correction or the testedIP release setting. See [release checklist](../operations/RELEASE_CHECKLIST.md), [real-stack evidence](verification/collaboration-v2-real-stack.md) and [backup evidence](verification/collaboration-v2-backup.md). The older epic/ticket tables below preserve v1 planning history and do not represent v2 acceptance.

### Epic E01: Self-hosted Dashboard

| Rank | ID | Type | Lane | Title | Priority | Status | Links | Risk Flags | Next Artifact | Notes |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | EXC-001 | ticket | normal | Go Backend Scaffold | High | done | [EXC-001](tickets/phase-01/EXC-001/EXC-001-go-backend-scaffold.md) | | Ticket | Backend build and health smoke verified. |
| 2 | EXC-002 | ticket | normal | Frontend Scaffold Vite+React | High | done | [EXC-002](tickets/phase-01/EXC-002/EXC-002-frontend-scaffold-vite-react.md) | | Ticket | Frontend build and dependency checks verified. |
| 3 | EXC-003 | ticket | high-risk | DB Migrations & Multi-tenant Schema | High | done | [EXC-003](tickets/phase-01/EXC-003/EXC-003-db-migrations-and-multi-tenant-schema.md) | Data model, Migration | Ticket + Detail Design | Tenant schema and migration checks verified. |
| 4 | EXC-004 | ticket | high-risk | Tenant Middleware & Workspace Repository | High | done | [EXC-004](tickets/phase-01/EXC-004/EXC-004-tenant-middleware-and-workspace-repository.md) | Authorization | Ticket + Detail Design | Tenant middleware and repository tests pass. |
| 5 | EXC-005 | ticket | normal | Shadcn UI & Tailwind Setup | Medium | done | [EXC-005](tickets/phase-01/EXC-005/EXC-005-shadcn-ui-and-tailwind-setup.md) | | Ticket | UI dependencies and production build verified. |
| 6 | EXC-006 | ticket | high-risk | Legacy Google OAuth Backend | High | done | [EXC-006](tickets/phase-01/EXC-006/EXC-006-google-oauth-backend.md) | Auth | Ticket + Detail Design | Historical delivery; Google OAuth is retired from the active runtime. |
| 7 | EXC-007 | ticket | normal | Frontend Auth Store | Medium | done | [EXC-007](tickets/phase-01/EXC-007/EXC-007-frontend-auth-store.md) | Auth | Ticket | Store now performs email/password token handoff and authenticated route hydration. |
| 8 | EXC-008 | ticket | normal | Workspace CRUD Full Stack | Medium | done | [EXC-008](tickets/phase-01/EXC-008/EXC-008-workspace-crud-full-stack.md) | Data model | Ticket | Workspace CRUD backend and Zustand UI verified. |
| 9 | EXC-009 | ticket | normal | Folder Backend Materialized Path | Medium | completed | [EXC-009](tickets/phase-01/EXC-009/EXC-009-folder-backend-materialized-path.md) | Data model | Ticket | |
| 10 | EXC-010 | ticket | normal | Folder Frontend Tree+Nav | Medium | completed | [EXC-010](tickets/phase-01/EXC-010/EXC-010-folder-frontend-tree-nav.md) | | Ticket | |
| 11 | EXC-011 | ticket | normal | Scene Backend | Medium | completed | [EXC-011](tickets/phase-01/EXC-011/EXC-011-scene-backend.md) | Data model | Ticket | |
| 12 | EXC-012 | ticket | normal | Excalidraw Integration | Medium | done | [EXC-012](tickets/phase-01/EXC-012/EXC-012-excalidraw-integration.md) | Upstream transitive advisories recorded in ADR-0007 | [Verified Detail Design](tickets/phase-01/EXC-012/EXC-012-design.md) | |
| 13 | EXC-013 | ticket | normal | S3 Assets | Medium | done | [EXC-013](tickets/phase-01/EXC-013/EXC-013-s3-assets.md) | External system | [Detail Design](tickets/phase-01/EXC-013/EXC-013-design.md) | |
| 14 | EXC-014 | ticket | normal | Search + Starred/Trash | Low | done | [EXC-014](tickets/phase-01/EXC-014/EXC-014-search-starred-trash.md) | | [Detail Design](tickets/phase-01/EXC-014/EXC-014-design.md) | Workspace-scoped search plus Starred and Trash dashboard views implemented. |
| 34 | EXC-034 | ticket | high-risk | Replace Temporary Admin Auth | High | done | [EXC-034](tickets/phase-01/EXC-034/EXC-034-replace-temporary-admin-auth-with-google-oauth.md) | Auth, Audit/security/privacy | [Detail Design](tickets/phase-01/EXC-034/EXC-034-design.md) | Historical OAuth work was superseded by verified email/password login, userctl, invite registration and default-workspace provisioning. Retired credential revocation remains operational. |

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
| 24 | EXC-024 | ticket | high-risk | Coturn & Docker Setup | Medium | done | [EXC-024](tickets/phase-01/EXC-024/EXC-024-coturn-and-docker-setup.md) | Deployment/runtime, External system | [Collaboration v2 release](phases/PHASE-COLLAB-V2.md) | Compose support and tested two-network TURN audio exist; production must use the proven endpoint until split DNS is fixed. |
| 25 | EXC-025 | ticket | high-risk | Historical Socket.IO Backend + Frontend Client | Medium | done | [EXC-025](tickets/phase-01/EXC-025/EXC-025-socket.io-backend---frontend-client.md) | Public contract | [ADR 0010](../decisions/0010-durable-yjs-writer.md) | Historical implementation was removed and superseded by the authenticated custom /collab/ WebSocket. |
| 26 | EXC-026 | ticket | normal | Collaborative Cursors & Presence | Medium | done | [EXC-026](tickets/phase-01/EXC-026/EXC-026-collaborative-cursors-and-presence.md) | | [Runtime plan](../superpowers/plans/2026-10-16-collaboration-runtime.md) | Native cursors, roster, selections and follow mode pass real-stack browser proof. |
| 27 | EXC-027 | ticket | high-risk | Historical WebRTC DataChannel Scene Sync | Medium | done | [EXC-027](tickets/phase-01/EXC-027/EXC-027-webrtc-datachannel---scene-sync.md) | Existing behavior | [ADR 0010](../decisions/0010-durable-yjs-writer.md) | Superseded by durable Yjs sync over /collab/; WebRTC remains only for voice media. |
| 28 | EXC-028 | ticket | normal | Voice Calling | Medium | done | [EXC-028](tickets/phase-01/EXC-028/EXC-028-voice-calling.md) | External system | [Runtime plan](../superpowers/plans/2026-10-16-collaboration-runtime.md) | Generated two-browser audio, TURN, mute/leave, denial and continued editing pass locally. |

### Epic E05: AI Features

| Rank | ID | Type | Lane | Title | Priority | Status | Links | Risk Flags | Next Artifact | Notes |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 29 | EXC-029 | ticket | normal | AI Domain & OpenAI-Compatible Repository | Medium | done | [EXC-029](tickets/phase-01/EXC-029/EXC-029-ai-domain-and-gemini-repository.md) | External system | [ADR 0008](../decisions/0008-openai-compatible-ai-provider.md) | Repository and provider-neutral contracts are tested. |
| 30 | EXC-030 | ticket | normal | AI Proxy Endpoint & FE Client | Medium | done | [EXC-030](tickets/phase-01/EXC-030/EXC-030-ai-proxy-endpoint-and-fe-client.md) | Public contract | [Phase 5](phases/PHASE-5.md) | Authenticated backend proxy and frontend client are implemented; live provider UAT remains blocked on a key. |
| 31 | EXC-031 | ticket | normal | Text-to-Diagram & Diagram-to-Code (Backend) | Medium | done | [EXC-031](tickets/phase-01/EXC-031/EXC-031-text-to-diagram-and-diagram-to-code-backend.md) | External system | [Phase 5](phases/PHASE-5.md) | Output cleaning and scene validation tests pass. |
| 32 | EXC-032 | ticket | normal | AI Command Panel & Code Preview (Frontend) | Medium | done | [EXC-032](tickets/phase-01/EXC-032/EXC-032-ai-command-panel-and-code-preview-frontend.md) | | [Phase 5](phases/PHASE-5.md) | In-canvas command panel and code preview build and test successfully. |
| 33 | EXC-033 | ticket | normal | Multimodal: Image-to-Diagram & Visual Chat | Medium | done | [EXC-033](tickets/phase-01/EXC-033/EXC-033-multimodal-image-to-diagram-and-visual-chat.md) | External system | [Phase 5](phases/PHASE-5.md) | Image validation and attachment UI are implemented; live provider UAT remains blocked on a key. |

### Harness Framework

| Rank | ID | Type | Lane | Title | Priority | Status | Links | Risk Flags | Next Artifact | Notes |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| - | BL-001 | framework | normal | Finalize first project-specific standards | Medium | done | [Standards](../standards/README.md) | Standards change | [Quality bar](../standards/QUALITY_BAR.md) | Project standards now cover quality, validation, debugging, security, docs and lifecycle rules. Harness enforcement remains a separate milestone. |
