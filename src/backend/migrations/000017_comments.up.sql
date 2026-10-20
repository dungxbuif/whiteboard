CREATE TABLE comment_threads (
  id uuid NOT NULL DEFAULT gen_random_uuid(),
  workspace_id uuid NOT NULL,
  scene_id uuid NOT NULL,
  creator_kind text NOT NULL CHECK (creator_kind IN ('user','public')),
  creator_id text NOT NULL CHECK (length(creator_id) BETWEEN 1 AND 255),
  creator_name text NOT NULL CHECK (length(creator_name) BETWEEN 1 AND 255),
  element_id text CHECK (element_id IS NULL OR length(element_id) BETWEEN 1 AND 255),
  anchor_x double precision NOT NULL,
  anchor_y double precision NOT NULL,
  status text NOT NULL DEFAULT 'open' CHECK (status IN ('open','resolved')),
  revision bigint NOT NULL DEFAULT 1 CHECK (revision > 0),
  resolved_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
  updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
  PRIMARY KEY (workspace_id, scene_id, id),
  FOREIGN KEY (workspace_id, scene_id) REFERENCES scenes(workspace_id,id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX comment_threads_id_key ON comment_threads(id);
CREATE INDEX comment_threads_scene_page ON comment_threads(workspace_id,scene_id,created_at DESC,id DESC);

CREATE TABLE comments (
  id uuid NOT NULL DEFAULT gen_random_uuid(),
  workspace_id uuid NOT NULL,
  scene_id uuid NOT NULL,
  thread_id uuid NOT NULL,
  author_kind text NOT NULL CHECK (author_kind IN ('user','public')),
  author_id text NOT NULL CHECK (length(author_id) BETWEEN 1 AND 255),
  author_name text NOT NULL CHECK (length(author_name) BETWEEN 1 AND 255),
  body text NOT NULL CHECK ((deleted_at IS NOT NULL AND body='') OR (char_length(body) BETWEEN 1 AND 4000)),
  revision bigint NOT NULL DEFAULT 1 CHECK (revision > 0),
  edited_at timestamptz,
  deleted_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
  updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
  PRIMARY KEY (workspace_id, scene_id, thread_id, id),
  UNIQUE (id),
  FOREIGN KEY (workspace_id,scene_id,thread_id)
    REFERENCES comment_threads(workspace_id,scene_id,id) ON DELETE CASCADE
);

CREATE INDEX comments_thread_page ON comments(workspace_id,scene_id,thread_id,created_at,id);

CREATE TABLE comment_subscriptions (
  workspace_id uuid NOT NULL,
  scene_id uuid NOT NULL,
  thread_id uuid NOT NULL,
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
  PRIMARY KEY (workspace_id,scene_id,thread_id,user_id),
  FOREIGN KEY (workspace_id,scene_id,thread_id)
    REFERENCES comment_threads(workspace_id,scene_id,id) ON DELETE CASCADE
);

CREATE INDEX comment_subscriptions_user ON comment_subscriptions(user_id,created_at DESC);
