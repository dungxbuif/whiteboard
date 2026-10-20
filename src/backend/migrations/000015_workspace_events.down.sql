DROP TRIGGER IF EXISTS workspace_event_outbox_notify ON workspace_event_outbox;
DROP FUNCTION IF EXISTS workspace_event_notify();
DROP TABLE IF EXISTS workspace_event_source_cursors;
DROP TABLE IF EXISTS workspace_event_outbox;
DROP TABLE IF EXISTS workspace_events;
