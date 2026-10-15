---
artifact_type: adr
id: ADR-0007
status: accepted
owner: shared
trace:
  requirement: REQ-10
  phase: PHASE-1
  tickets_or_bugs: [EXC-012]
  detail_design: ../work/tickets/phase-01/EXC-012/EXC-012-design.md
  master_docs: [../architecture/ARCHITECTURE.md, ../architecture/SDD.md]
  release_notes: ../releases/CHANGELOG.md
---

# 0007 Excalidraw Runtime Dependency

## Status

- ID: ADR-0007
- Status: accepted through EXC-012 design approval
- Date: 2026-06-12
- Owner: shared

## Context

REQ-10 requires the whiteboard to run inside the React dashboard and persist scene JSON through the Go API. Reimplementing a drawing engine is outside Phase 1 and would not provide Excalidraw behavior parity.

## Decision

Use `@excalidraw/excalidraw` `0.18.x` as the embedded editor. Load its JavaScript through a dynamic import only after a scene opens. Keep scene metadata and save state in Zustand while Excalidraw owns live canvas state. Persist the latest serialized snapshot after 750 ms of inactivity using Phase 1 last-write-wins semantics.

## Alternatives Considered

- Reimplement drawing primitives: rejected because it is high risk and does not meet parity goals.
- Load Excalidraw in the dashboard entry bundle: rejected because the editor and optional diagram tooling are large.
- Use optimistic version conflicts: rejected because Phase 1 explicitly uses last-write-wins.

## Consequences

- Positive: Excalidraw-native editing is available inside the application with a small dashboard JavaScript entry impact.
- Negative: The package produces large lazy chunks and adds transitive dependencies.
- Negative: `npm audit --omit=dev` reports one high and eight moderate advisories through optional Mermaid parser/nanoid dependencies. EXC-012 does not expose Mermaid conversion, and npm offers no compatible fix; `--force` would downgrade Excalidraw to `0.17.6`. Re-audit on upgrades.
- Neutral: Binary asset upload remains owned by EXC-013.

## Linked Work

- Ticket: [EXC-012](../work/tickets/phase-01/EXC-012/EXC-012-excalidraw-integration.md)
- Design: [EXC-012 design](../work/tickets/phase-01/EXC-012/EXC-012-design.md)
- Architecture: [ARCHITECTURE](../architecture/ARCHITECTURE.md), [SDD](../architecture/SDD.md)
