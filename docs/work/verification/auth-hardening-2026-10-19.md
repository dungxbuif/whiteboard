# Backend authentication hardening — 2026-10-19

Status: implemented and verified

## Intended behavior

- The API refuses to start unless `JWT_SECRET` contains at least 32 bytes. Login, collaboration, authority, and mutation secrets remain distinct.
- Every production route protected by the login JWT middleware revalidates the subject against the active-user store. A missing, soft-deleted, or unreadable user record fails closed with the standard `401 UNAUTHENTICATED` response before workspace, AI, invitation, room-token, asset, or other authenticated handlers run.
- Password login performs one bcrypt comparison for both known and unknown accounts. Unknown accounts use a fixed dummy bcrypt hash so account lookup does not skip the expensive password check.
- Password-login JSON is capped at 4 KiB, rejects unknown fields and multiple JSON values, and reaches neither bcrypt nor account lookup when malformed.
- Invitation registration validates token shape and current invitation state before password hashing. Password hashing happens outside the workspace authority lock, then the invitation is fetched and validated again under the lock before user creation and membership changes.
- Password login, invitation registration, and public scene-session exchange are throttled by both client IP and a privacy-preserving digest of the submitted account or token. Exhausted limits return a stable JSON `429 RATE_LIMITED` response with `Retry-After`.
- The in-memory limiter uses a five-minute fixed window, HMACs account and token identifiers with an ephemeral process key, has a 20,000-entry hard cap, and expires idle entries so attacker-controlled identities cannot grow memory without bound. Login allows 30 attempts per IP and 10 per account; invitation registration allows 20 per IP and 5 per token; public scene exchange allows 60 per IP and 15 per token. It is intentionally per API process; deployment-level distributed limiting remains a reverse-proxy concern.

## Technical decision and affected flow

- Extend the login JWT middleware with an active-user lookup dependency and wire it from the production user repository. Existing test-only callers may omit the dependency, while the production server always supplies it. The same middleware instance protects the main authenticated route group plus direct room-token and upload-capability handlers.
- Add a small bounded fixed-window limiter in HTTP middleware. Route-specific middleware extracts only `email` or `token`, hashes the normalized identifier, restores the request body, and evaluates both the IP and identifier buckets atomically.
- Add injectable password hashing to the invitation use case for deterministic security tests. The production constructor continues to use bcrypt at the default cost.
- No database schema or public success payload changes are required. The only new public response is HTTP 429 with `{code,message,requestId}` and `Retry-After`.

## Verification plan

1. Red phase confirmed: the focused command initially failed to compile for absent active-user, injectable hasher, and limiter APIs, while the startup test proved 5-byte and 31-byte `JWT_SECRET` values were accepted.
2. Focused suites passed for `cmd/server`, `internal/usecase`, `internal/delivery/http`, `internal/delivery/http/middleware`, and `internal/delivery/http/handler`.
3. `go vet ./...` passed with no findings.
4. `go test ./...` passed across all backend packages.
5. `go test -race ./...` passed across all backend packages, including the synchronized bounded limiter.

The tests prove malformed, unknown, revoked, and expired invitations do not invoke the password hasher; a valid invitation hashes exactly once outside the authority lock and is rejected when it becomes revoked before locked revalidation. Router coverage proves a soft-deleted JWT subject reaches neither workspace creation nor AI. Handler coverage proves each exhausted identity bucket returns 429 before its use case executes.

No schema or migration changed. Human user documentation does not need a workflow update because successful login, invitation, and public-link flows are unchanged. The API reference and agent-readable endpoint contract must include the new 429 response; that shared document is reconciled by the release documentation owner to avoid overwriting concurrent edits.
