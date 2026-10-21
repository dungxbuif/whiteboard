# Dashboard style parity — 2026-10-21

## Intended behavior

The workspace dashboard should use the old Whiteboard dashboard's presentation: a compact workspace switcher, search field, icon-led navigation, collection rows, and account footer in a light gray sidebar. The current dashboard routes, workspace switching, collection actions, settings links, notifications, theme toggle, mobile drawer, and authorization behavior remain unchanged. Scene/canvas styling is outside this change.

## Reference and decision

The reference is `references/devtool/src/components/Sidebar/` from the legacy Whiteboard app. We are porting its structure and spacing into the current React dashboard components instead of copying legacy data or routing code. The current route/API surface remains the source of truth.

## Verification plan

- Run the focused dashboard Playwright suite at desktop and mobile sizes.
- Run the frontend unit/build checks.
- Capture a production smoke check after deployment and verify the authenticated dashboard shell renders with the new sidebar.

## Implementation and verification

Implemented the parity pass in the current dashboard components. The sidebar now uses the legacy visual hierarchy and spacing while preserving current navigation handlers, workspace and collection actions, notification inbox, account menu, and theme controls. The scene canvas and its presentation controls were not changed.

Verification completed in the worktree:

- `npm run build:contracts` — passed.
- `npm run build --workspace ./frontend` — passed.
- `npx playwright test e2e/dashboard-style.spec.ts --config playwright.config.ts` — 4 passed, including desktop, dark theme, and mobile drawer screenshots.
- Dashboard behavior subset covering session restore, workspace switching, and legacy sidebar rename flows — 5 passed.

The protected public-link test remains a pre-existing failure in the broader dashboard suite: the scene opens, but the test's expected `Read-only` label is absent. This styling change does not touch public-scene rendering or authentication.

## VM100 deployment evidence

The merged dashboard change was promoted to VM100 by synchronizing the frontend UI sources into the existing production checkout (the checkout contains operator-managed Compose overlays and secrets, so it was not overwritten with a git pull). The frontend image rebuilt successfully and `whiteboard-frontend` was recreated; backend, collab, feature-worker, and MCP remained healthy. Public checks returned:

- `GET https://whiteboard.dungxbuif.com/healthz` — `200 ok`.
- `GET https://whiteboard.dungxbuif.com/` — `200` with the new frontend asset bundle.
- Authenticated browser dashboard — new legacy-style sidebar visible with workspace switcher, search, icon navigation, file/collection groups, settings, notifications, and account footer.
- `POST https://whiteboard.dungxbuif.com/mcp` without a bearer — `401 UNAUTHENTICATED`.
- Authenticated MCP `initialize` and `tools/list` with the newly issued workspace-scoped personal key — `200`; 13 tools advertised.
- `GET https://whiteboard.dungxbuif.com/docs/mcp.md` — public personal-agent guide served after the frontend rebuild.

The personal key was issued for the owner’s default workspace with `workspace:read`, `scene:read`, `scene:create`, `scene:write`, `scene:move`, and `scene:export`. Its one-time secret is intentionally absent from repository and deployment files.
