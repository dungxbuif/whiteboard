CREATE INDEX workspace_events_workspace_activity_cursor
    ON workspace_events(workspace_id, occurred_at DESC, id DESC);

CREATE INDEX workspace_events_scene_activity_cursor
    ON workspace_events(workspace_id, scene_id, occurred_at DESC, id DESC)
    WHERE scene_id IS NOT NULL;
