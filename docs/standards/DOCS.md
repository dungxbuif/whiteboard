# Documentation Standards

## Quality Bar

All project artifacts MUST follow `docs/standards/QUALITY_BAR.md`.

## Master Doc Reconciliation

After implementation, agents must compare actual changes with master docs.

Update:

- `docs/requirements/` when product behavior, requirements, or acceptance criteria change.
- `docs/architecture/ARCHITECTURE.md` when modules, boundaries, dependencies, or runtime flow change.
- `docs/architecture/API.md` when endpoints, events, CLI flags, request/response shapes, errors, auth, or versioning change.
- `docs/architecture/ERD.md` when entities, tables, relationships, constraints, or migrations change.
- `docs/architecture/SDD.md` when design changes in `DETAIL_DESIGN.md` affect the high-level system/architectural specifications during the sprint.
- `docs/decisions/` when durable technical decisions change.
- `docs/CONTEXT.md` after every completed task.

## Docs Review Checklist

Every completed ticket or bug MUST include a docs review result:

- [ ] Code changed but docs unchanged: reason recorded
- [ ] User-facing behavior changed: requirements updated or not needed reason recorded
- [ ] API/contract changed: `docs/architecture/API.md` updated or not needed reason recorded
- [ ] Data model changed: `docs/architecture/ERD.md` updated or not needed reason recorded
- [ ] Architecture/runtime changed: `docs/architecture/ARCHITECTURE.md` updated or not needed reason recorded
- [ ] Design changes: `DETAIL_DESIGN.md` changes consolidated into `docs/architecture/SDD.md` or not needed reason recorded
- [ ] Durable decision changed: ADR added or not needed reason recorded
- [ ] `docs/CONTEXT.md` updated

Use `docs/templates/DOCS_REVIEW.md` when the ticket or bug does not already contain a full docs review section.

## Brownfield Rule

For brownfield projects, document only the touched module and directly related contracts. Do not attempt full-system documentation unless the task asks for it.

## Brownfield Scope

Touched scope MUST be recorded for brownfield work:

- Touched modules/files
- Direct dependencies inspected
- Contracts affected
- Known unknowns
- Reason if scope was expanded

Agents MUST NOT scan or summarize the whole repository unless the task explicitly requires full-codebase mapping.

## Release Docs

Create release notes for completed phases or epics. Update changelog for major versions or externally visible releases.
