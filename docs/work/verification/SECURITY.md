# I2 OIDC and session-assurance security audit

Status: REMEDIATED_PENDING_PRODUCTION_WIRING

Scope: commit `840d95e` and the I2 contracts in the approved enterprise identity
plan/design. The plan has no machine-readable `<threat_model>` block, so the
register below is derived from its Review Focus and the binding contracts in
`2026-10-20-enterprise-parity-design.md` lines 267-294. ASVS review baseline: L2.

## Threat verification

| Threat ID | Disposition | Status | Evidence |
| --- | --- | --- | --- |
| I2-T01 OIDC request forgery/replay | mitigate | CLOSED | `sso_ucase.go:105-165`, `sso_repo.go:101-123,170-193`: random state/nonce/PKCE, hashed browser binding, ten-minute transaction and sixty-second exchange, locked single-use consumption. |
| I2-T02 forged or invalid provider identity | mitigate | CLOSED | `provider.go:125-148`: official go-oidc verifier checks signature/issuer/audience/time and code checks exact nonce plus verified email. |
| I2-T03 issuer/discovery/JWKS SSRF | mitigate | CLOSED | `provider.go:80-112,151-185`: HTTPS-only issuer, private/special IP rejection on every dial, pinned resolved IP, redirects disabled, same-origin discovery endpoints and bounded HTTP timeout. |
| I2-T04 existing-email account takeover | mitigate | CLOSED | `sso_ucase.go:180-200,257-295`: unlinked same-email login is rejected and linking uses a browser-bound transaction created by an authenticated user. Current-authority race remains separately open as I2-T09. |
| I2-T05 provider-secret disclosure | mitigate | CLOSED | `provider.go:26-65`, `sso_ucase.go:305-347`: 32-byte AES-GCM key, random nonce, encrypted secret persistence, secret digest only in canonical receipt input, secret omitted from JSON responses. |
| I2-T06 public links inheriting member assurance | mitigate | CLOSED | `auth_session_ucase.go:107-118`, `access_ucase.go:114-127`: public links use the explicit per-workspace switch and never a member session. |
| I2-T07 legacy/password session entering enforced workspace | mitigate | CLOSED | `auth_session_ucase.go:62-105`: no-sid/password/wrong-connection sessions fail for enforced workspaces while remaining usable elsewhere. |
| I2-T08 revoked SSO identity retaining HTTP authority | mitigate | CLOSED | `AuthMiddlewareWithSessions` rechecks every session-backed JWT, `AccessService.RequireOwner` composes assurance for the current request principal, and scene creation authorizes before writing. `TestRevokedSessionDeniedAcrossWorkspaceAuthorityOperations`, `TestRevokedSessionCannotMintCurrentRoomOrAssetAuthority` and the scene handler negative test cover the authority matrix. Production must use the session-aware middleware constructor. |
| I2-T09 link completion after revocation/removal | mitigate | CLOSED | Link transactions persist `source_session_id`; callback holds the workspace lock while rechecking the source session, current assurance, membership and connection before linking or issuing a session. Negative tests cover revoked source sessions and removed membership. |
| I2-T10 scoped break-glass recovery | mitigate | CLOSED | Recovery sessions and JWTs bind the exact workspace, expire after 15 minutes, and are rejected for optional, unconfigured and different workspaces. Migration constraints reject unscoped recovery rows. |
| I2-T11 revoked one-time exchange | mitigate | CLOSED | Exchange locks the session workspace and rechecks current session expiry/revocation, method, user and current connection before issuing the bearer token. Revoked-session exchange has a negative test. |
| I2-T12 provider reconfiguration invalidates old assurance | mitigate | CLOSED | Issuer, client ID or secret changes revoke all sessions for the connection inside the configuration workspace lock. The guarded PostgreSQL concurrency test verifies serialization against in-flight workspace authority. |

## Unregistered flags

- Break-glass credentials are read from a hidden terminal or protected file
  descriptor; the argv credential option and operations example were removed.
- Production constructors/routes and the browser IdP fixture are intentionally
  absent from commit `840d95e`; this audit does not treat unwired code or unit
  fixtures as production evidence.

## Required remediation

All five required remediations are implemented and covered by focused tests.
Production wiring and real external IdP UAT remain release gates rather than
code-remediation findings.

## Verification run

- `go test ./internal/usecase ./internal/delivery/http/middleware ./internal/delivery/http/handler ./cmd/userctl`: 283 passed.
- Guarded PostgreSQL `go test -race ./internal/repository/postgres -run 'TestSSO|TestAuthSession' -count=1`: passed, including workspace-lock serialization.
- `go test ./...`: 324 passed across 14 backend packages; `go vet ./...`: clean.
- Guarded PostgreSQL full backend `go test ./... -count=1`: passed; complete repository race suite `go test -race ./internal/repository/postgres -count=1`: passed in 45.9 seconds.
- Dependency pin: `github.com/coreos/go-oidc/v3 v3.21.0`, module checksum present; cached module license is Apache-2.0.
- Protected terminal input: `golang.org/x/term v0.43.0`, BSD-3-Clause; transitive `golang.org/x/sys v0.45.0`, BSD-3-Clause.

