-- Outbox table for transactional outbox pattern
-- This table stores events that need to be published reliably as part of database transactions.
-- Events are stored here and then published asynchronously by the outbox publisher worker.

CREATE TABLE IF NOT EXISTS outbox (
    id VARCHAR(255) PRIMARY KEY,
    event_name VARCHAR(255) NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    published_at TIMESTAMP
);

-- Index for querying unpublished events
CREATE INDEX IF NOT EXISTS idx_outbox_unpublished ON outbox(created_at) WHERE published_at IS NULL;

-- Index for cleanup of old published events (optional, for maintenance)
CREATE INDEX IF NOT EXISTS idx_outbox_published_at ON outbox(published_at);

