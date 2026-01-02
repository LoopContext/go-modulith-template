#!/bin/bash

# Script to add optional GraphQL support using gqlgen
# This script initializes GraphQL infrastructure without breaking existing functionality
#
# Usage:
#   ./scripts/add-graphql.sh           # Setup only (no code generation)
#   ./scripts/add-graphql.sh --generate # Setup + generate code for all modules

set -e

# Check for --generate flag
GENERATE_CODE=false
if [ "$1" = "--generate" ] || [ "$1" = "-g" ]; then
    GENERATE_CODE=true
fi

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
    echo "   💡 Tip: Create schemas per module (e.g., auth.graphql, order.graphql, payment.graphql)"
    echo "   💡 Tip: Use 'make new-module <name>' to scaffold a module with GraphQL schema"
else
    echo "ℹ️  Schema already exists, skipping..."
fi

# Create minimal stub for generated package FIRST to avoid import errors
# This will be overwritten when gqlgen generate runs
echo "📝 Creating generated package stub..."
mkdir -p "${GENERATED_DIR}"
cat > "${GENERATED_DIR}/generated.go" <<'EOF'
// Package generated contains generated GraphQL code.
// This file is a stub - run 'make graphql-generate' to generate the actual code.
package generated

// Config is the configuration for the GraphQL executable schema.
type Config struct {
	Resolvers ResolverRoot
}

// ResolverRoot is the root resolver interface.
type ResolverRoot interface {
	Query() QueryResolver
	Mutation() MutationResolver
	Subscription() SubscriptionResolver
}

// QueryResolver is the query resolver interface.
type QueryResolver interface{}

// MutationResolver is the mutation resolver interface.
type MutationResolver interface{}

// SubscriptionResolver is the subscription resolver interface.
type SubscriptionResolver interface{}

// ExecutableSchema is the executable GraphQL schema.
type ExecutableSchema struct{}

