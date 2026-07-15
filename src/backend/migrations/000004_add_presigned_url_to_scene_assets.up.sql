ALTER TABLE scene_assets ADD COLUMN presigned_url TEXT;
ALTER TABLE scene_assets ADD COLUMN url_expires_at TIMESTAMPTZ;
