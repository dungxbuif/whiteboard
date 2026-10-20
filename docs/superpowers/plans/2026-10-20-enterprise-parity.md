# Enterprise Parity Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver the complete ENT-P1-01–04 / ENT-P2-01–03 scope, beginning with canvas-integrated controls.

**Architecture:** Go owns feature metadata/authority; Node remains the exclusive durable Yjs writer. Metadata events/outbox support comments/activity/jobs; screen and presentation state use the existing authenticated socket. Identity, versions and retention extend the same workspace boundary.

**Tech Stack:** Existing Go/Chi/GORM/Casbin, PostgreSQL, TypeScript/React/Vite/Zustand, Excalidraw/Yjs, WebRTC/TURN and S3. OIDC adds a reviewed `coreos/go-oidc/v3` client with existing OAuth2.

**Spec:** [Enterprise detail design](../specs/2026-10-20-enterprise-parity-design.md).

## Global Constraints

- Status: approved for implementation on 2026-10-20; execute every work package and preserve the dependency order below.
- Org means workspace; preserve UUID and alias scene routes.
- Canvas fills `100dvh`; controls/panels float inside/on the canvas without a separate height-consuming header.
- Preserve legacy dashboard styling and Excalidraw native scene styling.
- Go remains authority; Node remains exclusive canonical Yjs writer; no RelayHub or second realtime socket.
- Preserve documentId, receipts, stable element IDs/types/maps and offline queues during restore.
- Feature metadata mutation retries bind requestId, principal and canonical bytes; decimal revision strings.
- Defaults and limits are exactly those in the approved spec; changes require a recorded design ruling.
- Run shell through `rtk`; use guarded loopback `*_test` PostgreSQL; no production test database.
- Every task includes internal/public/AI-readable docs reconciliation, regression tests and an atomic commit.

## Review Focus

1. Revocation or SSO/SCIM deprovision while a browser, guest or job is already active: check current authority before read/fanout/commit.
2. Reconnect/duplicate requests and lost HTTP responses: no duplicated comments, notifications, restores or SCIM membership side effects.
3. Images, bindings, frames and offline updates through restore/library insertion: no replaced maps, foreign assets or dangling references.
4. Mobile/keyboard controls and browser capture cancellation: preserve full canvas, meaningful focus, native tools and drawing after errors.
5. Concurrent retention, policy changes and backups: reference/hold checks prevent deleting assets still needed by scenes/versions/libraries/backups.

## Decision boundaries

The user requested planning first and then approved the complete programme with
the instruction “implement all” on 2026-10-20. The linked spec and defaults are
accepted runtime contracts for this implementation. Enterprise migrations still
ship behind the staged feature flags and verification gates defined here.

User already permits agent work. Recommended execution: subagent-driven by
bounded task, with a fresh reviewer for auth, migrations, restores and retention.
No agents are required to review this planning package itself. Avoid parallel
writers in router/main/protocol/App; the integration owner serializes them.

## Work packages and dependency order

| Wave | Package/task IDs | Prerequisite | Independently shippable result |
| --- | --- | --- | --- |
| 0 | [Canvas UI: UI1–UI2](2026-10-20-enterprise-foundation.md) | None | Full-viewport editor, floating existing controls and shared panel coordinator |
| 1 | [Foundation: F1–F3](2026-10-20-enterprise-foundation.md) | UI is independent | Feature policy/idempotency, events/outbox and bounded job worker |
| 2A | [Comments/activity: C1–C3](2026-10-20-enterprise-engagement.md) | F1–F3, UI1–UI2 | Anchored guest/member discussion, moderation, notifications, authorized activity |
| 2B | [Media/presentation: M1–M3](2026-10-20-enterprise-live.md) | F1, UI1–UI2 | Independent screen media and hosted presentation/waiting-room controls |
| 3 | [Content: V1–V2, L1–L2](2026-10-20-enterprise-content.md) | F1–F3, UI1–UI2 | User checkpoints/diff/restore, reusable native library and legacy-style collections |
| 4 | [Identity/governance: I1–I3, R1–R2](2026-10-20-enterprise-identity.md) | F1–F3; R depends on C/V/L reference models | Domain/OIDC/SCIM with revocation, previews/holds/retention and protected asset deletion |
| 5 | E1–E2 below | All implementation slices | Integrated acceptance, staged VM100 rollout and complete docs |

Wave 2A and 2B may execute independently after the foundation review. V/L schema
allocation and all `000014+` migration numbers are serialized by one owner;
never infer migration order from task order alone. Identity assurance and
retention have mandatory cross-package reviews before activation.

## Requirement coverage

