---
artifact_type: phase
id: ENTERPRISE-PARITY
status: in_progress
owner: shared
updated: 2026-10-20
approval: approved_2026-10-20
---

# Enterprise parity programme

## Goal and requested scope

Plan and then deliver ENT-P1-01–04 and ENT-P2-01–03: comments and guest comments,
screen sharing, hosted presentation/spotlight/waiting room/reactions/view lock,
activity, version history/diff/restore, libraries/collections, domain capture,
SSO/SCIM and retention. Org is workspace. On 2026-10-20 the user additionally
requested all scene controls within/floating over the canvas, without a separate
control header consuming viewport height.

## Current stage

The detail design and 22 bounded execution tasks were approved for implementation
on 2026-10-20. Wave 0/1 foundation work is active; task checkboxes remain tied to
real verification evidence. The already approved v2 baseline is reused rather
than relabeled as enterprise parity.

## Trace

- [Requirements](../../requirements/REQUIREMENTS.md#approved-product-backlog)
- [Proposed detail design](../../superpowers/specs/2026-10-20-enterprise-parity-design.md)
- [Programme/tasks](../../superpowers/plans/2026-10-20-enterprise-parity.md)
- [Foundation/UI](../../superpowers/plans/2026-10-20-enterprise-foundation.md)
- [Comments/activity](../../superpowers/plans/2026-10-20-enterprise-engagement.md)
- [Live media/presentation](../../superpowers/plans/2026-10-20-enterprise-live.md)
- [Versions/libraries/collections](../../superpowers/plans/2026-10-20-enterprise-content.md)
- [Identity/retention](../../superpowers/plans/2026-10-20-enterprise-identity.md)
- [Backlog](../BACKLOG.md), [roadmap](../ROADMAP.md),
  [validation matrix](../VALIDATION_MATRIX.md),
  [planning evidence](../verification/enterprise-planning-2026-10-20.md)
- [Changelog](../../releases/CHANGELOG.md); enterprise rollout record is created
  by E2 only after deployment, so no unshipped release note is presented as proof.

Task IDs in the subsystem plans are the execution work items (same convention
as Collaboration v2); do not create duplicate placeholder tickets. Each owns
files/interfaces, RED/GREEN/negative cases, doc review and an atomic commit.
ADR proposals live in the detail design; accepted ADR numbers are allocated
when each reviewed boundary ships, avoiding invented accepted decisions.

## Slices and dependencies

| Slice | Tasks | Dependency | Status |
| --- | --- | --- | --- |
| Full-viewport canvas controls | UI1–UI2 | Existing editor | complete |
| Feature authority/events/jobs | F1–F3 | Existing ACL/audit | complete |
| Comments/guest comments/inbox/activity | C1–C3 | F1–F3/UI1–UI2 | in_progress |
| Screen/live presentation/audience | M1–M3 | F1/UI1–UI2 | in_progress |
| Version capture/diff/restore | V1–V2 | F1–F3/UI1–UI2 | draft |
| Collections/native libraries | L1–L2 | F1–F3/UI1–UI2 | draft |
| Domain/SSO/SCIM | I1–I3 | F1–F3 | draft |
| Retention/referenced storage | R1–R2 | F1–F3/C/V/L models | draft |
| Integrated acceptance/release | E1–E2 | Every implementation slice | draft |

UI can ship before feature schema work. Comments and media may run in parallel
after prerequisites; router/App/protocol integration and migration allocation
remain serialized. Use fresh reviewers for migration, identity, restore and
deletion boundaries. No execution agent was started for the planning task.

## Acceptance and evidence gates

- Every requirement maps to tasks and concrete tests in the programme.
- Native/legacy UI parity, complete canvas viewport, accessible focus/keyboard/
  mobile and reloadable routes have screenshots and browser proof.
- User/guest/agent role isolation survives current-authority revocation during
  writes, media and background jobs; waiting audience receives no scene data.
- Restores preserve document/element identity, durable receipts and offline
  replay, including image/binding/frame references.
- DNS/OIDC/SCIM fixture proof and actual provider UAT are labeled separately;
  missing provider credentials do not block unrelated feature implementation.
- Retention defaults, preview/holds/grace and protected backups prevent live
  asset deletion; canonical Yjs history is outside automatic purge.
- Unit/integration/real-stack/security review, migration13 upgrade+fresh install,
  backup/restore, VM100 health/reboot/canaries, Homelab and public/agent docs are
  required before marking this programme done.

## Review decisions

The approved defaults are comment access policy,
ten-connection media capacity/one publisher, editor restore rights, private
collection visibility, OIDC-first SSO, SCIM membership provenance, and retention
duration/grace. These are written choices with tests; changing one requires a
recorded design ruling. The user's canvas-control placement and existing dashboard
preference are direct constraints.

## Out of this planning goal

Harness enforcement, AI-provider live UAT and retired Google credential
revocation keep their existing backlog entries. This does not exclude any of
the seven requested enterprise requirement groups from the programme.