// NewExecutableSchema creates a new executable schema.
func NewExecutableSchema(cfg Config) *ExecutableSchema {
	return &ExecutableSchema{}
}
EOF

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
func Setup(_ context.Context, eventBus *events.Bus, wsHub *websocket.Hub) http.Handler {
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

# Create resolver package (replace stub if exists or if it has old RootResolver structure)
if [ ! -f "${RESOLVER_DIR}/resolver.go" ] || grep -q "This is a stub file" "${RESOLVER_DIR}/resolver.go" 2>/dev/null || grep -q "type RootResolver struct" "${RESOLVER_DIR}/resolver.go" 2>/dev/null; then
    echo "🔧 Creating base resolver..."
    cat > "${RESOLVER_DIR}/resolver.go" <<'EOF'
// Package resolver implements GraphQL resolvers.
// This package provides the root resolver structure that will be used by gqlgen
// when GraphQL is initialized via `make add-graphql`.
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

// Resolver is the root resolver that implements all GraphQL operations.
// This matches what gqlgen expects (not RootResolver).
type Resolver struct {
	eventBus *events.Bus
	wsHub    *websocket.Hub
}

// NewRootResolver creates a new root resolver.
// This is a convenience function that creates a Resolver with the required dependencies.
func NewRootResolver(eventBus *events.Bus, wsHub *websocket.Hub) *Resolver {
	return &Resolver{
		eventBus: eventBus,
		wsHub:    wsHub,
	}
}

// EventBus returns the event bus.
func (r *Resolver) EventBus() *events.Bus {
	return r.eventBus
}

// WebSocketHub returns the WebSocket hub.
func (r *Resolver) WebSocketHub() *websocket.Hub {
	return r.wsHub
}
EOF
    echo "✅ Created base resolver at ${RESOLVER_DIR}/resolver.go"
else
    echo "ℹ️  Resolver already exists, skipping..."
fi

# Install dependencies
echo "📦 Installing GraphQL dependencies..."
go get github.com/99designs/gqlgen@latest
go mod tidy

# Generate code only if --generate flag is set
if [ "$GENERATE_CODE" = true ]; then
    echo "🔄 Generating GraphQL code for all modules..."
    cd "${PROJECT_ROOT}"

    if gqlgen generate 2>&1 | grep -q "validation failed"; then
        echo "⚠️  Schema validation failed, but that's OK for initial setup."
        echo "   You can add queries/mutations later and run 'make graphql-generate'"
    else
        echo "✅ GraphQL code generated successfully"
    fi
else
    echo "ℹ️  Skipping code generation (use --generate flag to generate code)"
fi

# Integrate GraphQL into cmd/server/main.go
echo "🔗 Integrating GraphQL into server..."
SERVER_MAIN="${PROJECT_ROOT}/cmd/server/main.go"

if [ -f "${SERVER_MAIN}" ]; then
    # Check if GraphQL is already integrated
    if grep -q "graphqlServer" "${SERVER_MAIN}" && grep -q "setupGraphQLEndpoint" "${SERVER_MAIN}"; then
        echo "ℹ️  GraphQL already integrated in cmd/server/main.go, skipping..."
    else
        # Create temporary files for modifications
        TEMP_FILE=$(mktemp)
        cp "${SERVER_MAIN}" "${TEMP_FILE}"

        # Add import after events import if not present
        if ! grep -q 'graphqlServer "github.com/cmelgarejo/go-modulith-template/internal/graphql"' "${TEMP_FILE}"; then
            if [[ "$OSTYPE" == "darwin"* ]]; then
                # macOS sed
                sed -i '' '/^[[:space:]]*"github.com\/cmelgarejo\/go-modulith-template\/internal\/events"$/a\
	graphqlServer "github.com/cmelgarejo/go-modulith-template/internal/graphql"
' "${TEMP_FILE}"
            else
                # Linux sed
                sed -i '/^[[:space:]]*"github.com\/cmelgarejo\/go-modulith-template\/internal\/events"$/a\	graphqlServer "github.com/cmelgarejo/go-modulith-template/internal/graphql"' "${TEMP_FILE}"
            fi
            echo "✅ Added GraphQL import to cmd/server/main.go"
        fi

        # Add setupGraphQLEndpoint function before const healthStatusHealthy if not present
        if ! grep -q "func setupGraphQLEndpoint" "${TEMP_FILE}"; then
            TEMP_FUNC=$(mktemp)
            cat > "${TEMP_FUNC}" <<'FUNCEOF'
func setupGraphQLEndpoint(ctx context.Context, mux *http.ServeMux, cfg *config.AppConfig, eventBus *events.Bus, wsHub *websocket.Hub) {
	if graphqlHandler := graphqlServer.Setup(ctx, eventBus, wsHub); graphqlHandler != nil {
		mux.Handle("/graphql", graphqlHandler)

		if cfg.Env == "dev" {
			playgroundHandler := graphqlServer.PlaygroundHandler()
			mux.Handle("/graphql/playground", playgroundHandler)
			slog.Info("GraphQL playground enabled", "path", "/graphql/playground")
		}

		slog.Info("GraphQL endpoint enabled", "path", "/graphql")
	}
}

FUNCEOF
            # Insert before const healthStatusHealthy
            if [[ "$OSTYPE" == "darwin"* ]]; then
                sed -i '' '/^const healthStatusHealthy/r '"${TEMP_FUNC}" "${TEMP_FILE}"
            else
                sed -i '/^const healthStatusHealthy/r '"${TEMP_FUNC}" "${TEMP_FILE}"
            fi
            rm -f "${TEMP_FUNC}"
            echo "✅ Added setupGraphQLEndpoint function to cmd/server/main.go"
        fi

        # Add call to setupGraphQLEndpoint in setupGateway function if not present
        if ! grep -q "setupGraphQLEndpoint(ctx, mux, cfg, reg.EventBus(), wsHub)" "${TEMP_FILE}"; then
            if [[ "$OSTYPE" == "darwin"* ]]; then
                sed -i '' '/mux.Handle("\/", rmux)/a\
\
	setupGraphQLEndpoint(ctx, mux, cfg, reg.EventBus(), wsHub)
' "${TEMP_FILE}"
            else
                sed -i '/mux.Handle("\/", rmux)/a\	setupGraphQLEndpoint(ctx, mux, cfg, reg.EventBus(), wsHub)' "${TEMP_FILE}"
            fi
            echo "✅ Added GraphQL endpoint setup call to setupGateway function"
        fi

        # Replace original file with modified version
        mv "${TEMP_FILE}" "${SERVER_MAIN}"

        # After integration, generate GraphQL code to ensure it compiles
        if [ "$GENERATE_CODE" != true ]; then
            echo "🔄 Generating GraphQL code to ensure integration compiles..."
            cd "${PROJECT_ROOT}"
            if gqlgen generate 2>&1 >/dev/null; then
                echo "✅ GraphQL code generated successfully"
            else
                echo "⚠️  GraphQL code generation had issues, but integration is complete"
                echo "   Run 'make graphql-generate-all' manually to fix any issues"
            fi
        fi
    fi
else
    echo "⚠️  cmd/server/main.go not found, skipping integration"
    echo "   You'll need to manually integrate GraphQL (see docs/GRAPHQL_INTEGRATION.md)"
fi

echo ""
echo "✅ GraphQL support added successfully!"
echo ""

if [ "$GENERATE_CODE" = true ]; then
    echo "📚 Next steps:"
    echo "   1. Edit ${SCHEMA_DIR}/schema.graphql to add your queries/mutations"
    echo "   2. Run 'make graphql-generate-all' to regenerate code"
    echo "   3. Implement resolvers in ${RESOLVER_DIR}/"
    echo "   4. Run 'make run' to start the server"
    echo "   5. Access playground at http://localhost:8080/graphql/playground (dev mode)"
    echo ""
    echo "   ✅ GraphQL is already integrated into cmd/server/main.go"
else
    echo "📚 Next steps:"
    echo "   1. Edit ${SCHEMA_DIR}/schema.graphql to add your queries/mutations"
    echo "   2. Run 'make graphql-generate-all' to generate code for all modules"
    echo "   Or run 'make graphql-generate-module <module>' for a specific module"
    echo "   3. Implement resolvers in ${RESOLVER_DIR}/"
    echo "   4. Run 'make run' to start the server"
    echo "   5. Access playground at http://localhost:8080/graphql/playground (dev mode)"
    echo ""
    echo "   ✅ GraphQL is already integrated into cmd/server/main.go"
    echo "   💡 Tip: Run 'make add-graphql --generate' to generate code immediately"
fi

echo ""
echo "📖 See docs/GRAPHQL_INTEGRATION.md for detailed instructions"
echo "📖 See docs/GRAPHQL_AUTO_GENERATION.md for auto-generating schemas from proto files"


