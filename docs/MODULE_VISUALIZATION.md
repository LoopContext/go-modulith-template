# Module Visualization Tool

A development tool to visualize connections between modules in the modulith architecture, similar to Encore.dev's service graph visualization.

## Overview

The visualization tool analyzes your codebase to discover:

-   **Modules**: All registered modules in `modules/`
-   **gRPC Services**: Services defined in proto files
-   **Event Connections**: Event publications and subscriptions between modules
-   **Module Dependencies**: Inter-module communication patterns

## Usage

### Quick Start

Generate an HTML visualization and open it in your browser:

```bash
make visualize
```

This will:

1. Analyze your codebase
2. Generate `docs/module-graph.html` in the docs directory
3. Display instructions to open it in your browser

### Output Formats

#### HTML (Default)

Interactive web-based visualization with D3.js:

```bash
make visualize FORMAT=html
```

Opens an interactive graph where you can:

-   Drag nodes to rearrange
-   Hover to see module details
-   See gRPC and event connections

#### JSON

Raw graph data for programmatic use:

```bash
make visualize FORMAT=json
```

Generates `docs/module-graph.json` with complete graph structure.

#### GraphViz DOT

Generate a DOT file for rendering with GraphViz:

```bash
make visualize FORMAT=dot
```

Then render with:

```bash
dot -Tsvg docs/module-graph.dot -o docs/graph.svg
# or
dot -Tpng docs/module-graph.dot -o docs/graph.png
```

### Web Server Mode

Start a local web server to view the visualization:

```bash
make visualize SERVE=true
```

This will start a server on port 8081 (default) and open the visualization in your browser.

## What It Analyzes

### Module Discovery

The tool scans the `modules/` directory to find all modules:

-   Looks for `module.go` files
-   Extracts module names from `Name()` method

### gRPC Services

Analyzes proto files in `proto/{module}/v1/` to find:

-   Service definitions
-   RPC methods
-   Creates connections showing which modules expose which services

### Event Connections

Scans Go code to find:

-   **Event Publications**: `bus.Publish(ctx, events.Event{Name: "event.name", ...})`
-   **Event Subscriptions**: `bus.Subscribe("event.name", handler)`
-   Creates connections from publishers to subscribers

### Example Output

The visualization shows:

-   **Nodes**: Each module as a circle
-   **Solid Lines**: gRPC service connections (inbound)
-   **Dashed Lines**: gRPC client connections (outbound)
-   **Dotted Blue Lines**: Event connections

## Graph Structure

The generated graph contains:

```json
{
    "modules": [
        {
            "name": "auth",
            "services": ["AuthService"],
            "events": ["user.created", "auth.session.created"]
        }
    ],
    "connections": [
        {
            "from": "auth",
            "to": "order",
            "type": "event",
            "event": "user.created"
        },
        {
            "from": "external",
            "to": "auth",
            "type": "grpc",
            "service": "AuthService",
            "direction": "inbound"
        }
    ]
}
```

## Integration with Development Workflow

### Before Committing

Visualize your module structure to ensure:

-   No circular dependencies
-   Clear event flow
-   Proper module boundaries

```bash
make visualize FORMAT=html
# Review the graph, then commit
```

### Documentation

Generate a static visualization for documentation:

```bash
make visualize FORMAT=dot
dot -Tsvg docs/module-graph.dot -o docs/module-graph.svg
```

### CI/CD Integration

Generate JSON output for automated analysis:

```bash
make visualize FORMAT=json
# Use docs/module-graph.json in CI to validate architecture
```

## Limitations

The current implementation uses pattern matching to detect:

-   Event publications and subscriptions
-   gRPC service definitions

For more accurate analysis, consider:

-   Using `go/ast` for proper Go code parsing
-   Using protobuf reflection for service discovery
-   Runtime analysis during development

## Future Enhancements

Potential improvements:

-   [ ] Real-time updates during development
-   [ ] Metrics overlay (request counts, latency)
-   [ ] Export to various formats (PNG, SVG, PDF)
-   [ ] Integration with IDE plugins
-   [ ] Runtime dependency tracking
-   [ ] Circular dependency detection
-   [ ] Module health status

## Troubleshooting

### No modules found

Ensure your modules are in `modules/` and have `module.go` files.

### Missing connections

The tool uses pattern matching. If connections aren't detected:

-   Check event names match exactly
-   Verify gRPC client usage follows patterns
-   Consider using event constants from `internal/events/types.go`

### Graph too cluttered

Use filters or generate separate graphs for:

-   gRPC connections only
-   Event connections only
-   Specific modules

## Examples

### View current architecture

```bash
make visualize
# Opens docs/module-graph.html
```

### Generate documentation image

```bash
make visualize FORMAT=dot
dot -Tpng docs/module-graph.dot -o docs/architecture.png
```

### Analyze specific module

```bash
make visualize FORMAT=json
# Filter docs/module-graph.json for specific module
```

## See Also

-   [Module Communication Guide](MODULE_COMMUNICATION.md)
-   [Modulith Architecture](MODULITH_ARCHITECTURE.md)
-   [Event Bus Documentation](MODULITH_ARCHITECTURE.md#13-asynchronous-communication-events)
