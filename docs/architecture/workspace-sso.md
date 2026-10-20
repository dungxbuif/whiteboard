# Workspace OIDC SSO and session assurance

Status: implemented locally (I2); production wiring and external IdP UAT remain

## Intended behavior

Workspace owners can configure one OIDC connection in `optional` or `enforced`
mode. A connection is identified by its exact HTTPS issuer and client ID. Login
identity is `(issuer, subject)`; an email claim, even when verified, never links
an existing account automatically. Linking requires an authenticated user and a
single-use, idempotent link operation. Linking starts from an authenticated
session and binds that user plus the request ID into the browser-bound OIDC
transaction. Browser requests never supply trusted provider claims.

The authorization-code flow uses state, nonce and PKCE. The browser receives an
opaque one-time exchange code in the callback URL, never an application bearer
token. Transactions expire after ten minutes; exchange codes expire after sixty
seconds and both are stored only as SHA-256 hashes bound to the initiating
browser. Provider secrets are encrypted with the backend SSO key.

Login JWTs carry an opaque `sid`. Workspace authorization rechecks that session
against current connection mode and revocation state. Existing JWTs without a
`sid` remain valid only for password-allowed workspaces during migration. An
enforced workspace rejects password and legacy sessions, while public links
continue to use their own explicit scene policy and never inherit member SSO.

## Authority and recovery

OIDC discovery accepts only public HTTPS issuers and rejects redirects or
discovered endpoints that leave the issuer origin or resolve to private,
loopback, link-local, multicast or unspecified IP space. ID tokens are verified
with the official `coreos/go-oidc/v3` verifier for issuer, audience, signature,
expiry and nonce.

Changing connection mode, revoking sessions or linking identities occurs under
the workspace authority lock and rechecks current ownership. Enabling enforced
mode requires recent owner authentication and a tested operator-created
break-glass session. Break-glass credentials are created only by `userctl`, are
audited and cannot be created through future SCIM routes.

## Data and integration

Migration `000025_workspace_sso` adds encrypted connection configuration,
hashed flow transactions, external identities, revocable login sessions,
one-time exchanges, mutation receipts and break-glass audit rows. The backend
constructs `SSOService` and `SessionService`, injects session assurance into the
access service, registers the SSO handler, and lets the password auth use case
create password sessions.

Room, asset, media and command grants all pass through current workspace access;
therefore revoked or insufficient sessions cannot mint or renew downstream
credentials. Existing grants remain bounded by their current short TTL and ACL
checks.

The enterprise sessions use `enterprise_login_sessions`. The legacy
`auth_sessions` table from migration `000001` has refresh-token semantics and is
left intact for upgrade compatibility.

Authenticated linking starts at `POST /api/v1/sso/link/begin`. The callback
may link only the user stored in that consumed transaction and only after the
provider identity passes signature, issuer, audience, expiry, nonce, PKCE and
browser-binding verification.

## Dependency decision

Use `github.com/coreos/go-oidc/v3` v3.21.0. It is Apache-2.0 licensed and its
official verifier validates signed ID tokens and standard issuer/audience/time
claims. OAuth2 PKCE uses the existing `golang.org/x/oauth2` dependency.

