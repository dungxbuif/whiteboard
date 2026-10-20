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
| `POST /scenes/{id}/duplicate` | Login user with scene edit access and workspace owner/editor membership | `{ "title": "Copy name", "folderId": null }` (folder UUID or `null`) | `201` new scene with copied content and images |
| `POST /scenes` | Workspace creation access | `{ "title": "New scene", "folderId": null, "content": { ... } }` | `201` new scene or import; not a duplication route |

The projection includes acknowledged collaboration updates after the last
scene-row compaction. The ordinary projection is for bounded authenticated v2
reads; it has a 500-element/256 KiB limit. Use the duplicate endpoint for a
copy: the server reads the larger bounded export projection, copies scene-owned
image objects and rows, and rewrites image references. It keeps the new scene
hidden and out of Trash/search/Restore until the copy completes. If your edit
access is revoked during the copy, the server rechecks scene and workspace
rights before publication and removes the hidden copy. For legacy scenes, the
copy retains durable canvas settings and scene-owned image files while dropping
renderer/session fields and inline file bytes. An image without a scene-owned
asset, a thumbnail in place of a file asset, a mismatched image MIME, malformed
durable scene fields (including unsafe nested metadata, bindings, or pressure
values), or a broken image-to-file reference makes the copy fail instead of
producing a broken image. Nested metadata follows the collaboration scene's
32-level limit counted from the scene root. If cleanup
fails after an unsuccessful copy, the API returns `500 DUPLICATE_CLEANUP_FAILED`;
the partial scene stays hidden for operator recovery. If the server cannot
verify whether publication committed, it preserves the copy and returns
`503 DUPLICATE_STATE_UNCERTAIN`; reload before retrying to avoid another copy.
Presence, collaborator, and session state are
not copied. Projection requests accept scene UUIDs and public aliases within
the selected workspace. All projection responses, including authentication
failure and service errors, set `Cache-Control: no-store`; clients must not
cache the content or state vector. A missing collaboration writer fails closed.

Folder moves reject self/descendant and depth-invalid targets. A missing parent
returns 404. Invalid names or target shapes return structured
`400 INVALID_INPUT`; current access failures use scoped 403/404 responses.
Collections policy denial is `403 FEATURE_DISABLED`: show an explanatory
disabled state, without Retry or a creation action. Transport and server
failures, and other 403 responses, remain retryable.
