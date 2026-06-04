CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT NOT NULL,
    name TEXT NOT NULL,
    avatar_url TEXT,
    google_subject TEXT,
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX users_email_lower_key ON users (lower(email));
CREATE UNIQUE INDEX users_google_subject_key ON users (google_subject)
    WHERE google_subject IS NOT NULL;

CREATE TABLE auth_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token_hash TEXT NOT NULL UNIQUE,
    user_agent TEXT,
    ip_address INET,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_auth_sessions_user_active
    ON auth_sessions (user_id, revoked_at, expires_at);

CREATE TABLE workspaces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_workspaces_owner_id ON workspaces (owner_id);
CREATE UNIQUE INDEX workspaces_owner_name_active_key
    ON workspaces (owner_id, lower(name))
    WHERE deleted_at IS NULL;

CREATE TABLE workspace_members (
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (workspace_id, user_id),
    CONSTRAINT workspace_members_role_check CHECK (role IN ('owner', 'member'))
);

CREATE INDEX idx_workspace_members_user_id ON workspace_members (user_id);

CREATE TABLE folders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    parent_id UUID REFERENCES folders(id) ON DELETE SET NULL,
    name TEXT NOT NULL,
    path TEXT NOT NULL,
    depth INTEGER NOT NULL DEFAULT 0,
    is_starred BOOLEAN NOT NULL DEFAULT false,
    deleted_at TIMESTAMPTZ,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_folders_workspace_id ON folders (workspace_id);
CREATE INDEX idx_folders_workspace_parent ON folders (workspace_id, parent_id);
CREATE INDEX idx_folders_workspace_path ON folders (workspace_id, path text_pattern_ops);
CREATE INDEX idx_folders_workspace_starred
    ON folders (workspace_id, is_starred)
    WHERE deleted_at IS NULL;
CREATE INDEX idx_folders_workspace_deleted
    ON folders (workspace_id, deleted_at)
    WHERE deleted_at IS NOT NULL;
CREATE UNIQUE INDEX folders_workspace_parent_name_active_key
    ON folders (workspace_id, COALESCE(parent_id, '00000000-0000-0000-0000-000000000000'::uuid), lower(name))
    WHERE deleted_at IS NULL;
CREATE INDEX idx_folders_workspace_name_trgm
    ON folders USING gin (lower(name) gin_trgm_ops);

CREATE TABLE scenes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    folder_id UUID REFERENCES folders(id) ON DELETE SET NULL,
    title TEXT NOT NULL,
    content JSONB NOT NULL DEFAULT '{}'::jsonb,
    content_version BIGINT NOT NULL DEFAULT 1,
    is_starred BOOLEAN NOT NULL DEFAULT false,
    deleted_at TIMESTAMPTZ,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    updated_by UUID REFERENCES users(id) ON DELETE SET NULL,
    last_opened_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_scenes_workspace_id ON scenes (workspace_id);
CREATE INDEX idx_scenes_workspace_folder_id ON scenes (workspace_id, folder_id);
CREATE INDEX idx_scenes_workspace_updated ON scenes (workspace_id, updated_at DESC);
CREATE INDEX idx_scenes_workspace_starred
    ON scenes (workspace_id, is_starred)
    WHERE deleted_at IS NULL;
CREATE INDEX idx_scenes_workspace_deleted
    ON scenes (workspace_id, deleted_at)
    WHERE deleted_at IS NOT NULL;
CREATE UNIQUE INDEX scenes_workspace_folder_title_active_key
    ON scenes (workspace_id, COALESCE(folder_id, '00000000-0000-0000-0000-000000000000'::uuid), lower(title))
    WHERE deleted_at IS NULL;
CREATE INDEX idx_scenes_workspace_title_trgm
    ON scenes USING gin (lower(title) gin_trgm_ops);

CREATE TABLE scene_assets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    scene_id UUID REFERENCES scenes(id) ON DELETE SET NULL,
    uploaded_by UUID REFERENCES users(id) ON DELETE SET NULL,
    asset_kind TEXT NOT NULL,
    file_id TEXT,
    storage_bucket TEXT NOT NULL,
    storage_key TEXT NOT NULL,
    mime_type TEXT NOT NULL,
    byte_size BIGINT NOT NULL CHECK (byte_size >= 0),
    sha256 TEXT,
    width INTEGER CHECK (width IS NULL OR width >= 0),
    height INTEGER CHECK (height IS NULL OR height >= 0),
    deleted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT scene_assets_asset_kind_check CHECK (asset_kind IN ('excalidraw_file', 'scene_thumbnail'))
);

CREATE UNIQUE INDEX scene_assets_bucket_key_key
    ON scene_assets (storage_bucket, storage_key);
CREATE INDEX idx_scene_assets_workspace_scene
    ON scene_assets (workspace_id, scene_id);
CREATE INDEX idx_scene_assets_workspace_deleted
    ON scene_assets (workspace_id, deleted_at);
CREATE UNIQUE INDEX scene_assets_workspace_file_active_key
    ON scene_assets (workspace_id, file_id)
    WHERE asset_kind = 'excalidraw_file'
      AND deleted_at IS NULL
      AND file_id IS NOT NULL;
