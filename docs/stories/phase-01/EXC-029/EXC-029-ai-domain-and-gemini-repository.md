# EXC-029: AI Domain & Gemini Repository

## Status

planned

## Lane

normal

## Product Contract

Package check, AIRepository interface, GeminiRepository implementation using Google Generative AI SDK

## Relevant Product Docs

- `05-01-PLAN.md`

## Acceptance Criteria

- AIRepository interface defines content and multimodal generation methods.
- GeminiRepository compiles and initializes Google Generative AI SDK connection.

## Design Notes

A Detail Design document following `docs/templates/detail_design.md` must be created at `docs/stories/phase-05/EXC-029-design.md` and approved by the reviewer before implementation starts.

- **Link to Detail Design**: [Design Document](EXC-029-design.md)

- **Modified Files**: `internal/domain/ai.go`, `internal/repository/gemini_repository.go`
- **Sprint Phase**: phase-05

## Validation

| Layer | Expected proof |
| --- | --- |
| Unit | go build ./internal/repository/... |
| Integration | |
| E2E | |
| Platform | |
| Release | |

## Harness Delta

- Registered EXC-029 in the harness database.

## Evidence

(No evidence yet)
