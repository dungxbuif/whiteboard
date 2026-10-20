DROP TRIGGER IF EXISTS library_current_revision_exists ON library_items;
DROP FUNCTION IF EXISTS enforce_library_current_revision();
DROP TRIGGER IF EXISTS library_revisions_immutable ON library_revisions;
DROP FUNCTION IF EXISTS reject_library_revision_update();
DROP TABLE IF EXISTS library_asset_stages;
DROP TABLE IF EXISTS library_assets;
DROP TABLE IF EXISTS library_revisions;
DROP TABLE IF EXISTS library_items;
ALTER TABLE collab_receipts DROP CONSTRAINT collab_receipts_action_check;
ALTER TABLE collab_receipts ADD CONSTRAINT collab_receipts_action_check
  CHECK(action IN ('scene:update','scene:command','version:restore'));
