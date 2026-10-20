DROP TRIGGER IF EXISTS scene_versions_immutable ON scene_versions;
DROP FUNCTION IF EXISTS reject_scene_version_content_update();
DROP TABLE IF EXISTS version_assets;
DROP TABLE IF EXISTS scene_versions;
ALTER TABLE scene_assets DROP CONSTRAINT IF EXISTS scene_assets_workspace_scene_id_key;
