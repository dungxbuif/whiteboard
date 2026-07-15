---
artifact_type: detail_design
id: DESIGN-EXC-013
status: approved
owner: human
approval: human
human_fields:
  - approval
  - constraints
  - scope_decisions
ai_fields:
  - problem
  - context_loaded
  - brownfield_scope
  - proposed_approach
  - design_tradeoffs
  - architecture_overview
  - execution_flow
  - api_data_model
  - security
  - test_plan
  - reconciliation_plan
shared_fields:
  - status
  - trace
  - small_task_exemption
trace:
  backlog_item: EXC-013
  requirement: REQ-03
  phase: PHASE-1
  ticket_or_bug: EXC-013
  test_verification: TBD
  validation_matrix: docs/work/VALIDATION_MATRIX.md
  docs_review: TBD
  adrs: []
  master_docs_touched:
    - docs/architecture/API.md
    - docs/architecture/ERD.md
    - docs/architecture/ARCHITECTURE.md
    - docs/CONTEXT.md
    - docs/work/BACKLOG.md
    - docs/releases/CHANGELOG.md
---

# DETAIL DESIGN: EXC-013 S3 Assets

## Status

- ID: DESIGN-EXC-013
- Status: in_review
- Ticket/Bug: [EXC-013](EXC-013-s3-assets.md)
- Approval: pending
- Author: AI
- Updated: 2026-07-15

## Trace Links

- Backlog item: [BACKLOG.md](../../../BACKLOG.md)
- Requirement: REQ-03 in [SPEC.md](../../../../requirements/SPEC.md)
- Phase: [PHASE-1](../../../phases/PHASE-1.md)
- Ticket/Bug: [EXC-013](EXC-013-s3-assets.md)
- Test verification: TBD
- Validation matrix: [VALIDATION_MATRIX.md](../../../VALIDATION_MATRIX.md)
- Docs review: TBD
- ADRs: none planned
- Master docs touched: API, ERD, Architecture, Context, Backlog, Changelog

---

## 1. Context & Scope

### Problem Statement

- Problem statement: Whiteboard image assets are currently kept inside Excalidraw scene JSON and have no backend-managed binary storage path. Phase 1 requires PostgreSQL to store asset metadata only while S3-compatible storage owns binary bytes.
- Why now: EXC-012 completed embedded Excalidraw and debounced scene saves. EXC-013 is the next Phase 1 item and adds the asset path needed before search/trash/star completion and release hardening.
- Success outcome: A scene can request bulk upload URLs for image files, upload blobs to S3/MinIO under workspace-scoped keys, save metadata in `scene_assets`, and later request bulk view URLs keyed by Excalidraw `fileId`.

### Context Loaded

- `docs/CONTEXT.md`
- `docs/work/BACKLOG.md`
- `docs/standards/README.md`
- `docs/standards/QUALITY_BAR.md`
- `docs/standards/VALIDATION.md`
- `docs/work/phases/PHASE-1.md`
- `docs/work/tickets/phase-01/EXC-013/EXC-013-s3-assets.md`
- `docs/work/tickets/phase-01/01-10-PLAN.md`
- `docs/architecture/ARCHITECTURE.md`
- `docs/architecture/API.md`
- `docs/architecture/ERD.md`
- `docs/architecture/phase-01-database-design.md`
- Direct code in backend domain/usecase/router and frontend Excalidraw wrapper/API client.

### Brownfield Scope

- Touched modules/files:
  - `src/backend/internal/domain/constants.go`
  - `src/backend/internal/domain/scene.go`
  - `src/backend/internal/repository/s3/s3_repo.go`
  - `src/backend/internal/repository/s3/s3_repo_test.go`
  - `src/backend/internal/repository/postgres/scene_asset_repo.go`
  - `src/backend/internal/repository/postgres/scene_asset_repo_test.go`
  - `src/backend/internal/usecase/asset_ucase.go`
  - `src/backend/internal/usecase/asset_ucase_test.go`
  - `src/backend/internal/delivery/http/handler/asset_handler.go`
  - `src/backend/internal/delivery/http/router.go`
  - `src/frontend/src/lib/api.ts`
  - `src/frontend/src/components/whiteboard/excalidraw-wrapper.tsx`
- Direct dependencies inspected: scene domain/usecase, router structure, API client, Excalidraw wrapper, Phase 1 database design.
- Contracts affected: New authenticated workspace-scoped `/api/v1/assets/*` HTTP routes; `scene_assets` repository behavior; S3 runtime configuration.
- Known unknowns: Existing migration status for `scene_assets`; exact frontend Excalidraw file callback support in the installed package version; whether local MinIO is running during verification.
- Scope expansion reason: The ticket names S3 repository and HTTP handlers, but durable metadata requires a PostgreSQL `scene_assets` repository to enforce workspace and scene ownership before generating URLs.

### Small Task Exemption

- Small task exemption: no
- Reason: This changes API contracts, runtime S3 integration, workspace-scoped authorization, and persistent asset metadata.
- Impact checked: API=yes, DB=yes, Security=yes, Runtime=yes, Standards=no

---