| Requirement | Tasks | Proof |
| --- | --- | --- |
| User canvas controls | UI1, UI2, E1 | viewport bounds/native toolbar/focus/mobile/keyboard browser cases |
| ENT-P1-01 comments/guests | F1, F2, C1, C2 | guest view-only comment, anchor, moderation, subscribe/revoke cases |
| ENT-P1-02 screen sharing | M1, E1 | permission cancel, relay video, ten/eleven, cleanup and drawing |
| ENT-P1-03 live presentation | M2, M3, E1 | host handoff, pending admission data denial, spotlight, reactions/view lock |
| ENT-P1-04 activity | F2, C3, R1 | source projection replay, ACL filtering and retention cutoff |
| ENT-P2-01 versions | V1, V2, R1 | immutable checkpoint/image, diff, stable-map restore, offline replay |
| ENT-P2-02 libraries/collections | L1, L2, R2 | private collection denial, imported library IDs/assets/search/export |
| ENT-P2-03 domain/SSO/SCIM/retention | I1, I2, I3, R1, R2 | DNS/OIDC negative cases, SCIM protocol/revoke, hold/reference/backup purge races |

## Task E1: Complete integrated acceptance and security review

**Files:** Create `src/frontend/e2e/real-stack/enterprise.spec.ts`,
`enterprise-revocation.spec.ts`, `enterprise-retention.spec.ts`,
`tests/packaging/enterprise-media.mjs`,
`docs/work/verification/enterprise-acceptance.md`; modify real-stack `harness.ts`
and `browser.ts` only to add isolated worker/IdP/DNS fixtures.

**Interfaces:** Consumes every package's public contracts and existing room,
scene, asset and agent APIs. Produces release acceptance evidence; no API.

- [ ] Write failing full-stack cases: comments/guests + native concurrent edits;
  two-network screen video + existing voice + presentation admission; library
  image insertion -> manual checkpoint -> edit -> restore -> offline replay;
  OIDC -> SCIM disable during a media call; retention with pinned backup assets.
  Assert no lost acknowledged write, no pending-audience content, no reference
  leak and zero owned canary resources after cleanup.
- [ ] Run `rtk npm run test:e2e:collab --workspace ./frontend` in `src`;
  the existing Playwright configuration discovers the new real-stack specs.
  Absence of a required fixture is a failed gate, never a passing skip.
- [ ] Resolve defects in their owning tasks; rerun focused failures, then
  existing real-stack regressions plus all enterprise cases. Record actual
  counts/commands and generated/local versus public/two-network proof.
- [ ] Run `rtk npm test`, `rtk npm run test:unit --workspace ./frontend`,
  `rtk npm run build --workspace ./frontend` in `src`, and
  `rtk go test ./...` in `src/backend` with guarded DB variables configured.
  Add whole-branch review of tenant boundaries, assurance, SSRF and deletion.
- [ ] Update validation matrix, reconcile ADR/API/ERD/protocol/guides and commit
  as `test(enterprise): verify integrated parity acceptance`.

## Task E2: Stage migrations and VM100 release

**Files:** Modify `docker-compose.prod.yml`, `src/.env.example`,
`docs/operations/DEPLOYMENT.md`, `ENVIRONMENTS.md`, `RELEASE_CHECKLIST.md`;
create `docs/releases/enterprise-parity.md`,
`docs/operations/enterprise.md`, `src/frontend/public/docs/enterprise.md`;
modify `src/frontend/public/llms.txt`. Reconcile Homelab's actual whiteboard
template/runbook paths after read-only inventory; never invent a deployment file.

**Interfaces:** Consumes E1 evidence, migration manifest, worker readiness,
feature flags, DNS/OIDC/SCIM fixtures/config and retention dry-run report.
Produces exact deployed commit/image manifest and per-feature canary evidence.

- [ ] Verify migration13 upgrade and fresh install, DB+S3 restore, limits,
  private listeners, IdP secret storage, job lease/reboot recovery and existing
  TURN service ownership. Capture backup protection set for retention jobs.
- [ ] Deploy flags off, observe health; enable UI/foundation/comments/activity,
  then media/presentation, versions/libraries, then configured identity.
  Keep destructive retention disabled through its seven-day preview period.
- [ ] Run disposable public-origin owner/editor/viewer/guest tests and two-network
  voice+screen test. Use an explicitly configured real OIDC/SCIM tenant for
  provider UAT; if unavailable record that rollout dependency rather than claim
  a real-provider pass from a fixture.
- [ ] Enable retention only after its preview/grace, backup/reference proof and
  reviewed policy; prove restart/revoke replay without duplicate jobs. Rollback
  disables features and drains workers; never roll old images over new writes.
- [ ] Reconcile human and agent guides, Homelab, statuses and release sign-off;
  run `rtk git diff --check`, record cleanup, then commit
  `docs(release): record enterprise parity rollout`.

## Planning self-review / execution handoff

Planning checks and known decisions live in the
[verification record](../../work/verification/enterprise-planning-2026-10-20.md).
Task checkboxes remain unchecked until real implementation evidence exists.
Review this package's proposed defaults before executing UI1/F1; product
implementation and deployment are separate from completing this planning goal.
