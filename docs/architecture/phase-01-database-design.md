# Phase 1 Database Design

## Status

Proposed high-level product contract for Phase 1 implementation. This document
does not replace story-level detail designs; it gives EXC-003, EXC-004,
EXC-006, EXC-008, EXC-009, EXC-011, EXC-013, and EXC-014 a shared database
shape before migrations and repositories are finalized.

## Scope

Phase 1 covers the self-hosted dashboard rewrite:

- Google OAuth user identity and persisted sessions.
- Multi-tenant workspace isolation.
- Workspace-owned folders with materialized paths.
- Workspace-owned Excalidraw scenes stored as JSONB.
- S3-backed scene assets with relational metadata.
- Search, starred views, trash views, and public scene permissions.

Out of scope for this document:

- Real-time collaboration state from Phase 4.
- AI request history or prompt artifacts from Phase 5.
- Workspace invitations and complex role management UI.

## Design Principles

- PostgreSQL is the system of record for identity, workspace metadata, folder
  metadata, scene metadata, scene JSON, asset metadata, and lifecycle flags.
- S3-compatible storage is the system of record for binary asset bytes.
- Every workspace-owned table must carry `workspace_id`.
- Repository queries must filter by the active workspace from Go context before
  applying folder, scene, search, star, or trash predicates.
- UUIDs are the primary identifiers for public and internal entity IDs.
- Soft delete uses `deleted_at`; hard delete is reserved for explicit future
  retention policy work.
- JSONB stores Excalidraw scene payloads because element schema evolution should
  not require database migrations for every Excalidraw-level change.

## Entity Map

```text
users
  1--N auth_sessions
  1--N workspaces.owner_id
  1--N workspace_members

workspaces
  1--N workspace_members
  1--N folders
  1--N scenes
  1--N scene_assets
  1--N scene_permissions

folders
  0/1--N folders.parent_id
  0/1--N scenes.folder_id

scenes
  1--N scene_assets
  1--1 scene_permissions
```

## Core Tables

### users

Stores the app user account created or updated by the password-auth user CLI.

| Column | Type | Notes |
| --- | --- | --- |
| `id` | `uuid` | Primary key, default `gen_random_uuid()`. |
| `email` | `text` | Required, unique case-insensitive login email. |
| `name` | `text` | Required display name. |
| `avatar_url` | `text` | Nullable Google profile image URL. |
| `password_hash` | `text` | Nullable bcrypt hash used by the temporary password login flow. |
| `google_subject` | `text` | Nullable, unique Google OAuth `sub` claim. |
| `last_login_at` | `timestamptz` | Nullable. |
| `created_at` | `timestamptz` | Required, default `now()`. |
| `updated_at` | `timestamptz` | Required, default `now()`. |

Indexes and constraints:

- Unique lower-case email index: `unique (lower(email))`.
- Unique `google_subject` where not null.

### auth_sessions

Stores revocable session records for JWT or secure-cookie sessions. Access
tokens can remain short-lived, but the persisted session row gives logout,
rotation, and future audit work a concrete anchor.

| Column | Type | Notes |
| --- | --- | --- |
| `id` | `uuid` | Primary key, default `gen_random_uuid()`. |
| `user_id` | `uuid` | Required FK to `users(id)`. |
| `refresh_token_hash` | `text` | Required if refresh tokens are issued. Never store raw tokens. |
| `user_agent` | `text` | Nullable. |
| `ip_address` | `inet` | Nullable. |
| `expires_at` | `timestamptz` | Required. |
| `revoked_at` | `timestamptz` | Nullable logout or forced revocation timestamp. |
| `created_at` | `timestamptz` | Required, default `now()`. |
| `updated_at` | `timestamptz` | Required, default `now()`. |

Indexes and constraints:

- `idx_auth_sessions_user_active` on `(user_id, revoked_at, expires_at)`.
- Unique `refresh_token_hash`.

### workspaces

