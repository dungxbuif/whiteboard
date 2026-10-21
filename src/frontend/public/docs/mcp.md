# MCP live scene tools

Status: collaboration v2 is deployed on VM100. Public MCP authentication,
discovery, workspace/scene reads and revoked-key rejection pass; native
production write/export remains a final release canary.

## Personal agent quick start

The production frontend exposes Streamable HTTP at `/mcp`. Create a scoped
workspace key with the normal authenticated API, copy the one-time `key` field,
and configure it only in the agent's protected environment:

```sh
export WHITEBOARD_ORIGIN=https://whiteboard.dungxbuif.com
export WHITEBOARD_LOGIN_TOKEN='...'
export WORKSPACE_ID='...'
curl --fail -H "Authorization: Bearer $WHITEBOARD_LOGIN_TOKEN" \
  -H 'Content-Type: application/json' -X POST \
  "$WHITEBOARD_ORIGIN/api/v1/workspaces/$WORKSPACE_ID/agent-keys" \
  -d '{"scopes":["workspace:read","scene:read","scene:create","scene:write","scene:export"],"sceneIds":[]}'
```

Set the returned value as `WHITEBOARD_AGENT_KEY` and connect to
`$WHITEBOARD_ORIGIN/mcp`. Keys are workspace-scoped bearer credentials, are
stored server-side only as SHA-256 digests, and are shown once at creation.
Do not put a key in a URL, source file, prompt, or logs.

- Scopes are independent: `workspace:read`, `scene:read`, `scene:create`, `scene:write`, `scene:move`, `scene:trash`, `scene:export`.
- Build from `src/` in order: `npm run build --workspace ./packages/collab-contracts`, `npm run build --workspace ./packages/diagram-kit`, then `npm run build --workspace ./mcp` (includes export renderer).
- Configure `WHITEBOARD_API_URL` for Go, `WHITEBOARD_PUBLIC_URL` for the public frontend/reverse-proxy serving `/w/...` and `/mcp/...`, `WHITEBOARD_AGENT_KEY`, and a unique absolute `MCP_AUDIT_PATH`. Both origins require HTTPS remotely (loopback HTTP is allowed), with no credentials, path, query or fragment. Scene/artifact links and remote exports use the public origin; it does not fall back to the API origin. Never give remote adapters internal service secrets.
- Discover with `list_diagram_types`, `get_diagram_template({preset})`, `validate_diagram({preset,data,instanceId,...})`. Catalog/schema resources provide structured inputs.
- `generate_diagram({sceneRef:{workspaceId,sceneId},preset,data,instanceId,requestId,expectedSequence,dryRun})` compiles native elements and submits one durable mutation, or returns a preview when dryRun is true.
- `patch_elements({sceneRef,patches:[{id,fields}],requestId,expectedSequence})` patches targeted fields.
- `delete_elements({sceneRef,ids,requestId,expectedSequence})` writes tombstones.
- `get_scene_activity({sceneRef,cursor?,limit?})` returns at most50 metadata events.
- `get_scene({workspaceId,sceneId})` returns current durable `sequence`, content and document identity; maximum500 elements/256KiB.
- `expectedSequence` is a decimal string, not contentVersion. New empty scenes start at `"0"`. A write-only key can write with a supplied sequence; it cannot read content/activity.
- Successful writes return `{receipt:{requestId,sequence,stateVector},sceneUrl,createdIds,warnings}`. Keep identical request IDs and bytes on uncertain retries. Conflict requires a fresh read and deliberate new request; never blindly replace a whole scene.
- Dry-run compiles without changing revision. No permission/key/share-management tools are exposed.

Read the human agent guide for setup, scopes, audit retention and failure
handling. Real MCP → Go → Node → PostgreSQL integration, live socket
convergence, restart, native browser IDs, rendered export and revoked downloads
pass locally. VM100 authentication/read/revocation verification also passes;
native production write/export remains a release gate.

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

Render timeouts terminate the job's browser and worker. The adapter removes temporary browser files before admitting another job. If cleanup itself fails, that renderer remains unavailable until the service operator restarts it; do not keep retrying a cleanup failure.
