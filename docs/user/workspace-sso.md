# Sign in with workspace SSO

Workspace owners can connect an OpenID Connect provider from the workspace
identity settings. Enter the exact HTTPS issuer, client ID and client secret,
then choose a mode:

- **Optional:** members may use SSO or their existing password session.
- **Enforced:** members must use this workspace's SSO connection. Password and
  legacy sessions still work in other workspaces that allow them.

Choose explicitly whether existing public scene links remain enabled. Public
links keep their own scene permission and never become SSO member sessions.

Before selecting enforced mode, sign in again and ask the operator to create and
test a break-glass credential with `userctl`. The app refuses to enforce SSO
until both checks pass.

To sign in, paste the connection ID supplied by the workspace owner and choose
**Continue with workspace SSO**. The identity provider returns a one-time code;
the callback URL never contains your Whiteboard login token.

Whiteboard identifies an SSO account by provider issuer and subject. A matching
email address does not merge accounts. Sign in to the existing account before
linking a new provider identity.

If an owner revokes your SSO session, new room, upload, export, media and command
credentials stop immediately. Refresh and sign in again through the configured
provider.

To add another provider identity, sign in to the existing account and start
**Link identity**. Whiteboard carries your account and retry ID in the protected,
single-use login transaction and accepts identity claims only from the verified
provider callback.

