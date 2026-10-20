# Dashboard management API

Base path: `/api/v1`. Use a login bearer token. Folder and scene requests also
send `X-Workspace-ID` with the current workspace UUID or public alias. Every
request is checked against current membership and scene access; hiding an
action in the dashboard is not authorization.

| Request | Access | Body | Success |
| --- | --- | --- | --- |
| `PATCH /workspaces/{id}` | Workspace owner | `{ "name": "New name" }` (trimmed, 3–100 characters) | `200` updated workspace; ID stays the same |
| `PATCH /folders/{id}` | Workspace owner/editor | Optional `name` (trimmed, 1–100 characters), optional `newParentId` (folder UUID or `null` for root) | `204`; omitted parent leaves location unchanged |
| `PATCH /scenes/{id}` | Current scene edit access | `{ "folderId": "folder UUID" }` or `{ "folderId": null }` | `200` updated scene |
| `GET /scenes/{id}/projection` | Login user with current `scene:read` capability | None | `200` durable projection with `content` and `sequence` |
| `POST /scenes` | Current scene creation access | `{ "title": "Copy name", "folderId": null, "content": { ... } }` | `201` new scene |

The projection includes acknowledged collaboration updates after the last
scene-row compaction. It is for authenticated v2 scene reads, including scene
duplication. For legacy scenes, read `GET /scenes/{id}`. Copy only canonical
`content` into the create request; do not copy presence, collaborator, or
session state. Projection requests accept scene UUIDs and public aliases within
the selected workspace. All projection responses, including authentication
failure and service errors, set `Cache-Control: no-store`; clients must not
cache the content or state vector. A missing collaboration writer fails closed.

Folder moves reject self/descendant and depth-invalid targets. A missing parent
returns 404. Invalid names or target shapes return structured
`400 INVALID_INPUT`; current access failures use scoped 403/404 responses.
Collections policy denial is `403 FEATURE_DISABLED`: show an explanatory
disabled state, without Retry or a creation action. Transport and server
failures, and other 403 responses, remain retryable.
