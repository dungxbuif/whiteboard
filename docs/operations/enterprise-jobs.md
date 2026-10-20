# Enterprise feature worker

The feature worker runs bounded asynchronous metadata jobs for enterprise
features. It uses the backend image and the `/app/feature-worker` entrypoint.
It does not write canonical Yjs scene state; Node remains the only durable scene
writer.

## Runtime contract

- Required environment: `DATABASE_URL`, `FEATURE_WORKER_ID`, and
  `FEATURE_WORKER_ADDR`. Production Compose passes those variables explicitly;
  the worker does not inherit account, collaboration, TURN, AI, or unrelated S3
  secrets from the application environment.
- Production Compose binds health endpoints to `127.0.0.1:8090` inside the
  container. `/healthz` proves the process is alive; `/readyz` is successful only
  after a database claim cycle succeeds.
- One run processes at most 100 due rows. It claims each row with `FOR UPDATE
  SKIP LOCKED` immediately before its handler, using a 60-second single-job
  lease, so later work never waits behind another handler with an aging lease.
  Each operation has a 30-second deadline: handlers receive the first 25 seconds
  and the final five seconds are reserved for completion, retry, or dead-letter
  persistence.
- Failures retry after 1, 2, 4, 8, 16, 32, then 60 minutes. Ten claimed attempts
  move the row to dead letter. Handler side effects must be idempotent because a
  process may stop after the effect and before completion is recorded.
- Shutdown marks readiness false and prevents the next claim. The active
  operation retains its bounded operation context; the handler yields before the
  reserved bookkeeping window, and the process allows up to 30 seconds for the
  handler plus completion/retry write to drain.
- Stop and claim admission share one serialized gate. Shutdown closes that gate
  before starting the drain timer, so a loop already preparing its next poll
  cannot begin a new database lease after shutdown admission.

## Operations

Check state without exposing the worker publicly:

```sh
docker compose -f docker-compose.prod.yml ps feature-worker
docker compose -f docker-compose.prod.yml exec feature-worker \
  wget -qO- http://127.0.0.1:8090/readyz
```

Inspect counts and error classes through database/operator tooling; do not log
job payloads because future handlers may carry private identifiers. An unready
worker does not roll back already committed web mutations. Restore database
access, confirm `/readyz`, then check that expired leases resume and idempotent
handlers do not duplicate effects.

Destructive retention handlers remain disabled until the retention policy,
hold, reference, backup and seven-day activation safeguards ship and pass their
release gates.
