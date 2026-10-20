ALTER TABLE scenes
    ADD COLUMN pending_duplicate boolean NOT NULL DEFAULT false,
    ADD COLUMN duplicate_cleanup_started boolean NOT NULL DEFAULT false;

-- Copies staged by the prior build already have a durable target reference.
UPDATE scenes AS target
SET pending_duplicate = TRUE
WHERE target.deleted_at IS NOT NULL
  AND target.writer_mode = 'v1'
  AND target.content_version = 1
  AND target.content = '{"elements":[],"appState":{},"files":{}}'::jsonb
  AND EXISTS (
      SELECT 1 FROM library_asset_stages AS stage
      WHERE stage.workspace_id = target.workspace_id AND stage.target_scene_id = target.id
  );

-- A copy can be between hidden-scene creation and its first asset stage.
-- Refuse ambiguous near-creation soft deletes, including abandoned old copies;
-- operators must drain the old writer and classify these rows before upgrade.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM scenes
        WHERE deleted_at IS NOT NULL
          AND pending_duplicate = FALSE
          AND content_version = 1
          AND writer_mode = 'v1'
          AND content = '{"elements":[],"appState":{},"files":{}}'::jsonb
          AND ABS(EXTRACT(EPOCH FROM (deleted_at - created_at))) <= 60
    ) THEN
        RAISE EXCEPTION 'Ambiguous hidden empty scenes require old-writer drain and classification before pending-duplicate migration';
    END IF;
END $$;
