# ADR 0012 — Reuse the original dashboard presentation

Status: accepted; user confirmed on 2026-10-16.
Trace: [C1 plan](../superpowers/plans/2026-10-16-collaboration-dashboard.md), [phase](../work/phases/PHASE-COLLAB-V2.md).

## Decision

Use the original devtool shadcn UI as the presentation source: Arial/Helvetica, spacing, light/dark CSS variables, sidebar, header actions and aspect-video scene cards. The user explicitly removed the previous monochrome override; original colored accents remain.

Keep the current Vite/Zustand routing, password authentication and folder/scene APIs. Apply tokens through a dashboard context, including Radix portal roots; do not theme body/html or the Excalidraw scene.

## Consequences

The old Next/Jotai application is a design/code reference, not a runtime dependency. Dead controls are omitted until their backend exists. Existing folder actions and scene actions remain accessible. Imported scenes are validated before one atomic creation request. No schema, API endpoint or deployment changes.

See [technical behavior](../architecture/dashboard-ui.md), [user guide](../user/dashboard.md) and [verification](../work/verification/collaboration-v2.md).