## 2. Design Considerations & Trade-offs

| Consideration / Alternative | Pros | Cons | Decision |
| --- | --- | --- | --- |
| Browser uploads directly to presigned S3 URL after backend metadata reservation | Keeps binary bytes out of Go API memory; works with MinIO/S3; scales better for large files | Requires frontend to perform a second upload request and then save scene JSON | chosen |
| Proxy raw uploads through the Go API | Simpler browser flow; easier validation of exact bytes | Increases API memory/bandwidth load; less aligned with S3 signed URL goal | rejected |
| Generate one signed URL per HTTP call | Small handler surface | Too chatty for scenes with many assets | rejected |
| Bulk presign one scene at a time | Verifies workspace and scene once; aligns with ticket requirement; keeps request bounded | Requires request cap and response map handling | chosen |
| Store raw `dataURL` payloads in PostgreSQL scene JSON | Lowest implementation effort | Violates Phase 1 data contract and bloats JSONB | rejected |
| Create an ADR for S3 assets | Durable storage decision is already in master architecture and Phase 1 database design | A new ADR would mostly duplicate approved architecture | rejected for now |

---

## 3. Architecture Overview

### Component Responsibilities

| Component | Role |
| --- | --- |
| `domain.SceneAsset` and asset interfaces | Define asset metadata, request/response DTOs, repository interfaces, and usecase contract. |
| `repository/postgres.SceneAssetRepository` | Create and query `scene_assets` rows with workspace and scene filters. |
| `repository/s3.S3Repository` | Create presigned PUT/GET URLs and delete S3 objects using AWS SDK Go v2 against S3 or MinIO. |
| `usecase.AssetUsecase` | Validate workspace-scoped scene ownership, cap bulk requests, build namespaced object keys, reserve metadata rows, and request presigned URLs. |
| `handler.AssetHandler` | Decode/validate HTTP JSON, read workspace/user context, call usecase, and serialize responses. |
| `router.go` | Mount authenticated, tenant-scoped asset routes under `/api/v1/assets`. |
| `excalidraw-wrapper.tsx` | Detect new Excalidraw files, request upload URLs in bulk, PUT blobs to S3, replace persisted file `dataURL` with view URLs where supported, and keep scene saves debounced. |

```text
React Excalidraw
    |
    | POST /api/v1/assets/presign-upload
    v
Go Asset Handler -> Asset Usecase
    |                  |
    |                  +--> Scene repository: verify workspace-scoped scene
    |                  +--> Scene asset repository: reserve/query metadata
    |                  +--> S3 repository: presign PUT/GET/delete
    v
S3/MinIO direct browser upload/view
```

---

## 4. Execution Flow

### Upload Flow

1. User inserts one or more images into Excalidraw.
2. Frontend identifies files whose `fileId` has not already been uploaded for the active scene.
3. Frontend sends `POST /api/v1/assets/presign-upload` with `sceneId` and up to 100 asset metadata entries.
4. Handler requires auth and `X-Workspace-ID`; tenant middleware injects workspace context.
5. Usecase verifies the scene belongs to the workspace once.
6. Usecase creates or reuses active `scene_assets` metadata rows keyed by workspace and `fileId`.
7. Usecase asks S3 repository for presigned PUT URLs using keys shaped as `workspaces/{workspace_id}/scenes/{scene_id}/assets/{asset_id}`.
8. Frontend uploads blobs directly to S3/MinIO using the returned URLs and headers.
9. Frontend schedules the normal scene save so scene JSON references the Excalidraw file IDs without persisting base64 data URLs.

### View Flow

1. Frontend opens a scene and extracts needed Excalidraw `fileId` values from scene JSON.
2. Frontend sends `POST /api/v1/assets/presign-view` with `sceneId` and file IDs.
3. Usecase verifies scene ownership once and queries matching active assets in one repository call.
4. Usecase checks if each asset has a valid cached `presigned_url` (where `url_expires_at` is safely in the future).
5. For assets with expired or missing URLs, the S3 repository generates new presigned GET URLs in memory, and the Usecase updates the `scene_assets` table with the new `presigned_url` and `url_expires_at`.
6. Frontend hydrates Excalidraw file data with returned URLs where the Excalidraw runtime accepts URL-backed files.

---

## 5. API & Data Model Design

### API Changes

- Endpoint: `POST /api/v1/assets/presign-upload`
- Auth: JWT + `X-Workspace-ID`
- Request:

```json
{
  "sceneId": "uuid",
  "assets": [
    {
      "fileId": "excalidraw-file-id",
      "mimeType": "image/png",
      "byteSize": 12345,
      "sha256": "optional-hex",
      "width": 640,
      "height": 480
    }
  ]
}
```

- Response:

```json
{
  "urls": {
    "excalidraw-file-id": {
      "assetId": "uuid",
      "uploadUrl": "https://...",
      "storageKey": "workspaces/.../scenes/.../assets/...",
      "expiresAt": "2026-07-15T10:00:00Z"
    }
  }
}
```

- Endpoint: `POST /api/v1/assets/presign-view`
- Auth: JWT + `X-Workspace-ID`
- Request:

