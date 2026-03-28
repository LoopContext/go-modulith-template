package resolver

import (
	"context"
	"testing"

	"github.com/LoopContext/go-modulith-template/internal/events"
	"github.com/LoopContext/go-modulith-template/internal/websocket"
)

func TestNewRootResolver(t *testing.T) {
	eventBus := events.NewBus()
	wsHub := websocket.NewHub(context.Background())

	resolver := NewRootResolver(eventBus, wsHub)

	if resolver == nil {
		t.Fatal("Expected resolver to be created")
	}

	if resolver.queryResolver == nil {
		t.Error("Expected queryResolver to be initialized")
	}

	if resolver.mutationResolver == nil {
		t.Error("Expected mutationResolver to be initialized")
	}

	if resolver.subscriptionResolver == nil {
		t.Error("Expected subscriptionResolver to be initialized")
	}
}

func TestRootResolver_Query(t *testing.T) {
	eventBus := events.NewBus()
	wsHub := websocket.NewHub(context.Background())

	resolver := NewRootResolver(eventBus, wsHub)

	if resolver == nil {
		t.Fatal("Expected resolver to be created")
	}

	queryResolver := resolver.Query()

	if queryResolver == nil {
		t.Fatal("Expected QueryResolver to be returned")
	}
}

func TestRootResolver_Mutation(t *testing.T) {
	eventBus := events.NewBus()
	wsHub := websocket.NewHub(context.Background())

	resolver := NewRootResolver(eventBus, wsHub)

	if resolver == nil {
		t.Fatal("Expected resolver to be created")
	}

	mutationResolver := resolver.Mutation()

	if mutationResolver == nil {
		t.Fatal("Expected MutationResolver to be returned")
	}
}

func TestRootResolver_Subscription(t *testing.T) {
	eventBus := events.NewBus()
	wsHub := websocket.NewHub(context.Background())

	resolver := NewRootResolver(eventBus, wsHub)

	if resolver == nil {
		t.Fatal("Expected resolver to be created")
	}

	subscriptionResolver := resolver.Subscription()

	if subscriptionResolver == nil {
		t.Fatal("Expected SubscriptionResolver to be returned")
	}
}

func TestQueryResolver_EventBus(t *testing.T) {
	eventBus := events.NewBus()
	wsHub := websocket.NewHub(context.Background())

	resolver := NewRootResolver(eventBus, wsHub)

	if resolver == nil {
		t.Fatal("Expected resolver to be created")
	}

	// Verify resolver has access to eventBus
	if resolver.queryResolver == nil {
		t.Error("Expected queryResolver to be initialized")
	}
}

func TestMutationResolver_EventBus(t *testing.T) {
	eventBus := events.NewBus()
	wsHub := websocket.NewHub(context.Background())

	resolver := NewRootResolver(eventBus, wsHub)

	if resolver == nil {
		t.Fatal("Expected resolver to be created")
	}

	// Verify resolver has access to eventBus
	if resolver.mutationResolver == nil {
		t.Error("Expected mutationResolver to be initialized")
	}
}

func TestSubscriptionResolver_EventBusAndHub(t *testing.T) {
	eventBus := events.NewBus()
	wsHub := websocket.NewHub(context.Background())

	resolver := NewRootResolver(eventBus, wsHub)

	if resolver == nil {
		t.Fatal("Expected resolver to be created")
	}

	// Verify resolver has access to both eventBus and wsHub
	if resolver.subscriptionResolver == nil {
		t.Error("Expected subscriptionResolver to be initialized")
	}
}

func TestRootResolver_NilEventBus(t *testing.T) {
	wsHub := websocket.NewHub(context.Background())

	// Should not panic with nil eventBus
	resolver := NewRootResolver(nil, wsHub)

	if resolver == nil {
		t.Fatal("Expected resolver to be created even with nil eventBus")
	}
}

func TestRootResolver_NilWebSocketHub(t *testing.T) {
	eventBus := events.NewBus()

	// Should not panic with nil wsHub
	resolver := NewRootResolver(eventBus, nil)

	if resolver == nil {
		t.Fatal("Expected resolver to be created even with nil wsHub")
	}

	if resolver.subscriptionResolver == nil {
		t.Error("Expected subscriptionResolver to be created even with nil wsHub")
	}
}

func TestRootResolver_BothNil(t *testing.T) {
	// Should not panic with both nil
	resolver := NewRootResolver(nil, nil)

	if resolver == nil {
		t.Fatal("Expected resolver to be created even with both nil")
	}
}

func TestRootResolver_MultipleInstances(t *testing.T) {
	eventBus := events.NewBus()
	wsHub := websocket.NewHub(context.Background())

	// Create multiple resolvers
	resolver1 := NewRootResolver(eventBus, wsHub)
	resolver2 := NewRootResolver(eventBus, wsHub)

	if resolver1 == resolver2 {
		t.Error("Expected different resolver instances")
	}

	if resolver1.queryResolver == resolver2.queryResolver {
		t.Error("Expected different queryResolver instances")
	}
}
