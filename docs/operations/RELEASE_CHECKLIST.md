---
artifact_type: release_checklist
id: RELEASE_CHECKLIST
status: active
owner: shared
human_fields: [release_approval, rollback_acceptance]
ai_fields: [pre_release_checks, post_release_checks, verification_notes]
shared_fields: [status]
---

# Release Checklist

## Field Ownership

- Human owns release approval and rollback acceptance.
- AI records pre-release checks, post-release checks, and verification notes.

## Pre-Release

- [x] Backend tests passed (`go test ./...`)
- [x] Frontend unit tests passed (`npm run test:unit`)
- [x] Frontend production build passed (`npm run build`)
- [x] Browser smoke tests passed (`npx playwright test`)
- [x] Docker Compose syntax passed (`docker compose config`)
- [x] Master docs reconciled
- [x] ADRs updated
- [x] Release notes prepared in `docs/releases/CHANGELOG.md` and `docs/releases/2026-10-15-whiteboard-vm100.md`
- [ ] Production `.env.prod` populated in the secret manager
- [ ] JWT/TURN/S3 secrets rotated
- [ ] PostgreSQL backup and restore point confirmed
- [ ] Rollback image tags confirmed

## Smoke Verification

- [x] `GET /healthz` returns HTTP 200 with body `ok`
- [x] Password login completes on the HTTPS origin with a user provisioned by `userctl`
- [x] Workspace list and scene create/save work
- [x] Authenticated scene deep link opens only for the owning workspace
- [x] Public share and password challenge work (automated; live public viewer/revoke smoke also passed)
- [ ] Socket.IO connection and presence update work
- [ ] TURN voice call works from two networks
- [ ] AI chat and diagram generation work with `AI_API_KEY`

## Post-Release

- [x] Deployment verified (VM100 health check and live public-share smoke)
- [ ] Monitoring checked
- [ ] Incidents recorded if any
- [x] `docs/CONTEXT.md` updated

## Evidence Notes — 2026-10-15

- Live public-share smoke used a temporary token and revoked it after the check;
  the revoked token returned `404 Shared scene not found`.
- Socket.IO two-browser presence, cross-network TURN voice, and AI generation
  with a real provider key remain human UAT items.
- Secret rotation, PostgreSQL backup/restore, and rollback image acceptance
  remain human-owned production gates.
