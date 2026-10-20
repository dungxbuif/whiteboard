# Workspace SCIM architecture

SCIM 2.0 provisioning is scoped to one workspace and authenticated with a
dedicated revocable bearer token. Tokens are generated once, stored as SHA-256
hashes, and are never accepted by normal login, agent, or collaboration routes.

`SCIMService` implements RFC 7643/7644 Users and Groups, stable UUID resources,
ETags, bounded paging, equality filters, and PATCH add/replace/remove. A SCIM
mapping records the external ID and managed membership provenance. Existing
manual membership is preserved unless it is explicitly adopted. Group-derived
roles are limited to editor/viewer; SCIM cannot create or remove an owner.

Membership provenance has four source kinds: direct SCIM user, SCIM group,
explicitly adopted membership, and manual owner/invitation grant. Canonical
member-role and invitation acceptance paths record late manual grants. Effective
access is the strongest active editor/viewer source. Deprovision and group
deletion remove only their managed sources before recomputing the membership;
the polymorphic source ID intentionally has no invalid cross-table foreign key.

All mutations serialize on the workspace lock. Deprovision removes only
SCIM-managed access, revokes that workspace's SSO sessions, bumps the existing
ACL version through membership triggers, and records a safe event. Existing
room, media, command, and asset grants therefore fail their next current-
authority check. Other workspaces and a user's default My Workspace are not
modified.

Create retries with the same workspace-scoped `externalId` return the stable
resource when the identity still matches. Conflicting identities return the
SCIM uniqueness error. Updates use weak ETags; stale `If-Match` values return
`412`. Owners can explicitly adopt a retained manual membership through
`POST /api/v1/workspaces/{workspace}/scim/users/{scimUser}/adopt`. Adoption is
idempotent and makes later SCIM deprovision responsible for that membership.

Migration `000027_workspace_scim` owns credentials, users, groups, group
members, and membership-source tables. Root wiring mounts discovery and SCIM
resource routes under `/api/v1/scim/v2` with SCIM bearer middleware, and mounts
owner token/configuration routes under ordinary session authentication.

Public machine-readable User and Group schemas are published at
`/scim/user-schema.json` and `/scim/group-schema.json`; the required access
extension is `/scim/workspace-access-schema.json`. Runtime discovery is
authoritative for provider negotiation. The core Group schema contains only
RFC attributes. Workspace role and retained-manual state live under
`urn:ietf:params:scim:schemas:extension:whiteboard:2.0:WorkspaceAccess`.

SCIM repository SQL logging is scoped off because user names, display names,
filters, and external IDs are PII. Errors still propagate to the handler and
service logger as generic operational failures without SQL values.
