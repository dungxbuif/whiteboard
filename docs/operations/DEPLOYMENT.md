# Production Deployment

## Deployment Target

The release bundle targets one Linux host or VM running Docker Compose. The
frontend nginx container serves the built SPA and proxies `/api/` and
`/socket.io/` to the Go backend. PostgreSQL, S3-compatible storage, and Coturn
are external services supplied through environment configuration. The shared
Homelab gateway runs Coturn on `turn.relayhub.dungxbuif.com:5349`; the release
stack must not start another TURN listener.

## Required secrets and settings

Create an untracked `.env.prod` at the repository root from `src/.env.example`.
Set production values for `DATABASE_URL`, `JWT_SECRET`, S3 credentials,
`FRONTEND_URL`, `AI_API_KEY`, `TURN_URL`, `TURN_USERNAME`, `TURN_AUTH_SECRET`,
and `TURN_REALM`. `TURN_AUTH_SECRET` must match the
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
2. Build and start the migration job, backend, and frontend:
   `docker compose --env-file .env.prod -f docker-compose.prod.yml up -d --build`.
   The production manifest attaches the services to the pre-created
   `homelab-net`, `whiteboard-app`, and `edge-http-net` networks; the frontend
   stays portless and is reached through the public gateway.
3. Confirm the migration job exits successfully before accepting traffic.
4. Verify `/healthz`, password sign-in, an owned workspace request, scene save,
   an authenticated deep link at `/w/{workspaceId}/scenes/{sceneId}`, public
   share viewer, a JWT-authenticated `/socket.io/` connection that can join
   only an owned workspace room, TURN allocation from an external
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
plan. A3 is implemented on the development branch; the existing release steps above remain v1.
Public refresh returns a renewed scene session before expiry, capped to link
expiry. A monotonic link revision invalidates existing sessions after password
or link changes. Release wiring must pass COLLAB_JWT_SECRET to both Go and
the collaboration service, and redact legacy token paths/query strings in
reverse-proxy and tracing logs. Go strips legacy query strings and redacts
RequestURI; routing still retains URL.Path. Credential SQL queries suppress
parameter logging.
