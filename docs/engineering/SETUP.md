# Local setup

The application source is under `src/`: a Go API, React/Vite frontend, Node
collaboration runtime, MCP adapter, and shared TypeScript packages. Use a
disposable PostgreSQL database and S3-compatible bucket for integration work.

## Prerequisites

- Go and Node/npm versions compatible with the checked-in `go.mod` and
  `src/package-lock.json` (the frontend pins TypeScript 7.0.2).
- PostgreSQL and S3-compatible storage for a real stack. Focused frontend tests
  can run without them.
- Docker Compose for production-manifest checks; Playwright browser binaries for
  browser tests.

## Prepare the checkout

```sh
cd src
npm ci
npm run build:contracts
cp .env.example .env.local
```

Replace example values in the untracked local file with disposable database,
storage, and distinct development secrets. `src/.env.example` is an inventory
with placeholders; it is not directly sourceable and currently uses some
legacy S3 names. For the Go server, provide `S3_ENDPOINT`, `S3_ACCESS_KEY`,
`S3_SECRET_KEY`, and `S3_BUCKET` alongside the required keys checked in
`src/backend/cmd/server/main.go`. Export the corrected file's values into the
server process environment explicitly; the Go binary does not load `.env.local`
itself. Keep `VITE_API_URL` unset to use the Vite `/api` proxy and set
`FRONTEND_URL=http://localhost:5173`. Collaboration and MCP also need private
service credentials described in [Environments](../operations/ENVIRONMENTS.md).

The root `docker-compose.yml` starts only a disposable Coturn service. The
production manifest depends on existing Homelab networks and private
credentials, so it is not a local all-service bootstrap. For real integration,
use the isolated [real-stack harness](../../tests/recovery/README.md).

## Verify before development

```sh
cd src/backend
go test ./...
cd ../
npm run test:unit --workspace ./frontend
npm run build --workspace ./frontend
```

Database-backed Go tests use optional `ACCESS_TEST_DATABASE_URL` and must
target a dedicated disposable database. See [local workflow](LOCAL_DEVELOPMENT.md)
for browser commands and [troubleshooting](TROUBLESHOOTING.md) for setup failures.
