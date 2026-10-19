CREATE TABLE agent_keys (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 owner_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 token_hash text NOT NULL UNIQUE CHECK(token_hash ~ '^[0-9a-f]{64}$'),
 scopes jsonb NOT NULL CHECK(jsonb_typeof(scopes)='array'), scene_ids jsonb NOT NULL DEFAULT '[]' CHECK(jsonb_typeof(scene_ids)='array'),
 created_at timestamptz NOT NULL DEFAULT now(), revoked_at timestamptz
);
CREATE INDEX agent_keys_workspace ON agent_keys(workspace_id);
CREATE FUNCTION collab_agent_key_changed() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE w uuid; k uuid;
BEGIN
 w:=COALESCE(NEW.workspace_id,OLD.workspace_id); k:=COALESCE(NEW.id,OLD.id);
 PERFORM 1 FROM workspaces WHERE id=w FOR UPDATE;
 PERFORM collab_bump_acl(w,NULL,k,'agent_key_' || lower(TG_OP));
 RETURN COALESCE(NEW,OLD);
END $$;
CREATE TRIGGER agent_key_acl AFTER INSERT OR UPDATE OR DELETE ON agent_keys FOR EACH ROW EXECUTE FUNCTION collab_agent_key_changed();
