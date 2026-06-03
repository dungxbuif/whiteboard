# EXC-005: Shadcn UI & Tailwind Setup

## Status

planned

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

- **Modified Files**: `dashboard/components.json`, `dashboard/tailwind.config.ts`, `dashboard/src/index.css`, `dashboard/src/lib/utils.ts`, `dashboard/src/components/ui/button.tsx`
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

(No evidence yet)
