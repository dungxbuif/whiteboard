# Agent access

Owners can create a workspace-bound bearer key with selected scopes and optional scene targets. Copy the key once; only its SHA-256 digest is stored. Revoking a key or removing/deactivating its owner prevents subsequent requests. Agents never manage members, permissions, sharing, or keys.

Scopes are `workspace:read`, `scene:read`, `scene:create`, `scene:write`, `scene:move`, `scene:trash`, and `scene:export`. Tools require their named scope independently; write does not imply read or create. A scene-targeted key cannot create arbitrary scenes. Current owner permissions also apply.

MCP exposes `list_workspaces`, `list_scenes`, `get_scene`, `create_scene`, native diagram discovery/validation, `generate_diagram`, `patch_elements`, `delete_elements`, and `get_scene_activity`. Lists return at most 50 results and a cursor, without global totals. Scene projection returns at most 500 elements and 256 KiB. Oversized projection is rejected rather than silently truncated.

Use Streamable HTTP `/mcp` with `Authorization: Bearer <key>` in clients that support explicit headers. For stdio clients, run the remote adapter with `WHITEBOARD_API_URL`, `WHITEBOARD_PUBLIC_URL` and `WHITEBOARD_AGENT_KEY` in protected client environment/configuration. It does not start a database. Keep credentials out of arguments, URLs, logs, and source control. This is PAT-compatible access; OAuth discovery and PKCE are a separate extension, not implemented OAuth compliance.

Owner management: GET/POST `/api/v1/workspaces/{id}/agent-keys`, DELETE `/api/v1/workspaces/{id}/agent-keys/{keyId}` using normal login authentication. Creation body: `{ "scopes": ["workspace:read", "scene:read"], "sceneIds": [] }`. Empty sceneIds permits the workspace; an explicit list narrows scene tools. The response `key` is shown once.

Implementation verification: isolated PostgreSQL tests check scope/target intersection, revoked owner/key, hash-only storage and room credential revocation. Real MCP HTTP and stdio tests check initialization, tool discovery, authentication on each call, origin validation, principal-bound sessions and bounded tenant-filtered pagination. Dashboard route and deployment integration are programme follow-ups.


### Remote stdio setup

From `src/`, build generated workspace dependencies first: `npm run build --workspace ./packages/collab-contracts`, then `npm run build --workspace ./packages/diagram-kit`, then `npm run build --workspace ./mcp`. The final command also builds the isolated export renderer. Native compilation/generation in stdio also needs the pinned Playwright Chromium runtime: from `src`, run `./node_modules/.bin/playwright install chromium` and install the browser system libraries required by your OS. HTTP clients use the deployed service's browser runtime. Configure your client to launch `node` with the absolute path to `src/mcp/dist/stdio.js`. Set `WHITEBOARD_API_URL` to your Go API origin (for example `https://api.whiteboard.example`), `WHITEBOARD_PUBLIC_URL` to the public frontend/reverse-proxy origin serving `/w/...` and `/mcp/...` (for example `https://whiteboard.example`), and `WHITEBOARD_AGENT_KEY` to your copied key in the client's protected environment settings. Public and API origins may differ; public scene/artifact links and remote exports use `WHITEBOARD_PUBLIC_URL`, while authenticated Go requests use `WHITEBOARD_API_URL`. Remote API origins require HTTPS; loopback HTTP is allowed for local development. The adapter never follows redirects with the key.

For the HTTP service, set `WHITEBOARD_API_URL`, `WHITEBOARD_PUBLIC_URL`, comma-separated exact `MCP_ALLOWED_ORIGINS`, and optionally `PORT` (default 3002). It binds to loopback by default; deployment may set `MCP_BIND_HOST` behind its configured reverse proxy. HTTP sessions expire after ten minutes, and clients initialize a fresh session after a 404. A key is checked again for every HTTP request. Keep normal login tokens and agent keys separate.

Scene creation needs `scene:create`, reading needs `scene:read`, and diagram updates need `scene:write`. Move and trash scopes remain reserved for their dedicated tools. Export uses its independent scope and separately authorized projection; rendering and downloads use the bounded export service described below. A projection error means the scene is too large for this discovery response; it is not a partial or empty scene. Dashboard route integration and deployment remain pending in the collaboration-v2 programme.

`get_scene` returns the live durable projection with `sequence`, `documentId`, `stateVector` and `content`. The sequence is a decimal string and is independent of the older materialized `contentVersion`. If the durable service is unavailable, reads fail; they do not substitute a potentially stale snapshot.


### Audit storage and failed calls

For HTTP, set `MCP_AUDIT_PATH` to an absolute path on persistent storage before starting the service. For stdio, the default is `~/.local/state/whiteboard-mcp/audit.jsonl`; set a separate absolute path for each simultaneously running adapter. One process must own each journal path. The adapter creates files with mode 0600 and retains an active file plus two archives, each bounded to 1 MiB. Collect these files before rotation removes older events. This adds synchronous disk writes to each call; unavailable audit storage blocks new operations.

