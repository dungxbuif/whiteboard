# Workspace invitations — Collaboration v2

Status: implemented on the development branch; deployment and browser invitation controls are pending.

Owners create an email-bound editor or viewer invitation and copy the `/invite/:token` link to the recipient. The app sends no email. A link works once and expires exactly seven days after creation; accepting at the expiry timestamp fails. Owners can list pending metadata and revoke invitations. A list cannot recover the raw link.

Existing account: log in with the invited email, then accept. New account: register with a matching email, name and password; the account, personal workspace and invited membership commit together. Email is trimmed and lowercased. Passwords require at least eight characters and at most 72 UTF-8 bytes. A typed email and link do not establish verified email ownership. Existing passwords are never changed by invitation registration. A lower-role invitation never reduces an existing membership.

## Agent-readable API

All paths begin `/api/v1`. Login JWTs remain required for authenticated routes. Workspace route IDs accept UUIDs or public aliases; invitation management takes workspace scope from the path.

| Method and route | Authorization | Request | Success |
| --- | --- | --- | --- |
| POST `/workspaces/{id}/invitations` | Owner | `{email,role:"editor"\|"viewer"}` | 201 `{invitation,inviteUrl}` |
| GET `/workspaces/{id}/invitations` | Owner | none | 200 pending metadata array |
| DELETE `/workspaces/{id}/invitations/{inviteId}` | Owner | none | 204 |
| POST `/invitations/accept` | Login | `{token}` | 200 `{workspaceId,userId,role,createdAt}` |
| POST `/invitations/register` | Public, valid invite | `{token,email,password,name}` | 201 User; then use existing password login |

Metadata: `id`, `workspaceId`, `email`, `role`, `inviterId`, `createdAt`, `expiresAt`, optional `acceptedAt`/`revokedAt`. Pending listing excludes consumed, revoked and expired invitations. Only creation returns a raw token URL. No API returns a token hash or password hash.

Errors are `{code,message,requestId}`. 400 invalid input/link or generic registration rejection; 401 login required; 403 wrong email or prohibited operation; 404 inaccessible workspace; 410 expired invitation; 413 oversized input; 408 stalled input; 503 storage/authority unavailable. Registration errors do not disclose whether an account exists. DTOs reject unknown fields and trailing JSON; JSON body limit is 4 KiB, with a ten-second prebuffer deadline before authority locking.

Runtime `FRONTEND_URL` must be an HTTP(S) origin without credentials, a path prefix, query or fragment; optional trailing slash is accepted.
