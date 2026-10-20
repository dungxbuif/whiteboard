# Workspace SCIM 2.0

Base path: `/api/v1/scim/v2`. Authenticate with the dedicated workspace SCIM
bearer token. Supported resources: ServiceProviderConfig, ResourceTypes,
Schemas, Users, and Groups. List paging uses `startIndex` (minimum 1) and
`count` (maximum 100). User filters support `userName eq`, `externalId eq`, and
`id eq`; group filters support `displayName eq`, `externalId eq`, and `id eq`.

Mutations return ETags. Send `If-Match` on replace, patch, and delete. Group
roles are `editor` or `viewer`; owner is rejected. Bulk is unsupported. SCIM
deprovision removes only SCIM-managed membership and revokes current workspace
SSO sessions while preserving manual and other-workspace access.

Create retries with the same external ID return the stable resource. A
conflicting identity returns `409`; a stale ETag returns `412`. Existing manual
memberships are retained until a workspace owner explicitly adopts them.

Machine-readable references: `/scim/user-schema.json` and
`/scim/group-schema.json`.
