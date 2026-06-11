# Debugging Standard

This file defines how agents handle bugs, test failures, build failures, production issues, and unexpected behavior.

## Principle

No fixes without root cause investigation.

Agents MUST NOT patch symptoms, stack multiple guesses, or claim a bug is fixed without reproduction and verification evidence.

## Required Debugging Flow

```text
Reproduce (or Add Telemetry) -> Investigate Root Cause -> Compare Patterns -> Form Hypothesis -> Create Failing Test (Red) -> Assess Impact -> Fix -> Verify (Green) -> Proactive Search -> Reconcile & Update Anti-patterns
```

## Phase 1: Reproduce

Before proposing a fix:

- [ ] Record exact steps, command, input, or environment that triggers the issue.
- [ ] Record expected behavior and actual behavior separately.
- [ ] Determine whether the issue is consistent, intermittent, or environment-specific.
- [ ] Capture relevant error messages, stack traces, logs, screenshots, or terminal output summaries.

If the bug cannot be reproduced but still occurs in higher environments:
- Do NOT mark it blocked immediately.
- Create a PR/Ticket to add Telemetry/Logs/Tracing to the suspected area.
- Deploy and gather data before returning to this phase.
If the bug simply lacks reproduction steps locally or is a known environment issue, gather more evidence or mark the bug blocked. Do not guess.

## Phase 2: Root Cause Investigation

Before changing implementation:

- [ ] Read error messages and stack traces completely.
- [ ] Check recent changes, dependencies, config, and environment.
- [ ] Trace data/control flow backward from symptom to source.
- [ ] For multi-component systems, inspect each boundary where data/config/state crosses.
- [ ] Identify the earliest source of incorrect behavior.

Root cause must be written in the bug or detail design.

## Phase 3: Pattern Comparison

Before choosing a fix:

- [ ] Find similar working code or previous decisions in the repo.
- [ ] Compare broken and working paths.
- [ ] Identify differences that could explain the failure.
- [ ] Check whether the current architecture or contract is being violated.

## Phase 4: Hypothesis

Write a single hypothesis:

```text
I think <root cause> causes <symptom> because <evidence>.
```

Then test one variable at a time. Do not bundle multiple speculative changes.

## Phase 5: Regression Proof (Red)

Before writing any fix code:

- [ ] Create or identify an automated test/reproduction that PROVES the bug.
- [ ] The test MUST fail (Red) and output should be recorded in the ticket/PR.
- [ ] Define the regression check that must pass after the fix.
- [ ] Update `docs/work/VALIDATION_MATRIX.md` when behavior proof changes.

## Phase 5.5: Assess Impact (Blast Radius Analysis)

Before implementing the fix:

- [ ] Identify all references to the file/function/component being changed.
- [ ] Verify that the proposed fix will not break other dependent features or cause a chain reaction of failures.

## Phase 6: Fix And Verify

After root cause and regression proof:

- [ ] Implement the smallest fix that addresses the root cause.
- [ ] Run regression proof to ensure it now passes (Green).
- [ ] Run relevant surrounding tests to guarantee no side effects.
- [ ] Record command/manual evidence.

## Phase 7: Proactive Search (Horizontal Fix)

After the local fix is verified:

- [ ] Search the codebase (regex/AST) for the same problematic pattern.
- [ ] If found, either fix them in the same PR or create new backlog items to address them proactively.

## Phase 8: Reconcile Docs & Anti-Patterns

- [ ] Reconcile requirements, architecture, API, ERD/data, ADR, and context as needed.
- [ ] If the bug was complex or architectural, log the root cause pattern into `docs/standards/ANTI_PATTERNS.md` (or equivalent) so future agents avoid the same mistake.

## Stop Conditions

Stop and ask for human/design review when:

- The same failure path still fails after 3 fix attempts.
- The task reaches 5 total fix/test cycles.
- Each fix reveals a new shared-state, coupling, or architecture problem.
- The issue cannot be reproduced but impact is high.
- The fix requires broad refactoring not covered by the active ticket/bug.
- The root cause implies an architecture, API, data, security, or standards decision.

When stopping, mark the work item `blocked` and record:

- Attempt count
- Commands run
- Changes made in each attempt
- Failure output summaries
- Current root-cause hypothesis
- Human/design decision needed

## Red Flags

If an agent writes or thinks any of these, return to root cause investigation:

- "Try this and see."
- "Quick fix for now."
- "It is probably X."
- "I will skip the test."
- "I do not fully understand, but..."
- "One more fix attempt."
- "Docs can be updated later."
