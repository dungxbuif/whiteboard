---
artifact_type: software_design_document
id: SDD-PHASE-1
status: verified
owner: shared
approval: approved
trace:
  backlog_item: EXC-012
  requirement: REQ-10
  phase: PHASE-1
  ticket_or_bug: EXC-012
  validation_matrix: ../work/VALIDATION_MATRIX.md
  docs_review: ../work/tickets/phase-01/EXC-012/EXC-012-excalidraw-integration.md
  adrs: [ADR-0007]
  master_docs_touched: [ARCHITECTURE.md]
---

# Software Design Document: Phase 1 Whiteboard Client

## Status

- ID: SDD-PHASE-1
- Status: verified
- Approval: inherited from the approved EXC-012 detail design
- Updated: 2026-06-12

## 1. Executive Summary

The React dashboard embeds Excalidraw for workspace-scoped scenes. The browser owns transient editor state, Zustand owns scene metadata and save lifecycle, and the Go API remains the persistence and tenancy boundary.

## 2. System Context & Boundary

```text
+----------------+     +--------------------+     +----------------+
| React Dashboard| --> | Excalidraw Wrapper | --> | Zustand Store  |
+----------------+     +--------------------+     +-------+--------+
                                                        |
                                                        v
                                                +---------------+
                                                | Go Scene API  |
                                                +-------+-------+
                                                        |
                                                        v
                                                +---------------+
                                                | PostgreSQL    |
                                                +---------------+
```

Binary assets are outside this boundary until EXC-013.

## 3. Architectural Decisions & Key Choices

- Dynamic import keeps editor JavaScript out of the initial dashboard entry.
- Zustand stores scene lists, active scene metadata, pending snapshots, and save status; it does not subscribe React to every live canvas mutation.
- A 750 ms debounce coalesces rapid changes.
- A client generation token ignores stale save completions in UI state.
- The server remains last-write-wins and increments `contentVersion` for observability.
- See [ADR-0007](../decisions/0007-excalidraw-runtime-dependency.md).

## 4. Component Design

- `scene-store.ts`: workspace-scoped API operations and save lifecycle.
- `scene-list.tsx`: current-folder scene cards and open action.
- `scene-actions.tsx`: rename, star, and trash operations.
- `excalidraw-wrapper.tsx`: editor loading, initialization guard, serialization, status, and back navigation.
- `App.tsx`: dashboard/editor composition and workspace/folder fetch orchestration.

## 5. Data Design

No schema change. The client persists `{elements, appState, files}` into the existing scene JSONB `content` field.

## 6. Security & Compliance

All requests include the active `X-Workspace-ID`; backend middleware and repositories enforce tenancy. Transitive dependency advisories and reachability are recorded in ADR-0007.

## 7. Infrastructure & Deployment Design

No service topology change. Vite emits the editor and optional diagram tooling as lazy chunks loaded from the same frontend deployment.

## 8. Failure Modes & Resilience

- Save failures remain visible as `Save failed`; the UI does not claim success.
- Pending content is flushed when leaving the editor.
- Older request completions cannot replace newer local save status.
- Initial Excalidraw normalization callbacks are ignored for 250 ms to avoid saving an unchanged scene on open.

## 9. Migration & Compatibility

No database migration. Existing scene JSON uses Excalidraw defaults for missing optional values. Rollback removes the frontend dependency and scene UI without changing persisted rows.

## 10. Open Questions

- Re-audit and upgrade Excalidraw when upstream resolves Mermaid parser and nanoid advisories.
- EXC-013 must define asset file hydration and upload behavior.

## 11. Feedback Log & Revision History

| Date | Reviewer | Feedback Item | Resolution / Action Taken | Status |
| --- | --- | --- | --- | --- |
| 2026-06-12 | Human | Approved EXC-012 detail design | Implemented and verified the approved slice | resolved |
