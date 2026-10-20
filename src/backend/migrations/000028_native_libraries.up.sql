CREATE TABLE library_items (
    id UUID PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    name TEXT NOT NULL CHECK (char_length(name) BETWEEN 1 AND 100),
    current_revision BIGINT NOT NULL CHECK (current_revision >= 1),
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    UNIQUE (workspace_id, id)
);
CREATE INDEX library_items_workspace_updated ON library_items(workspace_id, updated_at DESC, id DESC);
CREATE UNIQUE INDEX library_items_workspace_name ON library_items(workspace_id, lower(name));

CREATE TABLE library_revisions (
    library_item_id UUID NOT NULL,
    workspace_id UUID NOT NULL,
    revision BIGINT NOT NULL CHECK (revision >= 1),
    content JSONB NOT NULL CHECK (jsonb_typeof(content) = 'object'),
    content_bytes BYTEA NOT NULL CHECK (octet_length(content_bytes) <= 26214400),
    content_hash TEXT NOT NULL CHECK (content_hash ~ '^[a-f0-9]{64}$'),
    byte_size BIGINT NOT NULL CHECK (byte_size = octet_length(content_bytes) AND byte_size <= 26214400),
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY (library_item_id, revision),
    FOREIGN KEY (workspace_id, library_item_id) REFERENCES library_items(workspace_id, id) ON DELETE CASCADE,
    CHECK (convert_from(content_bytes, 'UTF8')::jsonb = content)
);

CREATE TABLE library_assets (
    library_item_id UUID NOT NULL,
    revision BIGINT NOT NULL,
    workspace_id UUID NOT NULL,
    source_scene_id UUID NOT NULL,
    asset_id UUID NOT NULL,
    file_id TEXT NOT NULL CHECK (char_length(file_id) BETWEEN 1 AND 256),
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY (library_item_id, revision, file_id),
    FOREIGN KEY (library_item_id, revision) REFERENCES library_revisions(library_item_id, revision) ON DELETE CASCADE,
    FOREIGN KEY (workspace_id, source_scene_id, asset_id) REFERENCES scene_assets(workspace_id, scene_id, id) ON DELETE RESTRICT
);
CREATE INDEX library_assets_asset_pin ON library_assets(asset_id);

CREATE TABLE library_asset_stages (
    workspace_id UUID NOT NULL,
    target_scene_id UUID NOT NULL,
    request_id UUID NOT NULL,
    source_asset_id UUID NOT NULL,
    source_file_id TEXT NOT NULL,
    target_asset_id UUID NOT NULL,
    target_file_id TEXT NOT NULL,
    storage_bucket TEXT NOT NULL,
    storage_key TEXT NOT NULL,
    state TEXT NOT NULL CHECK (state IN ('pending','ready')),
    cleanup_after TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY (workspace_id, target_scene_id, request_id, source_file_id),
    UNIQUE (target_asset_id),
    FOREIGN KEY (workspace_id, target_scene_id) REFERENCES scenes(workspace_id, id) ON DELETE CASCADE
);
CREATE INDEX library_asset_stages_cleanup ON library_asset_stages(cleanup_after);

CREATE FUNCTION reject_library_revision_update() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'library revisions are immutable' USING ERRCODE = '23000';
END $$;
CREATE TRIGGER library_revisions_immutable BEFORE UPDATE ON library_revisions
FOR EACH ROW EXECUTE FUNCTION reject_library_revision_update();

ALTER TABLE collab_receipts DROP CONSTRAINT collab_receipts_action_check;
ALTER TABLE collab_receipts ADD CONSTRAINT collab_receipts_action_check
  CHECK(action IN ('scene:update','scene:command','version:restore','library:insert'));

CREATE FUNCTION enforce_library_current_revision() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM library_revisions r
        WHERE r.workspace_id = NEW.workspace_id AND r.library_item_id = NEW.id AND r.revision = NEW.current_revision
    ) THEN
        RAISE EXCEPTION 'library current revision missing' USING ERRCODE = '23503';
    END IF;
    RETURN NEW;
END $$;
CREATE CONSTRAINT TRIGGER library_current_revision_exists
AFTER INSERT OR UPDATE OF current_revision ON library_items DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW EXECUTE FUNCTION enforce_library_current_revision();
