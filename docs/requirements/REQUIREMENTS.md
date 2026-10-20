---
artifact_type: requirements_index
id: REQUIREMENTS
status: active
owner: shared
updated: 2026-10-19
human_fields: [priority, acceptance, requirement_source]
ai_fields: [requirement_rows, status_updates, trace_links]
shared_fields: [functional_requirements, non_functional_requirements]
---

# Requirements

## Field Ownership

- Human owns priority, acceptance, and requirement source.
- AI maintains rows, trace links, and evidence-backed status updates.

Status follows the repository lifecycle: `ready`, `in_progress`, `blocked`,
`in_review`, `verified`, or `done`. `Verified` below means implementation
and local evidence exist. It does not imply that Collaboration v2 is deployed to
VM100; that boundary is tracked by `V2-REQ-08`.

## Functional Requirements

| ID | Requirement | Priority | Source | Status | Evidence or release boundary |
| --- | --- | --- | --- | --- | --- |
| REQ-01 | The backend uses Go Clean Architecture and the web client uses React, Vite, Zustand, and the project UI system. | P0 | [Master spec](SPEC.md) | verified | Backend and frontend suites/build pass; [validation](../work/VALIDATION_MATRIX.md). |
| REQ-02 | Workspace-scoped data is isolated in PostgreSQL and every protected request resolves current workspace authority. | P0 | [Master spec](SPEC.md), [ADR 0009](../decisions/0009-collaboration-access.md) | verified | Repository, lock/race, alias, denial, and migration tests are recorded in [Collaboration v2 evidence](../work/verification/collaboration-v2.md). |
| REQ-03 | Scene binary assets use S3-compatible storage with scoped, short-lived upload/view capabilities. | P0 | [Master spec](SPEC.md), [asset guide](../user/scene-assets.md) | verified | Native paste, signed PUT/GET, reload, renewal, failed-upload retention and byte equality pass locally; production `whiteboard` bucket write/read/delete also passes. |
| REQ-04 | Users authenticate with email/password; operator provisioning and invite registration create a default `My Workspace` transactionally. Google OAuth is retired from the runtime. | P0 | User decision; [API](../architecture/API.md) | verified | Password login, `userctl`, invitation registration, default-workspace idempotency, browser login, and negative cases pass. Retired OAuth credential revocation remains operational backlog. |
| REQ-05 | Users can create, list, select, and deep-link workspaces through UUIDs or public aliases without aliases granting authority. | P0 | [Master spec](SPEC.md) | verified | Workspace repository/browser route evidence is in the [validation matrix](../work/VALIDATION_MATRIX.md). |
| REQ-06 | Users can create and navigate nested folders represented by application-managed materialized paths. | P1 | [Master spec](SPEC.md) | verified | EXC-009/010 and dashboard browser flows. |
| REQ-07 | Scenes and folders support soft-delete Trash views and restore while enforcing workspace/role rules. | P1 | [Master spec](SPEC.md) | verified | Repository/usecase/browser coverage; owner-only Trash behavior is documented in [API](../architecture/API.md). |
| REQ-08 | Users can star scenes/folders and open a workspace-scoped Starred view. | P1 | [Master spec](SPEC.md) | verified | EXC-014 unit/browser evidence. |
| REQ-09 | Users can search workspace scene/folder names and navigate a result to its stable route. | P1 | [Master spec](SPEC.md) | verified | EXC-014 and Collaboration v2 dashboard route evidence. |
| REQ-10 | The native Excalidraw editor persists scene content and stable scene routes; promoted Collaboration v2 scenes write through the durable Yjs writer rather than legacy HTTP content PATCH. | P0 | [Master spec](SPEC.md), [ADR 0010](../decisions/0010-durable-yjs-writer.md) | verified | Scene/domain tests, canvas browser tests, and real-stack durable writer acceptance pass locally. |
| P2-REQ-01 | Excalidraw frames and their presentation order persist as scene data. | P1 | Phase 2 approved spec | verified | EXC-015/016 unit and browser proof. |
| P2-REQ-02 | Presentation mode focuses the selected frame and provides visible next/previous controls. | P1 | Phase 2 approved spec | verified | EXC-017 browser proof. |
| P2-REQ-03 | Users can reorder/jump to slides and navigate them from the keyboard. | P1 | Phase 2 approved spec | verified | EXC-018 browser proof. |
| P3-REQ-01 | Owners can create and revoke independently scoped public `view` and `edit` links for one scene. | P0 | Phase 3 plus [Collaboration v2 design](../superpowers/specs/2026-10-15-collaboration-v2-design.md) | verified | Legacy and independent-link repository/browser cases pass locally. |
| P3-REQ-02 | A public link opens at `/p/{token}` without an account; edit links require a display name and preserve a guest identity only for that link/session. | P0 | Collaboration v2 approved design | verified | Public view/edit/password/session browser cases pass in the real-stack suite. |
| P3-REQ-03 | Public links may require a password; password input is accepted only in JSON request bodies and stored only as a hash. | P0 | Phase 3 approved spec | verified | bcrypt, log-redaction, query stripping, and browser challenge cases pass. |
| P3-REQ-04 | Revocation, expiry, password change, or authority loss invalidates affected sessions and live room access without affecting unrelated grants. | P0 | Collaboration v2 approved design | verified | PostgreSQL races and real-stack connected-session denial pass locally. |
| V2-REQ-01 | Workspace roles are `owner`, `editor`, and `viewer`; owners manage members, invitations, scene overrides, public links, and agent keys. | P0 | [Collaboration v2 design](../superpowers/specs/2026-10-15-collaboration-v2-design.md) | verified | A1–A4 and C2 evidence; production rollout is tracked separately. |
| V2-REQ-02 | Copyable invitation links are email-bound, single-use, valid for seven days, and can register a new password account atomically. | P0 | Collaboration v2 approved design | verified | Fifteen invitation tests plus browser real-stack acceptance. |
| V2-REQ-03 | One authenticated custom `/collab/` WebSocket carries Yjs sync, durable acknowledgements, awareness, presence, follow, and voice signaling; Socket.IO is retired. | P0 | [ADR 0010](../decisions/0010-durable-yjs-writer.md) | verified | Runtime suites and 17/17 real-stack acceptance pass; VM100 is deployed and deep-link save/reload passes. Runtime restart and deployed voice remain release canaries. |
| V2-REQ-04 | Offline edits are scoped in IndexedDB, replay only after state-vector resynchronization, and survive browser/runtime restart without losing acknowledged updates. | P0 | Collaboration v2 approved design | verified | Offline, downgrade, commit-before-ACK, epoch-replacement, and recovery cases pass locally. |
| V2-REQ-05 | Presence exposes native cursors, selections, roster, and follow viewport; voice stays peer-to-peer and admits at most ten active voice participants. | P1 | Collaboration v2 approved design | verified | Native two-browser presence/follow/audio, mute/leave, denial, and continued editing pass locally. |
| V2-REQ-06 | Agents use scoped, revocable keys and MCP HTTP/stdio transports to discover workspaces/scenes, create scenes, generate or patch native diagrams, inspect activity, and export current state through the same authority and durable writer as browsers. | P0 | [ADR 0011](../decisions/0011-native-mcp.md) | verified | MCP discovery, scope denial, mutation, idempotency/conflict, human-edit preservation, export, and revoke tests pass. |
| V2-REQ-07 | The diagram catalog produces editable native Excalidraw elements for 19 typed presets and the documented restricted Mermaid subset. | P1 | Collaboration v2 approved design | verified | Forty-one compiler/validation tests and renderer integration evidence. |
| V2-REQ-08 | A release is complete only after reboot recovery, backup/restore proof, distinct service secrets, frozen Linux/amd64 images, migrations 7–13, VM100 deployment, public-origin authorization/S3/MCP/WebSocket/voice canaries, dependency/security review, and docs reconciliation. | P0 | [Release checklist](../operations/RELEASE_CHECKLIST.md) | in_progress | Backup/restore, secrets, images, migration13, deploy, review, deep-link/S3/MCP/internal denial/log, Docker DNS restart, authenticated TURN, controlled reboot and Homelab pass. Public share/invite/roles, native MCP write/export, browser voice/media and mobile/keyboard remain. |
| P5-REQ-01 | The backend exposes OpenAI-compatible chat and text/image-to-diagram operations with DeepSeek-compatible defaults and validates model output before returning it. | P1 | [ADR 0008](../decisions/0008-openai-compatible-ai-provider.md) | verified | Repository/handler validation and output-cleaning tests pass. |
| P5-REQ-02 | The canvas exposes an AI command panel and previewable diagram-to-code result without exposing provider keys to the browser. | P1 | Phase 5 approved spec | verified | Frontend tests/build and backend-only configuration contract pass. |
| P5-REQ-03 | Image-to-diagram and visual chat accept bounded validated image input. | P1 | Phase 5 approved spec | verified | Base64/media validation tests and attachment UI evidence. |
| P5-REQ-04 | Public-origin AI chat, diagram generation, image reasoning, and code generation must pass against the configured live provider before release sign-off. | P1 | Release gate | blocked | Requires the user-provided OpenAI-compatible API key; the key remains backend-only. |

