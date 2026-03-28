// Package main provides a development tool to visualize module connections
// in the modulith architecture, similar to Encore.dev's service graph.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/LoopContext/go-modulith-template/cmd/visualize/analyzer"
)

var (
	outputFile = flag.String("output", "", "Output file (auto-determined by format if not specified)")
	port       = flag.Int("port", 8081, "Port for web server (if serving)")
	serve      = flag.Bool("serve", false, "Start web server to view visualization")
	format     = flag.String("format", "html", "Output format: json, dot, html")
)

func main() {
	flag.Parse()

	projectRoot, err := findProjectRoot()
	if err != nil {
		slog.Error("Failed to find project root", "error", err)
		os.Exit(1)
	}

	slog.Info("Analyzing modulith architecture", "root", projectRoot)

	determineOutputFile()

	graph, err := analyzer.Analyze(projectRoot)
	if err != nil {
		slog.Error("Failed to analyze codebase", "error", err)
		os.Exit(1)
	}

	if err := outputGraph(graph, *outputFile, *format); err != nil {
		slog.Error("Failed to output graph", "error", err)
		os.Exit(1)
	}

	if *serve {
		if err := serveWebUI(graph, *port); err != nil {
			slog.Error("Failed to start web server", "error", err)
			os.Exit(1)
		}
	}
}

func determineOutputFile() {
	if *outputFile != "" {
		return
	}

	switch *format {
	case "html":
		*outputFile = "docs/module-graph.html"
	case "json":
		*outputFile = "docs/module-graph.json"
	case "dot":
		*outputFile = "docs/module-graph.dot"
	default:
		*outputFile = "docs/module-graph.html"
	}
}

func outputGraph(graph *analyzer.Graph, filename, format string) error {
	switch format {
	case "json":
		return outputJSONFormat(graph, filename)
	case "dot":
		return outputDOTFormat(graph, filename)
	case "html":
		return outputHTMLFormat(graph, filename)
	default:
		return fmt.Errorf("unknown format: %s", format)
	}
}

func outputJSONFormat(graph *analyzer.Graph, filename string) error {
	if err := outputJSON(graph, filename); err != nil {
		return fmt.Errorf("failed to write JSON output: %w", err)
	}

	slog.Info("Graph data written", "file", filename)

	if !*serve {
		fmt.Printf("\n✅ Module graph generated: %s\n", filename)
		fmt.Printf("   Run with --serve to view in browser\n")
	}

	return nil
}

func outputDOTFormat(graph *analyzer.Graph, filename string) error {
	if err := outputDOT(graph, filename); err != nil {
		return fmt.Errorf("failed to write DOT output: %w", err)
	}

	slog.Info("DOT graph written", "file", filename)

	fmt.Printf("\n✅ GraphViz DOT file generated: %s\n", filename)
	fmt.Printf("   Render with: dot -Tsvg %s -o graph.svg\n", filename)

	return nil
}

func outputHTMLFormat(graph *analyzer.Graph, filename string) error {
	if err := outputHTML(graph, filename); err != nil {
		return fmt.Errorf("failed to write HTML output: %w", err)
	}

	slog.Info("HTML visualization written", "file", filename)

	absPath, _ := filepath.Abs(filename)
	fmt.Printf("\n✅ HTML visualization generated: %s\n", filename)
	fmt.Printf("   Open in browser: file://%s\n", absPath)

	return nil
}

func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found")
		}

		dir = parent
	}
}

