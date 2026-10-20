ALTER TABLE scene_assets
    ADD CONSTRAINT scene_assets_workspace_scene_id_key UNIQUE (workspace_id, scene_id, id);

CREATE TABLE scene_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    scene_id UUID NOT NULL,
    document_id UUID NOT NULL,
    source_sequence BIGINT NOT NULL CHECK (source_sequence >= 0),
    content JSONB NOT NULL CHECK (jsonb_typeof(content) = 'object'),
    content_bytes BYTEA NOT NULL CHECK (octet_length(content_bytes) <= 26214400),
    content_hash TEXT NOT NULL CHECK (content_hash ~ '^[a-f0-9]{64}$'),
    byte_size BIGINT NOT NULL CHECK (byte_size >= 0 AND byte_size <= 26214400),
    label TEXT CHECK (label IS NULL OR char_length(label) BETWEEN 1 AND 255),
    pinned BOOLEAN NOT NULL DEFAULT FALSE,
    metadata_revision BIGINT NOT NULL DEFAULT 1 CHECK (metadata_revision >= 1),
    kind TEXT NOT NULL CHECK (kind IN ('manual', 'automatic')),
    creator_kind TEXT NOT NULL CHECK (creator_kind IN ('user', 'system')),
    creator_id TEXT NOT NULL CHECK (char_length(creator_id) BETWEEN 1 AND 256),
    request_id TEXT NOT NULL CHECK (char_length(request_id) BETWEEN 1 AND 128),
    payload_hash TEXT NOT NULL CHECK (payload_hash ~ '^[a-f0-9]{64}$'),
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    FOREIGN KEY (workspace_id, scene_id) REFERENCES scenes(workspace_id, id) ON DELETE CASCADE,
    UNIQUE (scene_id, request_id),
    UNIQUE (workspace_id, scene_id, id),
    CHECK (byte_size = octet_length(content_bytes)),
    CHECK (convert_from(content_bytes, 'UTF8')::jsonb = content)
);
CREATE INDEX scene_versions_scene_created ON scene_versions(scene_id, created_at DESC, id DESC);
CREATE INDEX scene_versions_automatic ON scene_versions(scene_id, created_at DESC) WHERE kind = 'automatic';

CREATE TABLE version_assets (
    version_id UUID NOT NULL,
    workspace_id UUID NOT NULL,
    scene_id UUID NOT NULL,
    asset_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY (version_id, asset_id),
    FOREIGN KEY (workspace_id, scene_id, version_id) REFERENCES scene_versions(workspace_id, scene_id, id) ON DELETE CASCADE,
    FOREIGN KEY (workspace_id, scene_id, asset_id) REFERENCES scene_assets(workspace_id, scene_id, id) ON DELETE RESTRICT
);
CREATE INDEX version_assets_asset ON version_assets(asset_id);

CREATE FUNCTION reject_scene_version_content_update() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.workspace_id IS DISTINCT FROM OLD.workspace_id
       OR NEW.scene_id IS DISTINCT FROM OLD.scene_id
       OR NEW.document_id IS DISTINCT FROM OLD.document_id
       OR NEW.source_sequence IS DISTINCT FROM OLD.source_sequence
       OR NEW.content IS DISTINCT FROM OLD.content
       OR NEW.content_bytes IS DISTINCT FROM OLD.content_bytes
       OR NEW.content_hash IS DISTINCT FROM OLD.content_hash
       OR NEW.byte_size IS DISTINCT FROM OLD.byte_size
       OR NEW.kind IS DISTINCT FROM OLD.kind
       OR NEW.creator_kind IS DISTINCT FROM OLD.creator_kind
       OR NEW.creator_id IS DISTINCT FROM OLD.creator_id
       OR NEW.request_id IS DISTINCT FROM OLD.request_id
       OR NEW.payload_hash IS DISTINCT FROM OLD.payload_hash
       OR NEW.created_at IS DISTINCT FROM OLD.created_at THEN
        RAISE EXCEPTION 'scene version content is immutable' USING ERRCODE = '23000';
    END IF;
    NEW.updated_at := clock_timestamp();
    RETURN NEW;
END $$;
CREATE TRIGGER scene_versions_immutable BEFORE UPDATE ON scene_versions
FOR EACH ROW EXECUTE FUNCTION reject_scene_version_content_update();