## Approved Product Backlog

These requirements are approved direction, not delivered Collaboration v2 scope.
They remain visible so local v2 completion cannot be mistaken for enterprise
feature parity.

The [enterprise proposed detail design](../superpowers/specs/2026-10-20-enterprise-parity-design.md)
and [22-task programme](../superpowers/plans/2026-10-20-enterprise-parity.md)
now map every ENT requirement to code boundaries, API/schema proposals and
verification. The [phase](../work/phases/PHASE-ENTERPRISE.md) remains draft for
contract/default review; `ready` below records approved direction, not shipped
implementation or approval of new schema/security decisions. Canvas controls
must live inside/floating over the full canvas per the user on 2026-10-20.

| ID | Requirement | Priority | Source | Status | Next artifact |
| --- | --- | --- | --- | --- | --- |
| ENT-P1-01 | Durable scene comments and guest comments with element anchors, moderation, and notification UI. | P1 | Collaboration v2 parity decision | ready | [Draft C1/C2](../superpowers/plans/2026-10-20-enterprise-engagement.md). |
| ENT-P1-02 | Screen sharing with explicit media permissions and bounded participant admission. | P1 | Collaboration v2 parity decision | ready | [Draft M1](../superpowers/plans/2026-10-20-enterprise-live.md). |
| ENT-P1-03 | Live presentation leadership, spotlight, waiting-room admission, reactions, and view locking. | P1 | Collaboration v2 parity decision | ready | [Draft M2/M3](../superpowers/plans/2026-10-20-enterprise-live.md). |
| ENT-P1-04 | A user-facing activity feed backed by the audit stream and an explicit retention policy. | P1 | Collaboration v2 parity decision | ready | [Draft C3](../superpowers/plans/2026-10-20-enterprise-engagement.md), [R1](../superpowers/plans/2026-10-20-enterprise-identity.md). |
| ENT-P2-01 | User-visible scene version history, labels, diff, and restore. | P2 | Collaboration v2 parity decision | ready | [Draft V1/V2](../superpowers/plans/2026-10-20-enterprise-content.md). |
| ENT-P2-02 | Workspace libraries and collections with sharing and search contracts. | P2 | Collaboration v2 parity decision | ready | [Draft L1/L2](../superpowers/plans/2026-10-20-enterprise-content.md). |
| ENT-P2-03 | Domain capture plus enterprise SSO/SCIM and retention controls. | P2 | Collaboration v2 parity decision | ready | [Draft I1–I3/R1–R2](../superpowers/plans/2026-10-20-enterprise-identity.md). |
| HARNESS-REQ-01 | Harness CLI hard gates, loop guard, and security standard are enforced or explicitly removed by the owner. | P2 | [Harness phase](../work/phases/PHASE-2.md) | ready | Create implementation tickets or record the owner decision. |

