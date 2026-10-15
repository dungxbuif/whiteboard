# EXC-023: Sharing Integration Tests

## Status

done

## Lane

normal

## Product Contract

Integration tests: toggle sharing, public access, password challenge, read-only enforcement, legacy parity audit

## Relevant Product Docs

- `03-03-PLAN.md`

## Acceptance Criteria

- Integration tests cover all public sharing states (disabled, enabled, password-protected).
- Whiteboard read-only state is verified for public links.

## Design Notes

A Detail Design document following `docs/templates/detail_design.md` is recorded at `EXC-023-design.md` and approved for implementation.

- **Link to Detail Design**: [Design Document](EXC-023-design.md)

- **Modified Files**: `tests/integration/sharing_test.go`
- **Sprint Phase**: phase-03

## Validation

| Layer | Expected proof |
| --- | --- |
| Unit | go test ./tests/integration/sharing_test.go |
| Integration | |
| E2E | |
| Platform | |
| Release | |

## Harness Delta

- Registered EXC-023 in the harness database.

## Evidence

- Sharing usecase tests cover enable, revoke, password failure, and success;
  the browser suite covers the public read-only viewer.
