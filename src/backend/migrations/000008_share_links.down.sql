DO $$ BEGIN
 IF EXISTS (SELECT 1 FROM share_links GROUP BY scene_id HAVING count(*)>1)
 OR EXISTS (SELECT 1 FROM share_links WHERE legacy_permission_id IS NULL OR access<>'view' OR expires_at IS NOT NULL) THEN
  RAISE EXCEPTION 'share_links rollback would discard independent grants; retain schema or restore a verified backup';
 END IF;
END $$;
DROP TRIGGER legacy_share_sync ON scene_permissions;
DROP FUNCTION collab_legacy_share_sync();
DROP TRIGGER share_link_acl ON share_links;
DROP FUNCTION collab_share_link_changed();
DROP TRIGGER share_link_version ON share_links;
DROP FUNCTION collab_share_link_version();
DROP TABLE share_links;
