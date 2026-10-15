---
artifact_type: ticket
id: EXC-034
status: done
owner: human
priority: high
lane: high-risk
trace:
  requirement: REQ-04
  phase: PHASE-1
  detail_design: EXC-034-design.md
  validation_matrix: VALIDATION_MATRIX
---

# EXC-034: Replace Temporary Admin Auth With Google OAuth

## Status

done

## Lane

high-risk

## Context

Phase 1 temporarily used one hard-coded `admin/admin` account to unblock local
development. The runtime now uses Google OAuth and has no hard-coded login
credentials.

## Acceptance Criteria

- Google OAuth login and callback are the only interactive login path.
- Temporary admin username/password constants and login form are removed.
- OAuth configuration is loaded defensively from `src/.env`.
- Existing JWT middleware and authenticated API contracts continue to work.
- Backend integration tests and browser E2E cover successful login and failure states.
- Previously exposed Google and JWT credentials are rotated before deployment
  (operational release gate; requires deployment secret-store access).

## Impacted Areas

- Code: backend auth usecase/handler/router/config and frontend auth UI/store.
- Security: authentication provider, credentials, session issuance.
- External provider: Google OAuth.

## Detail Design

- Required: yes
- Link: [EXC-034 Detail Design](EXC-034-design.md)
- Approval: approved via user-requested project completion

## Test Expectations

- Unit tests for OAuth state, callback, user upsert, and JWT issuance.
- Integration test for user persistence.
- Playwright verification of the complete browser redirect/callback flow.

## Evidence

- `go test ./...` passed (43 tests across 8 packages), including provider-backed
  OAuth exchange, state handling, profile upsert, and JWT issuance.
- `npm run test:unit` passed (16 tests) and `npm run build` passed.
- `npx playwright test` passed (11 tests), including the Google-only login UI and
  mocked redirect/callback into the authenticated dashboard.
- Deployment credential rotation is intentionally left as an operational gate;
  no external secret store was modified.
