# EXC-033: Multimodal: Image-to-Diagram & Visual Chat

## Status

implemented

## Lane

normal

## Product Contract

BE: AnalyzeImage + VisualChat methods, endpoints /ai/analyze-image + /ai/visual-chat. FE: VisualChat sidebar, ai-store.ts, Image upload flow

## Relevant Product Docs

- `05-03-PLAN.md`

## Acceptance Criteria

- Image-to-Diagram API generates Excalidraw structures from screenshots/drawings.
- VisualChat sidebar exports canvas viewport to blob and sends it as visual context to AI chat.
- Store manages session logs and conversational history.

## Design Notes

A Detail Design document following `docs/templates/detail_design.md` must be created at `docs/stories/phase-05/EXC-033-design.md` and approved by the reviewer before implementation starts.

- **Link to Detail Design**: [Design Document](EXC-033-design.md)

- **Modified Files**: `src/backend/internal/usecase/ai_usecase.go`, `ai_handler.go`, `src/frontend/src/components/ai/VisualChat.tsx`, `src/frontend/src/store/ai-store.ts`
- **Sprint Phase**: phase-05

## Validation

| Layer | Expected proof |
| --- | --- |
| Unit | go build ./src/backend/cmd/server && cd dashboard && npm run build |
| Integration | |
| E2E | |
| Platform | |
| Release | |

## Harness Delta

- Registered EXC-033 in the harness database.

## Evidence

Gemini requests accept validated base64 images for image-to-diagram and visual chat; repository tests cover data URL validation.
