ALTER TABLE workspace_members DROP CONSTRAINT workspace_members_role_check;
UPDATE workspace_members SET role = 'editor' WHERE role = 'member';
ALTER TABLE workspace_members ADD CONSTRAINT workspace_members_role_check CHECK (role IN ('owner','editor','viewer'));
INSERT INTO workspace_members(workspace_id,user_id,role)
SELECT id,owner_id,'owner' FROM workspaces
ON CONFLICT (workspace_id,user_id) DO UPDATE SET role='owner';
ALTER TABLE workspaces ADD COLUMN public_id TEXT UNIQUE CHECK (public_id ~ '^[A-Za-z0-9_-]{21}$');
ALTER TABLE scenes ADD COLUMN public_id TEXT UNIQUE CHECK (public_id ~ '^[A-Za-z0-9_-]{21}$');
ALTER TABLE workspaces ADD COLUMN acl_version BIGINT NOT NULL DEFAULT 1 CHECK (acl_version > 0);
ALTER TABLE scenes ADD CONSTRAINT scenes_workspace_id_id_key UNIQUE(workspace_id,id);
CREATE TABLE scene_member_overrides (
 workspace_id UUID NOT NULL,
 scene_id UUID NOT NULL,
 user_id UUID NOT NULL,
 role TEXT NOT NULL CHECK (role IN ('editor','viewer')),
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 PRIMARY KEY(scene_id,user_id),
 FOREIGN KEY(workspace_id,scene_id) REFERENCES scenes(workspace_id,id) ON DELETE CASCADE,
 FOREIGN KEY(workspace_id,user_id) REFERENCES workspace_members(workspace_id,user_id) ON DELETE CASCADE
);
CREATE INDEX scene_member_overrides_membership ON scene_member_overrides(workspace_id,user_id);
CREATE TABLE acl_audit_events (
 id BIGSERIAL PRIMARY KEY,
 workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 scene_id UUID,
 actor_id UUID,
 subject_id UUID,
 event_type TEXT NOT NULL,
 acl_version BIGINT NOT NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX acl_audit_events_workspace ON acl_audit_events(workspace_id,id DESC);
-- Shared by A2/A3: acquire the workspace lock before mutating a grant, then
-- call this helper in the same transaction. Notifications are commit-bound.
CREATE FUNCTION collab_bump_acl(wid UUID, sid UUID, uid UUID, event TEXT) RETURNS BIGINT LANGUAGE plpgsql AS $$
DECLARE version BIGINT; actor UUID;
BEGIN
 UPDATE workspaces SET acl_version=acl_version+1 WHERE id=wid RETURNING acl_version INTO version;
 IF version IS NOT NULL THEN
  actor := NULLIF(current_setting('app.actor_id',true),'')::uuid;
  INSERT INTO acl_audit_events(workspace_id,scene_id,actor_id,subject_id,event_type,acl_version)
  VALUES(wid,sid,actor,uid,event,version);
  PERFORM pg_notify('acl_changed',json_build_object('workspaceId',wid,'aclVersion',version::text)::text);
 END IF;
 RETURN version;
END $$;
-- All authority writers acquire workspace before changing membership/scene rows.
CREATE FUNCTION collab_acl_changed() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE wid UUID; sid UUID; uid UUID;
BEGIN
 IF TG_TABLE_NAME = 'workspace_members' OR TG_TABLE_NAME = 'scene_member_overrides' THEN
  IF TG_OP = 'DELETE' THEN wid := OLD.workspace_id; uid := OLD.user_id; ELSE wid := NEW.workspace_id; uid := NEW.user_id; END IF;
  IF TG_TABLE_NAME = 'scene_member_overrides' THEN
   IF TG_OP = 'DELETE' THEN sid := OLD.scene_id; ELSE sid := NEW.scene_id; END IF;
  END IF;
 ELSE wid := NEW.workspace_id; sid := NEW.id;
 END IF;
 PERFORM collab_bump_acl(wid,sid,uid,TG_TABLE_NAME || '.' || lower(TG_OP));
 RETURN NULL;
END $$;
CREATE TRIGGER workspace_members_acl AFTER INSERT OR UPDATE OR DELETE ON workspace_members FOR EACH ROW EXECUTE FUNCTION collab_acl_changed();
CREATE TRIGGER scene_member_overrides_acl AFTER INSERT OR UPDATE OR DELETE ON scene_member_overrides FOR EACH ROW EXECUTE FUNCTION collab_acl_changed();
CREATE TRIGGER scene_status_acl AFTER UPDATE OF deleted_at ON scenes FOR EACH ROW WHEN (OLD.deleted_at IS DISTINCT FROM NEW.deleted_at) EXECUTE FUNCTION collab_acl_changed();
CREATE FUNCTION collab_owner_membership() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 INSERT INTO workspace_members(workspace_id,user_id,role) VALUES(NEW.id,NEW.owner_id,'owner')
 ON CONFLICT(workspace_id,user_id) DO UPDATE SET role='owner';
 RETURN NULL;
END $$;
CREATE TRIGGER workspace_owner_membership AFTER INSERT ON workspaces FOR EACH ROW EXECUTE FUNCTION collab_owner_membership();

CREATE FUNCTION collab_resource_status_changed() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE wid UUID;
BEGIN
 IF TG_TABLE_NAME = 'users' THEN
  FOR wid IN SELECT workspace_id FROM workspace_members WHERE user_id=NEW.id ORDER BY workspace_id LOOP
   PERFORM collab_bump_acl(wid,NULL,NEW.id,'users.status');
  END LOOP;
 ELSE
  PERFORM collab_bump_acl(NEW.id,NULL,NULL,'workspaces.status');
 END IF;
 RETURN NULL;
END $$;
CREATE TRIGGER user_status_acl AFTER UPDATE OF deleted_at ON users FOR EACH ROW WHEN (OLD.deleted_at IS DISTINCT FROM NEW.deleted_at) EXECUTE FUNCTION collab_resource_status_changed();
CREATE TRIGGER workspace_status_acl AFTER UPDATE OF deleted_at ON workspaces FOR EACH ROW WHEN (OLD.deleted_at IS DISTINCT FROM NEW.deleted_at) EXECUTE FUNCTION collab_resource_status_changed();
