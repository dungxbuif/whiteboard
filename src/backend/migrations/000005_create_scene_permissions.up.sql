CREATE TABLE scene_permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    scene_id UUID NOT NULL REFERENCES scenes(id) ON DELETE CASCADE,
    token TEXT NOT NULL UNIQUE,
    active BOOLEAN NOT NULL DEFAULT true,
    password_hash TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (workspace_id, scene_id)
);

CREATE INDEX idx_scene_permissions_workspace_scene ON scene_permissions (workspace_id, scene_id);
CREATE INDEX idx_scene_permissions_active ON scene_permissions (token) WHERE active = true;
