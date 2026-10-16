# Joining a workspace

Collaboration v2 invitations are implemented on the development branch; deployment and browser invitation controls are pending. Workspace owners can create an invitation for an email address with editor or viewer access and copy the returned `/invite/:token` link to the intended person. The app does not send email. Save the link when creating it: the invitation list shows metadata and cannot recover it. Owners can revoke unused invitations.

A link works once and expires seven days after creation. If you already have an account, log in with the invited email and accept the invitation. If you are new, submit the invite token, matching email, display name and password (at least eight characters, at most 72 UTF-8 bytes) to invitation registration; your account, personal workspace and shared membership are created together. Registering with an existing email does not reset its password: use the normal login path. Typed email and possession of the link do not mark an email address as verified.

Editors can change workspace content; viewers can read it. An invitation with less access does not reduce access you already have. Only owners manage invitations and membership. If acceptance fails, check the account/email and ask the owner for a fresh link if it expired or was revoked.

API consumers: see [API reference](../architecture/API.md#collaboration-a2-invitation-contract-implemented-release-pending). Human browser invitation controls are integrated separately in the dashboard task.
