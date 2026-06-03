# EXC-002: Frontend Scaffold Vite+React

## Status

planned

## Lane

normal

## Product Contract

Create Vite React TS project, install core dependencies (zustand, axios, lucide-react), setup directory structure

## Relevant Product Docs

- `01-01-PLAN.md`

## Acceptance Criteria

- React TS application is scaffolded in dashboard/ directory.
- Dependencies (zustand, axios, lucide-react) are installed.
- Frontend builds successfully without errors.

## Design Notes

A Detail Design document following `docs/templates/detail_design.md` must be created at `docs/stories/phase-01/EXC-002-design.md` and approved by the reviewer before implementation starts.

- **Link to Detail Design**: [Design Document](EXC-002-design.md)

- **Modified Files**: `dashboard/package.json`, `dashboard/vite.config.ts`, `dashboard/src/main.tsx`, `dashboard/src/App.tsx`, `dashboard/index.html`
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

- Registered EXC-002 in the harness database.

## Evidence

(No evidence yet)
