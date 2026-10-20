# Collaboration undo and keyboard recovery — 2026-10-19

## Intended behavior

- A local Excalidraw edit committed through collaboration v2 can be undone and
  redone by the scene header controls and by `Ctrl/Cmd+Z` and
  `Ctrl/Cmd+Y`/`Ctrl/Cmd+Shift+Z`.
- Undo/redo mutations use the same durable Yjs writer, admission checks,
  recovery queue, acknowledgement path, and scene-scoped authorization as
  ordinary edits.
- Another actor's fields and agent-authored elements remain outside the local
  undo scope.

## Production finding

The 2026-10-19 release canary drew and durably projected one rectangle, but an
immediate local undo changed the client to `Recovery required` with
`INVALID_UPDATE`; the authoritative projection remained unchanged. The same
failure occurs from the explicit **Undo my edit** button, so this is a mutation
validation problem rather than a browser shortcut problem.

## Technical decision and affected flow

Reproduce the exact lifecycle with a client Y.Doc, a server Y.Doc, and the
server validation worker. Replace raw structural Yjs undo with bounded semantic
scene history. Merge only the local before/desired delta into the current scene,
write deletion tombstones into the existing element map, and revive that same
map on redo. Keep `gc: false` and do not bypass the worker, schema validation,
access checks, or durable receipt path.

Affected flow:

```text
Excalidraw local edit -> semantic history -> LOCAL_ORIGIN Yjs update -> durable ACK
  -> semantic undo/redo merge -> validation worker
  -> durable ACK -> all connected projections converge
```

## Verification plan

1. Add a regression that validates and applies add, undo, and redo updates
   across separate client/server Y.Docs.
2. Preserve the existing remote-edit and agent-edit undo isolation test.
3. Run the focused frontend/collab suites, full source gates, and production
   browser canary against the authoritative agent projection.
4. Reconcile public keyboard/collaboration docs and the AI-readable release
   surfaces with the behavior that actually ships.

## Local reconciliation

The frontend now stores bounded semantic before/desired scene pairs and merges
only the local delta into the current collaborative scene. Undo of a local
insertion writes an `isDeleted` tombstone into the existing element map; redo
revives that same map. Concurrent remote fields and agent-created elements are
preserved. History is capped at 25 entries and 32 MiB.

Focused validation passes 51 collaboration document tests and 6 frontend
undo/scene-diff tests. The complete frontend suite passes 25 files / 94 tests
plus the production build. The complete collaboration suite passes 15 files /
150 tests against the guarded loopback `*_test` database. Production commit
`76fcfd5` is deployed with a healthy frontend image. On the user-authenticated
production scene, a rectangle reached `Saved`, semantic Undo reached `Saved`,
semantic Redo reached `Saved`, and a final Undo restored the original empty
scene. A separate temporary-scene harness attempt failed before drawing because
its interaction did not produce an update; cleanup confirmed zero temporary
scenes and zero agent keys, so it is not used as product evidence.
