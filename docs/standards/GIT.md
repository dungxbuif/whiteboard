# Git Conventions

## Branches

Use these branch patterns:

- `feature/<ticket-id>-<short-name>`
- `bugfix/<ticket-id>-<short-name>`
- `hotfix/<ticket-id>-<short-name>`
- `docs/<ticket-id>-<short-name>`

Before work, agents must check the current branch and avoid mixing unrelated changes.

## Commits

Use concise conventional commit prefixes:

- `feat:`
- `fix:`
- `docs:`
- `test:`
- `refactor:`
- `chore:`

## Pull Requests

A PR or merge request should include:

- Summary of behavior changes
- Test commands and results
- Docs updated or reason docs were not needed
- Linked ticket, bug, phase, and ADRs

## Protected Work

Agents must not rewrite history, reset branches, or discard user changes unless the user explicitly requests it.
