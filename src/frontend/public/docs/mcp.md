# MCP live scene tools

Status: collaboration v2 implementation; deployment pending.

- Scopes are independent: `workspace:read`, `scene:read`, `scene:create`, `scene:write`, `scene:move`, `scene:trash`, `scene:export`.
- Configure `WHITEBOARD_API_URL`, `WHITEBOARD_AGENT_KEY`, and a unique absolute `MCP_AUDIT_PATH`. Never give remote adapters internal service secrets.
- Discover with `list_diagram_types`, `get_diagram_template({preset})`, `validate_diagram({preset,data,instanceId,...})`. Catalog/schema resources provide structured inputs.
- `generate_diagram({sceneRef:{workspaceId,sceneId},preset,data,instanceId,requestId,expectedSequence,dryRun})` compiles native elements and submits one durable mutation, or returns a preview when dryRun is true.
- `patch_elements({sceneRef,patches:[{id,fields}],requestId,expectedSequence})` patches targeted fields.
- `delete_elements({sceneRef,ids,requestId,expectedSequence})` writes tombstones.
- `get_scene_activity({sceneRef,cursor?,limit?})` returns at most50 metadata events.
- `get_scene({workspaceId,sceneId})` returns current durable `sequence`, content and document identity; maximum500 elements/256KiB.
- `expectedSequence` is a decimal string, not contentVersion. New empty scenes start at `"0"`. A write-only key can write with a supplied sequence; it cannot read content/activity.
- Successful writes return `{receipt:{requestId,sequence,stateVector},sceneUrl,createdIds,warnings}`. Keep identical request IDs and bytes on uncertain retries. Conflict requires a fresh read and deliberate new request; never blindly replace a whole scene.
- Dry-run compiles without changing revision. No permission/key/share-management tools are exposed.

Read the human agent guide for setup, scopes, audit retention and failure handling. Real MCP → Go → Node → PostgreSQL integration, live socket convergence and restart are verified in D3. Rendered browser/deployment verification remains a release gate.
