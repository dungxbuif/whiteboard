# Workspace SSO operations

Set a backend-only 32-byte `SSO_ENCRYPTION_KEY` and the exact public HTTPS
`SSO_CALLBACK_URL` (`https://<origin>/api/v1/sso/callback`). Rotate the provider
secret through the owner settings; it is encrypted at rest and never returned.
Changing the backend encryption key requires decrypting and resealing every
connection in a maintenance window before restarting with the new key.

Create and test recovery before enforcement:

```sh
DATABASE_URL=... userctl --email owner@example.com \
  --break-glass-workspace <workspace-uuid> --break-glass-action create
DATABASE_URL=... userctl --email owner@example.com \
  --break-glass-workspace <workspace-uuid> --break-glass-action test \
  --break-glass-credential '<shown-once-value>'
```

The credential is shown once; store it in the existing secret manager. Creating,
testing, revoking and using recovery is audited. Supply it through protected
terminal input or `--break-glass-credential-fd`; never place it in argv or shell
history. Revoke it with `--break-glass-action revoke`.

During an actual provider outage, run recovery only from the trusted operator
host. `JWT_SECRET` must match the backend and contain at least 32 bytes:

```sh
DATABASE_URL=... JWT_SECRET=... userctl --email owner@example.com \
  --break-glass-workspace <workspace-uuid> --break-glass-action recover
```

The command prints one short-lived bearer token after the transaction commits
and writes a `used` audit event. Handle that output as a secret, resolve the
incident, then revoke and replace the recovery credential.

Rollout order: apply migration `000025`; configure secrets; leave every
connection optional; complete issuer/client/redirect UAT; create and test
recovery; revoke a disposable SSO session and confirm room/upload/media/command
renewal fails; then enable enforced mode for one canary workspace.

On provider outage, keep other workspaces on password login. For an enforced
workspace, use the tested operator recovery path. Do not weaken issuer, audience,
nonce, PKCE, browser binding or private-network discovery checks.

## Production wiring checklist

- Construct `AESGCMCipher` from exactly 32 key bytes, `SSORepository`,
  `AuthSessionRepository`, `SessionService`, the OIDC provider factory and
  `SSOService`.
- Construct access with `NewAccessServiceWithAssurance`; the legacy constructor
  does not enable current-session assurance.
- Set `OAuthConfig.SessionIssuer` so password login receives a `sid`, then
  register `SSOHandler` below `/api/v1`.
- Keep the callback byte-for-byte equal to the IdP allowlist entry. Do not use a
  private origin, wildcard or redirecting URL.
- Apply migration 25 on a backup-tested copy. It adds
  `enterprise_login_sessions` and leaves legacy `auth_sessions` unchanged.

Alert on discovery failures, callback replay/expiry, revoked-session denials and
break-glass use. Do not log state, nonce, verifier, exchange codes, provider
secrets or recovery credentials.