Stores tenant roots. Phase 1 supports create/list/switch; only ownership is
needed by UI, but the schema keeps membership explicit for repository checks.

| Column | Type | Notes |
| --- | --- | --- |
| `id` | `uuid` | Primary key, default `gen_random_uuid()`. |
| `name` | `text` | Required. |
| `owner_id` | `uuid` | Required FK to `users(id)`. |
| `created_at` | `timestamptz` | Required, default `now()`. |
| `updated_at` | `timestamptz` | Required, default `now()`. |
| `deleted_at` | `timestamptz` | Nullable. Phase 1 may omit UI for deleting workspaces. |

Indexes and constraints:

- `idx_workspaces_owner_id` on `(owner_id)`.
- Optional uniqueness: `(owner_id, lower(name)) where deleted_at is null`.

### workspace_members

Makes workspace access checks explicit without committing to invitation flows in
Phase 1. Workspace creation inserts an `owner` row for the creator.

| Column | Type | Notes |
| --- | --- | --- |
| `workspace_id` | `uuid` | Required FK to `workspaces(id)`. |
| `user_id` | `uuid` | Required FK to `users(id)`. |
| `role` | `text` | Required. Phase 1 values: `owner`, `member`. |
| `created_at` | `timestamptz` | Required, default `now()`. |

Indexes and constraints:

- Primary key `(workspace_id, user_id)`.
- `idx_workspace_members_user_id` on `(user_id)`.
- Check `role in ('owner', 'member')`.

## Workspace-Owned Tables

### folders

Stores Google Drive-like folders. The app owns path mutation rules, including
descendant path updates on move.

| Column | Type | Notes |
| --- | --- | --- |
| `id` | `uuid` | Primary key, default `gen_random_uuid()`. |
| `workspace_id` | `uuid` | Required FK to `workspaces(id)`. |
| `parent_id` | `uuid` | Nullable FK to `folders(id)`. Null means workspace root. |
| `name` | `text` | Required. |
| `path` | `text` | Required materialized path. Use stable ID segments, e.g. `/{folder_id}/`, not names. |
| `depth` | `integer` | Required, default `0`. |
| `is_starred` | `boolean` | Required, default `false`. |
| `deleted_at` | `timestamptz` | Nullable trash marker. |
| `created_by` | `uuid` | Nullable FK to `users(id)`. |
| `created_at` | `timestamptz` | Required, default `now()`. |
| `updated_at` | `timestamptz` | Required, default `now()`. |

Indexes and constraints:

- `idx_folders_workspace_parent` on `(workspace_id, parent_id)`.
- `idx_folders_workspace_path` on `(workspace_id, path text_pattern_ops)`.
- `idx_folders_workspace_starred` on `(workspace_id, is_starred)` where
  `deleted_at is null`.
- `idx_folders_workspace_deleted` on `(workspace_id, deleted_at)` where
  `deleted_at is not null`.
- Unique active sibling name: `(workspace_id, parent_id, lower(name)) where
  deleted_at is null`.

Rules:

- `parent_id` must point to a folder in the same workspace. Enforce in
  repository/usecase, because PostgreSQL cannot express this with a simple FK.
- Moving a folder must update `path` and `depth` for all descendants in the
  same transaction.
- Trashing a folder should trash descendants and contained scenes in one
  workspace-scoped transaction.

### scenes

Stores Excalidraw drawing metadata and JSON payload.

| Column | Type | Notes |
| --- | --- | --- |
| `id` | `uuid` | Primary key, default `gen_random_uuid()`. |
| `workspace_id` | `uuid` | Required FK to `workspaces(id)`. |
| `folder_id` | `uuid` | Nullable FK to `folders(id)`. Null means root. |
| `title` | `text` | Required. |
| `content` | `jsonb` | Required, default `'{}'::jsonb`. Excalidraw scene data. |
| `content_version` | `bigint` | Required, default `1`. Increment on save for observability. |
| `is_starred` | `boolean` | Required, default `false`. |
| `deleted_at` | `timestamptz` | Nullable trash marker. |
| `created_by` | `uuid` | Nullable FK to `users(id)`. |
| `updated_by` | `uuid` | Nullable FK to `users(id)`. |
| `last_opened_at` | `timestamptz` | Nullable. |
| `created_at` | `timestamptz` | Required, default `now()`. |
| `updated_at` | `timestamptz` | Required, default `now()`. |