The Go database records authoritative operation metadata and commits successful scene creation together with its audit entry. Adapter journals also cover validation, denial and unavailable-upstream failures, including malformed `tools/call` envelopes rejected before the SDK handler. Journal records identify themselves as `adapter_observed`; a local stdio client controls its own journal, so it is not authoritative server evidence. Neither layer records keys, token hashes, titles, documents, arguments or raw error messages. Correlation UUIDs link the layers: MCP returns `_meta.agentCallId` (or `error.data.agentCallId` for a malformed tool-call envelope), and Go accepts and returns `X-Agent-Call-ID`. Correlation does not grant access. Unknown identities remain null; callers cannot claim another key ID.

If the initial journal write fails, the tool returns `AUDIT_UNAVAILABLE` without running the operation. If writing the completion fails, `AUDIT_COMPLETION_UNCERTAIN` includes a correlation ID. Check authoritative audit/state before retrying `create_scene`, which can otherwise create duplicates. Idempotent mutations may retry their exact stable request ID and bytes; changing either starts a different operation. Existing action errors remain intact when their audit writes succeed.

### Live diagram workflow (D3 implementation)

Use `list_diagram_types`, `get_diagram_template` and `validate_diagram` to prepare native editable diagrams. Create an empty scene with `create_scene` if needed, then call `generate_diagram` with its `sceneRef`, preset/data, stable `instanceId`, stable `requestId` and decimal `expectedSequence`. New empty scenes start at sequence `"0"`. `dryRun:true` compiles a preview without saving. `get_scene` returns the current durable sequence; write scope does not imply read scope. A write-only key can use a revision supplied by the user.

Mutations return a durable receipt, scene URL, created element IDs and warnings. `patch_elements` changes only the named fields; `delete_elements` creates tombstones. Keep the exact arguments and request ID when retrying an uncertain result. A stale revision requires a fresh read and a deliberate new request; a reused ID with changed data is rejected. `get_scene_activity` returns bounded metadata pages with sequence cursors, never document content or credentials. Export uses its independent scope.

Verification uses real MCP calls, Go authorization/audit, Node private ingress, PostgreSQL and a live WebSocket replica. It covers retry receipts, conflicts, tombstones, preserved human fields, scoped reads/export, activity pages and restart. Browser visual acceptance and deployment remain separate programme gates.

Go server setup for live tools: configure `COLLAB_INTERNAL_URL` to the trusted private Node ingress origin and `COLLAB_INTERNAL_SECRET` to its distinct secret (at least32 bytes). Configure both together; leaving both unset keeps live agent routes unavailable. Neither value belongs in remote MCP clients. Go capability issuance precedes the Node request and releases database locks first. The durable Node receipt/audit is the proof of append; Go metadata records the observed remote operation result, and MCP journals are adapter observations.

## Export a scene

Call `compile_mermaid({"source":"flowchart LR\nA[Start] --> B[Finish]"})` for editable native elements without saving. The supported flowchart/sequence subset is documented in [the diagram catalog](./diagram-catalog.md). Other syntax returns `UNSUPPORTED_NATIVE_DIAGRAM` with a typed preset alternative.

Call `export_scene({"sceneRef":{"workspaceId":"<workspace UUID>","sceneId":"<scene UUID>"},"format":"png"})` for `png`, `svg` or editable `json`. The key needs `scene:export` for that scene; read access alone is insufficient. The response includes an absolute `url`, `id`, `mimeType`, `expiresAt` and a decimal-string `revision` for the exported durable snapshot.

Download the returned URL with the same key:

```sh
curl --fail --header "Authorization: Bearer $WHITEBOARD_AGENT_KEY" "$EXPORT_URL" --output scene.png
```

Do not share the bearer key. A URL without authentication will not download the file. Downloads expire within five minutes and immediately respect key, membership, target or export-scope revocation. Temporary exports disappear when the adapter restarts; create another export if needed. Use `.excalidraw` for JSON and `.svg` for SVG downloads.

Stdio forwards exports to authenticated `POST /mcp/exports` at `WHITEBOARD_PUBLIC_URL`; the HTTP service returns its temporary download URL. When Go is served through the frontend reverse proxy, set both URL variables to that public origin; otherwise keep the separately configured trusted HTTPS API origin. HTTP deployment may set `WHITEBOARD_API_URL` to private Go, but must set `WHITEBOARD_PUBLIC_URL` to the frontend origin serving scene and `/mcp` routes. No internal service secrets are required by clients.

Rendering is bounded to 10 seconds, 50,000 input characters, 500 nodes/1,000 edges, 4,096 pixels per axis and 20 MiB per artifact. PNG/JPEG/GIF/WebP assets are embedded after authorization; unsupported legacy SVG/AVIF, missing images or excessive dimensions fail explicitly. Shrink the scene or split a large diagram before retrying a limit error. A busy renderer asks you to retry after the active job. Generated elements always remain native; raster output is only the explicitly requested PNG export.
