ALTER TABLE collab_receipts DROP CONSTRAINT collab_receipts_action_check;
ALTER TABLE collab_receipts ADD CONSTRAINT collab_receipts_action_check
  CHECK(action IN ('scene:update','scene:command','version:restore'));
