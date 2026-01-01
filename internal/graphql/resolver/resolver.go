// Package resolver implements GraphQL resolvers.
// This file is created by the add-graphql script.
// This is a stub file - the actual implementation will be generated/created when GraphQL is initialized.
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
// This will implement generated.QueryResolver after running: make add-graphql
func (r *RootResolver) Query() interface{} {
	return r.queryResolver
}

// Mutation returns the mutation resolver.
// This will implement generated.MutationResolver after running: make add-graphql
func (r *RootResolver) Mutation() interface{} {
	return r.mutationResolver
}

// Subscription returns the subscription resolver.
// This will implement generated.SubscriptionResolver after running: make add-graphql
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

