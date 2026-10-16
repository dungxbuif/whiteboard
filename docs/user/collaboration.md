# Joining and sharing a workspace

Status: browser management controls and access APIs are implemented and verified locally on the collaboration-v2 branch. They are not deployed yet. Live guest editing and session revocation still require the B3 collaboration adapter and real-stack release verification.

## Invite someone

1. Open your workspace and choose **Members** in the sidebar. Only owners see workspace settings.
2. Select **Invite member**, enter the person's email, and choose **Editor** or **Viewer**.
3. Select **Create invitation**, then **Copy invitation**. Send the link yourself: the app does not send email.

Save the link when you create it. The invitation list shows email, role and expiry, but cannot recover the secret URL. You can revoke an unused invitation from **Pending invitations**.

An invitation works once and expires seven days after creation. Existing users sign in with the invited email and select **Accept invitation**. New users select **Create account with invitation**, enter the matching email, display name and a password with at least eight characters and at most 72 UTF-8 bytes, and join during registration. The server creates their personal workspace and invited membership together. Registering with an existing email does not reset its password: sign in instead. An invitation does not mark the email address as verified.

If acceptance fails, check the signed-in email. Ask the owner for a new link if it expired, was revoked or was already used. The invitation stays in the current URL while you sign in; successful acceptance replaces that URL and clears the transient invitation screen. It is never stored with the account credentials.

## Manage access

Owners can change a member between **Editor** and **Viewer**, remove a member, or apply a scene override. The owner cannot be removed or demoted. Member rows currently use account IDs returned by the API.

Editors create and change content; viewers have reading controls. A scene override can grant editor access to a workspace viewer or restrict a workspace editor to viewing that scene. Overrides require current membership and disappear when membership is removed. They never grant permission management. In **Members**, choose a scene from the loaded scene list, then choose its member override; **Inherit workspace role** removes the override. The selector applies changes explicitly and does not claim to display existing overrides.

## Share one scene

1. As the workspace owner, open a scene's action menu and select **Share**.
2. Create a **View link** or **Edit link**. Each can have its own password (at least six characters and at most 72 UTF-8 bytes) and expiry.
3. Copy the new URL immediately. Closing the dialog clears the one-time URL; existing links show metadata only.
4. Select **Revoke** beside the link to stop future access. Other links remain independent.

A view link reads only its scene. An edit link asks guests for a display name; this does not create an account or grant workspace membership. The browser exchanges the link and optional password in POST bodies, then holds a scoped, short-lived scene session in memory. Public password mistakes leave your normal account signed in. Public requests never attach the account login token; scoped scene credentials never replace it.

The current public edit screen shows a read-only snapshot while live collaboration is awaiting its adapter. Actual editing is enabled only after room admission in the B3 integration. Renewal, live revocation, offline replay and real WebSocket verification are release gates, not represented as completed by the management UI tests. Legacy public URLs still work through current access checks. Passwords in query strings are ignored.

For API consumers, see the internal [API reference](../architecture/API.md). An agent-readable copy of this guide is served at `/docs/collaboration.md`; the shared `SceneSession` contract is documented in the C2 API section.
