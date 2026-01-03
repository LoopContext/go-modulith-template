-- Rollback for outbox table

DROP INDEX IF EXISTS idx_outbox_published_at;
DROP INDEX IF EXISTS idx_outbox_unpublished;
DROP TABLE IF EXISTS outbox;

