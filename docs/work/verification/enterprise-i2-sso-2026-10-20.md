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

