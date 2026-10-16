# Environments

| Environment | Purpose | URL | Owner | Notes |
| --- | --- | --- | --- | --- |
| local | Development and mocked browser verification | `http://localhost:5173` | shared | `src/.env`; Vite proxies API to `:8080` |
| production | Public self-hosted release | `<public-origin>` | human | Docker Compose; HTTPS termination is required |

## Secrets

Document secret names only. Do not store secret values.

- `DATABASE_URL`
- `JWT_SECRET`
- `S3_ENDPOINT`, `S3_ACCESS_KEY`, `S3_SECRET_KEY`, `S3_BUCKET`, `S3_REGION`
- `FRONTEND_URL`
- Password users are provisioned with the backend `userctl` CLI; no Google OAuth credentials are required.
- `AI_API_KEY`, `AI_BASE_URL`, `AI_MODEL_FAST`, `AI_MODEL_REASONING`
- `TURN_URL`, `TURN_USERNAME`, `TURN_PASSWORD`, `TURN_AUTH_SECRET`, `TURN_REALM`, `TURN_MIN_PORT`, `TURN_MAX_PORT`

## A4 private mutation authority (backend verified; deployment pending)

Require separate >=32-byte `AUTHORITY_SERVICE_SECRET` and `MUTATION_GRANT_SECRET`, each different from `JWT_SECRET`, `COLLAB_JWT_SECRET`, and one another. `AUTHORITY_LISTEN_ADDR` defaults to `127.0.0.1:8081`; only explicit loopback/private IP binds are accepted. Expose this listener to trusted collab peers only and never in a public reverse proxy. Deployers must supply both new secrets before startup. Decision TTL is five seconds maximum; storage upload/view URLs use at most 60 seconds. TURN scene configuration requires `TURN_AUTH_SECRET` when `TURN_URL` is configured and returns short-lived credentials after scene authorization. See [protocol](../architecture/mutation-authority.md) for exact byte/hash and lock contract. Real storage signature enforcement and ingress isolation require E-stage deployment checks.


## MCP audit storage

The HTTP MCP process requires `MCP_AUDIT_PATH`, an absolute path on a persistent writable volume. Give each process its own path; do not share a journal between replicas. Files are mode 0600, and the default retention is an active file plus two archives at 1 MiB each. Collect these records centrally before rotation expires them. No installer or database is required in a remote stdio adapter; its default journal is `~/.local/state/whiteboard-mcp/audit.jsonl`, overridable with a unique `MCP_AUDIT_PATH`.

Migration `000013_agent_operation_audit` adds authoritative Go operation records to PostgreSQL. Apply migrations before enabling updated agents. Creation and success audit commit atomically; failed operations write bounded metadata after rollback. Adapter journals have `adapter_observed` provenance and include local validation/upstream failures. Client-controlled stdio journals are not authoritative. Correlate `_meta.agentCallId`/`X-Agent-Call-ID` with database `correlation_id`; these UUIDs never confer authorization.

Audit storage failure before a call prevents its operation. `AUDIT_COMPLETION_UNCERTAIN` means completion could not be journaled: inspect authoritative state before retrying non-idempotent creation. D3 operations with durable deduplication may retry their exact stable request ID and bytes. Synchronous fsync increases call latency; retain PostgreSQL audit history according to operator policy and collect adapter journals for full-call coverage. There is no automatic database audit pruning in migration 13.

## Durable agent bridge (D3)

Go accepts paired `COLLAB_INTERNAL_URL` and `COLLAB_INTERNAL_SECRET`. Set both to enable live agent reads/commands/activity/export projections; leaving both unset returns503 for these paths and never falls back to stale materialized scene content. Supplying only one fails startup. The secret must contain at least32bytes and differ from login, room, authority and mutation-grant secrets. The URL must be an HTTP(S) origin with no credentials, query, fragment or non-root path; redirects are refused and requests are bounded. Use the private collab listener (default127.0.0.1:8083) reachable only within the service boundary. This credential is shared by Go/collab only; MCP and remote stdio clients use their workspace-scoped agent key and never receive internal service credentials. Production ingress must deny all `/internal` routes.
