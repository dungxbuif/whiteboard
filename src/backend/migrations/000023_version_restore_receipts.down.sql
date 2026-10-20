DO $$ BEGIN
  IF EXISTS(SELECT 1 FROM collab_receipts WHERE action='version:restore') THEN
    RAISE EXCEPTION 'version restore receipts exist; preserve migration 000023 or restore a verified backup';
  END IF;
END $$;
ALTER TABLE collab_receipts DROP CONSTRAINT collab_receipts_action_check;
ALTER TABLE collab_receipts ADD CONSTRAINT collab_receipts_action_check
  CHECK(action IN ('scene:update','scene:command'));
