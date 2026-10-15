---
artifact_type: release_notes
id: REL-2026-10-15-WHITEBOARD-VM100
status: in_review
owner: shared
human_fields:
  - release_approval
  - known_issues_acceptance
  - rollback_acceptance
ai_fields:
  - summary
  - changes
  - verification
  - linked_work
shared_fields:
  - status
trace:
  backlog_items: [EXC-001..EXC-034]
  requirements: [REQ-01..REQ-10, P2-REQ-01..P2-REQ-03, P3-REQ-01..P3-REQ-04, P4-REQ-01..P4-REQ-03, P5-REQ-01..P5-REQ-03]
  phases: [1, 2, 3, 4, 5]
  tickets_or_bugs: [public-viewer-collaborators-hydration]
  test_verification: docs/work/VALIDATION_MATRIX.md
  validation_matrix: docs/work/VALIDATION_MATRIX.md
  docs_review: docs/operations/RELEASE_CHECKLIST.md
  adrs: [docs/architecture/decisions/]
  changelog: docs/releases/CHANGELOG.md
---

# Release Notes: Whiteboard VM100 — 2026-10-15

## Field Ownership

- Human owns release approval, known-issue acceptance, and rollback acceptance.
- AI records the shipped scope, evidence, operational notes, and documentation links.

## Status

- ID: `REL-2026-10-15-WHITEBOARD-VM100`
- Status: `in_review`
- Target: `https://whiteboard.dungxbuif.com/`
- Runtime: VM100, `/opt/apps/whiteboard`, Docker Compose production bundle
- Release commit: this commit (the final hash is recorded by Git after commit)

The application code and VM100 deployment have passed the automated and live
smoke checks listed below. Final human sign-off remains open for production
secret rotation, database backup/restore confirmation, and rollback image
acceptance.

## Summary

This release completes the self-hosted Excalidraw whiteboard core across
workspace management, authenticated scene routes, read-only public sharing,
realtime collaboration primitives, WebRTC voice signalling, and the
OpenAI-compatible AI proxy. The production authentication path is now
email/password with the `userctl` CLI; Google OAuth is not required at runtime.

## Changes

- Added workspace, folder, scene, search, Starred, Trash, restore, rename, and
  move flows with UUID-backed ownership checks.
- Added authenticated deep links at
  `/w/{workspaceId}/scenes/{sceneId}` and enforced workspace ownership on
  workspace-scoped API routes and realtime room joins.
- Added email/password login, default-workspace creation, and
  `userctl create`/`userctl password` operations for provisioning and password
  rotation.
- Added Excalidraw persistence, autosave, presentation mode, slide navigation,
  and keyboard controls.
- Added public read-only scene sharing at `/p/{token}`, optional bcrypt
  password protection, copy/revoke controls, and share-token invalidation.
- Added Socket.IO presence, cursors, scene-sync signalling, and WebRTC voice
  signalling with short-lived TURN credentials from the Homelab Coturn relay.
- Added authenticated OpenAI-compatible chat, text/image-to-diagram, and
  diagram-to-code proxy flows with DeepSeek as the default provider.
- Upgraded the frontend compiler/toolchain to TypeScript `7.0.2`.
- Fixed public viewer hydration for persisted Excalidraw scenes whose
  `appState.collaborators` value was JSON-serialized. The viewer now restores
  the runtime `Map` shape before rendering.
- Updated release, operations, context, and validation documentation for the
  VM100 deployment and the current email/password runtime.

## Verification

### Repository checks

- `go test ./...` — **59 tests passed across 11 packages**.
- `npm run test:unit` — **19 tests passed across 6 files**.
- `npm run build` (`tsc -b && vite build`) — **passed with TypeScript 7.0.2**.
- `npx playwright test e2e/dashboard.spec.ts` — **11 passed, 0 failed**.
- `docker compose config` — **passed for the production compose bundle**.
- `git diff --check` — **passed for the release documentation change**.

### VM100 and public origin

- `GET https://whiteboard.dungxbuif.com/healthz` returned HTTP 200 with body
  `ok`.
- The authenticated dashboard opened on the production origin and the public
  share dialog created a read-only link.
- The public viewer rendered `Untitled scene`, `Read-only`, and a canvas region;
  the browser reported no application console errors.
- The test share token was revoked after verification; fetching the revoked
  token returned `404 Shared scene not found`.
- The production frontend was rebuilt and recreated from the fixed image on
  VM100; the deployment retained the expected container aliases.

## Known Issues and Release Gates

- The current sharing product is public read-only sharing. Member invitations
  and collaborative edit permissions are not part of this release.
- Two-browser voice calling across separate networks still needs a manual
  production UAT pass.
- AI generation needs a real `AI_API_KEY` and provider quota for a production
  smoke pass; the proxy and automated contract tests are present.
- Human release approval still needs confirmation that production secrets were
  rotated, the PostgreSQL backup/restore point is valid, and rollback image
  tags are recorded. See `docs/operations/RELEASE_CHECKLIST.md` and
  `docs/operations/auth-secrets-rotation.md`.

## Rollback

Use the image tags recorded in the VM100 deployment directory and redeploy the
previous backend/frontend pair with:

```sh
cd /opt/apps/whiteboard
docker compose --env-file .env.prod -f docker-compose.prod.yml up -d --no-build
```

Do not remove the current images until the human owner accepts the rollback
point and confirms the database restore procedure.

## Linked Work

- `8b4f989` — normalize persisted Excalidraw collaborators in the public viewer.
- `eb30443` — record the production viewer fix and live share verification.
- `ad3a32b` — document canonical UUID route identifiers.
- `f97c225` — make the frontend healthcheck IPv4 explicit.
- `93e1341` — retain VM100 container aliases.
- `d766e1e` — connect the migration job to the Homelab database.
- [Changelog](CHANGELOG.md)
- [Release checklist](../operations/RELEASE_CHECKLIST.md)
- [Validation matrix](../work/VALIDATION_MATRIX.md)
- [VM100 operations runbook](../operations/ENVIRONMENTS.md)
