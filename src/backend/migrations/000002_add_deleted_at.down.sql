DROP INDEX idx_users_deleted_at;
ALTER TABLE users DROP COLUMN deleted_at;
