# Enterprise parity VM100 rollout — 2026-10-21

Status: deployed from application commit `325f9fd`; service health and forward
migrations passed. Authenticated browser UAT remains pending because the
production browser session is currently signed out.

## Rollout evidence

- VM100 source: `325f9fd` on `master`.
- Fresh database backup: `/opt/apps/whiteboard/backups/enterprise-cutover-20261020T180655Z`.
- Compose validation: passed with the production environment file.
- Migration job: exited `0`.
- Backend, collaboration runtime, feature worker, MCP and frontend: healthy.
- Public `GET /healthz`: HTTP `200`, body `ok`.
- S3 secret: installed in the VM100 secret file and loaded by the backend; the
  value is intentionally absent from Git and this record.

The production compose network patch remains local to VM100 so PostgreSQL and
RustFS use their existing shared networks. The feature worker is attached to
the PostgreSQL network and reached `postgres-pg` after recreation.

## Remaining UAT

The public login, deep-link, scene save/reload, guest sharing, native library,
presentation, SSO/SCIM provider and two-network media checks still require a
valid production browser session and the corresponding external provider
configuration. No provider credential or password is recorded here.

Rollback uses the preserved VM100 compose/environment backup and the existing
replacement-scene recovery runbook; image-only rollback is not sufficient for
promoted collaboration scenes.
