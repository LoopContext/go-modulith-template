package cache

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/LoopContext/go-modulith-template/internal/events"
)

// Invalidator connects cache invalidation rules to the in-process event bus.
type Invalidator struct {
	cache        Cache
	eventBus     *events.Bus
	unsubscribes []func()
}

// NewInvalidator creates an event-driven cache invalidator.
func NewInvalidator(c Cache, bus *events.Bus) *Invalidator {
	return &Invalidator{
		cache:        c,
		eventBus:     bus,
		unsubscribes: make([]func(), 0),
	}
}

// SubscribeKeys removes exact cache keys when the given event is published.
func (i *Invalidator) SubscribeKeys(eventName string, keys ...string) {
	if i == nil || i.cache == nil || i.eventBus == nil || len(keys) == 0 {
		return
	}

	unsubscribe := i.eventBus.Subscribe(eventName, func(ctx context.Context, event events.Event) error {
		if err := i.cache.DeleteMany(ctx, keys...); err != nil {
			return fmt.Errorf("invalidate keys for event %s: %w", event.Name, err)
		}

		return nil
	})

	i.unsubscribes = append(i.unsubscribes, unsubscribe)
}

// SubscribePrefixes removes cache keys matching prefixes when the given event is published.
func (i *Invalidator) SubscribePrefixes(eventName string, prefixes ...string) {
	if i == nil || i.cache == nil || i.eventBus == nil || len(prefixes) == 0 {
		return
	}

	unsubscribe := i.eventBus.Subscribe(eventName, func(ctx context.Context, event events.Event) error {
		for _, prefix := range prefixes {
			if err := i.cache.DeleteByPrefix(ctx, prefix); err != nil {
				return fmt.Errorf("invalidate prefix %s for event %s: %w", prefix, event.Name, err)
			}
		}

		return nil
	})

	i.unsubscribes = append(i.unsubscribes, unsubscribe)
}

// Close removes all event subscriptions created by the invalidator.
func (i *Invalidator) Close() {
	if i == nil {
		return
	}

	for _, unsubscribe := range i.unsubscribes {
		if unsubscribe != nil {
			unsubscribe()
		}
	}

	slog.Debug("Cache invalidator closed", "subscriptions", len(i.unsubscribes))
	i.unsubscribes = nil
}
