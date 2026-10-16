ALTER TABLE scene_assets ADD COLUMN uploaded_principal_id text, ADD COLUMN uploaded_principal_type text;
UPDATE scene_assets SET uploaded_principal_id=uploaded_by::text, uploaded_principal_type='user' WHERE uploaded_by IS NOT NULL;
ALTER TABLE scene_assets ADD CONSTRAINT scene_assets_principal_check CHECK (
 (uploaded_principal_id IS NULL AND uploaded_principal_type IS NULL) OR
 (uploaded_principal_id IS NOT NULL AND uploaded_principal_type IS NOT NULL AND
 ((uploaded_principal_type='user' AND (uploaded_by IS NULL OR uploaded_principal_id=uploaded_by::text)) OR
 (uploaded_principal_type='public' AND uploaded_by IS NULL AND uploaded_principal_id LIKE 'guest:%' AND length(uploaded_principal_id)>6))));
DROP INDEX scene_assets_workspace_file_active_key;
CREATE UNIQUE INDEX scene_assets_workspace_scene_file_active_key ON scene_assets(workspace_id,scene_id,file_id) WHERE asset_kind = 'excalidraw_file' AND deleted_at IS NULL AND file_id IS NOT NULL;
