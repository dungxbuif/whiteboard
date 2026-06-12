# EXC-022: Password Protection

## Status

planned

## Lane

normal

## Product Contract

Migration: add password_hash column, bcrypt hashing in usecase, PublicHandler 401/403 challenge, PasswordPrompt component, PublicView password flow

## Relevant Product Docs

- `03-03-PLAN.md`

## Acceptance Criteria

- Password hash column is successfully added to permissions schema.
- Password protection challenge prevents unauthorized viewing.
- PasswordPrompt UI prompts unauthenticated user for passcode to decrypt scene.

## Design Notes

A Detail Design document following `docs/templates/detail_design.md` must be created at `docs/stories/phase-03/EXC-022-design.md` and approved by the reviewer before implementation starts.

- **Link to Detail Design**: [Design Document](EXC-022-design.md)

- **Modified Files**: `src/backend/migrations/000005_add_password_to_permissions.up.sql`, `src/backend/internal/domain/scene_permission.go`, `scene_permission_ucase.go`, `public_handler.go`, `src/features/sharing/components/PasswordPrompt.tsx`, `src/pages/PublicView.tsx`
- **Sprint Phase**: phase-03

## Validation

| Layer | Expected proof |
| --- | --- |
| Unit | go build ./src/backend/cmd/server |
| Integration | |
| E2E | |
| Platform | |
| Release | |

## Harness Delta

- Registered EXC-022 in the harness database.

## Evidence

(No evidence yet)
