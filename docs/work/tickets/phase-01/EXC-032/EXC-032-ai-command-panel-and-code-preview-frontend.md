# EXC-032: AI Command Panel & Code Preview (Frontend)

## Status

planned

## Lane

normal

## Product Contract

AICommandPanel (command palette with Cmd+K), canvas element injection, CodePreview sidebar/modal, Selection to Code action button

## Relevant Product Docs

- `05-02-PLAN.md`

## Acceptance Criteria

- Cmd+K shortcut launches floating AI Command Panel.
- AI-generated whiteboard components inject directly onto canvas.
- CodePreview displays generated code with syntax rendering and copy option.

## Design Notes

A Detail Design document following `docs/templates/detail_design.md` must be created at `docs/stories/phase-05/EXC-032-design.md` and approved by the reviewer before implementation starts.

- **Link to Detail Design**: [Design Document](EXC-032-design.md)

- **Modified Files**: `src/frontend/src/components/ai/AICommandPanel.tsx`, `src/frontend/src/components/ai/CodePreview.tsx`
- **Sprint Phase**: phase-05

## Validation

| Layer | Expected proof |
| --- | --- |
| Unit | cd dashboard && npm run build |
| Integration | |
| E2E | |
| Platform | |
| Release | |

## Harness Delta

- Registered EXC-032 in the harness database.

## Evidence

(No evidence yet)
