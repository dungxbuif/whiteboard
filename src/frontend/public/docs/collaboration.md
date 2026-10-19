# Joining and sharing a workspace

Status: browser management, durable collaboration, presence/follow and voice are implemented and deployed on VM100. Authenticated deep-link save/reload, Docker DNS restart/replay, TURN relay allocation and controlled reboot recovery pass; public share/invite/roles, native MCP write/export, browser voice/media and mobile/keyboard acceptance remain before sign-off.

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

The public scene remains read-only until fresh room admission and synchronization. The B3 browser implementation has local protocol, guest reload/revocation UI, and native browser offline-replay verification. The runtime is deployed; full public guest mutation/revocation acceptance remains a release gate. Legacy public URLs still work through current access checks. Passwords in query strings are ignored.

Agent API contract: POST `/api/v1/invitations/accept` accepts `{token}` with account bearer. POST `/api/v1/invitations/register` accepts `{token,email,name,password}` without bearer. POST `/api/v1/public/scene-session` accepts `{token,password?,displayName?}` without account bearer and returns `{token,expiresAt,workspaceId,sceneId,access,principalId,displayName}`. POST `/api/v1/public/scenes/:token` accepts `{password?,displayName?}` for the initial snapshot. Use the scoped session bearer only for `/api/v1/scenes/:sceneId/room-token`; never store it as the account login token. Share lists contain `{id,access,active,expiresAt}` only; the create response additionally contains a one-time token. Workspace responses include current `role`; authenticated scene responses include effective `access` and `owner`.


## Agent-readable endpoint reference

All routes below begin `/api/v1`. Login JWTs are account credentials. Public scene sessions are distinct scoped credentials. Workspace/scene route IDs accept existing UUIDs or public aliases; member/invitation/link IDs remain canonical IDs.

| Method and route | Authorization / scope | Request | Success |
| --- | --- | --- | --- |
| GET `/workspaces` | Login, current memberships | none | Workspace array with current `role` |
| POST `/workspaces` | Login | `{name}` | Workspace, `role:owner` |
| GET `/workspaces/{id}/members` | Owner, workspace from path | none | `{workspaceId,userId,role,createdAt}[]` |
| PATCH `/workspaces/{id}/members/{userId}` | Owner, protected owner cannot be demoted | `{role:"editor"\|"viewer"}` | 204 |
| DELETE `/workspaces/{id}/members/{userId}` | Owner, protected owner cannot be removed | none | 204 |
| PUT `/scenes/{id}/members/{userId}` | Owner + `X-Workspace-ID` | `{role:"editor"\|"viewer"}` | 204; apply override |
| DELETE `/scenes/{id}/members/{userId}` | Owner + `X-Workspace-ID` | none | 204; inherit membership |
| POST `/workspaces/{id}/invitations` | Owner, workspace from path | `{email,role:"editor"\|"viewer"}` | 201 `{invitation,inviteUrl}` |
| GET `/workspaces/{id}/invitations` | Owner | none | 200 pending metadata array |
| DELETE `/workspaces/{id}/invitations/{inviteId}` | Owner | none | 204 |
| POST `/invitations/accept` | Login with invited email | `{token}` | 200 Membership |
| POST `/invitations/register` | Public, valid email-bound invite | `{token,email,password,name}` | 201 User; then ordinary password login |

Invitation metadata contains `id,workspaceId,email,role,inviterId,createdAt,expiresAt` and optional `acceptedAt,revokedAt`. Pending listing excludes consumed, revoked and expired invitations. Only creation returns the raw URL; no API returns hashes. Invitation lifetime is exactly seven days; accepting at expiry fails. Accepting a lower-role invitation never reduces an existing membership.

| Method and route | Authorization / scope | Input | Output |
| --- | --- | --- | --- |
| GET `/scenes/{id}/share-links` | Owner login + `X-Workspace-ID` | none | `{id,access,active,expiresAt}[]` |
| POST `/scenes/{id}/share-links` | Owner login + `X-Workspace-ID` | `{access:"view"\|"edit",password?,expiresAt?}` | `{id,access,active,expiresAt,token}`; token once |
| DELETE `/scenes/{id}/share-links/{linkId}` | Owner login + `X-Workspace-ID` | none | 204 |
| POST `/public/scene-session` | Public link, no account bearer | `{token,password?,displayName?}` | SceneSession |
| POST `/public/scenes/{token}` | Public link, no account bearer | `{password?,displayName?}` | `{title,content}` snapshot after current authority checks |
| POST `/scenes/{id}/room-token` | Login or scoped scene-session bearer | `{}`; optional `X-Workspace-ID` for scoped guest, required for login | `{token,expiresAt,sceneSession?}` |