```json
{
  "sceneId": "uuid",
  "fileIds": ["excalidraw-file-id"]
}
```

- Response:

```json
{
  "urls": {
    "excalidraw-file-id": {
      "assetId": "uuid",
      "viewUrl": "https://...",
      "mimeType": "image/png",
      "expiresAt": "2026-07-15T10:00:00Z"
    }
  }
}
```

- Endpoint: `GET /api/v1/assets/{id}`
- Auth: JWT + `X-Workspace-ID`
- Behavior: Debug/fallback route that verifies workspace ownership and redirects to a presigned GET URL for the asset ID.

### Data Model Changes

- Table / Collection: `scene_assets`
- Added / Modified fields: 
  - Add `presigned_url` (text, nullable): to cache the generated S3 view URL.
  - Add `url_expires_at` (timestamp, nullable): to track when the cached URL expires.
- Schema Migration needed: Implementation must add a migration to alter `scene_assets` (or include in its initial creation) to support `presigned_url` and `url_expires_at`.

---

## 6. Security & Authorization

- Authentication changes: none. Existing JWT auth remains required.
- Authorization / Permissions: Every asset route must require tenant middleware and verify the active workspace owns the scene or asset before any URL is returned.
- Data Privacy / PII impact: Assets may contain user-provided image content. Do not log signed URLs, raw bytes, or base64 payloads.
- Input Validation: Validate UUIDs, non-empty scene ID, request count `1..100`, non-empty `fileId`, allowed MIME type prefixes for Phase 1 images, non-negative byte size, optional positive width/height, and optional SHA-256 format.
- URL Lifetime: Signed URL expiration should be a named constant, with production override through strict config if configuration already exists.
- Object Namespace: Never accept a client-provided storage key. Use server-generated workspace/scene/asset ID paths only.

---

## 7. Implementation & Verification Plan

### Impacted Areas

- Code/modules: backend domain, usecase, S3 repository, scene asset metadata repository, HTTP handler/router, frontend API and Excalidraw wrapper.
- Product behavior: Images inserted into whiteboards are stored in S3/MinIO and can be reloaded by URL.
- API/contracts: Adds `/api/v1/assets/presign-upload`, `/api/v1/assets/presign-view`, and `/api/v1/assets/{id}`.
- Data/schema: Uses existing Phase 1 `scene_assets` contract; may require migration implementation if absent.
- Security/auth: Workspace-scoped access checks before signed URL generation.
- Deployment/runtime: Requires S3/MinIO configuration and CORS that allows browser PUT/GET for signed URLs.
- Docs: API, ERD, architecture if implementation differs, validation matrix, context, backlog, changelog.

### Bug Analysis (If Applicable)

- Reproduction: not applicable.
- Root cause: not applicable.
- Impact scope: not applicable.
- Fix strategy: not applicable.
- Regression risk: not applicable.

### Test Plan

- Unit:
  - `go test ./src/backend/internal/usecase/...`
  - S3 repository tests with a fake presign client or local MinIO when available.
- Integration:
  - `go test ./src/backend/internal/repository/postgres/...` for workspace-scoped asset metadata.
  - Handler/router tests proving authenticated workspace routes, request caps, and ownership failures.
- E2E:
  - Frontend automated test or Playwright path that inserts an image into the canvas when a running backend and MinIO are available.
- UAT:
  - Open a scene, insert image, observe save success, reload scene, verify image appears.
- Manual/platform:
  - Verify uploaded object exists under `workspaces/{workspace_id}/scenes/{scene_id}/assets/{asset_id}` in MinIO/S3.
- Docs review:
  - Record API, ERD, validation matrix, context, backlog, and changelog reconciliation.

### Validation Matrix Impact

- Update required: yes
- Row(s): EXC-013 S3 asset upload, bulk view URL generation, workspace isolation, frontend image insertion/reload.
- Reason if no update required: not applicable.

### Verification Results

Fill after execution.

- Command: TBD
- Result: TBD
- Notes: TBD

---

## 8. Reconciliation Plan

Update master docs and decisions to reflect this design.

- Requirements docs: no change expected; EXC-013 implements existing REQ-03.
- Architecture docs: update only if implementation diverges from existing S3 asset architecture.
- API docs: update with new asset routes.
- ERD docs: update with `scene_assets` entity details if still TBD after implementation.
- SDD docs: no change expected unless component boundaries change.
- ADR: no change expected; create only if the implementation introduces a new durable storage or security tradeoff.
- Context: update active work, recent changes, and next work.

### Docs Review Checklist

- Code changed but docs unchanged reason: TBD after implementation.
- Requirements updated or not needed reason: Existing REQ-03 already covers asset persistence.
- Architecture updated or not needed reason: Existing architecture already names S3-backed binary assets; update only for implementation-specific constraints.
- API updated or not needed reason: Required, new route contract.
- ERD/data updated or not needed reason: Required if master ERD remains placeholder.
- SDD updated or not needed reason: No SDD boundary change expected.
- ADR created or not needed reason: No new durable architectural decision beyond approved Phase 1 design unless implementation diverges.
