# EXC-007: Frontend Auth Store

## Status

completed

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

- **Link to Detail Design**: [Design Document](../EXC-007-design.md)

- **Modified Files**: `src/frontend/src/store/auth-store.ts`, `src/frontend/src/lib/api.ts`, `src/frontend/src/App.tsx`
- **Sprint Phase**: phase-01

## Validation

| Layer | Expected proof | Result |
| --- | --- | --- |
| Unit | cd src/frontend && npm run build | PASS |
| Integration | | |
| E2E | | |
| Platform | | |
| Release | | |

## Harness Delta

- Registered EXC-007 in the harness database.
- Implemented Zustand auth store with session persistence.
- Configured Axios interceptor for JWT injection.
- Updated the login action to exchange temporary `admin/admin` credentials for a JWT.

## Evidence

Frontend build passed locally:
```text
vite v7.2.6 building for production...
✓ 40 modules transformed.
dist/index.html                   0.45 kB │ gzip:  0.30 kB
dist/assets/index-D7K_R_j1.css   20.55 kB │ gzip:  5.18 kB
dist/assets/index-Ba_L-z_u.js   195.42 kB │ gzip: 65.12 kB
✓ built in 1.25s
```
