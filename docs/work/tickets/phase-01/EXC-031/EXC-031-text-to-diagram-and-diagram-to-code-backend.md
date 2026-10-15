# EXC-031: Text-to-Diagram & Diagram-to-Code (Backend)

## Status

implemented

## Lane

normal

## Product Contract

GenerateDiagram + GenerateCode methods in usecase, prompt templates, endpoints /ai/generate-diagram + /ai/to-code, JSON cleaning

## Relevant Product Docs

- `05-02-PLAN.md`

## Acceptance Criteria

- Prompt engineering converts text descriptions into valid Excalidraw element JSON schemas.
- /ai/to-code endpoint returns React/Tailwind code based on scene diagrams.

## Design Notes

A Detail Design document following `docs/templates/detail_design.md` must be created at `docs/stories/phase-05/EXC-031-design.md` and approved by the reviewer before implementation starts.

- **Link to Detail Design**: [Design Document](EXC-031-design.md)

- **Modified Files**: `src/backend/internal/usecase/ai_usecase.go`, `src/backend/internal/delivery/http/ai_handler.go`
- **Sprint Phase**: phase-05

## Validation

| Layer | Expected proof |
| --- | --- |
| Unit | go build ./src/backend/cmd/server |
| Integration | |
| E2E | |
| Platform | |
| Release | |

## Harness Delta

- Registered EXC-031 in the harness database.

## Evidence

Prompt cleaning, fenced JSON extraction, Excalidraw validation, and diagram-to-code generation are implemented with unit coverage.
