// Package outbox implements the transactional outbox pattern for reliable event publishing.
// The outbox pattern ensures that events are stored in the database as part of the same
// transaction as business data, then published asynchronously by a background worker.
//
// This pattern is optional - enable it when you need:
// - Reliable event delivery (events must survive application restarts)
// - Multi-instance deployments (events must reach all instances)
// - Event durability (critical business events)
package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Entry represents a single event stored in the outbox table.
type Entry struct {
	ID          string
	EventName   string
	Payload     json.RawMessage
	CreatedAt   time.Time
	PublishedAt *time.Time
}

// StoreRepository provides methods for storing outbox entries.
type StoreRepository interface {
	// Store stores an event in the outbox table within the current transaction.
	Store(ctx context.Context, tx pgx.Tx, eventName string, payload interface{}) error
}

// PublisherRepository provides methods for retrieving and marking outbox entries.
type PublisherRepository interface {
	// GetUnpublished retrieves unpublished events (for publisher worker).
	GetUnpublished(ctx context.Context, limit int) ([]Entry, error)

	// MarkPublished marks events as published.
	MarkPublished(ctx context.Context, ids []string) error
}

// Repository combines StoreRepository and PublisherRepository.
type Repository interface {
	StoreRepository
	PublisherRepository
}

// SQLRepository implements the Repository interface using pgx.
type SQLRepository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new outbox repository.
func NewRepository(db *pgxpool.Pool) *SQLRepository {
	return &SQLRepository{
		db: db,
	}
}

// Store stores an event in the outbox table within the provided transaction.
func (r *SQLRepository) Store(ctx context.Context, tx pgx.Tx, eventName string, payload interface{}) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	id := fmt.Sprintf("outbox_%d", time.Now().UnixNano())

	query := `
		INSERT INTO outbox (id, event_name, payload, created_at)
		VALUES ($1, $2, $3, NOW())
	`

	_, err = tx.Exec(ctx, query, id, eventName, payloadBytes)
	if err != nil {
		return fmt.Errorf("failed to store outbox entry: %w", err)
	}

	return nil
}

// GetUnpublished retrieves unpublished events, ordered by creation time.
func (r *SQLRepository) GetUnpublished(ctx context.Context, limit int) ([]Entry, error) {
	query := `
		SELECT id, event_name, payload, created_at, published_at
		FROM outbox
		WHERE published_at IS NULL
		ORDER BY created_at ASC
		LIMIT $1
	`

	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query unpublished events: %w", err)
	}

	defer rows.Close()

	var entries []Entry

	for rows.Next() {
		var (
			entry       Entry
			publishedAt *time.Time
		)

		if err := rows.Scan(
			&entry.ID,
			&entry.EventName,
			&entry.Payload,
			&entry.CreatedAt,
			&publishedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan outbox entry: %w", err)
		}

		entry.PublishedAt = publishedAt
		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return entries, nil
}

// MarkPublished marks the specified events as published.
func (r *SQLRepository) MarkPublished(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	query := `
		UPDATE outbox
		SET published_at = NOW()
		WHERE id = ANY($1)
	`

	_, err := r.db.Exec(ctx, query, ids)
	if err != nil {
		return fmt.Errorf("failed to mark events as published: %w", err)
	}

	return nil
}

// Publisher publishes events from the outbox table to the event bus.
type Publisher struct {
	repo         PublisherRepository
	publisher    PublisherFunc
	batchSize    int
	pollInterval time.Duration
	stopChan     chan struct{}
}

// PublisherFunc is a function type for publishing events.
// This allows the publisher to work with any event bus implementation.
type PublisherFunc func(ctx context.Context, eventName string, payload interface{})

// NewPublisher creates a new outbox publisher.
// The publisher function should publish events to the event bus.
//
//	Example: NewPublisher(repo, func(ctx context.Context, name string, payload interface{}) {
//	    bus.Publish(ctx, events.Event{Name: name, Payload: payload})
//	})
func NewPublisher(repo PublisherRepository, publisher PublisherFunc) *Publisher {
	return &Publisher{
		repo:         repo,
		publisher:    publisher,
		batchSize:    100,             // Default batch size
		pollInterval: 5 * time.Second, // Default poll interval
		stopChan:     make(chan struct{}),
	}
}

// Start starts the outbox publisher background worker.
func (p *Publisher) Start(ctx context.Context) {
	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.stopChan:
			return
		case <-ticker.C:
			if err := p.Process(ctx); err != nil {
				slog.ErrorContext(ctx, "failed to process outbox", "error", err)
			}
		}
	}
}

// Stop stops the outbox publisher.
func (p *Publisher) Stop() {
	close(p.stopChan)
}

// SetBatchSize sets the number of events to process per batch.
func (p *Publisher) SetBatchSize(size int) {
	p.batchSize = size
}

// SetPollInterval sets the interval between polls.
func (p *Publisher) SetPollInterval(interval time.Duration) {
	p.pollInterval = interval
}

// Process processes unpublished events from the outbox and publishes them.
// This should be called periodically by a background worker.
func (p *Publisher) Process(ctx context.Context) error {
	entries, err := p.repo.GetUnpublished(ctx, p.batchSize)
	if err != nil {
		return fmt.Errorf("failed to get unpublished events: %w", err)
	}

	if len(entries) == 0 {
		return nil // No events to process
	}

	publishedIDs := make([]string, 0, len(entries))

	for _, entry := range entries {
		var payload map[string]interface{}

		if err := json.Unmarshal(entry.Payload, &payload); err != nil {
			// Log error but continue processing other events
			continue
		}

		// Publish event using the publisher function
		p.publisher(ctx, entry.EventName, payload)

		// Event published successfully (publisher function doesn't return error)
		publishedIDs = append(publishedIDs, entry.ID)
	}

	// Mark successfully published events
	if len(publishedIDs) > 0 {
		if err := p.repo.MarkPublished(ctx, publishedIDs); err != nil {
			return fmt.Errorf("failed to mark events as published: %w", err)
		}
	}

	return nil
}
