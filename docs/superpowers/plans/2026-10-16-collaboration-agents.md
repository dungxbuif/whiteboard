# D — Native MCP and diagram catalog Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let authorized agents discover, create and edit scenes using editable diagram presets.

**Architecture:** MCP TypeScript service calls Go APIs and obtains operation grants; collab command ingress handles writes. A shared diagram package validates and lays out native elements; bounded browser workers provide exports.

**Tech Stack:** Go, PostgreSQL, TypeScript 7, MCP SDK, Excalidraw, Yjs, Docker, Playwright.

**Spec:** [Approved design](../specs/2026-10-15-collaboration-v2-design.md), approved by “xác nhận”; not implemented.

## Global Constraints

- Read [programme contracts](2026-10-16-collaboration-v2.md); exact interfaces, Go policy and documentation requirements apply.
- Custom Node/TypeScript WebSocket service; RelayHub is not involved. Go owns authorization.
- Roles: owner/editor/viewer; management owner-only. Invite TTL seven days, single-use/email-bound. Room JWT ten minutes.
- Update frame 1 MiB; document 25 MiB; 120 updates/minute/principal; snapshot at 100 updates or five seconds.
- Ten voice participants; no product-level data cap. Retain TypeScript 7 and password/default-workspace provisioning.
- Dashboard copies old shadcn styling with its original colors and tokens. Scene components retain native style.

## Review Focus

- Agent key for workspace A enumerates/mutates B — D1/D3.
- Diagram generations reuse IDs or a retry inserts duplicates — D2/D3.
- Malformed Mermaid, huge graph or unsafe external assets exhaust renderer — D2/D4.
- Stale agent patch overwrites concurrent human changes — D3.
- Client without custom-header support cannot use PAT auth — D1 compatibility docs.

---

### D1: Agent keys and MCP discovery transports

**Files:** Create `src/backend/migrations/000010_agent_keys.{up,down}.sql`, Go `internal/domain/agent.go`, `internal/usecase/agent_ucase.go`, `internal/repository/postgres/agent_repo.go`, `internal/delivery/http/handler/agent_handler.go` and tests; create `src/mcp/{package.json,tsconfig.json,src/server.ts,src/stdio.ts,src/auth.ts,src/go-client.ts,src/server.test.ts}`; create frontend `components/collaboration/agent-settings.tsx`.

**Interfaces:** Go `CreateAgentKey(ctx,owner,scopes,targets) (KeyOnce,error)`, `AuthenticateAgent(ctx,bearer) (Principal,error)`; owner-only GET/POST/DELETE `/api/v1/workspaces/{id}/agent-keys[/{keyId}]`. MCP `list_workspaces`, `list_scenes`, `get_scene`, `create_scene`; page limit 50, projection cap 500 elements/256 KiB.

- [ ] **Step 1 — Record behavior:** update `docs/user/agents.md` (create), `docs/architecture/API.md`, `docs/decisions/0011-native-mcp.md` (create) with contracts and verification cases.
- [ ] **Step 2 — Add failing proof:** `TestAgentScopeIntersection`: read-only key cannot write; key for A cannot enumerate B; revoked owner/key denied; DB stores hashes only. `server.test.ts`: initialize/tools/list lifecycle, missing bearer 401, invalid Origin rejected, stdio stdout only MCP frames, pagination doesn't leak other tenants' totals.
- [ ] **Step 3 — Verify failure:** run `rtk npm test --workspace ./mcp -- server.test.ts`; confirm missing behavior causes failure, not broken infrastructure.
- [ ] **Step 4 — Implement:** Use server-managed bearer keys for clients supporting explicit headers and a stdio remote proxy with credential in secure config/environment. Do not call PAT support full OAuth compliance: OAuth discovery/PKCE is a separate compatibility extension. Authenticate every HTTP call and bind MCP session to principal. Stdio proxy does not start a local DB. UI uses legacy dashboard style, key shown once, audit stores only key ID. Run Go Agent tests in backend too.
- [ ] **Step 5 — Verify:** rerun `rtk npm test --workspace ./mcp -- server.test.ts`; require listed cases to pass and record actual results.
- [ ] **Step 6 — Reconcile and commit:** reconcile `docs/user/agents.md` (create), `docs/architecture/API.md`, `docs/decisions/0011-native-mcp.md` (create), stage named files and lockfiles only, commit `feat: expose scoped native MCP scene tools`.

### D2: Discoverable native diagram catalog

Required compiler fixture assertions (independent of rendering snapshots):

