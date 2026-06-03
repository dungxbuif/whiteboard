# EXC-013: S3 Assets

## Status

planned

## Lane

normal

## Product Contract

S3Repository (Upload, GetURL, Delete with AWS SDK Go v2), AssetUsecase, AssetHandler + routes, Excalidraw asset integration hook

## Relevant Product Docs

- `01-10-PLAN.md`

## Acceptance Criteria

- S3Repository connects to MinIO / S3 and uploads, fetches, and deletes blobs.
- AssetUsecase and Handler manage whiteboard assets (images).
- Excalidraw canvas uploads images to backend S3 API on insert.

## Design Notes

A Detail Design document following `docs/templates/detail_design.md` must be created at `docs/stories/phase-01/EXC-013-design.md` and approved by the reviewer before implementation starts.

- **Link to Detail Design**: [Design Document](EXC-013-design.md)

- **Modified Files**: `internal/repository/s3/s3_repo.go`, `s3_repo_test.go`, `internal/usecase/asset_ucase.go`, `asset_ucase_test.go`, `internal/delivery/http/handler/asset_handler.go`, `router.go`, `excalidraw-wrapper.tsx`
- **Sprint Phase**: phase-01

## Validation

| Layer | Expected proof |
| --- | --- |
| Unit | go test -v ./internal/repository/s3/... |
| Integration | |
| E2E | |
| Platform | |
| Release | |

## Harness Delta

- Registered EXC-013 in the harness database.

## Evidence

(No evidence yet)
