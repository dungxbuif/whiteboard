# C — Legacy dashboard and management UI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Restore the old dashboard styling and expose working invite/share/agent administration.

**Architecture:** Port reference shadcn presentation into existing Vite/Zustand application; preserve current APIs until A contracts are available. Scope theme tokens at dashboard shell so Excalidraw remains independent.

**Tech Stack:** React, shadcn/Radix, Tailwind, Zustand, TypeScript 7.

**Spec:** [Approved collaboration design](../specs/2026-10-15-collaboration-v2-design.md); approved in conversation by “xác nhận”; implementation has not started.

## Global Constraints

- Read [programme contracts](2026-10-16-collaboration-v2.md) before executing; its interface types, limits, dependency order and documentation requirements apply to every task.
- Custom Node/TypeScript WebSocket service; RelayHub is not involved.
- Workspace roles are `owner`, `editor`, and `viewer`; scene overrides are `editor` or `viewer`; management is owner-only.
- Invitation links: single-use, email-bound, seven-day expiry. Room JWT: ten minutes.
- Update frame: 1 MiB; materialized document: 25 MiB; 120 updates/minute/principal.
- Snapshot: first of 100 updates or five seconds. Voice: ten active participants; no product-level data-participant cap.
- Dashboard follows `references/devtool` styling; monochrome applies to dashboard only. Scene components retain native library styling.
- Retain TypeScript 7 toolchain, password login, default workspace provisioning, and OpenAI-compatible provider boundary.
- Every task updates its affected docs before implementation and reconciles them with verified behavior before commit.

## Review Focus

- Long workspace names, zero scenes and many folders overflow sidebar — C1.
- Mobile collapse and keyboard focus fail after port — C1.
- Theme globals leak onto native Excalidraw — C1.
- Viewer sees enabled owner controls or stale role after revocation — C2.
- Invitation expires while form is open or browser reloads deep link — C2.

---

### C1: Faithful dashboard style port

Minimum browser assertions, alongside inspected visual baselines:

```ts
await expect(page.getByRole("navigation", { name: "Workspace navigation" })).toBeVisible();
await expect(page).toHaveScreenshot("legacy-dashboard-desktop.png");
await page.setViewportSize({ width: 390, height: 844 });
await expect(page).toHaveScreenshot("legacy-dashboard-mobile.png");
```

**Files:** Create `src/frontend/src/components/dashboard/{dashboard-shell,dashboard-sidebar,dashboard-header,user-menu}.tsx`, `src/frontend/src/styles/dashboard.css`, `src/frontend/e2e/dashboard-style.spec.ts`; modify App, index.css, workspace-switcher, scene-list and UI primitives; port reference sidebar/separator/sheet/tooltip components as needed.

**Interfaces:** `DashboardShell({children})` owns `.dashboard-theme` tokens, responsive sidebar and footer; `DashboardSidebar({workspaceId})` consumes existing stores. Use reference `globals.css`, `tailwind.config.ts`, Sidebar subtree, DashboardHeader, SceneItem and EmptyState as the visual sources.

- [ ] **Step 1 — Record intended behavior:** update `docs/user/dashboard.md` (create), `docs/architecture/dashboard-ui.md` (create) with this task's API/data/flow decision and the cases below.
- [ ] **Step 2 — Add failing proof:** `dashboard-style.spec.ts`: desktop 1440px and mobile 390px; long names/empty/populated state; keyboard sidebar toggle returns focus; light/dark snapshots; open scene screenshot shows original Excalidraw styling. Baselines require visual inspection against rendered old-reference components, not blindly accepting screenshots of new code.
- [ ] **Step 3 — Verify failure:** run `rtk npm exec --workspace ./frontend -- playwright test dashboard-style.spec.ts`; confirm the new assertion fails for missing behavior, not a broken fixture.
- [ ] **Step 4 — Implement:** Port old typography, spacing, border radius, card composition, collapsible navigation, search and user footer. Preserve password auth and current folder/scene functionality; replace Next Link/Jotai plumbing with existing router/Zustand. Use monochrome focus/selection tokens in dashboard only. Import scenes button must invoke validated .excalidraw import if shown; implement `parseSceneImport(file:File):Promise<SceneContent>` and create a scene via current API, with malformed/oversized-file browser cases. Never copy dead handlers or literal stray characters.
- [ ] **Step 5 — Verify:** rerun `rtk npm exec --workspace ./frontend -- playwright test dashboard-style.spec.ts`; require all listed cases to pass. Record actual results in the phase verification record; skipped integration tests do not count as passes.
- [ ] **Step 6 — Reconcile and commit:** reconcile `docs/user/dashboard.md` (create), `docs/architecture/dashboard-ui.md` (create); stage only the named files and supporting lockfiles; commit `feat: restore legacy dashboard styling`.

### C2: Invites, members, public sharing and routes

**Files:** Create `src/frontend/src/components/collaboration/{member-list,invite-dialog,invite-accept,scene-permissions,share-links}.tsx`, `src/frontend/src/lib/collaboration-api.ts`, `src/frontend/e2e/access-flows.spec.ts`; modify App/router, share-dialog, public-view, workspace-store and dashboard settings links.

**Interfaces:** Consume A HTTP DTOs; produce `acceptInvite(token:string):Promise<Membership>`, `createPublicSession(input:PublicSessionInput):Promise<SceneSession>`; routes `/invite/:token`, `/w/:id/settings/members`, `/w/:id/scenes/:sceneId`, `/p/:token`.

- [ ] **Step 1 — Record intended behavior:** update `docs/user/collaboration.md`, `docs/user/dashboard.md`, `docs/architecture/API.md` with this task's API/data/flow decision and the cases below.
- [ ] **Step 2 — Add failing proof:** `access-flows.spec.ts`: owner copies invite, recipient registers/signs in and accepts, wrong email/expired token cannot join; viewer sees no edit/manage controls; scene override and public password/edit display-name flows; revoke live session removes edit capability. Reload/back/forward retain workspace and scene; old UUID route and new public alias resolve through ACL.
- [ ] **Step 3 — Verify failure:** run `rtk npm exec --workspace ./frontend -- playwright test access-flows.spec.ts`; confirm the new assertion fails for missing behavior, not a broken fixture.
- [ ] **Step 4 — Implement:** Integrate real A APIs; role-aware UI mirrors server checks. Keep invite token during login safely without logging it; clear from transient state after consume. New user gets personal default workspace plus invited membership. Use settings routes within dashboard shell and native-style public scene. Share dialog supports independent view/edit link rows, expiry/password and revoke; never retrieve a secret token from list API.
- [ ] **Step 5 — Verify:** rerun `rtk npm exec --workspace ./frontend -- playwright test access-flows.spec.ts`; require all listed cases to pass. Record actual results in the phase verification record; skipped integration tests do not count as passes.
- [ ] **Step 6 — Reconcile and commit:** reconcile `docs/user/collaboration.md`, `docs/user/dashboard.md`, `docs/architecture/API.md`; stage only the named files and supporting lockfiles; commit `feat: add workspace invitation and sharing workflows`.
