CREATE TABLE agent_operation_audit (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 correlation_id uuid NOT NULL,
 key_id uuid NOT NULL,
 principal_id text NOT NULL,
 workspace_id uuid NOT NULL,
 requested_workspace_id uuid,
 scene_id uuid,
 tool text NOT NULL CHECK (tool IN ('list_workspaces','list_scenes','get_scene','create_scene','generate_diagram','patch_elements','delete_elements','get_scene_activity','list_diagram_types','get_diagram_template','validate_diagram','compile_mermaid','export_scene')),
 scope text NOT NULL,
 outcome text NOT NULL CHECK (outcome IN ('success','denied','invalid','failure')),
 failure_class text NOT NULL DEFAULT '',
 latency_ms bigint NOT NULL CHECK (latency_ms BETWEEN 0 AND 86400000),
 provenance text NOT NULL DEFAULT 'go_operation' CHECK (provenance='go_operation'),
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 CHECK (principal_id='agent:' || key_id::text)
);
CREATE INDEX agent_audit_workspace_created ON agent_operation_audit(workspace_id,created_at);
CREATE INDEX agent_audit_correlation ON agent_operation_audit(correlation_id);
