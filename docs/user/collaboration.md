# Joining and sharing a workspace

Status: browser management, durable collaboration, presence/follow and voice
are implemented and deployed on VM100. Authenticated deep-link save/reload is
verified publicly; Docker DNS restart/replay, TURN relay allocation and
controlled reboot recovery also pass. Public share/invite/roles, native MCP
write/export, mobile/keyboard and the public two-peer browser media canary pass.
Same-host browser media uses host/host candidates; cross-network relay proof is
recorded separately.

## Invite someone

1. Open your workspace and choose **Members** in the sidebar. Only owners see workspace settings.
2. Select **Invite member**, enter the person's email, and choose **Editor** or **Viewer**.
3. Select **Create invitation**, then **Copy invitation**. Send the link yourself: the app does not send email.

Save the link when you create it. The invitation list shows email, role and expiry, but cannot recover the secret URL. You can revoke an unused invitation from **Pending invitations**.

An invitation works once and expires seven days after creation. Existing users sign in with the invited email and select **Accept invitation**. New users select **Create account with invitation**, enter the matching email, display name and a password with at least eight characters and at most 72 UTF-8 bytes, and join during registration. The server creates their personal workspace and invited membership together. Registering with an existing email does not reset its password: sign in instead. An invitation does not mark the email address as verified.

If acceptance fails, check the signed-in email. Ask the owner for a new link if it expired, was revoked or was already used. The invitation stays in the current URL while you sign in; successful acceptance replaces that URL and clears the transient invitation screen. It is never stored with the account credentials.

Repeated password-login, invitation-registration, or public-link credential
attempts can return HTTP 429 with
`{code:"RATE_LIMITED",message,requestId}`. Wait for the integer number of
seconds in the `Retry-After` response header before trying again. A throttled
request does not consume an invitation or change scene access.

## Manage access

Owners can change a member between **Editor** and **Viewer**, remove a member, or apply a scene override. The owner cannot be removed or demoted. Member rows currently use account IDs returned by the API.

Editors create and change content; viewers have reading controls. A scene override can grant editor access to a workspace viewer or restrict a workspace editor to viewing that scene. Overrides require current membership and disappear when membership is removed. They never grant permission management. In **Members**, choose a scene from the loaded scene list, then choose its member override; **Inherit workspace role** removes the override. The selector applies changes explicitly and does not claim to display existing overrides.

## Share one scene

1. As the workspace owner, open a scene's action menu and select **Share**.
2. Create a **View link** or **Edit link**. Each can have its own password (at least six characters and at most 72 UTF-8 bytes) and expiry.
3. Copy the new URL immediately. Closing the dialog clears the one-time URL; existing links show metadata only.
4. Select **Revoke** beside the link to stop future access. Other links remain independent.

A view link reads only its scene. An edit link asks guests for a display name; this does not create an account or grant workspace membership. The browser exchanges the link and optional password in POST bodies, then retains the scoped, short-lived scene session in the current tab's sessionStorage so reload preserves its guest identity. Public password mistakes leave your normal account signed in. Public requests never attach the account login token; scoped scene credentials never replace it.

The public scene remains read-only until fresh room admission and
synchronization. Live guest edit, reload, renewal, revocation and native
offline replay pass the local 17/17 real-stack suite. The runtime is deployed;
public-origin guest mutation/revocation acceptance passes. Legacy
public URLs still work through current access checks. Passwords in query
strings are ignored.

For API consumers, see the internal [API reference](../architecture/API.md). An agent-readable copy of this guide is served at `/docs/collaboration.md`; the shared `SceneSession` contract is documented in the C2 API section.

## Browser integration (deployed; browser media canary pending)

The approved browser adapter uses durable scoped offline updates, fresh room admission and one Yjs content writer. AI additions and presentation order share that writer. See [browser design](../architecture/BROWSER_COLLABORATION.md) for scope, recovery, legacy fallback and verification.

For editing states, local undo, image retries and retained offline work, see [browser collaboration and recovery](../../src/frontend/public/docs/browser-collaboration.md).

## People, follow and voice

Open **People** in a v2 scene to see each connected tab, its effective Editor/Viewer role and voice status. Two tabs from the same account are separate participants. Choose **Follow** to track another participant's camera; **Stop following** or their departure ends it. Cursor, selection, camera and follow state are temporary and never enter scene history.

Choose **Join voice** to request one of ten room slots. Microphone permission is requested only after the server admits the slot. Denial or a full room leaves drawing and synchronization available. Use **Mute**, **Unmute** and **Leave voice** from the same toolbar. Voice media is peer-to-peer with short-lived TURN credentials; signaling shares the admitted custom collaboration socket. A browser autoplay block exposes **Play voice audio**. Closing, revocation or connection loss stops tracks, peers and audio elements.

### Interrupted image preparation

Moving an image while its upload or save is pending preserves the latest position and independent remote edits. If the canvas closes or local storage fails before image preparation finishes, automatic image replay pauses. The browser keeps the original draft and image bytes and shows a recovery message after reopening. Choose **Export recovery** and keep the download before clearing browser data; reopening or pressing Retry does not resolve this incomplete preparation automatically. Normally completed preparation resumes automatically after reload. Recovery exports include `blobs[].draft.pendingBaseline=true` for this interrupted case, together with the retained draft and encoded image bytes.
