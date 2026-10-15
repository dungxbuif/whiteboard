---
artifact_type: phase
id: PHASE-2
status: ready
owner: human
human_fields:
  - goal
  - scope
  - out_of_scope
  - priority
  - success_criteria
ai_fields:
  - risks
  - dependencies
  - verification_plan
  - completion_summary
shared_fields:
  - status
  - trace
  - tickets_and_bugs
trace:
  backlog_items: []
  roadmap: ROADMAP
  requirements: []
  tickets: []
  bugs: []
  test_verification: TBD
  validation_matrix: TBD
  adrs: []
  release_notes: TBD
---

# Phase: PHASE-2 Develop Harness CLI (Automated Hard Gates) & Security Guardrails

## Field Ownership

- Human fills goal, scope, out of scope, priority, and success criteria.
- AI fills risks, dependencies, verification plan, and completion summary.
- Shared fields include status, trace links, and ticket/bug list.

## Status

- ID: PHASE-2
- Status: ready
- Owner: human
- Created: 2026-06-05
- Updated: 2026-06-07

## Trace Links

- Backlog items: TBD
- Roadmap: [ROADMAP.md](../ROADMAP.md)
- Requirements: TBD
- Tickets: TBD
- Bugs: TBD
- Test verification: TBD
- Validation matrix: TBD
- ADRs: TBD
- Release notes: TBD

## Goal

Develop the `harness` CLI tool to definitively solve the LLM "Honor System". This physical script will parse the defined YAML frontmatter structures and act as a "Hard Gate", preventing AI Agents from bypassing or executing the SDLC incorrectly. Additionally, establish strict Security Guardrails to govern AI behavior when writing code.

## Scope

- Design and develop the `harness` CLI (can be written in Bash, Python, or Node.js).
- Implement the `harness check-ticket <id>` command:
  - Scan the ticket's YAML file.
  - Verify the presence of sufficient Trace links (Forward/Backward links).
  - Verify that Human has set `Approved: true` (if required).
  - Return an error code (`exit 1`) if the above conditions are not met.
- Implement the `harness guard` command:
  - Hook integrated into the testing process (Execution & Test phase).
  - Count the number of test runs/AI bug fix loops.
  - If the number of test failures exceeds the limit (e.g., > 5 times), throw an error locking the screen, forcing the AI to stop the session and wait for Human review.
- Update documentation (`README`/`AGENTS.md`) to force AI to call these CLI commands in Phase 2 (Detail Design) and Phase 3 (Execution).
- **Security Guardrails Implementation**:
  - Create the `docs/standards/SECURITY.md` standard for strict security compliance.
  - Integrate language-specific security checks and skills (Go, Node.js, Python) into the `DETAIL_DESIGN.md` phase.

## Out Of Scope

- Implementing a complete Harness Orchestrator for multi-agent management.
- Automated CI/CD features on a server (Focusing only on IDE/Local hooks).

## Tickets And Bugs

| ID | Type | Title | Status | Link |
| --- | --- | --- | --- | --- |
| TBD | Ticket | Implement `harness check-ticket` | draft | TBD |
| TBD | Ticket | Implement `harness guard` | draft | TBD |
| TBD | Ticket | Update AGENTS.md to integrate CLI | draft | TBD |
| TBD | Ticket | Create Security Guardrails standard (`SECURITY.md`) | draft | TBD |
| TBD | Ticket | Integrate language-specific security skills into `DETAIL_DESIGN.md` | draft | TBD |

## Dependencies

- Depends on the YAML Frontmatter structure standardized in Phase 1.
- Node.js or Python (depending on the language chosen for easy YAML parsing).

## Risks

- If AI still has permission to "edit" the CLI script files or bypass calling the CLI commands, vulnerabilities remain. (Needs Read-Only configuration or IDE-level hooks).
- YAML parser compatibility across different operating systems (Windows, MacOS, Linux) if using pure Bash.

## Success Criteria

- Successfully block an Agent attempting to write code when `DETAIL_DESIGN` has not been approved.
- Successfully block an Agent caught in a fix loop (Infinite Loop) at the 6th cycle.
- Scripts can run directly on the Local Repo without overly complex installation.
- AI correctly references `SECURITY.md` and runs language-specific security checks during Detail Design.

## Verification Plan

- Simulate an Agent intentionally generating code without Approval -> `harness check-ticket` must throw an `exit 1` error.
- Simulate an Agent creating a continuous 6-time test failure loop -> `harness guard` must throw a block execution error.
- Verify an Agent applies security constraints appropriately during a mock Detail Design task.

## Gate Checklist

- [x] Phase has linked requirements or explicit discovery goal
- [ ] Tickets/bugs are created or planned
- [ ] Risks and dependencies are recorded
- [ ] Verification plan is defined
- [ ] Release/changelog need is identified

## Completion Summary

Complete this section when the phase is done.
