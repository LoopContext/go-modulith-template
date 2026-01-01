#!/bin/bash

# Script to add optional GraphQL support using gqlgen
# This script initializes GraphQL infrastructure without breaking existing functionality

set -e

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GRAPHQL_DIR="${PROJECT_ROOT}/internal/graphql"
SCHEMA_DIR="${GRAPHQL_DIR}/schema"
RESOLVER_DIR="${GRAPHQL_DIR}/resolver"
GENERATED_DIR="${GRAPHQL_DIR}/generated"

echo "🚀 Adding GraphQL support to go-modulith-template..."

# Check if gqlgen is installed
if ! command -v gqlgen &> /dev/null; then
    echo "📦 Installing gqlgen..."
    go install github.com/99designs/gqlgen@latest
fi

# Create directory structure
echo "📁 Creating directory structure..."
mkdir -p "${SCHEMA_DIR}"
mkdir -p "${RESOLVER_DIR}"
mkdir -p "${GENERATED_DIR}"

# Create gqlgen.yml if it doesn't exist
if [ ! -f "${PROJECT_ROOT}/gqlgen.yml" ]; then
    echo "⚙️  Creating gqlgen.yml..."
    cat > "${PROJECT_ROOT}/gqlgen.yml" <<EOF
# Configuration for gqlgen
# See: https://gqlgen.com/config/