```ts
expect(first.elements.map(e => e.id)).toEqual(repeated.elements.map(e => e.id));
expect(intersectIds(first.elements, otherInstance.elements)).toEqual([]);
expect(unresolvedBindings(first.elements)).toEqual([]);
expect(first.elements.every(e => Number.isFinite(e.x) && Number.isFinite(e.y))).toBe(true);
```

**Files:** Create `src/packages/diagram-kit/{package.json,tsconfig.json,src/types.ts,src/catalog.ts,src/layout.ts,src/compile.ts,src/mermaid.ts,src/catalog.test.ts,src/fixtures/}`; create `src/presets/{graph,sequence,erd,class,state,tree,timeline,journey,swimlane}.ts` inside diagram-kit and curated library manifest; create `docs/research/2026-10-16-diagram-dependencies.md`.

**Interfaces:** Produce `listDiagramTypes():PresetDescriptor[]`, `getDiagramTemplate(id:string):{schema:JSONSchema,example:unknown}`, `compileDiagram(input:DiagramInput):Promise<CompiledDiagram>`; input `{preset,instanceId,schemaVersion:1,theme:'sketch'|'clean'|'pastel',direction:'LR'|'TB',data}`; result `{elements:NativeElement[],files:AssetReference[],bounds,warnings}`. MCP tools `list_diagram_types`, `get_diagram_template`, `validate_diagram`; resources `whiteboard://diagrams/catalog` and `whiteboard://diagrams/{id}/schema`.

- [ ] **Step 1 — Record behavior:** update `docs/research/2026-10-16-diagram-dependencies.md`, `docs/user/diagram-catalog.md`, `src/frontend/public/docs/diagram-catalog.md` (create missing docs) with contracts and verification cases.
- [ ] **Step 2 — Add failing proof:** `catalog.test.ts` iterates flowchart, sequence, architecture, deployment, infrastructure, c4-context, c4-container, c4-component, erd, class, state, mindmap, timeline, user-journey, swimlane, org-chart, network, data-flow, agent-workflow. Assert native elements, stable IDs for same instance, disjoint IDs between instances, resolved bindings, finite geometry, no unintended overlaps in canonical fixtures. Cycle/missing-node/Unicode/long-label cases produce actionable diagnostics; invalid schema writes nothing.
- [ ] **Step 3 — Verify failure:** run `rtk npm test --workspace ./packages/diagram-kit`; confirm missing behavior causes failure, not broken infrastructure.
- [ ] **Step 4 — Implement:** Typed schemas: graph nodes/edges/groups, sequence participants/messages, ER entities/fields/relations, class members/methods, tree nodes/parents, timeline events, journey actors/stages, swimlane lanes/steps. Graph engine supports C4/network/architecture/agent semantics. ELK handles layered routing; dedicated ordered sequence/timeline renderers handle those layouts. Use Excalidraw skeleton conversion and binding repair. IDs derive from instanceId+semantic ID with group/file/binding remap. No provider key required. Dependency pin/license/security decision recorded before imports.
- [ ] **Step 5 — Verify:** rerun `rtk npm test --workspace ./packages/diagram-kit`; require listed cases to pass and record actual results.
- [ ] **Step 6 — Reconcile and commit:** reconcile `docs/research/2026-10-16-diagram-dependencies.md`, `docs/user/diagram-catalog.md`, `src/frontend/public/docs/diagram-catalog.md` (create missing docs), stage named files and lockfiles only, commit `feat: add editable diagram presets and discovery`.

### D3: Live mutation tools and safe retries

**Files:** Create `src/mcp/src/{scene-tools,mutation-client,diagram-tools}.ts`, `src/mcp/tests/scene-tools.integration.test.ts`; extend B2 private commands and A4 Go grants.

**Interfaces:** MCP `generate_diagram({sceneRef,preset,data,instanceId,requestId,expectedSequence,dryRun})`, `patch_elements({sceneRef,patches,requestId,expectedSequence})`, `delete_elements({sceneRef,ids,requestId,expectedSequence})`, `get_scene_activity({sceneRef,cursor,limit})` use shared SceneMutation; return receipt, scene URL, created IDs, warnings.

