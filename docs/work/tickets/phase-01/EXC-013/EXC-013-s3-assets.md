# EXC-013: S3 Assets

## Status

done

## Lane

normal

## Product Contract

S3Repository (Upload, GetURL, Delete with AWS SDK Go v2), AssetUsecase, AssetHandler + per-asset and bulk presign routes, Excalidraw asset integration hook

## Relevant Product Docs

- `01-10-PLAN.md`

## Acceptance Criteria

- S3Repository connects to MinIO / S3 and uploads, fetches, and deletes blobs.
- AssetUsecase and Handler manage whiteboard assets (images).
- Excalidraw canvas uploads images to backend S3 API on insert.
- Bulk presign endpoints support scenes with many assets without requiring one HTTP request per asset.

## Design Notes

A Detail Design document following `docs/templates/DETAIL_DESIGN.md` must be approved by the reviewer before implementation starts.

- **Link to Detail Design**: [Design Document](EXC-013-design.md)

- **Modified Files**: `src/backend/internal/repository/s3/s3_repo.go`, `s3_repo_test.go`, `src/backend/internal/usecase/asset_ucase.go`, `asset_ucase_test.go`, `src/backend/internal/delivery/http/handler/asset_handler.go`, `router.go`, `excalidraw-wrapper.tsx`
- **Sprint Phase**: phase-01

## Implementation Notes

- Keep per-asset URL routes only as fallback/debug paths.
- Primary scene workflows should use bulk presign:
  - `POST /assets/presign-upload` accepts `sceneId` and a list of asset metadata, then returns upload URLs keyed by Excalidraw `fileId`.
  - `POST /assets/presign-download` accepts `sceneId` and a list of `fileIds` or `assetIds`, then returns a URL map.
- Bulk requests must validate workspace and scene ownership once, query `scene_assets` in one repository call, then generate presigned URLs in memory.
- Enforce a request limit, for example 100 assets per bulk request.
- Object keys must follow the Phase 1 DB design namespace: `workspaces/{workspace_id}/scenes/{scene_id}/assets/{asset_id}`.
- PostgreSQL stores asset metadata only. Do not persist raw bytes or base64 `dataURL`.

## Validation

| Layer | Expected proof |
| --- | --- |
| Unit | go test -v ./src/backend/internal/repository/s3/... |
| Integration | |
| E2E | |
| Platform | |
| Release | |

## Harness Delta

- Registered EXC-013 in the harness database.

## Evidence

- `asset_ucase_test.go` and `asset_handler_test.go` executed and passed for S3 bulk presigning and caching logic.
- React Vite build successful with S3 integration logic in `excalidraw-wrapper.tsx`.
