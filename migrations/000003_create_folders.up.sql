CREATE TABLE folders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    parent_id UUID,
    name TEXT NOT NULL,
    path TEXT NOT NULL,
    depth INT NOT NULL DEFAULT 0,
    is_starred BOOLEAN NOT NULL DEFAULT false,
    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_folders_workspace_id ON folders (workspace_id);
CREATE INDEX idx_folders_parent_id ON folders (parent_id);
CREATE INDEX idx_folders_path ON folders (path);
CREATE INDEX idx_folders_deleted_at ON folders (deleted_at);

-- Foreign key constraints (optional but recommended)
-- ALTER TABLE folders ADD CONSTRAINT fk_folders_workspace FOREIGN KEY (workspace_id) REFERENCES workspaces(id);
-- ALTER TABLE folders ADD CONSTRAINT fk_folders_parent FOREIGN KEY (parent_id) REFERENCES folders(id);
