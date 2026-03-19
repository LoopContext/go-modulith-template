// Package resolver implements GraphQL resolvers.
// This package provides the root resolver structure that will be used by gqlgen
// when GraphQL is initialized via `just add-graphql`.
//
// The resolver structure is ready to use and provides:
// - Query resolver for read operations
// - Mutation resolver for write operations
// - Subscription resolver for real-time subscriptions via WebSocket
package resolver

import (
	"github.com/cmelgarejo/go-modulith-template/internal/events"
	"github.com/cmelgarejo/go-modulith-template/internal/websocket"
)

// RootResolver is the root resolver that implements all GraphQL operations.
type RootResolver struct {
	queryResolver        *queryResolver
	mutationResolver     *mutationResolver
	subscriptionResolver *subscriptionResolver
}

// NewRootResolver creates a new root resolver.
func NewRootResolver(eventBus *events.Bus, wsHub *websocket.Hub) *RootResolver {
	return &RootResolver{
		queryResolver:        &queryResolver{eventBus: eventBus},
		mutationResolver:     &mutationResolver{eventBus: eventBus},
		subscriptionResolver: &subscriptionResolver{eventBus: eventBus, wsHub: wsHub},
	}
}

// Query returns the query resolver.
// This will implement generated.QueryResolver after running: just add-graphql
func (r *RootResolver) Query() interface{} {
	return r.queryResolver
}

// Mutation returns the mutation resolver.
// This will implement generated.MutationResolver after running: just add-graphql
func (r *RootResolver) Mutation() interface{} {
	return r.mutationResolver
}

// Subscription returns the subscription resolver.
// This will implement generated.SubscriptionResolver after running: just add-graphql
func (r *RootResolver) Subscription() interface{} {
	return r.subscriptionResolver
}

// queryResolver implements QueryResolver.
type queryResolver struct {
	eventBus *events.Bus
}

// mutationResolver implements MutationResolver.
type mutationResolver struct {
	eventBus *events.Bus
}

// subscriptionResolver implements SubscriptionResolver.
type subscriptionResolver struct {
	eventBus *events.Bus
	wsHub    *websocket.Hub
}
