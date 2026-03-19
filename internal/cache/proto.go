package cache

import (
	"context"
	"errors"
	"fmt"
	"time"

	"golang.org/x/sync/singleflight"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

var protoLoadGroup singleflight.Group

// SetProto marshals and stores a protobuf message in cache.
func SetProto[T proto.Message](ctx context.Context, c Cache, key string, value T, ttl time.Duration) error {
	data, err := protojson.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal proto cache value: %w", err)
	}

	if err := c.Set(ctx, key, data, ttl); err != nil {
		return fmt.Errorf("set proto cache value: %w", err)
	}

	return nil
}

// GetProto retrieves and unmarshals a protobuf message from cache.
func GetProto[T proto.Message](ctx context.Context, c Cache, key string, newValue func() T) (T, error) {
	var zero T

	data, err := c.Get(ctx, key)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return zero, ErrNotFound
		}

		return zero, fmt.Errorf("get proto cache value: %w", err)
	}

	value := newValue()
	if err := protojson.Unmarshal(data, value); err != nil {
		return zero, fmt.Errorf("unmarshal proto cache value: %w", err)
	}

	return value, nil
}

// GetOrLoadProto returns a cached protobuf message or loads and stores it.
func GetOrLoadProto[T proto.Message](
	ctx context.Context,
	c Cache,
	key string,
	ttl time.Duration,
	newValue func() T,
	loader func(context.Context) (T, error),
) (T, error) {
	value, err := GetProto(ctx, c, key, newValue)
	if err == nil {
		return value, nil
	}

	if err != nil && err != ErrNotFound {
		var zero T
		return zero, err
	}

	result, err, _ := protoLoadGroup.Do(key, func() (interface{}, error) {
		loaded, loadErr := loader(ctx)
		if loadErr != nil {
			return nil, loadErr
		}

		if setErr := SetProto(ctx, c, key, loaded, ttl); setErr != nil {
			return nil, setErr
		}

		return loaded, nil
	})
	if err != nil {
		var zero T
		return zero, fmt.Errorf("load proto cache value: %w", err)
	}

	typed, ok := result.(T)
	if !ok {
		var zero T
		return zero, fmt.Errorf("unexpected cached proto type for key %s", key)
	}

	return typed, nil
}
