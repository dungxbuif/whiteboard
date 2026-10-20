CREATE TABLE user_notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    scene_id UUID NOT NULL,
    event_id BIGINT NOT NULL REFERENCES workspace_events(id) ON DELETE CASCADE,
    kind TEXT NOT NULL,
    actor_name TEXT NOT NULL CHECK (length(actor_name) BETWEEN 1 AND 255),
    thread_id UUID NOT NULL,
    read_at TIMESTAMPTZ,
    read_request_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    UNIQUE (user_id,event_id),
    UNIQUE (user_id,read_request_id),
    FOREIGN KEY (workspace_id,scene_id) REFERENCES scenes(workspace_id,id) ON DELETE CASCADE,
    FOREIGN KEY (workspace_id,scene_id,thread_id) REFERENCES comment_threads(workspace_id,scene_id,id) ON DELETE CASCADE
);
CREATE INDEX user_notifications_inbox ON user_notifications(user_id,created_at DESC,id DESC);

CREATE FUNCTION enqueue_comment_notification_job() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.kind LIKE 'comment.%' THEN
        INSERT INTO feature_jobs(workspace_id,event_id,kind,payload,idempotency_key)
        VALUES (NEW.workspace_id,NEW.id,'notification.deliver','{}'::jsonb,'event:' || NEW.id::text)
        ON CONFLICT (workspace_id,kind,idempotency_key) DO NOTHING;
    END IF;
    RETURN NULL;
END $$;
CREATE TRIGGER workspace_event_notification_job
AFTER INSERT ON workspace_events FOR EACH ROW EXECUTE FUNCTION enqueue_comment_notification_job();
