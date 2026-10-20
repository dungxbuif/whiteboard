DROP TRIGGER IF EXISTS workspace_event_notification_job ON workspace_events;
DROP FUNCTION IF EXISTS enqueue_comment_notification_job();
DROP TABLE IF EXISTS user_notifications;
