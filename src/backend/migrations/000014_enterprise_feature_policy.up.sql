ALTER TABLE share_links
  ADD COLUMN allow_comments boolean NOT NULL DEFAULT false;

CREATE TABLE workspace_feature_policies (
  workspace_id uuid PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,
  revision bigint NOT NULL DEFAULT 1 CHECK (revision > 0),
  member_comments boolean NOT NULL DEFAULT true,
  member_history boolean NOT NULL DEFAULT true,
  comments_enabled boolean NOT NULL DEFAULT false,
  activity_enabled boolean NOT NULL DEFAULT false,
  versions_enabled boolean NOT NULL DEFAULT false,
  libraries_enabled boolean NOT NULL DEFAULT false,
  collections_enabled boolean NOT NULL DEFAULT false,
  presentation_enabled boolean NOT NULL DEFAULT false,
  screen_share_enabled boolean NOT NULL DEFAULT false,
  enterprise_identity_enabled boolean NOT NULL DEFAULT false,
  retention_enabled boolean NOT NULL DEFAULT false,
  updated_at timestamptz NOT NULL DEFAULT now()
);

INSERT INTO workspace_feature_policies(workspace_id)
SELECT id FROM workspaces
ON CONFLICT (workspace_id) DO NOTHING;

CREATE FUNCTION enterprise_feature_policy_version() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  NEW.revision := OLD.revision + 1;
  NEW.updated_at := clock_timestamp();
  RETURN NEW;
END $$;

CREATE TRIGGER enterprise_feature_policy_version
  BEFORE UPDATE ON workspace_feature_policies
  FOR EACH ROW EXECUTE FUNCTION enterprise_feature_policy_version();

CREATE FUNCTION enterprise_workspace_policy_create() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  INSERT INTO workspace_feature_policies(workspace_id) VALUES (NEW.id)
  ON CONFLICT (workspace_id) DO NOTHING;
  RETURN NEW;
END $$;

CREATE TRIGGER enterprise_workspace_policy_create
  AFTER INSERT ON workspaces
  FOR EACH ROW EXECUTE FUNCTION enterprise_workspace_policy_create();

CREATE TABLE feature_mutation_receipts (
  workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  request_id uuid NOT NULL,
  scene_id uuid,
  action text NOT NULL CHECK (action IN (
    'comment:read','comment:write','comment:moderate','activity:read',
    'version:read','version:create','version:restore',
    'library:read','library:write','collection:manage',
    'presentation:host','screen:publish','enterprise:manage'
  )),
  principal_kind text NOT NULL CHECK (principal_kind IN ('user','public','agent')),
  principal_id text NOT NULL CHECK (length(principal_id) BETWEEN 1 AND 255),
  payload_hash text NOT NULL CHECK (payload_hash ~ '^[0-9a-f]{64}$'),
  response_status integer NOT NULL CHECK (response_status BETWEEN 100 AND 599),
  response_body bytea NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (workspace_id, request_id),
  FOREIGN KEY (workspace_id, scene_id)
    REFERENCES scenes(workspace_id, id) ON DELETE CASCADE
);

CREATE INDEX feature_mutation_receipts_scene
  ON feature_mutation_receipts(workspace_id, scene_id, created_at);
