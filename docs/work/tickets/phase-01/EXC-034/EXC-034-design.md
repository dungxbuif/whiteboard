---
artifact_type: detail_design
id: EXC-034-DESIGN
status: approved
owner: shared
ticket: EXC-034
approval: user-requested-project-completion
---

# EXC-034 Detail Design: Replace Temporary Admin Auth With Google OAuth

## Intended behavior

The public login surface starts a Google OAuth authorization-code flow. The
callback validates a short-lived, HttpOnly state cookie, exchanges the code at
Google, upserts the Google profile by subject or email, issues the existing
JWT, and redirects the SPA with a one-time token fragment. The frontend stores
that token and calls the existing `/users/me` endpoint; all authenticated
workspace APIs remain unchanged.

## Technical decisions

- Use `golang.org/x/oauth2` with Google OpenID scopes (`openid`, `email`,
  `profile`).
- Keep the state value in a random 256-bit SameSite=Lax HttpOnly cookie and
  compare it in constant time on callback.
- Persist the stable Google `sub` claim on `users.google_subject`; use email
  only to link an existing account when the subject is not known.
- Keep JWT issuance and middleware contracts unchanged so existing API clients
  do not need a migration.
- Read OAuth and frontend redirect settings from environment variables. The
  real `.env` file remains local-only and must be rotated before deployment.

## Public contract

- `GET /api/v1/auth/google/login` redirects to Google and sets state.
- `GET /api/v1/auth/google/callback` validates state, exchanges the code, and
  redirects to `FRONTEND_URL#token=<jwt>`.
- `POST /api/v1/auth/login` and the username/password form are removed.

## Security and failure handling

- Missing, mismatched, or reused state returns `400`.
- Provider cancellation returns `401`; exchange, profile, and persistence
  failures return `401` without exposing provider details.
- State cookies are Secure when the request is HTTPS or carries the trusted
  `X-Forwarded-Proto: https` signal.
- The callback accepts only the configured redirect URL and Google profile
  subject/email; no client-provided identity is trusted.

## Verification plan

- Go tests cover login URL state, callback state validation, token exchange,
  profile fetch, user upsert, and JWT issuance with a local provider server.
- Frontend unit tests cover OAuth hash token consumption and session restore.
- Playwright covers the Google-only login surface and the mocked redirect/callback
  into the authenticated dashboard.
- Run `go test ./...`, `npm run test:unit`, `npm run build`, and
  `npx playwright test`.

## Operational follow-up

Google client credentials and any JWT secret that existed in local development
must be revoked and replaced in the deployment secret store before release.
