# Detail Design: DB Migrations and Multi-tenant Schema

## Status
Under Review

## Associated Story
- **Story ID**: EXC-003
- **Story Link**: [EXC-003 DB Migrations & Multi-tenant Schema](EXC-003-db-migrations-and-multi-tenant-schema.md)

---

## 1. Design Considerations & Trade-offs
- **App-specific database**:
  - *Issue/Challenge*: The local PostgreSQL service uses `dev/password`, but the app should own a separate database.
  - *Decision/Trade-off*: Target the existing `devtool` database created during EXC-001 validation.
- **UUID generation in Postgres**:
  - *Issue/Challenge*: Tables need UUID primary keys without forcing Go to generate every ID.
  - *Decision/Trade-off*: Enable `pgcrypto` and use `gen_random_uuid()` defaults in the migration.
- **Soft delete over hard delete**:
  - *Issue/Challenge*: Folder and scene trash behavior is planned in Phase 1.
  - *Decision/Trade-off*: Add nullable `deleted_at` columns and keep queries responsible for filtering active records in repository tickets.
- **Materialized folder path**:
  - *Issue/Challenge*: Folder tree navigation needs efficient descendant lookup.
  - *Decision/Trade-off*: Store `path` as text with workspace-scoped indexes; stricter path operations can be added later if needed.

---

## 2. Architecture Overview

### Component Responsibilities
| Component | Responsibility / Role |
| --- | --- |
| **Migration SQL** | Creates extensions, tables, constraints, and tenant indexes. |
| **Domain Entities** | Mirrors persistence fields needed by repositories without SQL annotations. |
| **PostgreSQL** | Enforces primary keys, foreign keys, uniqueness, and tenant indexes. |

---

## 3. Sequence / Execution Flow

```mermaid
sequenceDiagram
    autonumber
    actor Developer
    participant Migration as SQL Migration
    participant DB as PostgreSQL devtool
    participant Domain as Go Domain

    Developer->>Migration: Apply migrations/000001_init.up.sql
    Migration->>DB: CREATE EXTENSION pgcrypto
    Migration->>DB: CREATE users/workspaces/folders/scenes
    Migration->>DB: CREATE workspace indexes
    Developer->>Domain: Build Go package
    Domain-->>Developer: Folder and Scene include star/trash fields
```

---

## 4. API Specification

No HTTP API endpoints are introduced in EXC-003.

---

## 5. Data Model & Storage Design

### Relational Schema (PostgreSQL)
- **users**
  | Column | Type | Constraints | Description |
  | --- | --- | --- | --- |
  | `id` | `UUID` | PRIMARY KEY, DEFAULT `gen_random_uuid()` | User identity |
  | `email` | `TEXT` | NOT NULL UNIQUE | Login email |
  | `name` | `TEXT` | NOT NULL | Display name |
  | `avatar_url` | `TEXT` | NULL | Profile image URL |
  | `google_id` | `TEXT` | UNIQUE NULL | Future Google OAuth subject |
  | `created_at` | `TIMESTAMPTZ` | NOT NULL DEFAULT `now()` | Creation timestamp |
  | `updated_at` | `TIMESTAMPTZ` | NOT NULL DEFAULT `now()` | Update timestamp |

- **workspaces**
  | Column | Type | Constraints | Description |
  | --- | --- | --- | --- |
  | `id` | `UUID` | PRIMARY KEY, DEFAULT `gen_random_uuid()` | Tenant boundary |
  | `name` | `TEXT` | NOT NULL | Workspace name |
  | `owner_id` | `UUID` | NOT NULL REFERENCES `users(id)` | Workspace owner |
  | `created_at` | `TIMESTAMPTZ` | NOT NULL DEFAULT `now()` | Creation timestamp |
  | `updated_at` | `TIMESTAMPTZ` | NOT NULL DEFAULT `now()` | Update timestamp |

- **folders**
  | Column | Type | Constraints | Description |
  | --- | --- | --- | --- |
  | `id` | `UUID` | PRIMARY KEY, DEFAULT `gen_random_uuid()` | Folder ID |
  | `workspace_id` | `UUID` | NOT NULL REFERENCES `workspaces(id)` | Tenant scope |
  | `parent_id` | `UUID` | NULL REFERENCES `folders(id)` | Tree parent |
  | `name` | `TEXT` | NOT NULL | Folder name |
  | `path` | `TEXT` | NOT NULL | Materialized path |
  | `is_starred` | `BOOLEAN` | NOT NULL DEFAULT false | Starred marker |
  | `deleted_at` | `TIMESTAMPTZ` | NULL | Trash marker |
  | `created_at` | `TIMESTAMPTZ` | NOT NULL DEFAULT `now()` | Creation timestamp |
  | `updated_at` | `TIMESTAMPTZ` | NOT NULL DEFAULT `now()` | Update timestamp |

- **scenes**
  | Column | Type | Constraints | Description |
  | --- | --- | --- | --- |
  | `id` | `UUID` | PRIMARY KEY, DEFAULT `gen_random_uuid()` | Scene ID |
  | `workspace_id` | `UUID` | NOT NULL REFERENCES `workspaces(id)` | Tenant scope |
  | `folder_id` | `UUID` | NULL REFERENCES `folders(id)` | Containing folder |
  | `name` | `TEXT` | NOT NULL | Scene title |
  | `data` | `JSONB` | NOT NULL DEFAULT `'{}'::jsonb` | Excalidraw payload |
  | `is_starred` | `BOOLEAN` | NOT NULL DEFAULT false | Starred marker |
  | `deleted_at` | `TIMESTAMPTZ` | NULL | Trash marker |
  | `created_by` | `UUID` | NULL REFERENCES `users(id)` | Creator |
  | `created_at` | `TIMESTAMPTZ` | NOT NULL DEFAULT `now()` | Creation timestamp |
  | `updated_at` | `TIMESTAMPTZ` | NOT NULL DEFAULT `now()` | Update timestamp |

### Indexes
- `idx_folders_workspace_id`
- `idx_folders_workspace_path`
- `idx_scenes_workspace_id`
- `idx_scenes_workspace_folder_id`

---

## 6. Security Design
- **Authentication**: Not implemented in this ticket.
- **Authorization**: Schema carries `workspace_id`; enforcement happens in EXC-004 repositories/middleware.
- **Input Validation**: SQL constraints reject missing required fields and invalid foreign keys.
- **Data Isolation**: Tenant-specific tables are indexed by `workspace_id` to support repository filtering.

---

## 7. Verification & Test Plan

### Automated Verification
- `grep -i "is_starred" migrations/000001_init.up.sql`
- `grep -i "deleted_at" migrations/000001_init.up.sql`
- `go test ./internal/domain/...`

### Manual Verification Scenarios
- [ ] Apply migration to local `devtool` database.
- [ ] Confirm `folders` and `scenes` include `workspace_id`, `is_starred`, and `deleted_at`.

---

## 8. Review and Sign-off
- **Reviewer**: Human reviewer
- **Review Date**: 2026-06-04
- **Reviewer Decision**: Pending
- **Reviewer Comments**:
  ```text
  Pending review before implementation.
  ```