Indexes and constraints:

- `idx_scenes_workspace_folder` on `(workspace_id, folder_id)`.
- `idx_scenes_workspace_updated` on `(workspace_id, updated_at desc)`.
- `idx_scenes_workspace_starred` on `(workspace_id, is_starred)` where
  `deleted_at is null`.
- `idx_scenes_workspace_deleted` on `(workspace_id, deleted_at)` where
  `deleted_at is not null`.
- Optional active folder title uniqueness: `(workspace_id, folder_id,
  lower(title)) where deleted_at is null`.

Rules:

- `folder_id` must point to a folder in the same workspace. Enforce in
  repository/usecase.
- Phase 1 conflict policy is last-write-wins. `content_version` is not an
  optimistic lock in Phase 1.
- Scene content may include presentation/frame metadata later, but Phase 1 does
  not need separate frame tables.

### scene_assets

Stores metadata for S3-backed assets referenced by scene JSON. The binary bytes
live in S3-compatible storage.

| Column | Type | Notes |
| --- | --- | --- |
| `id` | `uuid` | Primary key, default `gen_random_uuid()`. |
| `workspace_id` | `uuid` | Required FK to `workspaces(id)`. |
| `scene_id` | `uuid` | Nullable FK to `scenes(id)`. Nullable allows staged upload before save. |
| `uploaded_by` | `uuid` | Nullable FK to `users(id)`. |
| `asset_kind` | `text` | Required. Phase 1 values: `excalidraw_file`, `scene_thumbnail`. |
| `file_id` | `text` | Nullable Excalidraw `FileId` for image elements. |
| `storage_bucket` | `text` | Required. |
| `storage_key` | `text` | Required object key. |
| `mime_type` | `text` | Required. |
| `byte_size` | `bigint` | Required, `>= 0`. |
| `sha256` | `text` | Nullable content hash for dedupe/integrity. |
| `width` | `integer` | Nullable image width. |
| `height` | `integer` | Nullable image height. |
| `deleted_at` | `timestamptz` | Nullable soft-delete marker. |
| `created_at` | `timestamptz` | Required, default `now()`. |
| `updated_at` | `timestamptz` | Required, default `now()`. |

Indexes and constraints:

- Unique `(storage_bucket, storage_key)`.
- `idx_scene_assets_workspace_scene` on `(workspace_id, scene_id)`.
- `idx_scene_assets_workspace_deleted` on `(workspace_id, deleted_at)`.
- Unique active Excalidraw file mapping: `(workspace_id, file_id)` where
  `asset_kind = 'excalidraw_file' and deleted_at is null`.
- Check `asset_kind in ('excalidraw_file', 'scene_thumbnail')`.

Rules:

- S3 object keys should include workspace, scene, and asset IDs, e.g.
  `workspaces/{workspace_id}/scenes/{scene_id}/assets/{asset_id}`.
- Deleting an asset in Phase 1 marks `deleted_at` and may delete the S3 object
  immediately if no scene references it. Retention policy can be added later.
- Signed URL generation should verify `workspace_id` before returning access.
- PostgreSQL must not store raw asset bytes or base64 `dataURL` payloads.
  `scene_assets` stores metadata only; S3-compatible storage owns binary bytes.
- The legacy reference server used a generic `files` table and object keys equal
  to raw file or scene IDs. Phase 1 should not copy that model because it lacks
  workspace ownership, scene linkage, asset kind, and storage-key namespacing.
