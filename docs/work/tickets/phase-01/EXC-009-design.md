# Detail Design: Optimized Folder Hierarchy via Materialized Path

## Status
Under Review

## Associated Story
- **Story ID**: EXC-009
- **Story Link**: `docs/stories/phase-01/EXC-009/EXC-009-folder-backend-materialized-path.md`

---

## 1. Rationale & Research Findings

### 1.1. Why Materialized Path?
Based on research of hierarchical data patterns in PostgreSQL, I evaluated three main strategies:
1.  **Adjacency List (`parent_id`)**: Easiest to implement, but requires expensive Recursive CTEs for subtree lookups.
2.  **Closure Table**: Fastest for deep tree joins, but requires a separate many-to-many table and complex triggers to maintain.
3.  **Materialized Path**: Stores the full lineage in a single column (e.g., `/A/B/C/`).

**Decision**: **Materialized Path** was selected because our use case (File Explorers/Dashboards) is **read-heavy** and requires frequent **subtree operations** (e.g., "Delete this folder and all its contents", "Move this entire branch").

### 1.2. Optimization: Standard `TEXT` vs. `ltree` Extension
I researched PostgreSQL's native `ltree` extension versus a standard indexed `TEXT` column:
-   **`ltree`**: Offers powerful operators (`<@`, `@>`) and GiST indexing. However, it mandates alphanumeric labels (no hyphens).
-   **Standard `TEXT` with B-Tree Index**: Standard B-Tree indexes in Postgres are highly optimized for **prefix matching** (`LIKE '/path/to/folder/%'`).

**Decision & Optimizations**:
-   **Storage Optimization**: We will use a standard **`TEXT` column with B-Tree indexing**. To minimize storage and index size, we use **stripped UUIDs (32 characters)** by removing hyphens from the folder IDs in the path.
-   **Depth Safety**: We will enforce a **maximum depth of 30 levels** in the Usecase layer. This ensures absolute safety for B-Tree index entries (which have a ~2.7KB limit) and prevents runaway nesting.
-   **Performance**: This approach avoids extension dependencies and allows us to use raw UUID segments (compacted) while maintaining near-instant lookup performance for subtrees.

---

## 2. Technical Strategy

### 2.1. Materialized Path Structure
-   **Format**: `/{stripped_uuid1}/{stripped_uuid2}/.../`
-   **Segment Length**: Each segment is exactly 32 characters + 1 slash.
-   **Trailing Slash**: Always include a trailing slash to avoid "false matches".
-   **Index**: A B-Tree index on the `path` column.

### 2.2. Optimized Operations
-   **Subtree Fetching**: `SELECT * FROM folders WHERE path LIKE '/parent-path/%'` (Utilizes Index Range Scan).
-   **Recursive Deletion**: A single SQL command: `UPDATE folders SET deleted_at = now() WHERE path LIKE '/deleted-folder-path/%'` (Optimized for GORM Soft Delete).
-   **Moving Branches**: When moving folder A to parent B, we use a string replacement query:
    `UPDATE folders SET path = REPLACE(path, 'old-prefix/', 'new-prefix/') WHERE path LIKE 'old-prefix/%'`.

---

## 3. Data Model & Storage Design

### 3.1. Folder Schema (GORM)
-   `id`: UUID (Primary Key)
-   `workspace_id`: UUID (Index) - **Mandatory Tenant Isolation**
-   `parent_id`: UUID (Nullable, Index) - Required for standard Adjacency List fallbacks.
-   `path`: TEXT (Index) - The Materialized Path segment.
-   `depth`: INT - For fast rendering of tree levels.
-   `is_starred`: BOOLEAN
-   `deleted_at`: GORM DeletedAt - Soft delete support.

---

## 4. API Specification

| Method | Endpoint | Description |
| :--- | :--- | :--- |
| `GET` | `/api/v1/folders` | List folders in a workspace (flat or by `parent_id`). |
| `POST` | `/api/v1/folders` | Create a new folder (calculates path based on parent). |
| `PATCH` | `/api/v1/folders/{id}/move` | Moves a folder and recursively updates all descendants' paths. |
| `DELETE` | `/api/v1/folders/{id}` | Recursively soft-deletes a folder and all descendants. |
| `POST` | `/api/v1/folders/{id}/star` | Toggles the starred status. |

---

## 5. Security & Validation
-   **Tenant Enforcement**: Every repository method **MUST** include `workspace_id = ?` to prevent cross-tenant path leakage.
-   **Circular Dependency Check**: The `Move` usecase will verify that a folder is not being moved into its own descendant by checking the path prefix.
-   **Maximum Depth Enforcement**: The `Create` and `Move` logic will block any operation that would result in a folder depth > 30.
-   **Input Validation**: `name` must be between 1-100 characters and contain no invalid path characters.

---

## 6. Verification Plan
-   **Unit Tests**: Mock repository to test path calculation and circular move prevention.
-   **Integration Tests**: Real Postgres tests for `REPLACE` path logic and `LIKE` subtree deletions.
-   **E2E (Playwright)**: Create a folder, move it into another, and verify the tree structure in the UI.

---

## 7. Review and Sign-off
-   **Reviewer**: Agent
-   **Review Date**: 2026-06-04
-   **Reviewer Decision**: Pending
-   **Reviewer Comments**:
  ```text
  Design focuses on high-performance Materialized Path implementation using standard Postgres indexing.
  ```
