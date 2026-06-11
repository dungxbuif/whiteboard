---
artifact_type: changelog
id: CHANGELOG
status: active
owner: shared
human_fields: [release_approval]
ai_fields: [change_entries, linked_releases]
shared_fields: [status]
---

# Changelog

## Field Ownership

- Human owns release approval.
- AI maintains change entries and links to release notes.

All notable changes should be recorded here.

## [Unreleased]

*(No unreleased changes)*

## [1.1.1] - 2026-06-07

### Changed
- **Debugging Standards Enhancements**: Upgraded `docs/standards/DEBUGGING.md` to include 5 major proactive bug-fixing practices for AI agents. The detailed rationale for these changes is as follows:
  - **Assess Impact (Blast Radius Analysis)**:
    - *Problem:* Fixing a local bug in a shared library or core utility can cause a chain reaction, breaking multiple other features.
    - *Solution:* Added a mandatory `Assess Impact` phase before implementing fixes, requiring agents to find references and verify that the proposed change will not break dependent features.
  - **Horizontal Fix & Proactive Search**:
    - *Problem:* Fixing a single instance of a bug (e.g., a missing null check) often leaves similar undiscovered vulnerabilities elsewhere in the codebase.
    - *Solution:* Added a `Proactive Search` phase. After fixing the local bug, agents must scan the entire codebase for the same problematic pattern and address it proactively.
  - **Telemetry First for Opaque Bugs**:
    - *Problem:* The previous standard mandated blocking a bug if it could not be reproduced locally, which could stall progress on environment-specific production issues.
    - *Solution:* If a bug lacks local reproduction steps but occurs in higher environments, the agent must create a PR/Ticket to add Telemetry/Logs/Tracing to gather data before marking it as blocked.
  - **Strict Red-Green Testing**:
    - *Problem:* Writing regression proofs after the fix or without explicit validation can lead to tests that always pass, providing a false sense of security.
    - *Solution:* Enforced strict Red-Green testing. The regression test MUST be written and fail (Red) with recorded output *before* any fix code is written, ensuring the test is actually catching the bug.
  - **Anti-Patterns Documentation (Post-mortem)**:
    - *Problem:* Valuable knowledge gained from complex architectural bugs is often lost after the fix is merged.
    - *Solution:* Required documenting root causes and gotchas as *Anti-Patterns* during the Reconcile phase so future agents avoid repeating the same mistakes.

## [1.1.0] - 2026-06-05

### Added
- **Feedback Intake Flow**: Added `PHASE 0: FEEDBACK INTAKE & TRIAGE` to the Agent Lifecycle.
- **Feedback Funnel**: Created `docs/work/FEEDBACK_LOG.md` to serve as the raw intake funnel for all user feedback before entering the backlog.
- **Feedback Template**: Created `docs/templates/FEEDBACK.md` for deep-dive analysis of complex user feedback.
- **SDD Template**: Added `docs/templates/SDD.md` (Software Design Document) to track high-level system architecture and required its synchronization upon detail design changes in `DOCS.md`.

### Changed
- **Triage Types**: Allowed triage types for user feedback in `WORKFLOW.md` now explicitly include `Enhancement`.
- **Detail Design Revamp**: Revamped `docs/templates/DETAIL_DESIGN.md` structure. It now integrates visual components (Architecture Overview, Execution Flow, API & Data Model Design, Security) while retaining strict Harness lifecycle metadata and reconciliation rules.
- **Agent Rules**: Updated `AGENTS.md` (Rule 9: Completion Rules) to strictly enforce writing framework and user-facing changes to `CHANGELOG.md`.
- **Docs Guide**: Updated `docs/README.md` to include the new templates and feedback intake funnel.

## [1.0.0] - 2026-06-05

### Added
- Initial SDLC agent framework scaffold.
