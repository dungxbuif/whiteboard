---
artifact_type: decision
id: ADR-0008
status: accepted
owner: shared
---

# ADR-0008: OpenAI-Compatible AI Provider Boundary

## Decision

The backend uses an OpenAI-compatible `chat/completions` adapter for all AI
features. DeepSeek is the default deployment target, configured with
`https://api.deepseek.com/v1`, `deepseek-chat`, and `deepseek-reasoner`.

The API key, base URL, and model names are environment configuration. The
frontend talks only to the authenticated `/api/v1/ai/*` proxy and never sees
the provider key.

## Rationale

- DeepSeek can be supplied without changing the domain or frontend contracts.
- A standard chat-completions payload keeps the repository portable to other
  compatible providers or a self-hosted gateway.
- Image flows remain available when the selected compatible model supports
  vision; text-only models return the provider's normal capability error.

## Consequences

- Production must set `AI_API_KEY` and should explicitly set `AI_BASE_URL` and
  model names when using a provider other than DeepSeek.
- Provider-specific output differences are normalized by the repository and
  usecase output cleaning/validation.
