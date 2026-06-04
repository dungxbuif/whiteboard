# Detail Design: Go Backend Scaffold

## Status
Approved

## Associated Story
- **Story ID**: EXC-001
- **Story Link**: [EXC-001 Go Backend Scaffold](EXC-001-go-backend-scaffold.md)

---

## 1. Design Considerations & Trade-offs
- **Clean Architecture scaffold first**:
  - *Issue/Challenge*: Phase 1 needs a backend foundation without prematurely implementing persistence or auth.
  - *Decision/Trade-off*: Add domain entities and repository interfaces now, with empty usecase and repository package anchors for later tickets.
- **Direct local Postgres**:
  - *Issue/Challenge*: The development environment already provides PostgreSQL locally and should not require Docker.
  - *Decision/Trade-off*: Default to `postgres://test:test@127.0.0.1:5432/whiteboard_test?sslmode=disable` and run the Go server directly.
- **Minimal HTTP contract**:
  - *Issue/Challenge*: Later API tickets will define real resources.
  - *Decision/Trade-off*: Expose only `/healthz` as a smoke endpoint and avoid committing CRUD response shapes in this ticket.

---

## 2. Architecture Overview

### Component Responsibilities
| Component | Responsibility / Role |
| --- | --- |
| **Backend (Go)** | Owns HTTP server startup, router construction, and domain type boundaries. |
| **Domain** | Defines User, Workspace, Folder, and Scene entities plus repository interfaces. |
| **Database / Storage** | Local PostgreSQL is provided outside the repo. S3-compatible storage is deferred to the asset ticket. |

---

## 3. Sequence / Execution Flow

```mermaid
sequenceDiagram
    autonumber
    actor Developer
    participant Server as Go Server
    participant Router as chi Router
    participant DB as Postgres

    Developer->>Server: go run ./cmd/server
    Server->>DB: Attempt pgx pool ping
    DB-->>Server: Success or warning
    Server->>Router: Mount /healthz
    Developer->>Router: GET /healthz
    Router-->>Developer: 200 {"status":"ok"}
```

---

## 4. API Specification

### HTTP Endpoints

- **GET `/healthz`** - Scaffold health check.
  - **Auth Required**: No
  - **Success Response (200 OK)**:
    ```json
    {
      "status": "ok"
    }
    ```

### WebSockets / Real-time Events
Not applicable for EXC-001.

---

## 5. Data Model & Storage Design

### Relational Schema (PostgreSQL)
No database schema is created in EXC-001. EXC-003 owns migrations and multi-tenant schema.

### Domain Entities
| Entity | Purpose |
| --- | --- |
| `User` | Authenticated user identity metadata. |
| `Workspace` | Tenant boundary and dashboard workspace. |
| `Folder` | Materialized-path folder node placeholder. |
| `Scene` | Excalidraw scene metadata and JSON payload placeholder. |

---

## 6. Security Design
- **Authentication**: No authenticated endpoints in this ticket.
- **Authorization**: No tenant middleware in this ticket; workspace IDs are present on domain models for later enforcement.
- **Input Validation**: `/healthz` accepts no input.
- **Secrets**: Development credentials are stored only in `.env.example`.

---

## 7. Verification & Test Plan

### Automated Verification
- **Unit/Compile Command**: `go mod verify && go build -o /private/tmp/devtool-server ./cmd/server`
- **Smoke Command**: `HTTP_ADDR=:18080 go run ./cmd/server`

### Manual Verification Scenarios
- [ ] Confirm local PostgreSQL is running with user `dev`, password `password`, and app database `devtool`.
- [ ] Start the backend directly with `go run ./cmd/server`.
- [ ] Request `GET http://localhost:8080/healthz` and confirm `200`.

---

## 8. Review and Sign-off
- **Reviewer**: Repository harness
- **Review Date**: 2026-06-04
- **Reviewer Decision**: Approved
- **Reviewer Comments**:
  ```text
  Scope is limited to scaffold, direct local PostgreSQL configuration, and smoke health endpoint.
  ```
