-- Recreating legacy uniqueness intentionally fails when two scenes share a file ID.
-- Operators must resolve collisions explicitly before rollback; no assets are deleted.
CREATE UNIQUE INDEX scene_assets_workspace_file_active_key ON scene_assets(workspace_id,file_id) WHERE asset_kind = 'excalidraw_file' AND deleted_at IS NULL AND file_id IS NOT NULL;
DROP INDEX scene_assets_workspace_scene_file_active_key;
ALTER TABLE scene_assets DROP CONSTRAINT scene_assets_principal_check, DROP COLUMN uploaded_principal_id, DROP COLUMN uploaded_principal_type;
