# Dashboard management and legacy parity

## Intent

Bring the production dashboard to the checked-in legacy presentation while
making the everyday workspace controls usable from the dashboard itself. The
canvas keeps the native Excalidraw surface; dashboard controls stay in the
sidebar, header, cards, and overflow menus.

## Scope

- Preserve the existing legacy tokens, typography, spacing, sidebar structure,
  card proportions, focus states, and light/dark behavior.
- Add owner-only workspace rename from the workspace switcher. Workspace
  deletion/archive is out of scope until a retention and restore contract is
  approved.
- Add editor/owner folder rename and move controls using the existing folder
  hierarchy and path invariants. Keep trash and restore semantics unchanged.
- Add editor/owner scene move and duplicate controls, retaining rename, star,
  share, and trash actions. Duplicate copies persisted scene content through
  the authenticated scene-create contract and never copies session-only state.
- Make feature-disabled collections render an explanatory, non-error state with
  a retry action for transient failures. Collection management remains behind
  the existing feature policy.
- Keep owner-only enterprise settings and member/agent controls visible through
  the same legacy sidebar presentation.

## Contracts and authorization

- Add `PATCH /api/v1/workspaces/{id}` for owner-only `{name}` updates. The
  response returns the updated workspace and uses the existing workspace access
  service; non-owners receive the structured access error.
- Add `PATCH /api/v1/folders/{id}` for editor/owner `{name?,newParentId?}`
  updates. Reuse the folder move depth/descendant checks and reject empty names,
  self/descendant moves, and invalid workspace scope.
- Reuse `PATCH /api/v1/scenes/{id}` for title/folder changes and
  `POST /api/v1/scenes` for duplicate creation. The duplicate request carries a
  new title, target folder, and canonical persisted content only.
- All mutations require the current workspace membership and use the same
  access decision as existing create/update/trash routes.

## UI behavior

- Workspace switcher exposes “Rename workspace” only to owners and refreshes
  the active workspace after a successful update.
- Folder and scene overflow menus open keyboard-accessible dialogs for rename
  and move/duplicate; destructive actions remain explicit and confirm before
  sending.
- Existing routes, public aliases, deep links, native scene styling, and local
  theme preferences remain unchanged.
- Collections show “Collections are unavailable for this workspace” when the
  API returns `FEATURE_DISABLED`; other errors retain a retry action and a
  diagnostic-safe message.

## Verification

- Backend handler/usecase/repository tests cover owner/editor/viewer policy,
  validation, folder path invariants, duplicate content filtering, and
  structured error responses.
- Frontend tests cover legacy layout, keyboard dialogs, permissions, retry
  state, and store refresh behavior.
- Playwright checks exercise workspace rename, folder rename/move, scene
  duplicate/move, collection disabled state, responsive sidebar, and direct
  dashboard/scene routes.
- Production smoke uses the two disposable test accounts, checks `/healthz`,
  password login, workspace-scoped list/create/update flows, and confirms no
  credential is written to Git or release evidence.

