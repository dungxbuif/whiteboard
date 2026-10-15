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
3. Confirm the migration job exits successfully before accepting traffic.
4. Verify `/healthz`, password sign-in, a workspace request, scene save, public
   share viewer, `/socket.io/` connection, TURN allocation from an external
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