- Scene thumbnails should either be rows in `scene_assets` with
  `asset_kind = 'scene_thumbnail'`, or be cached on `scenes` only as derived
  signed URL metadata while the canonical object metadata remains in
  `scene_assets`.

## Search, Starred, and Trash

Phase 1 can implement search directly over `folders.name` and `scenes.title`
with PostgreSQL `ILIKE`. If search becomes slow, add `pg_trgm` indexes before
changing the product API.

Recommended migration support:

- Enable `pg_trgm` if accepted by the deployment baseline.
- `idx_folders_workspace_name_trgm` on `lower(name) gin_trgm_ops`.
- `idx_scenes_workspace_title_trgm` on `lower(title) gin_trgm_ops`.

Search rules:

- Search defaults to active records only: `deleted_at is null`.
- Trash view queries `deleted_at is not null`.
- Starred view queries `is_starred = true and deleted_at is null`.
- Search result payloads should normalize folders and scenes into an item shape:
  `id`, `type`, `workspace_id`, `parent_id/folder_id`, `name/title`,
  `is_starred`, `deleted_at`, `updated_at`.

## Tenant Isolation Contract

The database schema supports isolation; repositories enforce it.

Mandatory repository rules:

- Every workspace-owned read/write accepts or derives `workspace_id`.
- `workspace_id` is part of every `where` clause for `folders`, `scenes`, and
  `scene_assets`.
- Cross-workspace foreign key mismatches must return a domain authorization or
  validation error, not a generic database leak.
- Middleware may use `X-Workspace-ID` before auth is complete, but after
  EXC-006 it must also verify the user has a `workspace_members` row for that
  workspace.

## Migration Order

Recommended Phase 1 migration sequence:

1. `000001_enable_extensions.up.sql`
   - `pgcrypto`
   - optional `pg_trgm`
2. `000002_identity_and_sessions.up.sql`
   - `users`
   - `auth_sessions`
3. `000003_workspaces.up.sql`
   - `workspaces`
   - `workspace_members`
4. `000004_folders.up.sql`
   - `folders`
   - materialized path indexes
5. `000005_scenes.up.sql`
   - `scenes`
   - JSONB and lifecycle indexes
6. `000006_scene_assets.up.sql`
   - `scene_assets`
7. `000007_search_indexes.up.sql`
   - optional trigram indexes if `pg_trgm` is enabled
8. `000005_create_scene_permissions.up.sql` in the current rewrite checkout
   - one opaque, revocable public-link permission per scene

Down migrations should reverse this order and must not be used against
production data without an explicit backup and restore plan.

## Repository Boundaries

The Go Clean Architecture split should keep SQL out of usecases:

- `domain`: entity structs and repository interfaces.
- `usecase`: path creation, folder move/trash cascade, scene save policy,
  workspace access checks, asset upload orchestration.
- `repository/postgres`: SQL, transactions, indexes, row mapping.
- `repository/s3`: binary upload/download/delete and signed URL generation.
- `delivery/http`: request parsing, response mapping, middleware context.

## Open Decisions Before Implementation

- Whether session persistence uses secure cookies with refresh tokens or
  stateless JWT only. The `auth_sessions` table is recommended for revocation.
- Whether `pg_trgm` is mandatory in all deployments or optional behind a
  migration flag.
- Whether workspace delete is exposed in Phase 1 UI. The schema allows soft
  delete but Phase 1 acceptance does not require it.
- Whether active sibling names must be unique. This document recommends
  uniqueness for predictable Drive-like behavior.

## Verification Expectations

At implementation time, Phase 1 DB work should verify:

- Migrations apply cleanly to a fresh PostgreSQL database.
- All workspace-owned tables include `workspace_id`.
- Repository tests prove workspace A cannot read or mutate workspace B data.
- Folder move updates descendant paths in one transaction.
- Folder trash cascades to descendants and contained scenes.
- Scene save persists JSONB and increments `content_version`.
- Asset metadata cannot return a signed URL unless workspace scope matches.
- Search excludes trashed records by default and includes only current
  workspace records.
