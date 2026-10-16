CREATE TABLE share_links (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 scene_id uuid NOT NULL REFERENCES scenes(id) ON DELETE CASCADE,
 token_hash text NOT NULL UNIQUE CHECK (token_hash ~ '^[0-9a-f]{64}$'),
 revision bigint NOT NULL DEFAULT 1 CHECK (revision>0),
 access text NOT NULL CHECK (access IN ('view','edit')),
 password_hash text NOT NULL DEFAULT '',
 active boolean NOT NULL DEFAULT true,
 expires_at timestamptz,
 legacy_permission_id uuid UNIQUE REFERENCES scene_permissions(id) ON DELETE CASCADE,
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now(),
 FOREIGN KEY (workspace_id,scene_id) REFERENCES scenes(workspace_id,id) ON DELETE CASCADE
);
CREATE INDEX share_links_scene ON share_links(workspace_id,scene_id);
INSERT INTO share_links(id,workspace_id,scene_id,token_hash,access,password_hash,active,legacy_permission_id,created_at,updated_at)
 SELECT id,workspace_id,scene_id,encode(sha256(convert_to(token,'UTF8')),'hex'),'view',COALESCE(password_hash,''),active,id,created_at,updated_at FROM scene_permissions;

CREATE FUNCTION collab_share_link_version() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN NEW.revision:=OLD.revision+1; NEW.updated_at:=clock_timestamp(); RETURN NEW; END $$;
CREATE TRIGGER share_link_version BEFORE UPDATE ON share_links FOR EACH ROW EXECUTE FUNCTION collab_share_link_version();

CREATE FUNCTION collab_share_link_changed() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE w uuid; s uuid;
BEGIN
 w := COALESCE(NEW.workspace_id,OLD.workspace_id); s := COALESCE(NEW.scene_id,OLD.scene_id);
 PERFORM 1 FROM workspaces WHERE id=w FOR UPDATE;
 PERFORM collab_bump_acl(w,s,NULL,'share_link_' || lower(TG_OP));
 IF TG_OP='UPDATE' AND NEW.legacy_permission_id IS NOT NULL AND pg_trigger_depth()=1 THEN
  UPDATE scene_permissions SET active=NEW.active,password_hash=NEW.password_hash,updated_at=now()
   WHERE id=NEW.legacy_permission_id AND (active IS DISTINCT FROM NEW.active OR password_hash IS DISTINCT FROM NEW.password_hash);
 END IF;
 RETURN COALESCE(NEW,OLD);
END $$;
CREATE TRIGGER share_link_acl AFTER INSERT OR UPDATE OR DELETE ON share_links FOR EACH ROW EXECUTE FUNCTION collab_share_link_changed();

CREATE FUNCTION collab_legacy_share_sync() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF pg_trigger_depth()>1 THEN RETURN NEW; END IF;
 PERFORM 1 FROM workspaces WHERE id=NEW.workspace_id FOR UPDATE;
 INSERT INTO share_links(id,workspace_id,scene_id,token_hash,access,password_hash,active,legacy_permission_id)
 VALUES (NEW.id,NEW.workspace_id,NEW.scene_id,encode(sha256(convert_to(NEW.token,'UTF8')),'hex'),'view',COALESCE(NEW.password_hash,''),NEW.active,NEW.id)
 ON CONFLICT (legacy_permission_id) DO UPDATE SET token_hash=EXCLUDED.token_hash,password_hash=EXCLUDED.password_hash,active=EXCLUDED.active,updated_at=now();
 RETURN NEW;
END $$;
CREATE TRIGGER legacy_share_sync AFTER INSERT OR UPDATE ON scene_permissions FOR EACH ROW EXECUTE FUNCTION collab_legacy_share_sync();
