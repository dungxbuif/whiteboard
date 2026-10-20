CREATE TABLE presentation_links (
  id uuid PRIMARY KEY,
  workspace_id uuid NOT NULL,
  scene_id uuid NOT NULL,
  session_id uuid NOT NULL,
  token_hash text NOT NULL UNIQUE CHECK (token_hash ~ '^[a-f0-9]{64}$'),
  created_by uuid NOT NULL REFERENCES users(id),
  revision bigint NOT NULL DEFAULT 1 CHECK (revision > 0),
  active boolean NOT NULL DEFAULT true,
  expires_at timestamptz NOT NULL,
  revoked_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
  CONSTRAINT presentation_links_scene_fk FOREIGN KEY (workspace_id, scene_id)
    REFERENCES scenes(workspace_id, id) ON DELETE CASCADE,
  CHECK (expires_at <= created_at + interval '24 hours 1 minute'),
  CHECK ((active AND revoked_at IS NULL) OR (NOT active AND revoked_at IS NOT NULL))
);
CREATE INDEX presentation_links_scene_idx ON presentation_links(workspace_id, scene_id, created_at, id);
CREATE INDEX presentation_links_session_idx ON presentation_links(session_id) WHERE active;
