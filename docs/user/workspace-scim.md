# Provision workspace members with SCIM

Workspace owners can create or rotate a dedicated SCIM token from **Settings →
Identity & SCIM**. Copy the token when it is created; the server stores only its
hash. Configure the IdP base URL shown by the app and use SCIM 2.0 Users and
Groups.

SCIM can grant viewer or editor access. It cannot grant owner access. Disabling
a provisioned user removes only access managed by that SCIM connection and
ends current SSO sessions for this workspace. Independently granted manual
access and access to other workspaces remain visible and active.

Rotate or revoke a token immediately if it is exposed. After revocation the old
token receives `401`. Provider retries should reuse the same external ID and
ETag; stale updates receive `412` and must be re-read before retrying.

The base URL is `/api/v1/scim/v2`. Discovery is available at
`ServiceProviderConfig`, `ResourceTypes`, and `Schemas`; provisioning uses
`Users` and `Groups`. Results start at index 1 and return at most 100 resources.
Bulk, password changes, and sorting are not supported. Machine-readable schema
references are available at `/scim/user-schema.json` and
`/scim/group-schema.json`. Role is supplied through the required
`urn:ietf:params:scim:schemas:extension:whiteboard:2.0:WorkspaceAccess`
extension, documented at `/scim/workspace-access-schema.json`.

When a SCIM record matches an existing manual workspace member, the app marks
it as retained and SCIM deprovision leaves it in place. An owner must explicitly
adopt that member before SCIM can remove the membership.

Manual role changes and accepted invitations made after provisioning are also
preserved. Adoption transfers that specific manual membership to SCIM; later
SCIM role changes then update it, including editor-to-viewer changes.