schema:
  - internal/graphql/schema/*.graphql

exec:
  filename: internal/graphql/generated/generated.go
  package: generated

model:
  filename: internal/graphql/generated/models_gen.go
  package: generated

resolver:
  layout: follow-schema
  dir: internal/graphql/resolver
  package: resolver

# Optional: Add custom scalars, directives, etc.
# See gqlgen documentation for more options
EOF
    echo "✅ Created gqlgen.yml"
else
    echo "ℹ️  gqlgen.yml already exists, skipping..."
fi

# Create base root schema (combines all module schemas)
if [ ! -f "${SCHEMA_DIR}/schema.graphql" ]; then
    echo "📝 Creating root schema..."
    cat > "${SCHEMA_DIR}/schema.graphql" <<'EOF'
# Root GraphQL schema
# This file combines all module schemas
# Each module should have its own schema file (e.g., auth.graphql, order.graphql)

# Import module schemas (gqlgen will merge them)
# Add your module schemas here as you create them

type Query {
  # Module schemas will extend this
  _empty: String # Placeholder - remove when adding real queries
}

type Mutation {
  # Module schemas will extend this
  _empty: String # Placeholder - remove when adding real mutations
}

type Subscription {
  # Module schemas will extend this
  _empty: String # Placeholder - remove when adding real subscriptions
}
EOF
    echo "✅ Created root schema at ${SCHEMA_DIR}/schema.graphql"

    # Create example module schema
    if [ ! -f "${SCHEMA_DIR}/auth.graphql" ]; then
        echo "📝 Creating example auth module schema..."
        cat > "${SCHEMA_DIR}/auth.graphql" <<'EOF'
# Auth Module GraphQL Schema
# This schema is specific to the auth module
# Each module should have its own schema file

extend type Query {
  me: User
}

extend type Mutation {
  requestLogin(email: String, phone: String): Boolean!
  completeLogin(email: String, phone: String, code: String!): AuthPayload!
}

extend type Subscription {
  userEvents: UserEvent!
}

type User {
  id: ID!
  email: String
  phone: String
  createdAt: String!
}

type AuthPayload {
  token: String!
  user: User!
}

type UserEvent {
  type: String!
  user: User!
}
EOF
        echo "✅ Created example auth schema at ${SCHEMA_DIR}/auth.graphql"
        echo "   💡 Tip: Create schemas per module (e.g., order.graphql, payment.graphql)"
    fi
else
    echo "ℹ️  Schema already exists, skipping..."
fi

# Create server setup file (replace stub if exists)
if [ ! -f "${GRAPHQL_DIR}/server.go" ] || grep -q "This is a stub file" "${GRAPHQL_DIR}/server.go" 2>/dev/null; then
    echo "🔧 Creating GraphQL server setup..."
    cat > "${GRAPHQL_DIR}/server.go" <<'EOF'
// Package graphql provides optional GraphQL API using gqlgen.
// This package is optional and can be integrated into cmd/server/main.go
package graphql

import (
	"context"
	"net/http"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/cmelgarejo/go-modulith-template/internal/events"
	"github.com/cmelgarejo/go-modulith-template/internal/graphql/generated"
	"github.com/cmelgarejo/go-modulith-template/internal/graphql/resolver"
	"github.com/cmelgarejo/go-modulith-template/internal/websocket"
)

// Setup initializes and returns a GraphQL handler.
// Returns nil if GraphQL is not properly configured.
func Setup(ctx context.Context, eventBus *events.Bus, wsHub *websocket.Hub) http.Handler {
	// Check if resolvers are available
	// If not, GraphQL is not set up yet
	resolvers := resolver.NewRootResolver(eventBus, wsHub)
	if resolvers == nil {
		return nil
	}

	cfg := generated.Config{
		Resolvers: resolvers,
	}

	es := generated.NewExecutableSchema(cfg)
	srv := handler.NewDefaultServer(es)

	// Add WebSocket transport for subscriptions
	srv.AddTransport(transport.Websocket{
		KeepAlivePingInterval: 10 * time.Second,
	})

	// Add other transports
	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.MultipartForm{})

	return srv
}

// PlaygroundHandler returns the GraphQL playground handler.
func PlaygroundHandler() http.Handler {
	return playground.Handler("GraphQL Playground", "/graphql")
}
EOF
    echo "✅ Created server setup at ${GRAPHQL_DIR}/server.go"
else
    echo "ℹ️  Server setup already exists, skipping..."
fi

# Create resolver package (replace stub if exists)
if [ ! -f "${RESOLVER_DIR}/resolver.go" ] || grep -q "This is a stub file" "${RESOLVER_DIR}/resolver.go" 2>/dev/null; then
    echo "🔧 Creating base resolver..."
    cat > "${RESOLVER_DIR}/resolver.go" <<'EOF'
// Package resolver implements GraphQL resolvers.
package resolver

import (
	"github.com/cmelgarejo/go-modulith-template/internal/events"
	"github.com/cmelgarejo/go-modulith-template/internal/graphql/generated"
	"github.com/cmelgarejo/go-modulith-template/internal/websocket"
)

// RootResolver is the root resolver that implements all GraphQL operations.
type RootResolver struct {
	*queryResolver
	*mutationResolver
	*subscriptionResolver
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
func (r *RootResolver) Query() generated.QueryResolver {
	return r.queryResolver
}

// Mutation returns the mutation resolver.
func (r *RootResolver) Mutation() generated.MutationResolver {
	return r.mutationResolver
}

// Subscription returns the subscription resolver.
func (r *RootResolver) Subscription() generated.SubscriptionResolver {
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
EOF
    echo "✅ Created base resolver at ${RESOLVER_DIR}/resolver.go"
else
    echo "ℹ️  Resolver already exists, skipping..."
fi

# Create .gitkeep for generated directory
touch "${GENERATED_DIR}/.gitkeep"

# Generate initial code
echo "🔄 Generating GraphQL code..."
cd "${PROJECT_ROOT}"

if gqlgen generate 2>&1 | grep -q "validation failed"; then
    echo "⚠️  Schema validation failed, but that's OK for initial setup."
    echo "   You can add queries/mutations later and run 'make graphql-generate'"
else
    echo "✅ GraphQL code generated successfully"
fi

# Install dependencies
echo "📦 Installing GraphQL dependencies..."
go get github.com/99designs/gqlgen@latest
go mod tidy

echo ""
echo "✅ GraphQL support added successfully!"
echo ""
echo "📚 Next steps:"
echo "   1. Edit ${SCHEMA_DIR}/schema.graphql to add your queries/mutations"
echo "   2. Run 'make graphql-generate' to regenerate code"
echo "   3. Implement resolvers in ${RESOLVER_DIR}/"
echo "   4. Integrate in cmd/server/main.go (see docs/GRAPHQL_INTEGRATION.md)"
echo "   5. Access playground at http://localhost:8080/graphql/playground"
echo ""
echo "📖 See docs/GRAPHQL_INTEGRATION.md for detailed instructions"

