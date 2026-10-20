CREATE TABLE workspace_events (
 id BIGSERIAL PRIMARY KEY,
 workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 scene_id UUID,
 actor_kind TEXT NOT NULL CHECK(actor_kind IN ('user','public','agent','system')),
 actor_id TEXT NOT NULL CHECK(length(actor_id) BETWEEN 1 AND 256),
 kind TEXT NOT NULL CHECK(kind ~ '^[a-z0-9][a-z0-9_.:-]{0,127}$'),
 subject_id TEXT NOT NULL DEFAULT '' CHECK(length(subject_id) <= 256),
 occurred_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
 safe_metadata JSONB NOT NULL DEFAULT '{}'::jsonb CHECK(jsonb_typeof(safe_metadata)='object' AND octet_length(safe_metadata::text) <= 4096),
 source TEXT NOT NULL CHECK(source ~ '^[a-z0-9][a-z0-9_.:-]{0,127}$'),
 source_key TEXT NOT NULL CHECK(length(source_key) BETWEEN 1 AND 256),
 UNIQUE(source,source_key),
 FOREIGN KEY(workspace_id,scene_id) REFERENCES scenes(workspace_id,id) ON DELETE CASCADE
);
CREATE INDEX workspace_events_workspace_cursor ON workspace_events(workspace_id,id DESC);
CREATE INDEX workspace_events_scene_cursor ON workspace_events(workspace_id,scene_id,id DESC) WHERE scene_id IS NOT NULL;

CREATE TABLE workspace_event_outbox (
 event_id BIGINT PRIMARY KEY REFERENCES workspace_events(id) ON DELETE CASCADE,
 workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 scene_id UUID,
 created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
 delivered_at TIMESTAMPTZ,
 attempts INTEGER NOT NULL DEFAULT 0 CHECK(attempts >= 0),
 FOREIGN KEY(workspace_id,scene_id) REFERENCES scenes(workspace_id,id) ON DELETE CASCADE
);
CREATE INDEX workspace_event_outbox_pending ON workspace_event_outbox(event_id) WHERE delivered_at IS NULL;

CREATE TABLE workspace_event_source_cursors (
 source TEXT PRIMARY KEY CHECK(source ~ '^[a-z0-9][a-z0-9_.:-]{0,127}$'),
 cursor_time TIMESTAMPTZ NOT NULL DEFAULT '1970-01-01T00:00:00Z',
 cursor_key TEXT NOT NULL DEFAULT '',
 updated_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);

CREATE FUNCTION workspace_event_notify() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE payload JSONB;
BEGIN
 payload := jsonb_build_object('workspaceId',NEW.workspace_id,'eventCursor',NEW.event_id::text);
 IF NEW.scene_id IS NOT NULL THEN payload := payload || jsonb_build_object('sceneId',NEW.scene_id); END IF;
 PERFORM pg_notify('feature_changed',payload::text);
 RETURN NULL;
END $$;
CREATE TRIGGER workspace_event_outbox_notify AFTER INSERT ON workspace_event_outbox FOR EACH ROW EXECUTE FUNCTION workspace_event_notify();
