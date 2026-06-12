---
artifact_type: ticket
id: EXC-034
status: planned
owner: human
priority: high
lane: high-risk
trace:
  requirement: REQ-04
  phase: PHASE-1
  detail_design: TBD
  validation_matrix: VALIDATION_MATRIX
---

# EXC-034: Replace Temporary Admin Auth With Google OAuth

## Status

planned

## Lane

high-risk

## Context

Phase 1 temporarily uses one hard-coded `admin/admin` account to unblock local
development. Before Phase 1 release, replace that temporary path with Google
OAuth and remove the hard-coded credential from runtime code.

## Acceptance Criteria

- Google OAuth login and callback are the only interactive login path.
- Temporary admin username/password constants and login form are removed.
- OAuth configuration is loaded defensively from `src/.env`.
- Existing JWT middleware and authenticated API contracts continue to work.
- Backend integration tests and browser E2E cover successful login and failure states.
- Previously exposed Google and JWT credentials are rotated before deployment.

## Impacted Areas

- Code: backend auth usecase/handler/router/config and frontend auth UI/store.
- Security: authentication provider, credentials, session issuance.
- External provider: Google OAuth.

## Detail Design

- Required: yes
- Link: TBD
- Approval: pending

## Test Expectations

- Unit tests for OAuth state, callback, user upsert, and JWT issuance.
- Integration test for user persistence.
- Playwright verification of the complete browser redirect/callback flow.
