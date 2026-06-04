# EXC-005: Shadcn UI & Tailwind Setup

## Status

implemented

## Lane

normal

## Product Contract

Init Shadcn UI, copy Tailwind config, port core components (button, input, card, dropdown-menu, dialog, scroll-area), setup cn utility

## Relevant Product Docs

- `01-03-PLAN.md`

## Acceptance Criteria

- Shadcn UI is initialized inside dashboard/ directory.
- Tailwind CSS is configured and index.css has basic utilities.
- Core shadcn components (button, input, card, dropdown-menu, dialog, scroll-area) compile and build.

## Design Notes

A Detail Design document following `docs/templates/detail_design.md` must be created at `docs/stories/phase-01/EXC-005-design.md` and approved by the reviewer before implementation starts.

- **Link to Detail Design**: [Design Document](EXC-005-design.md)

- **Modified Files**: `dashboard/components.json`, `dashboard/tailwind.config.ts`, `dashboard/postcss.config.js`, `dashboard/src/index.css`, `dashboard/src/lib/utils.ts`, `dashboard/src/components/ui/button.tsx`, `dashboard/src/components/ui/input.tsx`, `dashboard/src/components/ui/card.tsx`, `dashboard/src/components/ui/dropdown-menu.tsx`, `dashboard/src/components/ui/dialog.tsx`, `dashboard/src/components/ui/scroll-area.tsx`, `dashboard/src/App.tsx`, `dashboard/package.json`, `dashboard/package-lock.json`, `dashboard/vite.config.ts`, `dashboard/tsconfig.app.json`, `dashboard/tsconfig.node.json`
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

- Registered EXC-005 in the harness database.

## Evidence

- 2026-06-04: `cd dashboard && npm install class-variance-authority clsx tailwind-merge @radix-ui/react-slot @radix-ui/react-dropdown-menu @radix-ui/react-dialog @radix-ui/react-scroll-area tailwindcss-animate` completed with 0 vulnerabilities.
- 2026-06-04: `cd dashboard && npm install -D tailwindcss@3.4.17 postcss autoprefixer @types/node` completed with 0 vulnerabilities.
- 2026-06-04: `cd dashboard && npm list tailwindcss @radix-ui/react-slot @radix-ui/react-dropdown-menu @radix-ui/react-dialog @radix-ui/react-scroll-area class-variance-authority clsx tailwind-merge` confirmed core UI dependencies are installed.
- 2026-06-04: `cd dashboard && npm run build` passed.
