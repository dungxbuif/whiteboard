# Enterprise M2 presentation admission verification — 2026-10-20

## Intended boundary

M2 adds presentation-specific links and credentials without reusing public scene
links. PostgreSQL stores only hashed, revocable 24-hour invite links. A join
returns a five-minute `presentation-pending` credential that can receive only
its own waiting status. It cannot read the scene, mint room/asset/media
credentials, receive roster data, or send binary frames.

The collaboration owner keeps active sessions and the bounded 100-person
waiting queue in memory. Restart invalidates active sessions. Host disconnect
ends its session. Admission requires an audience credential bound to the same
workspace, scene, session, and pending principal. Current Go feature and scene
authority are checked before credentials are issued; disabling or revoking an
invite cannot affect independent workspace membership or public links.

## Verification plan

- Go service/repository/HTTP tests cover link hashing, expiry/revision, revoke,
  idempotency, cross-scene denial, pending-token purpose, and the 60/minute/IP
  join boundary.
- Node tests cover the waiting cap, metadata-only waiting messages, binary and
  media denial, audience binding, and host-disconnect teardown.
- Migration 000020 is exercised on the guarded disposable database. Router and
  server construction remain a root integration step after this slice commits.

## Executed evidence

- Focused Go presentation/capability/middleware/repository tests: 27 passed.
- Guarded full Go suite: 232 passed in 12 packages.
- Focused collab presentation tests: 4 passed; shared protocol tests: 8 passed.
- Guarded full collab suite: 183 passed in 24 files; shared contracts: 21 passed
  in 2 files.
- `git diff --check` passed.
- `go vet ./...` reached an unrelated concurrent collection slice and failed
  because `collection_repo_test.go` references its not-yet-present
  `NewCollectionRepository`; M2 packages compile and the full Go test suite
  passes.

Root integration constructs `PresentationService` with the Postgres repository,
feature service, access service and `COLLAB_JWT_SECRET`; sets the existing
remote collab `AgentClient` as runtime verifier; constructs
`PresentationHandler`; registers its routes inside `/api/v1`; and calls
`CapabilityHandler.SetPresentationAuthority` so admitted viewers can fetch
scoped assets. No new environment variable is required.

