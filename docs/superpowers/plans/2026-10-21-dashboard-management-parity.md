# Dashboard Management and Legacy Parity Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Complete the production dashboard's everyday management controls while preserving the checked-in legacy UI and native Excalidraw canvas behavior.

**Architecture:** Extend the existing Go domain/usecase/repository and chi routes with owner/editor-scoped metadata mutations, then expose those mutations through the existing Zustand stores and Radix dashboard menus/dialogs. Keep scene content duplication on the existing scene-create contract, and make collection feature-policy responses a stable sidebar state instead of a generic error.

**Tech Stack:** Go, chi, GORM/PostgreSQL, React 18, TypeScript, Vite, Zustand, Radix UI, Vitest, Playwright.

**Spec:** `docs/superpowers/specs/2026-10-21-dashboard-management-design.md`

## Global Constraints

- Preserve the legacy dashboard tokens, typography, spacing, sidebar structure, card proportions, focus states, and light/dark behavior.
- Keep the Excalidraw canvas native; dashboard controls remain in sidebar, header, cards, and overflow menus.
- Workspace deletion/archive is out of scope until a retention/restore contract exists.
- Every mutation must use current workspace membership and server-side access decisions.
- Do not persist session-only scene state when duplicating content.
- Never add credentials or production test passwords to source, docs, logs, or fixtures.

## Review Focus

- Owner-only workspace rename must reject editor/viewer callers and refresh the active workspace without changing its ID.
- Folder rename/move must preserve descendant paths and reject self/descendant moves and invalid depth.
- Scene duplication must produce a new scene with canonical persisted content and no collaborator/session fields.
- Feature-disabled collections must be an explanatory non-error state, while transient failures retain retry.
- Legacy visual behavior must survive desktop/mobile, keyboard focus, portals, and dark mode.

### Task 1: Workspace and folder metadata mutations

**Files:**
- Modify: `src/backend/internal/domain/workspace.go`
- Modify: `src/backend/internal/usecase/workspace_ucase.go`
- Modify: `src/backend/internal/repository/postgres/workspace_repo.go`
- Modify: `src/backend/internal/delivery/http/handler/workspace_handler.go`
- Modify: `src/backend/internal/domain/folder.go`
- Modify: `src/backend/internal/usecase/folder_ucase.go`
- Modify: `src/backend/internal/delivery/http/handler/folder_handler.go`
- Modify: `src/backend/internal/delivery/http/router.go`
- Test: `src/backend/internal/delivery/http/handler/workspace_handler_test.go`
- Test: `src/backend/internal/usecase/folder_ucase_test.go`
- Test: `src/backend/internal/delivery/http/router_test.go`

**Interfaces:**
- Produces `WorkspaceUsecase.UpdateName(ctx, userID, workspaceID, name) (*Workspace, error)` and `FolderUsecase.Update(ctx, workspaceID, folderID, name *string, newParentID **uuid.UUID) error`.
- Adds `PATCH /api/v1/workspaces/{id}` with `{name}` and `PATCH /api/v1/folders/{id}` with optional `{name,newParentId}`.
- The workspace access middleware resolves `{id}` and enforces owner access before the handler; folder routes reuse the existing workspace middleware and path invariants.

- [ ] **Step 1: Write failing handler/usecase tests** for owner rename success, non-owner denial, invalid names, folder rename, folder move, self/descendant rejection, and route registration.
- [ ] **Step 2: Run targeted Go tests** and confirm the new methods/routes fail to compile or return the expected missing behavior.
- [ ] **Step 3: Implement workspace update** by loading the workspace, requiring owner membership, validating trimmed length 3–100, saving only `Name`, and returning the updated record.
- [ ] **Step 4: Implement folder update** by applying an optional trimmed name and delegating parent changes to the existing move algorithm so path/depth updates remain atomic at the repository boundary.
- [ ] **Step 5: Register PATCH routes** beside existing workspace/folder routes and map validation/access errors to the existing structured HTTP error behavior.
- [ ] **Step 6: Run backend tests** with `go test ./internal/...` from `src/backend` and confirm all targeted tests pass.
- [ ] **Step 7: Commit** with `feat: add dashboard workspace and folder management api`.

### Task 2: Dashboard store and management controls

