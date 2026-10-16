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
