# Production Deployment

Status: this procedure describes the collaboration-v2 release candidate and its cutover from the existing v1 deployment. Until the recorded cutover finishes, production remains v1. After promotion, use the [replacement-scene recovery runbook](collaboration-recovery.md); image-only rollback is not valid for a promoted v2 scene.

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
`FRONTEND_URL`, `AI_API_KEY`, `TURN_URL`, `TURN_AUTH_SECRET`,
and `TURN_REALM`. Also configure distinct values of at least 32 bytes for
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
plan. A3 is implemented and locally verified on merged master; the existing
production deployment remains v1.
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


## V2 rollout procedure (prepared, not yet executed)

1. Complete task/whole-branch review and the current release checklist. Build frozen-source Linux/amd64 images for backend, collab, MCP and frontend; record their exact IDs. Keep the v1 manifest/environment/images captured in the private release directory.
2. Preserve existing external networks and verify the dedicated authority subnet remains unused. Generate four separate v2 secrets in the private environment, each at least32bytes and distinct from the login secret. Set both public origins to the public HTTPS app. Use the tested `turn:103.82.21.202:5349` endpoint with the existing matching TURN secret/realm. Start with `COLLAB_V2_ENABLED=false`.
3. Take a fresh cutover database backup. Validate the manifest using `docker compose --env-file .env.prod -f docker-compose.prod.yml config --quiet`; never print expanded configuration. Apply migrations7–13as one forward stream and verify version13with dirty=false. No partial stream or down migration is permitted.
4. Start backend, collab and MCP; verify Go health, collab readiness/ownership and MCP liveness. Recreate frontend after upstream changes so nginx resolves current addresses. Do not run `down -v` or replace shared overlay networks. Frontend remains portless and retains its established alias.
5. Enable new promotions for the canary window, promote a disposable test scene and exercise two-browser edits, guest view/edit, invitation/role/revoke, offline recovery, native agent write/export, restart and denied requests. A single global flag is not a scene allowlist; keep the window controlled and monitor any other promotions. Existing promoted scenes retain the v2 writer even if new promotions are later disabled.
6. Check native UI desktop/mobile/keyboard/deep links, transport heartbeat through the public proxy, image PUT/view/reload and real voice/TURN. Verify public `/internal` routes are denied and upstream logs do not record token-bearing paths/query strings. MCP401 alone proves liveness, not successful authenticated operations.
7. Record canary evidence, migration/image IDs and remaining limitations. On failure, disable new promotions and use the current-release recovery procedure; never deploy an old writer against promoted history. Publish final human and agent-readable docs only with accurate deployment status.

The existing v1 instructions above are retained for historical context; they are not the v2 content or authorization acceptance contract.
