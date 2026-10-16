# Joining a workspace

Collaboration v2 invitations are implemented on the development branch; deployment and browser invitation controls are pending. Workspace owners can create an invitation for an email address with editor or viewer access and copy the returned `/invite/:token` link to the intended person. The app does not send email. Save the link when creating it: the invitation list shows metadata and cannot recover it. Owners can revoke unused invitations.

A link works once and expires seven days after creation. If you already have an account, log in with the invited email and accept the invitation. If you are new, submit the invite token, matching email, display name and password (at least eight characters, at most 72 UTF-8 bytes) to invitation registration; your account, personal workspace and shared membership are created together. Registering with an existing email does not reset its password: use the normal login path. Typed email and possession of the link do not mark an email address as verified.

Editors can change workspace content; viewers can read it. An invitation with less access does not reduce access you already have. Only owners manage invitations and membership. If acceptance fails, check the account/email and ask the owner for a fresh link if it expired or was revoked.

API consumers: see [API reference](../architecture/API.md#collaboration-a2-invitation-contract-implemented-release-pending). Human browser invitation controls are integrated separately in the dashboard task.

## Sharing one scene

Collaboration v2 public-link APIs are implemented on the development branch. The new browser sharing controls and realtime editor are still being integrated. An owner can create separate view and edit links, optionally protect each with a password or expiry, and revoke each independently. Copy a new link immediately: the link list cannot recover its token. Revoking one link leaves other links working.

A viewer can read the shared scene. An editor must enter a display name before joining; this identifies the guest in the session and does not create an account. A public link grants access only to its scene. It never grants workspace membership or permission management.

The client exchanges the link and optional password in a POST body, then uses a short-lived scene session. Refreshing before expiry renews the session while the link remains valid; users do not need to re-enter a password every ten minutes during an active session. If the session expires, the link is revoked, or its password changes, exchange the link again with the current password. Legacy password-protected viewers also use POST JSON at their existing public scene path; passwords in query strings are ignored.

API details and agent contracts are in the [A3 API reference](../architecture/API.md#collaboration-a3-public-links-and-room-credentials-implemented-release-pending).
