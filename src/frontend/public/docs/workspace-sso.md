# Workspace SSO

## Modes

- `optional`: password, legacy and current SSO sessions may access the workspace.
- `enforced`: only a current session for the workspace's OIDC connection, or a
  tested operator break-glass session, may access member resources.

Existing JWTs without `sid` are transition credentials. They cannot access an
enforced workspace. Every JWT with a `sid` is checked against the current login
session before workspace authority is evaluated. Public links use their explicit
scene policy.

## Browser flow

1. `POST /api/v1/sso/begin` with `connectionId` and a relative `returnPath`.
2. Navigate to the returned provider URL. State, nonce, PKCE and the HttpOnly
   browser-binding cookie expire after ten minutes.
3. Provider redirects to `/api/v1/sso/callback`.
4. Callback redirects to `/auth/sso/exchange?code=...`; this is a one-time,
   browser-bound code valid for sixty seconds, not a login bearer token.
5. `POST /api/v1/sso/exchange`, store the returned login token, then remove the
   code from browser history.

## Identity and retries

Accounts link by exact `(issuer, subject)`. Verified email is required but never
auto-links an existing account. Configuration and link mutations bind request
ID, principal, action and canonical payload. Retry an uncertain response with
the same request ID; change it after a definitive result or explicit cancel.

Authenticated linking begins with `POST /api/v1/sso/link/begin`. Provider
profile claims are accepted only from the verified callback, never from a
browser-submitted identity object. Link completion rechecks the source session
and current workspace membership. Changing the issuer, client ID or provider
secret revokes sessions from the previous provider configuration.

## Operator dependency

Enforced mode requires recent owner authentication and a tested break-glass
credential created by `userctl`. Real identity-provider UAT is required before
production activation; the disposable test provider is fixture evidence only.
Recovery sessions last 15 minutes, apply only to the selected enforced
workspace, and are rejected by optional, unconfigured and other workspaces.
Enter the recovery credential at the hidden terminal prompt or supply a
protected file descriptor; never place it on the command line.

