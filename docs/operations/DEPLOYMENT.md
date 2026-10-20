# Production Deployment

Status: enterprise parity is deployed on VM100 from application commit
`325f9fd`. The backend, collaboration runtime, feature worker, MCP adapter and
frontend are healthy, and the forward migration job exits successfully. The
authenticated browser/provider UAT matrix remains on the release checklist.
For promoted scenes, use the
[replacement-scene recovery runbook](collaboration-recovery.md); image-only
rollback is not valid.

Enterprise metadata jobs use the separately health-checked backend-image worker
described in [enterprise-jobs.md](enterprise-jobs.md). Deploy it with feature
flags disabled until each owning feature package has passed its release gate.

## Deployment Target

The release bundle targets one Linux host or VM running Docker Compose. The
frontend nginx container serves the built SPA and proxies `/api/` to Go,
`/collab/` to the custom Node WebSocket runtime and `/mcp` to the MCP adapter. PostgreSQL, S3-compatible storage, and Coturn
are external services supplied through environment configuration. The shared
Homelab gateway runs Coturn on `turn.relayhub.dungxbuif.com:5349`; the release
stack must not start another TURN listener.

## Required secrets and settings

Create an untracked `.env.prod` at the repository root from `src/.env.example`.
Set production values for `DATABASE_URL`, `JWT_SECRET`, S3 credentials,
`FRONTEND_URL`, `TURN_URL`, `TURN_AUTH_SECRET`, and `TURN_REALM`. Configure
one stable `DOMAIN_RECEIPT_KEY` on every backend replica so exact retries can
decrypt prior workspace-domain responses; when omitted, the backend derives
this key from `JWT_SECRET`, so JWT rotation also invalidates those receipts. Configure
`AI_API_KEY` only when enabling the optional AI provider workflow. Also
configure distinct values of at least 32 bytes for
`COLLAB_JWT_SECRET`, `AUTHORITY_SERVICE_SECRET`, `MUTATION_GRANT_SECRET` and
`COLLAB_INTERNAL_SECRET`. `TURN_AUTH_SECRET` must match the
VPS-only Coturn secret and never be exposed to the browser. Never put secret
values in Git or in browser URLs.

Create or rotate password users with the backend `userctl` CLI after the
migration job completes. The CLI stores bcrypt hashes and never prints the
password.

Use `AI_BASE_URL=https://api.deepseek.com/v1` and the DeepSeek model names when
using the supplied DeepSeek key. Select a vision-capable compatible model if
image-to-diagram is required.

## Release process

### Upgrade through migration 000029

When upgrading from a build that already offered scene duplication, stop all
old backend replicas and let in-flight requests exit before starting migration
`000029`. The old build could have left a soft-deleted destination scene while
copying assets. The migration marks targets recorded in `library_asset_stages`
as pending; it refuses ambiguous empty scenes deleted at creation time that
have no stage record. Check the candidates before rerunning migration:

```sql
SELECT s.id, s.workspace_id, s.created_at, s.deleted_at,
       EXISTS (SELECT 1 FROM library_asset_stages a
               WHERE a.workspace_id=s.workspace_id AND a.target_scene_id=s.id) AS has_stage
FROM scenes s
WHERE s.deleted_at IS NOT NULL AND s.writer_mode='v1' AND s.content_version=1
  AND s.content='{"elements":[],"appState":{},"files":{}}'::jsonb
  AND ABS(EXTRACT(EPOCH FROM (s.deleted_at-s.created_at))) <= 60;
```

Review each unbacked row against request logs and its scene asset rows. An
abandoned duplicate with no stage record can be removed by a targeted,
workspace-scoped hard delete after confirming no object copy occurred. Preserve
a confirmed legitimate trashed scene by recording its ID and original
`deleted_at` in an operator table, temporarily clearing `deleted_at` while
all backends are stopped, then restoring the original timestamp after migration:

