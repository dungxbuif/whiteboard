# Validation Standard

This file defines validation policy. Runtime proof state belongs in `docs/work/VALIDATION_MATRIX.md`.

## Principle

Every accepted behavior needs proof. The proof can be automated, manual, or explicitly not required with a reason, but it must be recorded.

## Proof Types

| Proof Type | Use When | Evidence |
| --- | --- | --- |
| Unit | Pure domain logic, helpers, parsing, validation, isolated business rules | Test command and result |
| Integration | Backend enforcement, persistence, provider behavior, jobs, service contracts | Test command, fixture/setup notes, result |
| E2E | User-visible browser/app flow | Browser/app test command or manual steps |
| UAT | User-facing acceptance criteria or workflow sign-off | Expected behavior, verified behavior, sign-off or not-required reason |
| Platform/manual | Runtime, shell, deployment, mobile, desktop, environment-specific behavior | Manual steps, screenshots/log summary, or command output |
| Docs review | Behavior, contract, architecture, data, or decision changes | Docs review checklist result |

## Required Proof By Work Type

| Work Type | Required Proof |
| --- | --- |
| Tiny docs-only | Docs review or not-needed reason |
| Feature/ticket | Unit or integration where applicable, UAT for user-facing behavior, docs review |
| Bug | Reproduction evidence, regression proof, docs review |
| API/contract | Integration or contract proof, docs review, UAT if user-visible |
| Data/schema | Migration/schema-sensitive proof, docs review, ADR when ownership/tradeoff changes |
| Security/auth | Integration or E2E proof, negative cases, docs review, ADR when model changes |
| Deployment/runtime | Platform/manual proof, rollback or release check, docs review |
| Standards/framework | Docs review and trace links |

## Evidence Rules

- Record the exact command when a command was run.
- Record `pass`, `fail`, or `skipped`.
- If skipped, record why and what residual risk remains.
- Do not mark behavior `implemented` in `docs/work/VALIDATION_MATRIX.md` without evidence.
- If validation requirements change, update the matrix and create an ADR when the change is durable or risky.
