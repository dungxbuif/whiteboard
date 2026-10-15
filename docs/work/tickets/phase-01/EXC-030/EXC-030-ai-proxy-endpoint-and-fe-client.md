# EXC-030: AI Proxy Endpoint & FE Client

## Status

implemented

## Lane

normal

## Product Contract

AIUsecase (pass-through chat), AIHandler (POST /ai/chat), route registration, FE ai-client.ts client stubs

## Relevant Product Docs

- `05-01-PLAN.md`

## Acceptance Criteria

- AI REST proxy endpoint is registered on backend router.
- AI client handles requests and JWT auth headers on frontend.
- Chat endpoint successfully proxies prompts to the configured OpenAI-compatible repository.

## Design Notes

A Detail Design document following `docs/templates/detail_design.md` must be created at `docs/stories/phase-05/EXC-030-design.md` and approved by the reviewer before implementation starts.

- **Link to Detail Design**: [Design Document](EXC-030-design.md)

- **Modified Files**: `src/backend/internal/usecase/ai_usecase.go`, `src/backend/internal/delivery/http/ai_handler.go`, `router.go`, `src/frontend/src/lib/ai-client.ts`
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

- Registered EXC-030 in the harness database.

## Evidence

Authenticated AI routes and the frontend AI store are implemented; backend and frontend tests pass.
