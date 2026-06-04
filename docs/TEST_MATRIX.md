# Test Matrix

This file maps product behavior to proof.

No product behavior has been defined or implemented yet. Do not mark a row
implemented until tests or validation evidence exist.

## Status Values

| Status | Meaning |
| --- | --- |
| planned | Accepted as intended behavior, not implemented |
| in_progress | Actively being built |
| implemented | Implemented and proof exists |
| changed | Contract changed after earlier implementation |
| retired | No longer part of the product contract |

## Matrix

| Story | Contract | Unit | Integration | E2E | Platform | Status | Evidence |
| --- | --- | --- | --- | --- | --- | --- | --- |
| EXC-001 | Go Backend Scaffold | yes | yes | no | no | implemented | [EXC-001.md](stories/phase-01/EXC-001/EXC-001-go-backend-scaffold.md) |
| EXC-002 | Frontend Scaffold Vite+React | yes | no | no | no | implemented | [EXC-002.md](stories/phase-01/EXC-002/EXC-002-frontend-scaffold-vite-react.md) |
| EXC-003 | DB Migrations & Multi-tenant Schema | no | no | no | no | planned | none |
| EXC-004 | Tenant Middleware & Workspace Repository | no | no | no | no | planned | none |
| EXC-005 | Shadcn UI & Tailwind Setup | yes | no | no | no | implemented | [EXC-005.md](stories/phase-01/EXC-005/EXC-005-shadcn-ui-and-tailwind-setup.md) |
| EXC-006 | Google OAuth Backend | no | no | no | no | planned | none |
| EXC-007 | Frontend Auth Store | no | no | no | no | planned | none |
| EXC-008 | Workspace CRUD Full Stack | no | no | no | no | planned | none |
| EXC-009 | Folder Backend Materialized Path | no | no | no | no | planned | none |
| EXC-010 | Folder Frontend Tree+Nav | no | no | no | no | planned | none |
| EXC-011 | Scene Backend | no | no | no | no | planned | none |
| EXC-012 | Excalidraw Integration | no | no | no | no | planned | none |
| EXC-013 | S3 Assets | no | no | no | no | planned | none |
| EXC-014 | Search + Starred/Trash | no | no | no | no | planned | none |
| EXC-015 | Frame Persistence & Scene Store Extension | no | no | no | no | planned | none |
| EXC-016 | Frame Tool Activation | no | no | no | no | planned | none |
| EXC-017 | Presentation Engine & Controls | no | no | no | no | planned | none |
| EXC-018 | Slide Navigator & Keyboard Support | no | no | no | no | planned | none |
| EXC-019 | Sharing DB Schema & Domain | no | no | no | no | planned | none |
| EXC-020 | Sharing Backend | no | no | no | no | planned | none |
| EXC-021 | Public API & Share Dialog UI | no | no | no | no | planned | none |
| EXC-022 | Password Protection | no | no | no | no | planned | none |
| EXC-023 | Sharing Integration Tests | no | no | no | no | planned | none |
| EXC-024 | Coturn & Docker Setup | no | no | no | no | planned | none |
| EXC-025 | Socket.io Backend + Frontend Client | no | no | no | no | planned | none |
| EXC-026 | Collaborative Cursors & Presence | no | no | no | no | planned | none |
| EXC-027 | WebRTC DataChannel + Scene Sync | no | no | no | no | planned | none |
| EXC-028 | Voice Calling | no | no | no | no | planned | none |
| EXC-029 | AI Domain & Gemini Repository | no | no | no | no | planned | none |
| EXC-030 | AI Proxy Endpoint & FE Client | no | no | no | no | planned | none |
| EXC-031 | Text-to-Diagram & Diagram-to-Code (Backend) | no | no | no | no | planned | none |
| EXC-032 | AI Command Panel & Code Preview (Frontend) | no | no | no | no | planned | none |
| EXC-033 | Multimodal: Image-to-Diagram & Visual Chat | no | no | no | no | planned | none |

## Evidence Rules

- Unit proof covers pure domain and application rules.
- Integration proof covers backend enforcement, data integrity, provider
  behavior, jobs, or service contracts.
- E2E proof covers user-visible browser flows.
- Platform proof covers only shell, deployment, mobile, desktop, or runtime
  behavior that cannot be proven in lower layers.
- A story can be implemented without every proof column if the story packet
  explains why.
