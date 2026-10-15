# Phase 5 Detail Design: OpenAI-Compatible AI Features

## Intended behavior

- Authenticated users can submit text or image context through the backend proxy.
- Text-to-diagram returns validated Excalidraw scene content. Invalid model output is rejected with a useful error.
- Diagram-to-code returns a self-contained React/HTML snippet that reflects the selected scene elements.
- The frontend exposes a command panel, a code preview, and a visual chat surface without exposing the provider key.

## Technical decisions

- Keep an OpenAI-compatible provider behind a repository interface so tests use a deterministic fake and production uses the configured REST API.
- The backend owns prompt construction, output cleaning, JSON schema validation, and request limits.
- Responses use a discriminated payload `{kind, content, model, usage}` so the frontend can render diagram, code, and chat results consistently.
- Image input is accepted as a base64 data URL only after size and MIME validation.

## Affected configuration and API

- Required production secret: `AI_API_KEY` (DeepSeek key by default).
- Optional settings: `AI_BASE_URL`, `AI_MODEL_FAST`, `AI_MODEL_REASONING`, `AI_MAX_PROMPT_BYTES`.
- New endpoints: `POST /api/v1/ai/chat`, `POST /api/v1/ai/generate-diagram`, `POST /api/v1/ai/to-code`.

## Verification plan

- Unit tests validate fenced JSON cleaning, Excalidraw shape validation, prompt limits, and repository error mapping.
- Handler tests verify authentication, request validation, and stable response envelopes with a fake repository.
- Frontend tests cover command submission, diagram insertion, code preview, and visual chat error states.
- Build and the existing end-to-end suite must remain green without requiring a live provider key.