- [ ] **Step 1 — Record behavior:** update `docs/user/agents.md`, `docs/architecture/collaboration-protocol.md`, `src/frontend/public/docs/mcp.md` (create) with contracts and verification cases.
- [ ] **Step 2 — Add failing proof:** `scene-tools.integration.test.ts`: generate in allowed scene, browser sees same IDs, restart retains diagram; retry returns same receipt/one event; changed payload same requestId 409; stale sequence 409; fresh read/new request succeeds preserving human edits; delete creates tombstones; dry-run leaves revision unchanged.
- [ ] **Step 3 — Verify failure:** run `rtk npm test --workspace ./mcp -- scene-tools.integration.test.ts`; confirm missing behavior causes failure, not broken infrastructure.
- [ ] **Step 4 — Implement:** Create scene via Go, compile via D2, sign via A4, submit once through B2. Atomic compare-revision append and principal-bound dedup; retry keeps IDs. Paginate/redact activity. Accurate readOnly/destructive/idempotent tool hints. Return conflict guidance rather than blindly retrying stale patches. No whole-scene replacement or agent permission/key/share-management tools.
- [ ] **Step 5 — Verify:** rerun `rtk npm test --workspace ./mcp -- scene-tools.integration.test.ts`; require listed cases to pass and record actual results.
- [ ] **Step 6 — Reconcile and commit:** reconcile `docs/user/agents.md`, `docs/architecture/collaboration-protocol.md`, `src/frontend/public/docs/mcp.md` (create), stage named files and lockfiles only, commit `feat: apply agent diagrams through durable collaboration`.

### D4: Mermaid and bounded scene exports

**Files:** Create `src/mcp/src/{render-worker,export-tools}.ts`, `src/mcp/tests/render.integration.test.ts`, render page and renderer Dockerfile; extend Mermaid adapter/support matrix in diagram-kit.

**Interfaces:** `compileMermaid(source:string):Promise<CompiledDiagram>`, `exportScene(sceneRef,format:'json'|'svg'|'png'):Promise<ScopedArtifact>`; result `{id,mimeType,expiresAt,url,revision}` with authenticated or short-lived scene-scoped download.

- [ ] **Step 1 — Record behavior:** update `docs/user/diagram-catalog.md`, `docs/user/agents.md`, `docs/operations/mcp.md` (create), dependency research with contracts and verification cases.
- [ ] **Step 2 — Add failing proof:** `render.integration.test.ts`: supported Mermaid gives editable elements; unsupported native syntax returns `UNSUPPORTED_NATIVE_DIAGRAM` with DSL alternative; exports preserve fonts/labels/assets and MIME; arbitrary URL/loopback abuse blocked; malformed/oversize input fails within deadline; revoked principal cannot get fresh artifact grant.
- [ ] **Step 3 — Verify failure:** run `rtk npm test --workspace ./mcp -- render.integration.test.ts`; confirm missing behavior causes failure, not broken infrastructure.
- [ ] **Step 4 — Implement:** Pin tested parser/Excalidraw versions; fixture-based support matrix, reject image fallback for native generation. Bundle fonts; render generated local content with external network disabled and assets injected after scene authorization. Defaults: 500 nodes/1000 edges, 50,000 source chars, one render per worker, 10-second deadline, 4096px max dimension, 20 MiB artifact cap. Kill timeout workers. Curated library manifest records each asset's source/hash/license; no unreviewed plugin runtime.
- [ ] **Step 5 — Verify:** rerun `rtk npm test --workspace ./mcp -- render.integration.test.ts`; require listed cases to pass and record actual results.
- [ ] **Step 6 — Reconcile and commit:** reconcile `docs/user/diagram-catalog.md`, `docs/user/agents.md`, `docs/operations/mcp.md` (create), dependency research, stage named files and lockfiles only, commit `feat: add bounded Mermaid conversion and scene exports`.

## Research sources

Pin the tested version and preserve license notices before reuse. These sources are candidates/references, not claims of installed dependencies:

- [Official Mermaid parser](https://github.com/excalidraw/mermaid-to-excalidraw): skeleton conversion; native support needs fixtures.
- [ELK JS](https://github.com/kieler/elkjs): graph layout; inspect licensing/notice requirements.
- [Excalidraw libraries](https://github.com/excalidraw/excalidraw-libraries): curate individual shape provenance.
- [Official MCP](https://github.com/excalidraw/excalidraw-mcp): streamed preview pattern.
- [yctimlin](https://github.com/yctimlin/mcp_excalidraw), [debu-sinha](https://github.com/debu-sinha/excalidraw-mcp-server), [excaliwow](https://github.com/excaliwow/mcp), [hangarbay](https://github.com/hangarbay/excalidraw.mcp): interface references; do not assume hosted app source is public.
- [MCP authorization](https://modelcontextprotocol.io/specification/2025-06-18/basic/authorization): distinguish PAT-compatible clients from OAuth-discovery clients.
