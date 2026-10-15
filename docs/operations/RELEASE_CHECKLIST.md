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
- [x] Release notes prepared in `docs/releases/CHANGELOG.md`
- [ ] Production `.env.prod` populated in the secret manager
- [ ] JWT/TURN/S3 secrets rotated
- [ ] PostgreSQL backup and restore point confirmed
- [ ] Rollback image tags confirmed

## Smoke Verification

- [ ] `GET /healthz` returns `{"status":"ok"}`
- [ ] Password login completes on the HTTPS origin with a user provisioned by `userctl`
- [ ] Workspace list and scene create/save work
- [ ] Public share and password challenge work
- [ ] Socket.IO connection and presence update work
- [ ] TURN voice call works from two networks
- [ ] AI chat and diagram generation work with `AI_API_KEY`

## Post-Release

- [ ] Deployment verified
- [ ] Monitoring checked
- [ ] Incidents recorded if any
- [ ] `docs/CONTEXT.md` updated
