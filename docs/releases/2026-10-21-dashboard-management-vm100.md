---
artifact_type: release_notes
id: REL-2026-10-21-DASHBOARD-MANAGEMENT-VM100
status: deployed
owner: shared
human_fields: [release_approval, rollback_acceptance]
ai_fields: [deployment_evidence, verification_notes]
shared_fields: [status]
---

# Dashboard management VM100 release — 2026-10-21

Status: **deployed and smoke-verified**.

This release promotes workspace rename; folder rename and move; scene move and
server-side duplicate; disabled Collections with retry states; and
authenticated durable scene projection. The implementation is in application
commit `1847241`; the final pushed release head is `d639b15` (documentation
and verification commits only after the runtime build).

## Production evidence

- VM100 was synchronized from the clean `codex/dashboard-management-parity`
  workspace. Existing production compose, environment, backups, and private
  overrides were preserved.
- Pre-deploy backup: `/opt/apps/whiteboard/backups/dashboard-management-20261021T001701Z`.
- Compose validation passed and production images built for backend,
  collaboration, feature worker, MCP, and frontend.
- Migration `000029` completed after the documented ambiguous-row recovery:
  one qualifying legacy hidden scene was preserved, version `29` is clean
  (`dirty = false`), its deletion state was restored, and the temporary
  operator table was removed.
- Backend, collaboration, feature worker, MCP, and frontend are running.
- `GET https://whiteboard.dungxbuif.com/healthz` returned `200` with body
  `ok`; the public root and scene deep-link shells returned `200`.
- The existing authenticated Chrome session loaded the production workspace
  settings Members page and displayed the owner/member controls.

## Verification boundary

Local Go, frontend unit/build, and dashboard Playwright gates passed before the
build. The production checks above prove deployment and public routing.
Credentialed owner/editor/viewer mutation canaries, provider checks, and
cross-network media remain operator-owned follow-up and are not inferred from
HTTP, container health, or one authenticated settings-page load.

## Rollback

The prior manifest, private environment, and running image IDs are preserved in
the backup directory above. Do not roll back a promoted collaboration scene by
changing images alone; use `docs/operations/collaboration-recovery.md` when
durable scene state has been promoted.
