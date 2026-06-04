# EXC-007: Frontend Auth Store

## Status

planned

## Lane

normal

## Product Contract

Zustand auth store (user, login, logout, checkAuth), axios interceptor with JWT, API client setup

## Relevant Product Docs

- `01-04-PLAN.md`

## Acceptance Criteria

- Zustand auth store manages auth state (user, loading, authenticated).
- Axios API client includes Authorization header interceptor.
- Frontend builds successfully with auth store integration.

## Design Notes

A Detail Design document following `docs/templates/detail_design.md` must be created at `docs/stories/phase-01/EXC-007-design.md` and approved by the reviewer before implementation starts.

- **Link to Detail Design**: [Design Document](EXC-007-design.md)

- **Modified Files**: `src/frontend/src/store/auth-store.ts`, `src/frontend/src/lib/api.ts`
- **Sprint Phase**: phase-01

## Validation

| Layer | Expected proof |
| --- | --- |
| Unit | cd dashboard && npm run build |
| Integration | |
| E2E | |
| Platform | |
| Release | |

## Harness Delta

- Registered EXC-007 in the harness database.

## Evidence

(No evidence yet)
