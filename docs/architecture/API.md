---
artifact_type: api_contract_index
id: API-MASTER
status: draft
owner: shared
human_fields: [contract_intent, approval]
ai_fields: [contract_rows, errors, auth_notes, versioning, linked_decisions]
shared_fields: [status, trace]
---

# API

## Field Ownership

- Human owns public contract intent and approval for contract changes.
- AI maintains contract rows, errors, auth notes, versioning, and linked decisions.

## API Surface

Document HTTP endpoints, RPC methods, events, CLI commands, or any other public contract.

| Contract | Type | Auth | Status | Notes |
| --- | --- | --- | --- | --- |
| `POST /api/v1/auth/login` | HTTP | Public | active | Temporary single-admin login; replaced by EXC-034. |
| `GET, POST /api/v1/workspaces` | HTTP | JWT | active | List and create workspaces. |
| `GET, POST /api/v1/folders` | HTTP | JWT + workspace | active | List/create folders; `all=true` returns flat tree data. |
| `PATCH, DELETE /api/v1/folders/{id}` | HTTP | JWT + workspace | active | Folder move and trash lifecycle routes. |
| `GET, POST /api/v1/scenes` | HTTP | JWT + workspace | active | List/create scenes; `trash=true` lists deleted scenes. |
| `GET, PATCH, DELETE /api/v1/scenes/{id}` | HTTP | JWT + workspace | active | Read, update, and trash scene JSONB content. |
| `POST /api/v1/scenes/{id}/star` | HTTP | JWT + workspace | active | Explicitly set starred state. |
| `POST /api/v1/scenes/{id}/restore` | HTTP | JWT + workspace | active | Restore a trashed scene. |

## Errors

| Error | Meaning | Consumer Impact |
| --- | --- | --- |
| `400 Bad Request` | Invalid UUID, body, title, content, or folder mapping | Request must be corrected. |
| `401 Unauthorized` | Missing/invalid JWT or login credentials | Consumer must authenticate again. |
| `404 Not Found` | Workspace-scoped entity does not exist | Consumer must discard stale entity reference. |
| `500 Internal Server Error` | Repository or infrastructure failure | Consumer may retry safely where appropriate. |

## Versioning

Phase 1 HTTP contracts are versioned under `/api/v1`.

## Linked Decisions

- TBD
