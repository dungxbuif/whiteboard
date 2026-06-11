---
artifact_type: release_checklist
id: RELEASE_CHECKLIST
status: draft
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

- [ ] Tests passed
- [ ] Master docs reconciled
- [ ] ADRs updated
- [ ] Release notes prepared
- [ ] Rollback plan confirmed

## Post-Release

- [ ] Deployment verified
- [ ] Monitoring checked
- [ ] Incidents recorded if any
- [ ] `docs/CONTEXT.md` updated
