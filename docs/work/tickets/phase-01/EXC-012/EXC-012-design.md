---
artifact_type: detail_design
id: DESIGN-EXC-012
status: verified
owner: ai
approval: approved
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
  backlog_item: EXC-012
  requirement: REQ-10
  phase: PHASE-1
  ticket_or_bug: EXC-012
  test_verification: pending
  validation_matrix: ../../../VALIDATION_MATRIX.md
  docs_review: pending
  adrs: [ADR-0007]
  master_docs_touched:
    - ../../../../architecture/ARCHITECTURE.md
    - ../../../../architecture/SDD.md
---

# DETAIL DESIGN: EXC-012 Excalidraw Integration

## Status

- ID: DESIGN-EXC-012
- Status: verified
- Ticket/Bug: [EXC-012](EXC-012-excalidraw-integration.md)
- Approval: approved by reviewer on 2026-06-12
- Author: AI
- Updated: 2026-06-12

## Trace Links

- Backlog item: [EXC-012](../../../BACKLOG.md)
- Requirement: [REQ-10](../../../../requirements/SPEC.md#req-10-embedded-excalidraw-library)
- Phase: [PHASE-1](../../../phases/PHASE-1.md)
- Ticket/Bug: [EXC-012](EXC-012-excalidraw-integration.md)
- Test verification: 16 unit tests, production build, and 7 Playwright tests passed
- Validation matrix: [VALIDATION_MATRIX](../../../VALIDATION_MATRIX.md)
- Docs review: pending execution
- ADRs: [ADR-0007](../../../../decisions/0007-excalidraw-runtime-dependency.md)
- Master docs touched: `ARCHITECTURE.md`, `SDD.md`

---

## 1. Context & Scope

### Problem Statement

- Problem statement: The dashboard currently renders static scene cards and has no embedded drawing surface or scene persistence client.
- Why now: EXC-011 established the workspace-scoped scene API required by REQ-10.
- Success outcome: A user can create or open a scene, edit it in embedded Excalidraw, and have valid scene JSON saved through a debounced workspace-scoped API call.

### Context Loaded

- `docs/CONTEXT.md`
- `docs/work/BACKLOG.md`
- `docs/standards/{README,QUALITY_BAR,VALIDATION,CODE,TESTING,GIT,DOCS}.md`
- `docs/work/phases/PHASE-1.md`
- `docs/work/tickets/phase-01/EXC-012/EXC-012-excalidraw-integration.md`
- `docs/requirements/SPEC.md`
- `docs/architecture/{ARCHITECTURE,API}.md`
- Current frontend app, stores, API client, folder list, and Playwright suite
- Current backend scene handler and domain contract

### Brownfield Scope

- Touched modules/files: frontend package manifest/lockfile, app composition, scene store, scene list/actions, whiteboard wrapper, focused unit tests, and dashboard E2E tests.
- Direct dependencies inspected: workspace store, folder store, API client, scene REST contract, Tailwind layout, and current Playwright mocks.
- Contracts affected: frontend consumption of `GET/POST/PATCH/DELETE /scenes` and `POST /scenes/{id}/star`; no backend contract change planned.
- Known unknowns: exact Excalidraw package peer compatibility with React 19 will be verified by install/build and rendered tests.
- Scope expansion reason: none. Binary asset persistence is EXC-013; aggregate search/starred/trash views are EXC-014.

### Small Task Exemption

- Small task exemption: no
- Reason: Adds a major runtime dependency and user-facing persistence workflow.
- Impact checked: API=yes, DB=no, Security=no, Runtime=yes, Standards=no

## 2. Design Considerations & Trade-offs

| Consideration / Alternative | Pros | Cons | Decision |
| --- | --- | --- | --- |
| Store full Excalidraw state in Zustand | Single state source | High-frequency element updates cause broad React subscriptions | Rejected |
| Keep live canvas state inside Excalidraw and store scene metadata/save state in Zustand | Limits rerenders and keeps persistence explicit | Wrapper must serialize change payloads | Chosen |
| Save every `onChange` | Simple | Excessive API writes | Rejected |
| Debounced save with a per-scene generation token | Bounded writes; stale completions cannot overwrite UI save status | Slightly more state logic | Chosen |
| Add client optimistic locking | Detects conflicts | Conflicts with approved Phase 1 last-write-wins contract | Rejected |
| One dashboard route with selected-scene state | Minimal routing work | Scene URL is not directly shareable in Phase 1 | Chosen; public/deep links are outside this ticket |

## 3. Architecture Overview

### Component Responsibilities

| Component | Role |
| --- | --- |
| `scene-store.ts` | Workspace-scoped scene CRUD, active scene, save status, debounce scheduling, and stale-request generation tracking |
| `scene-list.tsx` | Render scenes in the current folder and expose create/open actions |
| `scene-actions.tsx` | Rename, star/unstar, and move-to-trash actions |
| `excalidraw-wrapper.tsx` | Lazy-load and render Excalidraw; translate initial API JSON and `onChange` payloads |
| `App.tsx` | Switch between dashboard and active whiteboard, and reset/fetch scene state on workspace/folder changes |

`@excalidraw/excalidraw` will be dynamically imported so the dashboard does not pay the editor bundle cost before a scene is opened.

## 4. Execution Flow

1. After authentication and workspace selection, fetch scenes for the current folder with `X-Workspace-ID`.
2. `New scene` creates a scene with empty Excalidraw content and opens it.
3. Opening a scene lazy-loads Excalidraw and initializes elements, app state, and files from the stored JSON.
4. Each canvas change sends a serializable snapshot to `scheduleSave`.
5. The store resets one debounce timer and saves the latest snapshot after 750 ms of inactivity.
6. PATCH completion updates the scene and `contentVersion`; a generation token prevents an older response from clearing a newer pending/saving state.
7. Back returns to the folder dashboard after flushing any pending snapshot.
8. Rename, star, and trash update the backend and then reconcile the local list; trash closes the active scene if applicable.

Last-write-wins means the backend accepts the latest completed PATCH. `contentVersion` remains observability metadata and is not sent as an optimistic precondition.

## 5. API & Data Model Design

### API Changes

- No backend API changes.
- Consume `GET /api/v1/scenes?folderId={id}` for folder scenes; omit `folderId` at workspace root.
- Consume `POST /api/v1/scenes/` with `{title, folderId, content}`.
- Consume `PATCH /api/v1/scenes/{id}/` with title and/or serialized content.
- Consume `POST /api/v1/scenes/{id}/star` with `{isStarred}`.
- Consume `DELETE /api/v1/scenes/{id}/` for soft delete.

### Data Model Changes

- Database schema migration needed: no.
- Frontend scene content shape: `{ elements, appState, files }` with defensive defaults for absent or malformed optional members.

## 6. Security & Authorization

- Authentication changes: none.
- Authorization/permissions: every scene request keeps the active `X-Workspace-ID`; backend remains authoritative for tenancy.
- Data privacy/PII impact: none introduced.
- Input validation: trim and require scene titles; serialize only JSON-compatible Excalidraw scene data; surface API failures without claiming content is saved.

## 7. Implementation & Verification Plan

### Impacted Areas

- Code/modules: React dashboard, Zustand scene store, Excalidraw wrapper, unit/E2E tests.
- Product behavior: create/open/edit/rename/star/trash scenes and autosave drawings.
- API/contracts: client consumption only; server contract unchanged.
- Data/schema: no change.
- Security/auth: no change; existing auth and workspace header reused.
- Deployment/runtime: frontend bundle gains `@excalidraw/excalidraw`; editor chunk lazy-loaded.
- Docs: ticket, detail design, ADR, architecture/SDD, validation matrix, context, backlog, phase, and changelog.

### Test Plan

- Unit: scene-store list/create/action state changes; fake-timer debounce; stale save completion; save error state.
- Integration: TypeScript build validates Excalidraw types and Vite production bundling.
- E2E: login -> create scene -> canvas renders -> draw/update -> one debounced PATCH -> rename/star/trash actions; desktop and mobile dashboard/canvas checks where practical.
- UAT: acceptance criteria verified through Playwright because Browser plugin is not available in this session.
- Manual/platform: inspect screenshots, console errors, clipping, overlays, and responsive behavior.
- Docs review: complete the ticket checklist and reconcile master docs.

### Validation Matrix Impact

- Update required: yes
- Row(s): EXC-012 / REQ-10

### Verification Results

- Command: `npm run test:unit`; result: pass, 16 tests in 5 files.
- Command: `npm run build`; result: pass, editor emitted as lazy chunks.
- Command: `npx playwright test --reporter=line --workers=1`; result: pass, 7 Chromium tests.
- Command: `npm audit --omit=dev`; result: known upstream risk, 1 high and 8 moderate transitive advisories; no compatible fix is available.

## 8. Reconciliation Plan

- Requirements docs: no change; implementation follows existing REQ-10.
- Architecture docs: update dependency loading and frontend scene-state responsibilities.
- API docs: no change; endpoints are already documented.
- ERD docs: no change; no schema change.
- SDD docs: update with the implemented client flow and component ownership.
- ADR: create for the major Excalidraw runtime dependency and lazy-loading decision.
- Context: update active/next work and evidence.

### Docs Review Checklist

- Code changed but docs unchanged reason: not applicable; affected docs will be updated.
- Requirements updated or not needed reason: no requirement change.
- Architecture updated or not needed reason: update required for frontend ownership/runtime dependency.
- API updated or not needed reason: no API contract change.
- ERD/data updated or not needed reason: no schema change.
- SDD updated or not needed reason: update required.
- ADR created or not needed reason: required by dependency standard.
