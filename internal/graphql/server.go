// Package graphql provides optional GraphQL API using gqlgen.
// This package is optional and can be integrated into cmd/server/main.go
// This is a stub file - the actual implementation will be created when GraphQL is initialized.
package graphql

import (
	"context"
	"net/http"

	"github.com/cmelgarejo/go-modulith-template/internal/events"
	"github.com/cmelgarejo/go-modulith-template/internal/websocket"
)

// Setup initializes and returns a GraphQL handler.
// Returns nil if GraphQL is not properly configured.
// This is a stub - the actual implementation requires running: just add-graphql
func Setup(_ context.Context, _ *events.Bus, _ *websocket.Hub) http.Handler {
	// Stub implementation - returns nil until GraphQL is initialized
	// After running: just add-graphql, this will return a proper handler
	return nil
}

// PlaygroundHandler returns the GraphQL playground handler.
// This is a stub - the actual implementation requires running: just add-graphql
func PlaygroundHandler() http.Handler {
	// Stub implementation - returns nil until GraphQL is initialized
	// After running: just add-graphql, this will return a proper handler
	return http.NotFoundHandler()
}
