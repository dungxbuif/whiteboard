---
artifact_type: agent_gateway
version: 1
status: active
owner: human
parse_hint: yaml_frontmatter_plus_markdown_body
---

# Agent Gateway

This file is the mandatory system-prompt style gateway for every AI agent working in this repository.

## Rule 0: Response Format

Every agent response MUST start with this exact prefix:

```text
Current Phase: <Hydration|Detail Design|Execution & Test|Reconciliation|Dehydration|Blocked> | Next action: <one concrete next action>
```

Then continue with the response body. If the agent is only answering a question, use:

```text
Current Phase: Hydration | Next action: answer the question from current repository context
```

## Rule 1: Required Reading Order

Before changing code or docs, read:

1. `docs/CONTEXT.md`
2. `docs/work/BACKLOG.md`
3. `docs/standards/README.md`
4. `docs/standards/QUALITY_BAR.md`
5. `docs/standards/VALIDATION.md`
6. Active phase in `docs/work/phases/`, if one exists
7. Active ticket or bug in `docs/work/tickets/` or `docs/work/bugs/`
8. Relevant master docs only: `docs/requirements/`, `docs/architecture/`, `docs/decisions/`

For bug, failure, or unexpected behavior work, also read `docs/standards/DEBUGGING.md` before proposing fixes.

## Rule 2: Do Not Over-Scan

Do not scan the whole repository by default.

For brownfield work:

- Inspect only the touched module and direct dependencies.
- Record known unknowns.
- Expand scope only when evidence requires it.

## Rule 3: Lifecycle

Every task follows this lifecycle:

1. Hydration: load context, backlog queue, standards, and active work.
2. Detail Design: write or update a design for non-small work.
3. Execution & Test: implement and verify with real commands.
4. Reconciliation: update master docs and decisions when behavior changes.
5. Dehydration: update `docs/CONTEXT.md`, backlog, validation matrix, and task status.

## Rule 4: Backlog Queue

`docs/work/BACKLOG.md` is the runtime scheduler. It may reorder requirements, tickets, bugs, maintenance, and framework work based on priority and risk.

Backlog order decides what to consider next. Non-tiny work still requires an execution artifact: requirement slice, phase, ticket, or bug.

## Rule 5: Status Model

Use this status flow for phases, tickets, and bugs:

```text
draft -> ready -> in_progress -> blocked -> in_review -> verified -> done
```

Bug work may additionally record `reproduced`, `fixed`, and `regression_verified`, but the primary status must still use the shared flow.

## Rule 6: Auto-Run Policy

Small tasks may proceed without human approval only when they do not change:

- Public API or user-facing contracts
- Database schema or migrations
- Architecture boundaries
- Authentication, authorization, security, or permissions
- Deployment/runtime configuration
- Major dependencies
- Repository standards

All other tasks require a `DETAIL_DESIGN.md` artifact with approval before implementation.

Every auto-run small task must record:

```text
Small task exemption: yes
Reason: <why this is small>
Impact checked: API=no, DB=no, Security=no, Runtime=no, Standards=no
```

## Rule 7: Human Vs AI Field Ownership

Templates use YAML frontmatter and body sections to separate responsibilities.

- Human fields define intent, priority, approval, acceptance, and business context.
- AI fields define analysis, proposed approach, evidence, docs reconciliation, and context updates.
- Shared fields may be updated by either human or AI, but AI must not overwrite human decisions silently.

If a human-owned field is missing and blocks execution, ask or mark the item `blocked`.

## Rule 8: Required Trace Links

Every work artifact must link backward and forward:

- Phase links to backlog, roadmap, tickets, bugs, verification, validation matrix, release notes, and ADRs.
- Ticket links to backlog, phase, detail design, tests, validation matrix, docs review, ADRs, and release notes.
- Bug links to backlog, phase, reproduction, detail design, regression tests, validation matrix, docs review, ADRs, and release notes.
- Detail design links back to ticket or bug and forward to tests, docs review, ADRs, and touched master docs.
- ADR links back to triggering ticket, bug, phase, and master docs.

## Rule 9: Completion Rules

A task is not complete until:

- Relevant tests were run and results were recorded.
- `docs/work/VALIDATION_MATRIX.md` was updated when behavior proof changed.
- Acceptance/UAT was completed or explicitly marked not required.
- Master docs were reconciled with actual changes.
- Docs review checklist was completed.
- New decisions were logged as ADRs when needed.
- `docs/CONTEXT.md` reflects current state and next steps.
- `docs/work/BACKLOG.md` and work item status were updated.
- `docs/releases/CHANGELOG.md` or release notes were updated with the user-facing or framework changes.

## Rule 10: Loop Guard

Agents MUST stop and ask for human/design review instead of continuing when:

- The same test or verification path fails after 3 fix attempts.
- The agent has run 5 total fix/test cycles on one work item.
- A fix attempt changes architecture, API, data, security, runtime, or standards beyond approved scope.
- Each fix reveals a new unrelated failure or broader design problem.

When this happens, set the work item to `blocked`, record the attempt log, summarize the best current root-cause hypothesis, and identify the decision or input needed.

## Rule 11: Output Body Requirements

After the required prefix, include only sections that are relevant:

- `Context`: what was read or assumed.
- `Decision`: decision made or needed.
- `Action`: work performed or proposed.
- `Evidence`: commands, checks, or docs reviewed.
- `Blocked`: blocker, attempt count, and required human input.

Do not claim work is done unless Rule 9 is satisfied.

---

# Project Context: Excalidraw Transformation

## 1. Quick Links
- **Product Spec:** [docs/requirements/SPEC.md](docs/requirements/SPEC.md)
- **Architecture:** [docs/architecture/ARCHITECTURE.md](docs/architecture/ARCHITECTURE.md)
- **Current Backlog:** [docs/work/BACKLOG.md](docs/work/BACKLOG.md)
- **Roadmap:** [docs/work/ROADMAP.md](docs/work/ROADMAP.md)
- **Validation Matrix:** [docs/work/VALIDATION_MATRIX.md](docs/work/VALIDATION_MATRIX.md)
- **Phase 1 Tickets:** [docs/work/tickets/phase-01/](docs/work/tickets/phase-01/)

## 2. Project Engineering Standards
This section mandates the architectural, code quality, and verification standards for the project. All agents must adhere to these rules strictly.

### 2.1. Code Quality & Refactoring
- **No Magic Values**: Never use magic strings or magic numbers. All literals (timeouts, URLs, header names, context keys, etc.) must be defined as named constants in `internal/domain/constants.go` or localized constants within the package.
- **Strict Configuration**: Environment variable loading must be defensive. Use a `loadConfig` pattern that **exits the application immediately** with a clear error message if a required variable is missing or invalid. Never use insecure fallbacks for credentials or security-critical settings.
- **No Redundant Comments**: Do not add comments for obvious logic. Use comments only to explain complex algorithms, business domain reasoning, or as part of formal documentation (e.g., Go doc comments).

### 2.2. Frontend Verification
- **Automated Verification**: Every ticket involving frontend changes **MUST** be verified using automated tools.
- **Playwright/MCP Requirement**: Use the `mcp-playwright-browser` tools to perform end-to-end verification of UI components, user flows, and state changes. Capturing screenshots and comparing them against design concepts is mandated for high-fidelity work.

### 2.3. Backend Architecture
- **Clean Architecture**: Follow the `Repository -> Usecase -> Handler` pattern.
- **Validation**: All DTOs and input structures must be validated using `go-playground/validator/v10`.
- **Database**: Use **GORM** for ORM operations and **Casbin v3** for authorization logic.