func outputJSON(graph *analyzer.Graph, filename string) error {
	data, err := json.MarshalIndent(graph, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	cleanPath := filepath.Clean(filename)
	if err := os.WriteFile(cleanPath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func outputDOT(graph *analyzer.Graph, filename string) error {
	var sb strings.Builder

	sb.WriteString("digraph modulith {\n")
	sb.WriteString("  rankdir=LR;\n")
	sb.WriteString("  node [shape=box, style=rounded];\n\n")

	// Add nodes
	for _, module := range graph.Modules {
		fmt.Fprintf(&sb, "  \"%s\" [label=\"%s\"];\n", module.Name, module.Name)
	}

	sb.WriteString("\n")

	// Add gRPC connections
	for _, conn := range graph.Connections {
		if conn.Type == "grpc" {
			style := "solid"
			if conn.Direction == "outbound" {
				style = "dashed"
			}

			fmt.Fprintf(&sb, "  \"%s\" -> \"%s\" [label=\"%s\", style=%s];\n",
				conn.From, conn.To, conn.Service, style)
		}
	}

	// Add event connections
	for _, conn := range graph.Connections {
		if conn.Type == "event" {
			fmt.Fprintf(&sb, "  \"%s\" -> \"%s\" [label=\"%s\", style=dotted, color=blue];\n",
				conn.From, conn.To, conn.Event)
		}
	}

	sb.WriteString("}\n")

	cleanPath := filepath.Clean(filename)

	if err := os.WriteFile(cleanPath, []byte(sb.String()), 0o600); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func outputHTML(graph *analyzer.Graph, filename string) error {
	// Generate HTML with embedded visualization
	html := generateHTML(graph)

	cleanPath := filepath.Clean(filename)

	if err := os.WriteFile(cleanPath, []byte(html), 0o600); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func serveWebUI(_ *analyzer.Graph, _ int) error {
	// This will be implemented to serve the web UI
	return fmt.Errorf("web server not yet implemented, use --format=html instead")
}

//nolint:funlen // HTML generation requires long function
func generateHTML(graph *analyzer.Graph) string {
	// Convert graph to JSON for embedding
	graphJSON, _ := json.Marshal(graph)
	graphJSONStr := string(graphJSON)

	return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Modulith Module Graph</title>
    <script src="https://d3js.org/d3.v7.min.js"></script>
    <style>
        html, body {
            height: 100%%;
            margin: 0;
            padding: 0;
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
            background: #f5f5f5;
        }
        body {
            display: flex;
            flex-direction: column;
            padding: 0;
            box-sizing: border-box;
        }
        .container {
            flex: 1;
            display: flex;
            flex-direction: column;
            width: 100%%;
            margin: 0;
            background: white;
            border-radius: 0;
            padding: 20px;
            box-shadow: none;
            box-sizing: border-box;
            min-height: 0;
        }
        h1 {
            margin-top: 0;
            margin-bottom: 0;
            color: #333;
            flex-shrink: 0;
        }
        .controls {
            flex-shrink: 0;
            margin-bottom: 20px;
            padding: 15px;
            background: #f9f9f9;
            border-radius: 4px;
        }
        .legend {
            display: flex;
            gap: 20px;
            margin-top: 10px;
            flex-wrap: wrap;
        }
        .legend-item {
            display: flex;
            align-items: center;
            gap: 8px;
        }
        .legend-line {
            width: 30px;
            height: 2px;
        }
        .legend-line.solid { border-top: 2px solid #333; }
        .legend-line.dashed { border-top: 2px dashed #333; }
        .legend-line.dotted { border-top: 2px dotted #0066cc; }
        #graph {
            flex: 1;
            width: 100%%;
            min-height: 0;
            border: 1px solid #ddd;
            border-radius: 4px;
        }
        @media (max-width: 768px) {
            body {
                padding: 10px;
            }
            .container {
                padding: 15px;
            }
            .legend {
                gap: 10px;
            }
        }
        .node {
            cursor: pointer;
        }
        .node circle {
            fill: #4a90e2;
            stroke: #2c5aa0;
            stroke-width: 2px;
        }
        .node text {
            font-size: 12px;
            fill: #333;
        }
        .link {
            fill: none;
            stroke: #999;
            stroke-width: 1.5px;
            opacity: 0.6;
        }
        .link.grpc {
            stroke: #333;
        }
        .link.event {
            stroke: #0066cc;
            stroke-dasharray: 5,5;
        }
        .tooltip {
            position: absolute;
            padding: 10px;
            background: rgba(0, 0, 0, 0.8);
            color: white;
            border-radius: 4px;
            pointer-events: none;
            font-size: 12px;
            display: none;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>🔗 Modulith Module Graph</h1>
        <div class="controls">
            <div class="legend">
                <div class="legend-item">
                    <div class="legend-line solid"></div>
                    <span>gRPC (inbound)</span>
                </div>
                <div class="legend-item">
                    <div class="legend-line dashed"></div>
                    <span>gRPC (outbound)</span>
                </div>
                <div class="legend-item">
                    <div class="legend-line dotted"></div>
                    <span>Events</span>
                </div>
            </div>
        </div>
        <svg id="graph"></svg>
    </div>
    <div class="tooltip" id="tooltip"></div>

    <script>
        const graphData = ` + graphJSONStr + `;

        const container = document.getElementById('graph');
        const svg = d3.select("#graph");
        const tooltip = d3.select("#tooltip");

        let width, height;
        let simulation;

        function updateDimensions() {
            // Use viewport width (100vw) with minimum of 800px
            width = Math.max(window.innerWidth - 42 || 800, 800);

            // Calculate height: 100vh - header height - controls height
            const header = document.querySelector('h1');
            const controls = document.querySelector('.controls');
            const headerHeight = header ? header.getBoundingClientRect().height : 0;
            const controlsHeight = controls ? controls.getBoundingClientRect().height : 0;
            const containerPadding = 40; // 20px padding top + 20px padding bottom
            const availableHeight = (window.innerHeight - 24 || 600) - headerHeight - controlsHeight - containerPadding;
            height = Math.max(availableHeight, 600);

            svg.attr("width", width)
               .attr("height", height);

            if (simulation) {
                simulation.force("center", d3.forceCenter(width / 2, height / 2));
                simulation.alpha(0.3).restart();
            }
        }

        // Create nodes - include "external" node for external connections
        const nodeMap = new Map();
        graphData.modules.forEach(m => {
            nodeMap.set(m.name, {id: m.name, module: m});
        });

        // Add external node if there are external connections
        const hasExternal = graphData.connections.some(c => c.from === "external" || c.to === "external");
        if (hasExternal) {
            nodeMap.set("external", {id: "external", module: {name: "external", services: [], events: []}});
        }

        const nodes = Array.from(nodeMap.values());

        // Create links with source/target as string IDs (D3 will convert them)
        const links = graphData.connections.map(c => ({
            source: c.from,
            target: c.to,
            type: c.type,
            service: c.service || "",
            event: c.event || "",
            direction: c.direction || ""
        }));

        // Initialize dimensions
        updateDimensions();

        // Create force simulation
        simulation = d3.forceSimulation(nodes)
            .force("link", d3.forceLink(links).id(d => d.id))
            .force("charge", d3.forceManyBody().strength(-300))
            .force("center", d3.forceCenter(width / 2, height / 2))
            .force("collision", d3.forceCollide().radius(50));

        // Update dimensions after layout to use actual container size
        requestAnimationFrame(function() {
            updateDimensions();
        });

        // Handle window resize
        let resizeTimeout;
        window.addEventListener('resize', function() {
            clearTimeout(resizeTimeout);
            resizeTimeout = setTimeout(updateDimensions, 100);
        });

        // Create links
        const link = svg.append("g")
            .selectAll("line")
            .data(links)
            .enter().append("line")
            .attr("class", d => d.type === "event" ? "link event" : "link grpc")
            .attr("stroke-width", 2);

        // Create link labels
        const linkLabel = svg.append("g")
            .selectAll("text")
            .data(links)
            .enter().append("text")
            .attr("class", "link-label")
            .attr("font-size", 10)
            .attr("fill", "#666")
            .text(d => d.type === "event" ? d.event : d.service);

        // Create nodes
        const node = svg.append("g")
            .selectAll("g")
            .data(nodes)
            .enter().append("g")
            .attr("class", "node")
            .call(d3.drag()
                .on("start", dragstarted)
                .on("drag", dragged)
                .on("end", dragended));

        node.append("circle")
            .attr("r", 30);

        node.append("text")
            .attr("dy", 50)
            .attr("text-anchor", "middle")
            .text(d => d.id);

        // Add tooltip on hover
        node.on("mouseover", function(event, d) {
            const services = (d.module && d.module.services) ? d.module.services : [];
            const moduleEvents = (d.module && d.module.events) ? d.module.events : [];
            const eventConnections = graphData.connections
                .filter(c => c.from === d.id && c.type === "event")
                .map(c => c.event);

            const serviceCount = services.length;
            const eventCount = moduleEvents.length;
            const eventConnectionCount = eventConnections.length;

            let tooltipContent = "<strong>" + d.id + "</strong><br/>";
            tooltipContent += "Services: " + serviceCount + "<br/>";
            tooltipContent += "Events: " + eventCount;
            if (eventCount > 0) {
                tooltipContent += " (" + moduleEvents.join(", ") + ")";
            }
            if (eventConnectionCount > 0) {
                tooltipContent += "<br/>Event Connections: " + eventConnectionCount;
            }

            tooltip
                .style("display", "block")
                .html(tooltipContent)
                .style("left", (event.pageX + 10) + "px")
                .style("top", (event.pageY - 10) + "px");
        })
        .on("mouseout", function() {
            tooltip.style("display", "none");
        });

        // Update positions on tick
        simulation.on("tick", () => {
            link
                .attr("x1", d => d.source.x)
                .attr("y1", d => d.source.y)
                .attr("x2", d => d.target.x)
                .attr("y2", d => d.target.y);

            linkLabel
                .attr("x", d => (d.source.x + d.target.x) / 2)
                .attr("y", d => (d.source.y + d.target.y) / 2);

            node
                .attr("transform", d => "translate(" + d.x + "," + d.y + ")");
        });

        function dragstarted(event, d) {
            if (!event.active) simulation.alphaTarget(0.3).restart();
            d.fx = d.x;
            d.fy = d.y;
        }

        function dragged(event, d) {
            d.fx = event.x;
            d.fy = event.y;
        }

        function dragended(event, d) {
            if (!event.active) simulation.alphaTarget(0);
            d.fx = null;
            d.fy = null;
        }
    </script>
</body>
</html>`
}