SceneSession is `{token,expiresAt,workspaceId,sceneId,access,principalId,displayName}`. Edit guests require a trimmed 1–100 character display name. Share passwords require at least six characters and at most 72 UTF-8 bytes. Link expiry is a future RFC3339 timestamp. Public access grants exactly one scene, with no membership or management rights. Lists expose metadata, never token/hash/password.

Room renewal validates current membership/link and scene/workspace status. For guests it returns a renewed sceneSession with the same guest ID; replace both credentials before expiry. Credentials last at most ten minutes and never outlive link expiry. Expired sessions require link/password exchange. Link/password changes invalidate earlier sessions. A room JWT is not a login or scene-session token. Browser live renewal and revoke/downgrade transport remain B3/E integration gates.

Room claims: HS256, issuer `whiteboard-api`, audience `whiteboard-collab`, purpose `room`, scene/workspace scope, `principalType`, `principalId`, `grantId`, `displayName`, `access`, decimal-string `aclVersion`, `iat/exp/jti`. Public IDs begin `guest:` and include decimal-string `grantVersion`. Scene-session audience is `whiteboard-scene-session` and purpose is `scene-session`. Go/collab use a dedicated `COLLAB_JWT_SECRET` of at least 32 bytes, distinct from `JWT_SECRET`.

Errors use `{code,message,requestId}`. Invitations: 400 invalid input/link or generic registration rejection; 401 login required; 403 wrong email/prohibited operation; 404 inaccessible workspace; 410 expiry; 413 oversized body; 408 stalled input; 503 unavailable authority. Registration errors do not enumerate accounts. Sharing: 400 invalid input; 401 password/session challenge; 403 forbidden; 404 inaccessible/revoked/expired resource; 413 oversized body; 503 unavailable authority. Invitation/public-link JSON bodies are bounded at 4 KiB; unknown fields/trailing JSON are rejected. Passwords in query strings are ignored. Do not log tokens, passwords or scoped credentials.

See [browser collaboration and recovery](browser-collaboration.md) for scoped queues, local undo, image retries, session lifetime and the single-writer migration policy.

## People, follow and voice

Open **People** in a v2 scene to list connected tabs and their effective Editor/Viewer role. Each tab has its own connection identity, even when two tabs use the same account. **Follow** tracks a participant's camera; **Stop following** or target departure ends it. Cursor, selection, camera, follow and voice metadata are ephemeral and never enter the scene document.

**Join voice** requests one of ten room slots before the browser asks for microphone permission. Denial or a full room leaves drawing available. **Mute**, **Unmute**, **Leave voice** and **Play voice audio** manage the current call. Media is peer-to-peer with short-lived scoped TURN credentials; signaling shares the custom `/collab/` WebSocket. Socket.IO is no longer an API or deployment dependency.


## Operator recovery limits

The merged release candidate includes a tested recovery procedure for a
verified replacement scene with a new link. Your original queued edits stay
associated with their original scene; they are not automatically replayed into
the replacement. Keep a downloaded recovery copy and ask the workspace owner
which link to use.

Scenes containing images/assets or scene-specific access overrides require an operator mapping procedure before replacement. The bounded rehearsal refuses those cases. This is not an automatic restore button or version-history feature. The production database cutover backup restored successfully, but that whole-database proof does not remove these per-scene mapping limits.

### Interrupted image preparation

Moving an image while its upload or save is pending preserves the latest position and independent remote edits. If the canvas closes or local storage fails before image preparation finishes, automatic image replay pauses. The browser keeps the original draft and image bytes and shows a recovery message after reopening. Choose **Export recovery** and keep the download before clearing browser data; reopening or pressing Retry does not resolve this incomplete preparation automatically. Normally completed preparation resumes automatically after reload. Recovery exports include `blobs[].draft.pendingBaseline=true` for this interrupted case, together with the retained draft and encoded image bytes.
