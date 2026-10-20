# Enterprise V1 version checkpoint verification — 2026-10-20

Scope: immutable manual and automatic scene checkpoint capture, asset pins,
Go history API boundary, private Node bridge, PostgreSQL migration 000018 and
human/agent documentation.

## Evidence ledger

| Gate | Command / evidence | Result |
| --- | --- | --- |
| Focused Node/Yjs/PostgreSQL | `COLLAB_TEST_DATABASE_URL=postgres://dungxbuif@127.0.0.1:55439/devtool_a1_test?sslmode=disable npx vitest run versions version-checkpointer` | PASS: 2 files, 11 tests. Exact sequence/hash, immutable retry, atomic image pin rollback, concurrent compaction survival, one-minute floor, five-minute coalescing, durable dirty-scene recovery, default-disabled suppression and enabled-to-disabled races. |
| Full backend | `ACCESS_TEST_DATABASE_URL=postgres://dungxbuif@127.0.0.1:55439/devtool_a1_test?sslmode=disable go test ./...` | PASS across command, HTTP, repository, S3 and usecase packages. |
| Migration/repository focus | `ACCESS_TEST_DATABASE_URL=... go test ./internal/repository/postgres -run TestVersionRepositoryReadsContentAndGuardsMetadataRevision -count=1` | PASS. Migration 000018 applied in a disposable random schema; canonical content/read, metadata conflict and immutable-content trigger verified. |
| Guest denial / Go binding | `go test ./internal/usecase ./internal/delivery/http/handler` | PASS. Public principal is rejected before repository access; login JWT expiry reaches capability issuance; create grant binds canonical payload, request ID and exact sequence; nil bridge returns typed `503 UNAVAILABLE`; GET mints fresh scoped asset URLs; label/pin writes emit safe metadata-only events. |
| Full collab suite | `COLLAB_TEST_DATABASE_URL=... npm test --workspace ./collab` | PASS: 22 files, 179 tests, including TypeScript build and policy-race regressions. |

Commits containing the work are `ac76f8d` (initial V1 snapshot, captured by a
shared-index coordination incident) and `6085a15` (durable recovery scan and
canonical-bytes constraint). No M1 room/voice/protocol files were staged by V1.

## Runtime integration

The production server constructs `VersionRepository` and `VersionService`,
injects the current scoped asset and workspace-event services, and registers
`VersionHandler` below `/api/v1` with login authentication. The same private
collaboration client used by durable agent commands is passed to the version
service when the configured bridge is available. List/get remain readable from
PostgreSQL when the optional bridge is disabled; create fails closed.
No public Node route is required: `server.ts` already owns the service-authenticated
`POST /internal/scenes/{sceneId}/versions` bridge and checkpointer lifecycle.
