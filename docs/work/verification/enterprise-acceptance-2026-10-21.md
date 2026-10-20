# Enterprise parity acceptance evidence

Status: local real-stack acceptance is progressing; production release remains
blocked on the documented E1/E2 external-provider and VM100 gates.

## Automated evidence

- Go backend: `go test ./...` — 345 passed in 14 packages.
- Go static checks: `go vet ./...` — passed.
- Frontend unit: `npm run test:unit` — 199 passed in 48 files.
- Frontend production build: `npm run build` — TypeScript 7 and Vite build passed.
- Collaboration runtime: guarded PostgreSQL `npm test --workspace @collab/runtime
  -- --run --maxWorkers=1 --no-file-parallelism` — 210 passed in 27 files.
- Real-stack browser flows against the guarded disposable PostgreSQL database:
  comments 1/1 passed, live presentation 2/2 passed, and the earlier library
  and collections cases passed after root bridge wiring.
- Canvas controls browser smoke: 2/2 passed at mobile and desktop viewports.

The focused three-case rerun (`enterprise-collections|enterprise-live`) passed
3/3. The aggregate eight-case run reached 6/8 twice: collections deep-link and
the first live case timed out only when run after the activity case; both pass
when isolated. This is recorded as a harness teardown/flakiness release gate,
not as a product pass.

## Release boundary

The local harness uses an ephemeral database schema, SSO encryption key, and
loopback callback. It proves behavior and authorization on the assembled Go,
collaboration, PostgreSQL, and browser stack; it does not prove a real IdP
tenant, production object storage, the VM100 deployment, or two-network media.
Those, plus a stable 8/8 aggregate E1 run, remain release gates and must be
recorded in the production rollout evidence before the enterprise phase is
marked shipped.
