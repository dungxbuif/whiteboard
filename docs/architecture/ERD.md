---
artifact_type: erd_master
id: ERD-MASTER
status: draft
owner: shared
human_fields: [data_ownership_decisions, migration_approval]
ai_fields: [entities, relationships, constraints, migrations, linked_decisions]
shared_fields: [status, trace]
---

# ERD

## Field Ownership

- Human owns data ownership and migration approval.
- AI maintains entities, relationships, constraints, migrations, and linked decisions.

## Entities

| Entity/Table | Purpose | Owner | Notes |
| --- | --- | --- | --- |
| users | Password identities; soft-delete disables access | Identity | UUID primary key; status changes invalidate membership workspaces. |
| workspaces | Tenant and authority lock | Workspace owner | UUID primary key, nullable unique public_id, positive acl_version. |
| workspace_members | Current workspace role | Owner management | Composite workspace_id/user_id primary key; owner/editor/viewer. |
| scenes | Scene content and metadata | Workspace | UUID key, nullable unique public_id; workspace_id/id unique pair. |
| scene_member_overrides | Per-scene content role | Owner management | scene_id/user_id primary key; editor/viewer only. |
| acl_audit_events | Durable permission metadata | Access authority | Sequence, workspace, optional scene/actor/subject, event type and ACL version; no raw credentials. |

## Relationships

| From | To | Relationship | Notes |
| --- | --- | --- | --- |
| workspace_members | workspaces, users | Many-to-one | Foreign keys; removal cascades dependent overrides. |
| scene_member_overrides | scenes | Composite foreign key | workspace_id/scene_id enforces scene tenancy. |
| scene_member_overrides | workspace_members | Composite foreign key | workspace_id/user_id requires current membership. |
| acl_audit_events | workspaces | Many-to-one | Cascades on physical workspace removal. |

## Constraints

- Workspace roles: owner/editor/viewer; override roles: editor/viewer. Public aliases are 21 URL-safe characters, unique within each resource table; UUIDs remain canonical.

## Migrations

- 000007_collab_access: maps member to editor, backfills owner memberships, creates override/audit storage and authority triggers; down maps editor/viewer back to member and removes v2-only ACL data.

## Linked Decisions

- [ADR 0009 — collaboration access](../decisions/0009-collaboration-access.md).

## Collaboration A1 implemented schema (release pending)

Migration 000007 retains UUID primary keys and adds nullable unique `public_id` aliases to workspaces/scenes, a positive workspace `acl_version`, constrained owner/editor/viewer memberships, member-dependent `scene_member_overrides`, and `acl_audit_events`. Legacy member becomes editor; missing owner memberships are backfilled. Scene override composite foreign keys enforce both workspace/scene relationship and current membership; removal cascades overrides. ACL changes increment the workspace authority version transactionally and notify listeners. Writes lock workspace before scene; no permission check may race a later revoke. Rollback maps editor/viewer back to legacy member and drops v2-only ACL data; rollout must back up first.

`collab_bump_acl(workspace_uuid, scene_uuid_or_null, subject_uuid_or_null, event_text)` is the shared SQL helper for future invite/share/agent grant mutations. It increments the workspace version, writes `acl_audit_events`, and publishes commit-bound `acl_changed` JSON with `workspaceId` and decimal-string `aclVersion`. Call under the workspace authority lock and transaction; the helper is not a replacement for permission validation. Existing triggers cover workspace membership, scene overrides, scene deletion/restoration, and user/workspace soft-delete changes. New workspace inserts create their owner membership automatically.

79 backend tests passed using disposable PostgreSQL 15, including actual up/down migrations and two-connection lock/rollback proof. Deployment and data backup remain release work.

## A2 invitation schema (implemented and reviewed; release pending)

Migration `000011_workspace_invitations` stores workspace/email/role/inviter, a unique SHA-256 token hash, creation/expiry and consumed/revoked timestamps. Invites only grant editor/viewer; tokens are never recovered by listing. The workspace is locked before the invitation row, and account creation, default personal workspace, invited membership and consumption commit or roll back together.

Version 11 follows reserved share-link, collab and agent migrations 8–10. Ship the complete v2 migration stream together from production version 6; do not apply 11 early and add lower versions afterward. A2 isolated fixtures explicitly apply 1–7 and 11. [Invitation behavior](../user/collaboration.md), [proof](../work/verification/collaboration-v2.md).

## A4 scene asset capabilities (deployed and production storage-canary verified)

`scene_assets` remains scene-owned. Migration 000012 adds `uploaded_principal_id` and `uploaded_principal_type` for guest audit while `uploaded_by` remains nullable and references real users only. Active file IDs are unique within `(workspace_id,scene_id,file_id)`; metadata reads require both workspace and scene. Grants are transient signed decisions, not durable ACL truth. [Authority protocol](mutation-authority.md) defines workspace-lock/version validation and revocation proof. Verification uses isolated PostgreSQL schemas and tests public guest uploads, scene isolation, and revoke races.

Migration 000012 changes active Excalidraw file-ID uniqueness to workspace+scene. Down migration recreates the legacy unique index first and intentionally fails if two scenes share a live file ID; operators must resolve collisions explicitly. It never deletes user assets to make rollback fit. User-principal audit strings survive user deletion; `uploaded_by` can become NULL through its existing foreign key. Historical rows without uploader keep nullable principal audit fields. Public rows require guest identity and nullable user UUID. No new durable capability table is introduced.


## Complete v2 durable schema (deployed at migration 13, clean)

| Migration / table | Durable contract |
| --- | --- |
| 000008 / `share_links` | Independent view/edit links, hash-only lookup, optional password/expiry, revision invalidation; legacy links migrate as view grants |
| 000009 / `scenes.writer_mode` | Exactly v1 or v2; promotion fences the old content PATCH writer |
| 000009 / `collab_documents` | One per scene; unique document UUID, latest and snapshot bigint sequences, snapshot bytes, original promotion JSON/version; composite workspace/scene foreign key |
| 000009 / `collab_updates` | Ordered binary updates keyed by scene/positive sequence; compacted after snapshot |
| 000009 / `collab_receipts` | Permanent scene/request ID dedup, exact principal/payload hash/action, unique sequence and state vector |
| 000009 / `collab_mutation_audit` | Receipt-linked principal, workspace, sequence, payload hash/action and ACL version; retained with receipts |
| 000010 / `agent_keys` | Workspace/owner, unique token hash, independent JSON scopes and optional scene targets; changes advance ACL version |
| 000011 / `workspace_invitations` | Seven-day email-bound single-use invitation metadata/hash, consumption and revocation timestamps |
| 000012 / `scene_assets` | Scene-scoped immutable file identity and user/guest uploader metadata; transient signed capabilities are not stored as authority |
| 000013 / `agent_operation_audit` | Go-authoritative operation outcome, key/principal/scope/correlation, latency and failure class; no raw token or scene contents |

Deleting a `collab_documents` row cascades its update log and receipts, then receipt-linked mutation audit. Therefore reset-by-delete is forbidden as a recovery shortcut. Trash leaves these rows intact. Replacement-scene recovery retains original rows and an independently verified archive. Mutation receipts are not a user-facing version-history product.

Presence, cursor, viewport/follow and voice signaling remain ephemeral. Agent operation audit and durable scene mutation audit have different provenance and purposes. Public short IDs remain aliases; UUIDs are canonical keys and foreign-key targets. All API sequences are decimal strings to avoid JavaScript number precision loss.
