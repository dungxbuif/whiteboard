CREATE TABLE collections (
  id uuid NOT NULL DEFAULT gen_random_uuid(),
  workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  creator_id uuid NOT NULL,
  name text NOT NULL CHECK (char_length(name) BETWEEN 1 AND 100 AND name=btrim(name)),
  visibility text NOT NULL CHECK (visibility IN ('workspace','private')),
  revision bigint NOT NULL DEFAULT 1 CHECK (revision > 0),
  created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
  updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
  PRIMARY KEY (workspace_id,id),
  FOREIGN KEY (workspace_id,creator_id) REFERENCES workspace_members(workspace_id,user_id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX collections_workspace_name_key
  ON collections(workspace_id,lower(name)) WHERE visibility='workspace';
CREATE UNIQUE INDEX collections_private_creator_name_key
  ON collections(workspace_id,creator_id,lower(name)) WHERE visibility='private';
CREATE INDEX collections_workspace_page
  ON collections(workspace_id,created_at DESC,id DESC);

CREATE TABLE collection_scenes (
  workspace_id uuid NOT NULL,
  collection_id uuid NOT NULL,
  scene_id uuid NOT NULL,
  position integer NOT NULL CHECK (position >= 0),
  created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
  PRIMARY KEY (workspace_id,collection_id,scene_id),
  UNIQUE (workspace_id,collection_id,position),
  FOREIGN KEY (workspace_id,collection_id) REFERENCES collections(workspace_id,id) ON DELETE CASCADE,
  FOREIGN KEY (workspace_id,scene_id) REFERENCES scenes(workspace_id,id) ON DELETE CASCADE
);

CREATE INDEX collection_scenes_scene
  ON collection_scenes(workspace_id,scene_id,collection_id);
