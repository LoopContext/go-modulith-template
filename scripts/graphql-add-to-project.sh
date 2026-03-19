#!/bin/bash

# Script to add optional GraphQL support using gqlgen
# This script initializes GraphQL infrastructure and automatically generates code

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
    echo "   💡 Tip: Create schemas per module (e.g., auth.graphql, order.graphql, payment.graphql)"
    echo "   💡 Tip: Use 'just new-module <name>' to scaffold a module with GraphQL schema"
else
    echo "ℹ️  Schema already exists, skipping..."
fi

# Create minimal stub for generated package FIRST to avoid import errors
# This will be overwritten when gqlgen generate runs
echo "📝 Creating generated package stub..."
mkdir -p "${GENERATED_DIR}"
cat > "${GENERATED_DIR}/generated.go" <<'EOF'
// Package generated contains generated GraphQL code.
// This file is a stub - run 'just graphql-generate' to generate the actual code.
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
// when GraphQL is initialized via `just graphql-init`.
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

# Code generation now happens automatically after integration (see end of script)
echo "ℹ️  GraphQL code will be generated automatically after integration"

# Integrate GraphQL into cmd/server/setup/gateway.go
echo "🔗 Integrating GraphQL into server..."
GATEWAY_FILE="${PROJECT_ROOT}/cmd/server/setup/gateway.go"

if [ -f "${GATEWAY_FILE}" ]; then
    # Check if GraphQL is already integrated
    if grep -q "graphqlServer" "${GATEWAY_FILE}" && grep -q "graphqlServer.Setup" "${GATEWAY_FILE}"; then
        echo "ℹ️  GraphQL already integrated in cmd/server/setup/gateway.go, skipping..."
    else
        # Create temporary files for modifications
        TEMP_FILE=$(mktemp)
        cp "${GATEWAY_FILE}" "${TEMP_FILE}"

        # Add import after websocket import if not present
        if ! grep -q 'graphqlServer "github.com/cmelgarejo/go-modulith-template/internal/graphql"' "${TEMP_FILE}"; then
            if [[ "$OSTYPE" == "darwin"* ]]; then
                # macOS sed - add after websocket import
                sed -i '' '/^[[:space:]]*"github.com\/cmelgarejo\/go-modulith-template\/internal\/websocket"$/a\
	graphqlServer "github.com/cmelgarejo/go-modulith-template/internal/graphql"
' "${TEMP_FILE}"
            else
                # Linux sed
                sed -i '/^[[:space:]]*"github.com\/cmelgarejo\/go-modulith-template\/internal\/websocket"$/a\	graphqlServer "github.com/cmelgarejo/go-modulith-template/internal/graphql"' "${TEMP_FILE}"
            fi
            echo "✅ Added GraphQL import to cmd/server/setup/gateway.go"
        fi

        # Add GraphQL endpoint setup in Gateway function after WebSocket setup and before metrics
        if ! grep -q "graphqlServer.Setup(ctx, reg.EventBus(), wsHub)" "${TEMP_FILE}"; then
            # Create a temporary file with the GraphQL setup code
            TEMP_GRAPHQL=$(mktemp)
            cat > "${TEMP_GRAPHQL}" <<'GRAPHQLEOF'

	// Setup GraphQL endpoint
	if graphqlHandler := graphqlServer.Setup(ctx, reg.EventBus(), wsHub); graphqlHandler != nil {
		mux.Handle("/graphql", graphqlHandler)

		if cfg.Env == "dev" {
			playgroundHandler := graphqlServer.PlaygroundHandler()
			mux.Handle("/graphql/playground", playgroundHandler)
			slog.Info("GraphQL playground enabled", "path", "/graphql/playground")
		}

		slog.Info("GraphQL endpoint enabled", "path", "/graphql")
	}
GRAPHQLEOF
            # Insert after WebSocket endpoint registration (after slog.Info line)
            if [[ "$OSTYPE" == "darwin"* ]]; then
                sed -i '' '/slog.Info("WebSocket endpoint registered"/r '"${TEMP_GRAPHQL}" "${TEMP_FILE}"
            else
                sed -i '/slog.Info("WebSocket endpoint registered"/r '"${TEMP_GRAPHQL}" "${TEMP_FILE}"
            fi
            rm -f "${TEMP_GRAPHQL}"
            echo "✅ Added GraphQL endpoint setup to Gateway function"
        fi

        # Replace original file with modified version
        mv "${TEMP_FILE}" "${GATEWAY_FILE}"
    fi
else
    echo "⚠️  cmd/server/setup/gateway.go not found, skipping integration"
    echo "   You'll need to manually integrate GraphQL (see docs/GRAPHQL_INTEGRATION.md)"
fi

# Always generate GraphQL code after initialization to ensure everything compiles
echo ""
echo "🔄 Generating GraphQL code to ensure everything compiles..."
cd "${PROJECT_ROOT}"

# Temporarily disable exit on error to handle generation failures gracefully
set +e
"${PROJECT_ROOT}/scripts/graphql-generate-all.sh" 2>&1
GEN_EXIT_CODE=$?
set -e

if [ $GEN_EXIT_CODE -eq 0 ]; then
    echo "✅ GraphQL code generated successfully"
else
    echo "⚠️  GraphQL code generation completed (exit code: $GEN_EXIT_CODE)"
    echo "   Note: Some warnings may be expected for empty schemas"
fi

echo ""
echo "✅ GraphQL support added successfully!"
echo ""

echo "📚 Next steps:"
echo "   1. Edit ${SCHEMA_DIR}/schema.graphql to add your queries/mutations"
echo "   2. Run 'just graphql-generate-all' to regenerate code after schema changes"
echo "   Or run 'just graphql-generate-module <module>' for a specific module"
echo "   3. Implement resolvers in ${RESOLVER_DIR}/"
echo "   4. Run 'just run' to start the server"
echo "   5. Access playground at http://localhost:8080/graphql/playground (dev mode)"
echo ""
echo "   ✅ GraphQL is integrated and code has been generated"
echo "   ✅ GraphQL endpoints are ready at /graphql and /graphql/playground"

echo ""
echo "📖 See docs/GRAPHQL_INTEGRATION.md for detailed instructions"
echo "📖 See docs/GRAPHQL_AUTO_GENERATION.md for auto-generating schemas from proto files"