```sql
CREATE TABLE IF NOT EXISTS scene_duplicate_upgrade_preserved
    (id uuid PRIMARY KEY, deleted_at timestamptz NOT NULL);
INSERT INTO scene_duplicate_upgrade_preserved
SELECT id, deleted_at FROM scenes WHERE id IN ('<reviewed-scene-uuid>');
UPDATE scenes s SET deleted_at=NULL
FROM scene_duplicate_upgrade_preserved p WHERE s.id=p.id;
-- Run migration 000029, then before restarting the backend:
UPDATE scenes s SET deleted_at=p.deleted_at
FROM scene_duplicate_upgrade_preserved p WHERE s.id=p.id AND s.pending_duplicate=FALSE;
DROP TABLE scene_duplicate_upgrade_preserved;
```

Replace the placeholder only with reviewed IDs; retain the operator table if
the migration or timestamp restoration fails. Do not let old backend
replicas resume before migration succeeds. After the upgrade, pending copies
are excluded from Trash/search/Restore, and cleanup failures carry a scene ID
in structured logs for operator reconciliation.

1. Run the pre-release checklist and all repository tests.
2. Build and start the migration job, backend, collaboration runtime, MCP adapter, and frontend:
   `docker compose --env-file .env.prod -f docker-compose.prod.yml up -d --build`.
   The production manifest attaches the services to the pre-created
   `homelab-net`, `whiteboard-app`, and `edge-http-net` networks; the frontend
   stays portless and is reached through the public gateway.
3. Confirm the migration job exits successfully before accepting traffic.
4. Verify `/healthz`, password sign-in, an owned workspace request, scene save,
   an authenticated deep link at `/w/{workspaceId}/scenes/{sceneId}`, public
   share viewer, a scoped `/collab/` WebSocket that can join only its authorized
   scene, MCP liveness plus an authenticated native mutation/export, TURN allocation from an external
   network, and the AI proxy with a real key.
5. Record the image tags, migration version, and verification results in the
   release notes.

## Rollback

Keep the previous application image tags available. Roll back by setting the
previous tags in the Compose file and running `docker compose up -d`; do not
run down migrations during an application rollback. Restore the database only
from a verified backup when a schema rollback is explicitly required.

Before release, verify PostgreSQL backups and a restore point, and rotate the
JWT and TURN secrets described in `docs/operations/auth-secrets-rotation.md`.

## Verification

- `docker compose --env-file .env.prod -f docker-compose.prod.yml config`
- `docker compose --env-file .env.prod -f docker-compose.prod.yml ps`
- `curl -fsS https://<public-origin>/healthz`
- Complete the smoke flows listed in `docs/operations/RELEASE_CHECKLIST.md`.

## Planned v2 public links and room credentials

Before enabling v2, configure a dedicated `COLLAB_JWT_SECRET` of at least 32 bytes,
distinct from `JWT_SECRET`, on Go and the private collaboration runtime. Go signs
ten-minute HS256 credentials for issuer `whiteboard-api` and audience
`whiteboard-collab`. Scene sessions have a distinct audience and purpose.
See the [public-link contract](../architecture/public-links.md).

Migration 000008 hashes existing tokens into independent view links while
retaining legacy storage for compatibility. Password hashes and inactive state
are preserved. Use POST JSON for password exchange; legacy query passwords
are no longer consumed and must be stripped before request logging. Keep old
raw tokens private until the compatibility stage is retired. New tokens are
returned once and cannot be recovered from link metadata.

Do not run its down migration as routine rollback: it refuses multiple links
or grants that cannot be represented by the legacy model. Preserve the schema
and disable v2 entry points, or use a verified backup and explicit migration
plan. A3 is implemented, deployed and covered by current authorization checks.
Public refresh returns a renewed scene session before expiry, capped to link
expiry. A monotonic link revision invalidates existing sessions after password
or link changes. Release wiring must pass COLLAB_JWT_SECRET to both Go and
the collaboration service, and redact legacy token paths/query strings in
reverse-proxy and tracing logs. Go strips legacy query strings and redacts
RequestURI; routing still retains URL.Path. Credential SQL queries suppress
parameter logging.


## E3 backup preparation (before production changes)