## Non-Functional Requirements

| ID | Requirement | Category | Status | Evidence or boundary |
| --- | --- | --- | --- | --- |
| NFR-01 | Authorization fails closed and is rechecked at mutation, renewal, export, asset, and room-admission boundaries. | Security | verified | Access, revoke-race, purpose-separated token, and denial suites. |
| NFR-02 | A collaboration ACK means the update and receipt are durable; retries with the same identity are idempotent and changed bytes conflict. | Durability | verified | Runtime/PostgreSQL fault tests and 650/650 ACK/receipt acceptance. |
| NFR-03 | Fifty clients sending one small update per second have ACK p95 below one second and converge exactly. | Performance | verified | Local result: p50 9.742 ms, p95 13.236 ms, 650/650 ACKs. |
| NFR-04 | Production recovers from controlled Docker/VM restart without manual compose intervention. | Reliability | in_progress | The 2026-10-19 reboot required manual recovery; `PROD-REBOOT-RECOVERY` is P0. |
| NFR-05 | Secrets are distinct, minimum-length, backend/private-service only, absent from logs/docs/history, and rotated or revoked after exposure. | Security | in_progress | Git history scrub is complete; Google credential revocation and v2 service-secret installation remain. |
| NFR-06 | Operators can back up and restore pre-migration data and must not use image-only rollback after a scene is promoted to v2. | Recoverability | verified | Fresh cutover dump restored with all14 table counts matching; rollback images/config and replacement-scene procedure are recorded. |
| NFR-07 | Human docs and stable Markdown/`llms.txt` agent surfaces describe the same released behavior and clearly label remaining release gates. | Documentation | verified | Public guides and `src/frontend/public/llms.txt` describe the VM100 deployment and final controlled-reboot/TURN/denial/role boundaries. |
| NFR-08 | Credential entry points bound JSON bodies, parse strictly, keep unknown-account login work comparable, reject inactive JWT users, and throttle abusive attempts without storing raw account or token identifiers. | Security | verified | [Authentication hardening verification](../work/verification/auth-hardening-2026-10-19.md); login, invitation registration, and public scene credential POSTs return stable JSON `429 RATE_LIMITED` plus `Retry-After`. |

## Traceability

- User outcomes: [USER_STORIES.md](USER_STORIES.md).
- Runtime proof: [VALIDATION_MATRIX.md](../work/VALIDATION_MATRIX.md) and
  [real-stack acceptance](../work/verification/collaboration-v2-real-stack.md).
- Current execution order: [BACKLOG.md](../work/BACKLOG.md) and
  [ROADMAP.md](../work/ROADMAP.md).
- Stable contracts: [API.md](../architecture/API.md),
  [ARCHITECTURE.md](../architecture/ARCHITECTURE.md), and ADRs
  [0008](../decisions/0008-openai-compatible-ai-provider.md) through
  [0012](../decisions/0012-legacy-dashboard-presentation.md).