**Files:**
- Modify: `src/frontend/src/store/workspace-store.ts`
- Modify: `src/frontend/src/store/folder-store.ts`
- Modify: `src/frontend/src/store/scene-store.ts`
- Modify: `src/frontend/src/components/workspace-switcher.tsx`
- Modify: `src/frontend/src/components/folder-actions.tsx`
- Create: `src/frontend/src/components/folder-management-dialog.tsx`
- Modify: `src/frontend/src/components/scene-actions.tsx`
- Create: `src/frontend/src/components/scene-duplicate-dialog.tsx`
- Test: `src/frontend/src/test/workspace-switcher.test.tsx`
- Test: `src/frontend/src/test/folder-store.test.ts`
- Test: `src/frontend/src/test/scene-store.test.ts`

**Interfaces:**
- Produces `workspaceStore.renameWorkspace(id, name)`, `folderStore.renameFolder(workspaceId, folderId, name)`, and `sceneStore.duplicateScene(workspaceId, sceneId, title, folderId)`.
- Dialogs accept explicit callbacks and keep Radix keyboard focus/Escape semantics.

- [ ] **Step 1: Write failing frontend tests** for owner-only rename visibility, successful store updates, folder rename/move dialog submission, scene duplicate title/folder submission, and viewer/editor control visibility.
- [ ] **Step 2: Run the focused Vitest files** and confirm failures identify the missing store methods and controls.
- [ ] **Step 3: Add store methods** using existing `api.patch`, `api.post`, workspace headers, and response replacement; preserve active IDs and refetch lists after mutations.
- [ ] **Step 4: Add workspace rename UI** to the switcher menu using the existing legacy button/dialog primitives and owner role guard.
- [ ] **Step 5: Add folder rename/move UI** to the overflow menu using a single management dialog that lists valid folders and blocks invalid descendants.
- [ ] **Step 6: Add scene duplicate/move UI** to scene actions; duplicate only `title`, `folderId`, and canonical persisted `content`, then navigate to the new scene.
- [ ] **Step 7: Run focused frontend tests** with `npm test -- --run ...` from `src/frontend` and confirm all pass.
- [ ] **Step 8: Commit** with `feat: expose dashboard management controls`.

### Task 3: Collections state and legacy presentation regression coverage

**Files:**
- Modify: `src/frontend/src/components/collections/collection-list.tsx`
- Modify: `src/frontend/src/components/dashboard/dashboard-shell.tsx`
- Modify: `src/frontend/src/styles/dashboard.css`
- Modify: `src/frontend/e2e/dashboard-style.spec.ts`
- Modify: `src/frontend/e2e/dashboard.spec.ts`
- Test: `src/frontend/src/components/collections/collections.test.tsx`

**Interfaces:**
- `CollectionList` distinguishes `FEATURE_DISABLED` from retryable failures and renders stable copy without treating a policy decision as a transport error.
- Existing dashboard shell/sidebar/header tokens remain the visual source of truth for all new dialogs and menus.

- [ ] **Step 1: Write failing collection tests** for disabled policy copy, retry action, and successful reload after a transient failure.
- [ ] **Step 2: Implement typed error inspection** in `CollectionList`, retaining the current safe fallback for unknown errors and adding a retry button.
- [ ] **Step 3: Add Playwright coverage** for desktop/mobile sidebar layout, workspace rename, folder/scene menus, and disabled collections.
- [ ] **Step 4: Run frontend unit tests and Playwright dashboard specs**; inspect screenshots against the checked-in legacy baselines.
- [ ] **Step 5: Commit** with `test: cover dashboard management and legacy parity`.

### Task 4: Documentation, build, and production verification

**Files:**
- Modify: `docs/user/dashboard.md`
- Modify: `src/frontend/public/docs/dashboard.md`
- Modify: `src/frontend/public/llms.txt`
- Modify: `docs/architecture/API.md`
- Modify: `docs/operations/RELEASE_CHECKLIST.md`
- Create: `docs/work/verification/dashboard-management-2026-10-21.md`

**Interfaces:**
- Human and AI-readable docs describe workspace rename, folder move/rename, scene duplicate/move, feature-disabled collections, and role limits.
- Verification evidence contains statuses and routes only; no credentials or tokens.

- [ ] **Step 1: Update docs** to match the shipped routes, controls, role guards, and error states.
- [ ] **Step 2: Run full checks**: backend `go test ./...`, frontend unit suite, TypeScript production build, and dashboard Playwright specs.
- [ ] **Step 3: Build the frozen production images** and run Compose config/migration gates on VM100.
- [ ] **Step 4: Verify production** with the two disposable test users: login, workspace rename, folder/scene management, collection disabled copy, `/healthz`, and all service health checks.
- [ ] **Step 5: Record redacted evidence** and deploy only after all checks pass; do not include passwords in evidence.
- [ ] **Step 6: Commit** with `docs: document dashboard management parity`.

