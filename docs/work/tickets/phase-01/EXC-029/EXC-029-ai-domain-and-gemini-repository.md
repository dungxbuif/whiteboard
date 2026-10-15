# EXC-029: AI Domain & OpenAI-Compatible Repository

## Status

implemented

## Lane

normal

## Product Contract

AIRepository interface and OpenAI-compatible chat-completions implementation with DeepSeek defaults

## Relevant Product Docs

- `05-01-PLAN.md`

## Acceptance Criteria

- AIRepository interface defines content and multimodal generation methods.
- The provider repository compiles and initializes from `AI_API_KEY`, `AI_BASE_URL`, and model settings.

## Design Notes

A Detail Design document following `docs/templates/detail_design.md` must be created at `docs/stories/phase-05/EXC-029-design.md` and approved by the reviewer before implementation starts.

- **Link to Detail Design**: [Design Document](EXC-029-design.md)

- **Modified Files**: `src/backend/internal/domain/ai.go`, `src/backend/internal/repository/openai/openai_repo.go`
- **Sprint Phase**: phase-05

## Validation

| Layer | Expected proof |
| --- | --- |
| Unit | go build ./src/backend/internal/repository/... |
| Integration | |
| E2E | |
| Platform | |
| Release | |

## Harness Delta

- Registered EXC-029 in the harness database.

## Evidence

AI contracts, OpenAI-compatible REST repository, and deterministic unit tests are implemented.
