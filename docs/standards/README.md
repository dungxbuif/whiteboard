# Standards

Human-maintained standards in this folder are source of truth for agents.

## Precedence

When instructions conflict, follow this order:

1. Direct user instruction for the current task
2. `AGENTS.md`
3. Files in `docs/standards/`
4. Active phase, ticket, or bug
5. Master docs in `docs/requirements/`, `docs/architecture/`, and `docs/decisions/`

If a conflict would bypass tests, docs reconciliation, or security rules, ask before proceeding.

## Files

- `WORKFLOW.md`: lifecycle, approval gates, done definition.
- `GIT.md`: branches, commits, PRs, merge rules.
- `CODE.md`: code style and dependency rules.
- `TESTING.md`: test expectations and evidence.
- `DOCS.md`: documentation update rules.
- `QUALITY_BAR.md`: what good tickets, designs, bugs, ADRs, and diagrams must contain.
- `VALIDATION.md`: proof policy for each kind of work.
- `DEBUGGING.md`: root-cause-first bug and failure investigation.

Agents may propose standard updates but must not silently change them unless the task explicitly asks for standard changes.
