# Enterprise I2 OIDC verification record

Status: local implementation verified; production wiring and real IdP UAT pending

## Required proof

- State, nonce, PKCE, browser binding, expiry and replay are enforced.
- Discovery rejects non-HTTPS/private-network/redirected issuer endpoints.
- ID tokens reject wrong issuer, audience, nonce and expiry.
- Same-email identities do not link accounts without an authenticated link.
- Password or legacy sessions cannot enter an enforced workspace.
- Revocation blocks new room, command, asset and media credentials.
- Enforcement requires recent owner authentication and tested operator
  break-glass recovery.
- Lost-response retries return the original result for the same principal,
  action and canonical body; cross-principal/action/body reuse is rejected.
- Fixture-provider browser evidence is labeled fixture evidence. Real provider
  UAT remains an operator release dependency.

## Verification evidence

- Focused handler/usecase/OIDC/PostgreSQL/middleware suite: 266 tests passed.
- Guarded PostgreSQL migration/repository SSO tests passed against disposable
  PostgreSQL, including legacy-table preservation, up/down migration, one-time
  transaction/exchange and exact retry/cross-action/body rejection.
- Focused frontend auth/settings unit suite: 8 tests passed; TypeScript/Vite
  production build passed.
- Official-verifier fixture tests cover signature, issuer, audience, expiry and
  nonce; SSRF tests cover non-HTTPS, private address and discovery redirect.

- Full backend suite: 301 tests passed across 14 packages; `go vet ./...` passed.
- Full frontend unit suite: 170 tests passed across 43 files; TypeScript/Vite
  production build passed. Vite emitted existing large-chunk advisory warnings.

Browser fixture proof depends on root production wiring and remains a release
gate. Unit fixture evidence does not prove a real external IdP configuration.
Production release still requires its exact issuer, client, secret and redirect
UAT.

## Production wiring reconciliation

The production server now constructs the session repository and SSO repository
before access policy, uses `NewAccessServiceWithAssurance` and
`AuthMiddlewareWithSessions`, and injects the same session authority into
password login and OIDC. The enterprise router registers public begin/callback/
exchange routes plus authenticated link and workspace configuration routes.
Startup requires a distinct exact 32-byte `SSO_ENCRYPTION_KEY` and an exact
HTTPS `/api/v1/sso/callback` URL. The owner dashboard exposes the workspace SSO
settings route. Router/config focused tests pass. Browser IdP fixture and
external-provider UAT remain open release evidence.

## Security remediation plan

The independent audit found five current-authority gaps. The remediation keeps
assurance centralized in `AccessService`, validates every login session at the
authentication boundary, and adds pre-write scene authorization. Owner/member,
invitation, feature, domain, room, command, media and asset operations must all
fail after source-session revocation.

Break-glass sessions will persist and sign one exact workspace, expire after 15
minutes, and be rejected for every other, optional or unconfigured workspace.
The CLI will read the recovery credential from protected terminal input or a
caller-selected file descriptor; command-line secret arguments are removed.

Authenticated link transactions will persist the source session as well as the
workspace/user/request binding. Callback completion will lock the workspace and
recheck that source session, current membership and assurance before linking or
issuing a replacement OIDC session. Exchange will re-read the session under the
same workspace lock and reject revoked, expired or stale-connection sessions.

Changing issuer, client ID or provider secret will revoke every session for the
connection in the configuration transaction. Verification adds negative unit
matrices plus guarded PostgreSQL race tests for reconfiguration, exchange and
current room/asset authority.

## Security remediation result

The implementation now validates `sid` at the authentication boundary and
composes workspace assurance through `AccessService`, including owner, member,
feature, domain, invitation and scene operations. Scene creation authorizes
before persistence. Negative tests cover the workspace operation matrix plus
current room and asset capability minting after revocation.

Link transactions persist the source session. Callback and exchange re-read the
session, membership and connection while holding the workspace lock. Provider
identity or secret changes revoke existing connection sessions in that same
transaction. A guarded PostgreSQL concurrency test proves rotation waits for an
in-flight workspace authority lock before revocation becomes visible.

Recovery sessions persist their exact workspace and use a 15-minute lifetime.
The login token carries the same scope, and assurance rejects recovery tokens
outside that enforced workspace. `userctl` reads recovery credentials from a
hidden terminal or bounded file descriptor using `golang.org/x/term`; its BSD
license and transitive `x/sys` dependency were audited with the module update.

