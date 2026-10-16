# Agent access

Owners can create a workspace-bound bearer key with selected scopes and optional scene targets. Copy the key once; only its SHA-256 digest is stored. Revoking a key or removing/deactivating its owner prevents subsequent requests. Agents never manage members, permissions, sharing, or keys.

Scopes are `workspace:read`, `scene:read`, `scene:create`, `scene:write`, `scene:move`, `scene:trash`, and `scene:export`. Tools require their named scope independently; write does not imply create. A scene-targeted key cannot create arbitrary scenes. Current owner permissions also apply.

MCP exposes `list_workspaces`, `list_scenes`, `get_scene`, and `create_scene`. Lists return at most 50 results and a cursor, without global totals. Scene projection returns at most 500 elements and 256 KiB. Oversized projection is rejected rather than silently truncated.

Use Streamable HTTP `/mcp` with `Authorization: Bearer <key>` in clients that support explicit headers. For stdio clients, run the remote adapter with `WHITEBOARD_API_URL` and `WHITEBOARD_AGENT_KEY` in protected client environment/configuration. It does not start a database. Keep credentials out of arguments, URLs, logs, and source control. This is PAT-compatible access; OAuth discovery and PKCE are a separate extension, not implemented OAuth compliance.

Owner management: GET/POST `/api/v1/workspaces/{id}/agent-keys`, DELETE `/api/v1/workspaces/{id}/agent-keys/{keyId}` using normal login authentication. Creation body: `{ "scopes": ["workspace:read", "scene:read"], "sceneIds": [] }`. Empty sceneIds permits the workspace; an explicit list narrows scene tools. The response `key` is shown once.

Implementation verification: isolated PostgreSQL tests check scope/target intersection, revoked owner/key, hash-only storage and room credential revocation. Real MCP HTTP and stdio tests check initialization, tool discovery, authentication on each call, origin validation, principal-bound sessions and bounded tenant-filtered pagination. Dashboard route and deployment integration are programme follow-ups.


### Remote stdio setup

Build the MCP workspace with `npm run build --workspace ./mcp` from `src/`. Configure your client to launch `node` with the absolute path to `src/mcp/dist/stdio.js`. Set `WHITEBOARD_API_URL` to your Go API origin (for example `https://whiteboard.example`) and `WHITEBOARD_AGENT_KEY` to your copied key in the client's protected environment settings. Remote API origins require HTTPS; loopback HTTP is allowed for local development. The adapter never follows redirects with the key.

For the HTTP service, set `WHITEBOARD_API_URL`, comma-separated exact `MCP_ALLOWED_ORIGINS`, and optionally `PORT` (default 3002). It binds to loopback by default; deployment may set `MCP_BIND_HOST` behind its configured reverse proxy. HTTP sessions expire after ten minutes, and clients initialize a fresh session after a 404. A key is checked again for every HTTP request. Keep normal login tokens and agent keys separate.

Scene creation needs `scene:create`, reading needs `scene:read`, and diagram updates need `scene:write`. Move, trash and export scopes are reserved for their dedicated tools; D1 does not yet expose those tools. A projection error means the scene is too large for this discovery response; it is not a partial or empty scene. Dashboard route integration and deployment remain pending in the collaboration-v2 programme.

D1 `get_scene` returns the Go materialized snapshot and its `contentVersion`. This version is not the collaboration sequence. Live mutation tooling must obtain the current durable projection/sequence from the collaboration service before using a revision precondition; that integration belongs to D3.


### Audit storage and failed calls

For HTTP, set `MCP_AUDIT_PATH` to an absolute path on persistent storage before starting the service. For stdio, the default is `~/.local/state/whiteboard-mcp/audit.jsonl`; set a separate absolute path for each simultaneously running adapter. One process must own each journal path. The adapter creates files with mode 0600 and retains an active file plus two archives, each bounded to 1 MiB. Collect these files before rotation removes older events. This adds synchronous disk writes to each call; unavailable audit storage blocks new operations.

The Go database records authoritative operation metadata and commits successful scene creation together with its audit entry. Adapter journals also cover validation, denial and unavailable-upstream failures, including malformed `tools/call` envelopes rejected before the SDK handler. Journal records identify themselves as `adapter_observed`; a local stdio client controls its own journal, so it is not authoritative server evidence. Neither layer records keys, token hashes, titles, documents, arguments or raw error messages. Correlation UUIDs link the layers: MCP returns `_meta.agentCallId` (or `error.data.agentCallId` for a malformed tool-call envelope), and Go accepts and returns `X-Agent-Call-ID`. Correlation does not grant access. Unknown identities remain null; callers cannot claim another key ID.

If the initial journal write fails, the tool returns `AUDIT_UNAVAILABLE` without running the operation. If writing the completion fails, `AUDIT_COMPLETION_UNCERTAIN` includes a correlation ID. Check authoritative audit/state before retrying `create_scene`, which can otherwise create duplicates. Future idempotent D3 mutations may retry their exact stable request ID and bytes; changing either starts a different operation. Existing action errors remain intact when their audit writes succeed.