Record the running whiteboard image IDs and protect a copy of the existing manifest/environment with mode0600. Read the database connection from the running backend environment in memory; never print the URL or pass it as an exposed command argument. Use a PostgreSQL client at least as new as the server to create a custom-format database dump in a private release directory. Capture the migration version, restore the dump into a new disposable local database, compare table row counts and migration state, then apply the complete forward migration stream7–13 in that disposable database and verify schema state. Never restore over production or apply down migrations.

The dump includes account and token data. Keep it private, outside Git and browser artifacts. This database-only proof does not back up S3 objects; existing immutable objects must remain available throughout release. Repeat the dump immediately before the migration cutover if production changed after this preparation. Record exact evidence separately; no successful restore is claimed until the commands pass.


## V2 rollout procedure and current result

1. Complete task/whole-branch review and the current release checklist. Build frozen-source Linux/amd64 images for backend, collab, MCP and frontend; record their exact IDs. Keep the v1 manifest/environment/images captured in the private release directory.
2. Preserve existing external networks and verify the dedicated authority subnet remains unused. Generate four separate v2 secrets in the private environment, each at least32bytes and distinct from the login secret. Set both public origins to the public HTTPS app. Use the tested `turn:103.82.21.202:5349` endpoint with the existing matching TURN secret/realm. Start with `COLLAB_V2_ENABLED=false`.
3. Take a fresh cutover database backup. Validate the manifest using `docker compose --env-file .env.prod -f docker-compose.prod.yml config --quiet`; never print expanded configuration. Apply migrations7–13as one forward stream and verify version13with dirty=false. No partial stream or down migration is permitted.
4. Start backend, collab and MCP; verify Go health, collab readiness/ownership and MCP liveness. Recreate frontend after upstream changes so nginx resolves current addresses. Do not run `down -v` or replace shared overlay networks. Frontend remains portless and retains its established alias.
5. Enable new promotions for the canary window, promote a disposable test scene and exercise two-browser edits, guest view/edit, invitation/role/revoke, offline recovery, native agent write/export, restart and denied requests. A single global flag is not a scene allowlist; keep the window controlled and monitor any other promotions. Existing promoted scenes retain the v2 writer even if new promotions are later disabled.
6. Check native UI desktop/mobile/keyboard/deep links, transport heartbeat through the public proxy, image PUT/view/reload and real voice/TURN. Verify public `/internal` routes are denied and upstream logs do not record token-bearing paths/query strings. MCP401 alone proves liveness, not successful authenticated operations.
7. Record canary evidence, migration/image IDs and remaining limitations. On failure, disable new promotions and use the current-release recovery procedure; never deploy an old writer against promoted history. Publish final human and agent-readable docs only with accurate deployment status.

The existing v1 instructions above are retained for historical context; they are not the v2 content or authorization acceptance contract.

The 2026-10-19 deployment completed the fresh backup, four-secret installation,
Linux/amd64 build, forward migration and four-service rollout. The database is
at migration 13 with `dirty=false`; `COLLAB_V2_ENABLED=true`. Exact image IDs,
cutover/rollback checks and completed public canaries are recorded in the
[release note](../releases/2026-10-19-collaboration-v2.md) and
[backup evidence](../work/verification/collaboration-v2-backup.md).

Docker Compose v5.6.0 exposed a YAML flow-sequence parsing problem for tmpfs
values containing commas. The manifest now uses block-sequence scalars; inspect
`HostConfig.Tmpfs` after future Compose upgrades because config validation alone
did not catch the invalid mount split. See
[VM100 tmpfs verification](../work/verification/compose-tmpfs-vm100-2026-10-19.md).

An installed Homelab systemd recovery unit waits for Swarm/network metadata and
re-applies the Compose stack. A controlled VM reboot verified that it restores
the two external overlay networks, local application bridge, all four healthy
services and public health without manual intervention.

Frontend nginx resolves backend, collaboration and MCP service names at runtime
through Docker's embedded DNS. A production collaboration-container address
change preserved public WebSocket upgrade and authenticated admission without a
frontend restart. Keep the isolated address-replacement smoke in the release
gate for future nginx/base-image changes. See
[DNS recovery evidence](../work/verification/nginx-docker-dns-recovery-2026-10-19.md).
