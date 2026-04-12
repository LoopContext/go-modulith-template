//nolint:revive,errcheck,gosec,wrapcheck,funlen,cyclop // This is a dev tool, lenient linting
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	"github.com/LoopContext/go-modulith-template/cmd/visualize/analyzer"
)

const defaultProjectName = "Modulith"

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

	projectName := determineProjectName(projectRoot)

	if err := outputGraph(graph, *outputFile, *format, projectName); err != nil {
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

func determineProjectName(projectRoot string) string {
	_ = godotenv.Load(filepath.Join(projectRoot, ".env"))

	projectName := os.Getenv("APP_NAME")
	if projectName == "" {
		modName, err := getProjectName(projectRoot)
		if err == nil && modName != "" && modName != defaultProjectName {
			projectName = modName
		}
	}

	if projectName == "" || projectName == defaultProjectName {
		projectName = defaultProjectName + " Project"
	}

	return projectName
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

func outputGraph(graph *analyzer.Graph, filename, format, projectName string) error {
	switch format {
	case "json":
		return outputJSONFormat(graph, filename)
	case "dot":
		return outputDOTFormat(graph, filename)
	case "html":
		return outputHTMLFormat(graph, filename, projectName)
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

func outputHTMLFormat(graph *analyzer.Graph, filename, projectName string) error {
	if err := outputHTML(graph, filename, projectName); err != nil {
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

func getProjectName(root string) (string, error) {
	f, err := os.Open(filepath.Join(root, "go.mod"))
	if err != nil {
		return "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			modPath := strings.TrimSpace(strings.TrimPrefix(line, "module "))

			parts := strings.Split(modPath, "/")
			if len(parts) > 0 {
				return parts[len(parts)-1], nil
			}

			return modPath, nil
		}
	}

	return defaultProjectName, nil
}

func outputJSON(graph *analyzer.Graph, filename string) error {
	data, err := json.MarshalIndent(graph, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return os.WriteFile(filepath.Clean(filename), data, 0o600)
}

func outputDOT(graph *analyzer.Graph, filename string) error {
	var sb strings.Builder
	sb.WriteString("digraph modulith {\n")
	sb.WriteString("  rankdir=LR;\n")
	sb.WriteString("  node [shape=box, style=rounded];\n\n")

	for _, module := range graph.Modules {
		fmt.Fprintf(&sb, "  \"%s\" [label=\"%s\"];\n", module.Name, module.Name)
	}

	sb.WriteString("\n")

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

	for _, conn := range graph.Connections {
		if conn.Type == "event" {
			fmt.Fprintf(&sb, "  \"%s\" -> \"%s\" [label=\"%s\", style=dotted, color=blue];\n",
				conn.From, conn.To, conn.Event)
		}
	}

	sb.WriteString("}\n")

	return os.WriteFile(filepath.Clean(filename), []byte(sb.String()), 0o600)
}

func outputHTML(graph *analyzer.Graph, filename, projectName string) error {
	html := generateHTML(graph, projectName)
	return os.WriteFile(filepath.Clean(filename), []byte(html), 0o600)
}

func serveWebUI(_ *analyzer.Graph, _ int) error {
	return fmt.Errorf("web server not yet implemented, use --format=html instead")
}

func generateHTML(graph *analyzer.Graph, projectName string) string {
	graphJSON, _ := json.Marshal(graph)
	graphJSONStr := string(graphJSON)

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>%s Module Graph</title>
    <script src="https://d3js.org/d3.v7.min.js"></script>
    <script src="https://cdnjs.cloudflare.com/ajax/libs/dagre/0.8.5/dagre.min.js"></script>
    <style>
        html, body {
            height: 100%%; margin: 0; padding: 0;
            font-family: 'Inter', -apple-system, blinkmacsystemfont, 'Segoe UI', roboto, monospace;
            background: #fdfdf5;
        }
        body { display: flex; flex-direction: column; padding: 0; box-sizing: border-box; }
        .container {
            flex: 1; display: flex; flex-direction: column; width: 100%%; margin: 0;
            background: transparent; padding: 20px; box-sizing: border-box; min-height: 0;
        }
        .header { display: flex; justify-content: space-between; align-items: flex-end; margin-bottom: 20px; flex-shrink: 0; }
        h1 { margin: 0; color: #000; font-size: 20px; font-weight: 800; letter-spacing: -0.5px; }
        .controls { flex-shrink: 0; margin-bottom: 15px; display: flex; justify-content: space-between; align-items: center; }
        .left-controls { display: flex; gap: 15px; align-items: center; }
        .view-btn {
            padding: 6px 14px; border: 1.5px solid #000; background: white; border-radius: 4px;
            cursor: pointer; font-size: 13px; font-weight: 700;
        }
        .view-btn.active { background: #000; color: #fdfdf5; }
        .orientation-toggle { display: flex; gap: 2px; border: 1.5px solid #000; padding: 2px; background: white; border-radius: 4px; }
        .orient-btn {
            padding: 2px 8px; border: none; background: transparent; cursor: pointer;
            font-size: 11px; font-weight: 800; color: #666;
        }
        .orient-btn.active { background: #000; color: #fff; border-radius: 2px; }
        .zoom-controls { display: flex; gap: 4px; }
        .zoom-btn {
            padding: 6px 10px; border: 1.5px solid #000; background: white; border-radius: 4px;
            cursor: pointer; font-size: 14px; font-weight: 800; display: flex; align-items: center; justify-content: center; min-width: 32px;
        }
        .legend { display: flex; gap: 15px; flex-wrap: wrap; }
        .legend-item { display: flex; align-items: center; gap: 6px; font-size: 11px; font-weight: 700; }
        .legend-line { width: 20px; height: 1.5px; background: #000; }
        .legend-line.dashed { background: transparent; border-top: 1.5px dashed #000; }
        .legend-rect { width: 12px; height: 12px; border: 1.5px solid #000; }
        #graph { flex: 1; width: 100%%; min-height: 0; border: 1.5px solid #000; background: #fdfdf5; cursor: grab; }
        #graph:active { cursor: grabbing; }
        .node { cursor: pointer; }
        .node rect.bg { fill: white; stroke: #000; stroke-width: 1.5px; }
        .node.gateway rect.bg { fill: #f0f4ff; stroke-dasharray: 4,2; }
        .node.gateway .title { fill: #0044cc; }
        .node.event rect.bg { fill: #000; }
        .node text { font-family: 'JetBrains Mono', 'SF Mono', monospace; }
        .node .title { font-size: 13px; fill: #000; font-weight: 800; }
        .node.event text { fill: #fff; font-size: 11px; font-weight: 700; text-anchor: middle; }
        .node.module .stats text { font-size: 9px; fill: #000; font-weight: 700; }
        .node.module .stats-val { font-weight: 800; }
        .node.module .sep { stroke: #000; stroke-width: 0.5px; opacity: 0.2; }
        .link { fill: none; stroke: #000; stroke-width: 1.5px; }
        .link.event { stroke-dasharray: 4,4; }
        .link.table-rel { stroke: #4a90e2; opacity: 0.4; }
        marker#arrowhead { fill: #000; }
        .tooltip {
            position: absolute; padding: 10px; background: #000; color: #fff; border-radius: 4px;
            pointer-events: none; font-size: 11px; display: none; z-index: 100; max-width: 250px;
        }
        .node, .link { transition: opacity 0.3s ease; }
        .node.dimmed, .link.dimmed { opacity: 0.1; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>%s</h1>
            <div class="legend" id="legend"></div>
        </div>
        <div class="controls">
            <div class="left-controls">
                <div class="view-toggle">
                    <button class="view-btn active" id="btn-service" onclick="switchView('service')">modules</button>
                    <button class="view-btn" id="btn-database" onclick="switchView('database')">database</button>
                </div>
                <div class="orientation-toggle" id="orient-controls">
                    <button class="orient-btn active" id="btn-lr" onclick="setOrientation('LR')">LR</button>
                    <button class="orient-btn" id="btn-tb" onclick="setOrientation('TB')">TB</button>
                </div>
            </div>
            <div class="zoom-controls">
                <button class="zoom-btn" onclick="zoomIn()">+</button>
                <button class="zoom-btn" onclick="zoomOut()">−</button>
                <button class="zoom-btn" onclick="resetZoom()">⟲</button>
            </div>
        </div>
        <svg id="graph"></svg>
    </div>
    <div class="tooltip" id="tooltip"></div>

    <script>
        const graphData = `+graphJSONStr+`;
        const svg = d3.select("#graph");
        const tooltip = d3.select("#tooltip");
        const legend = document.getElementById('legend');
        const orientControls = document.getElementById('orient-controls');

        svg.append("defs").append("marker")
            .attr("id", "arrowhead").attr("viewBox", "0 -5 10 10").attr("refX", 10).attr("refY", 0)
            .attr("markerWidth", 5).attr("markerHeight", 5).attr("orient", "auto")
            .append("path").attr("d", "M0,-5L10,0L0,5").attr("fill", "#000");

        const g = svg.append("g");
        let width, height;
        let simulation;
        let currentView = 'service';
        let rankDir = 'LR';

        const zoom = d3.zoom().scaleExtent([0.1, 8]).on("zoom", (event) => g.attr("transform", event.transform));
        svg.call(zoom);

        function zoomIn() { svg.transition().call(zoom.scaleBy, 1.3); }
        function zoomOut() { svg.transition().call(zoom.scaleBy, 0.7); }
        function resetZoom() { svg.transition().call(zoom.transform, d3.zoomIdentity); }

        function setOrientation(dir) {
            rankDir = dir;
            document.getElementById('btn-lr').classList.toggle('active', dir === 'LR');
            document.getElementById('btn-tb').classList.toggle('active', dir === 'TB');
            renderGraph();
            resetZoom();
        }

        function renderLegend() {
            if (currentView === 'service') {
                legend.innerHTML = `+"`"+`
                    <div class="legend-item"><div class="legend-line"></div><span>API</span></div>
                    <div class="legend-item"><div class="legend-line dashed"></div><span>Event</span></div>
                    <div class="legend-item"><div class="legend-rect" style="background:#000"></div><span>Event</span></div>
                `+"`"+`;
                orientControls.style.display = 'flex';
            } else {
                legend.innerHTML = `+"`"+`
                    <div class="legend-item"><div class="legend-rect" style="background:white"></div><span>Module</span></div>
                    <div class="legend-item"><div class="legend-rect" style="background:white;border-color:#4a90e2"></div><span>Table</span></div>
                `+"`"+`;
                orientControls.style.display = 'none';
            }
        }

        function switchView(view) {
            currentView = view;
            document.getElementById('btn-service').classList.toggle('active', view === 'service');
            document.getElementById('btn-database').classList.toggle('active', view === 'database');
            renderLegend();
            renderGraph();
            resetZoom();
        }

        function updateDimensions() {
            width = container.parentElement.clientWidth - 40;
            height = container.parentElement.clientHeight - 160;
            svg.attr("width", width).attr("height", height);
        }

        function renderGraph() {
            g.selectAll("*").remove();
            if (simulation) simulation.stop();

            let nodes = [];
            let links = [];

            if (currentView === 'service') {
                const nodeMap = new Map();
                const eventNodes = new Map();

                graphData.modules.forEach(m => {
                    nodeMap.set(m.name, {id: m.name, type: 'module', module: m, w: 150, h: 60});
                });

                graphData.connections.forEach(c => {
                    if (c.type === 'grpc') {
                        if (c.from === 'gateway') {
                            if (!nodeMap.has('gateway')) {
                                nodeMap.set('gateway', {id: 'gateway', type: 'gateway', module: {name: 'Gateway', services: []}, w: 150, h: 40});
                            }
                        }
                        const s = nodeMap.get(c.from);
                        const t = nodeMap.get(c.to);
                        if(s && t) links.push({source: s, target: t, type: 'grpc', id: c.from + "-" + c.to});
                    } else if (c.type === 'event') {
                        const eventId = "event_" + c.event;
                        if (!eventNodes.has(eventId)) {
                            // Dynamic width: approx 7-8px per char + padding
                            const w = Math.max(130, c.event.length * 8 + 10);
                            eventNodes.set(eventId, {id: eventId, type: 'event', name: c.event, w: w, h: 28});
                        }
                        const s = nodeMap.get(c.from);
                        const eNode = eventNodes.get(eventId);
                        const t = nodeMap.get(c.to);
                        if (s && eNode) links.push({source: s, target: eNode, type: 'event', id: s.id + "-" + eNode.id});
                        if (eNode && t) links.push({source: eNode, target: t, type: 'event', id: eNode.id + "-" + t.id});
                    }
                });

                nodes.push(...Array.from(nodeMap.values()));
                nodes.push(...Array.from(eventNodes.values()));

                const gDagre = new dagre.graphlib.Graph();
                // Ultra-compact spacing
                gDagre.setGraph({rankdir: rankDir, align: 'UL', nodesep: 10, ranksep: 20, marginx: 20, marginy: 20});
                gDagre.setDefaultEdgeLabel(() => ({}));

                nodes.forEach(n => gDagre.setNode(n.id, {width: n.w, height: n.h}));

                let gatewayOutCount = 0;
                links.forEach(l => {
                    let opts = {};
                    if (l.source.id === 'external' && l.target.type === 'module') {
                        gatewayOutCount++;
                        if (gatewayOutCount > 3) opts.minlen = 2;
                        if (gatewayOutCount > 6) opts.minlen = 3;
                    }
                    gDagre.setEdge(l.source.id, l.target.id, opts);
                });

                dagre.layout(gDagre);
                nodes.forEach(n => {
                    const d = gDagre.node(n.id);
                    n.x = d.x; n.y = d.y;
                });

            } else {
                graphData.modules.forEach(m => {
                    if (m.tables && m.tables.length > 0) {
                        // Dynamic height: base 50 + 16px per table
                        const tableCount = m.tables.length;
                        const h = Math.max(60, 40 + tableCount * 16);
                        nodes.push({id: m.name, type: 'module', module: m, w: 180, h: h});
                        m.tables.forEach(t => {
                            const tableId = m.name + "." + t;
                            nodes.push({id: tableId, type: 'table', name: t, parent: m.name, w: 140, h: 30});
                            links.push({source: m.name, target: tableId, type: 'table-rel', id: m.name + "-" + tableId});
                        });
                    }
                });
            }

            const pathFn = (d) => {
                if (currentView === 'service') {
                    if (rankDir === 'LR') {
                        const sx = d.source.x + d.source.w/2;
                        const sy = d.source.y;
                        const tx = d.target.x - d.target.w/2;
                        const ty = d.target.y;
                        const midX = (sx + tx) / 2;
                        return `+"`"+`M${sx},${sy} L${midX},${sy} L${midX},${ty} L${tx},${ty}`+"`"+`;
                    } else {
                        const sx = d.source.x;
                        const sy = d.source.y + d.source.h/2;
                        const tx = d.target.x;
                        const ty = d.target.y - d.target.h/2;
                        const midY = (sy + ty) / 2;
                        return `+"`"+`M${sx},${sy} L${sx},${midY} L${tx},${midY} L${tx},${ty}`+"`"+`;
                    }
                } else {
                    return `+"`"+`M${d.source.x},${d.source.y} L${d.target.x},${d.target.y}`+"`"+`;
                }
            };

            const link = g.append("g").selectAll("path").data(links).enter().append("path")
                .attr("class", d => "link " + d.type)
                .attr("marker-end", d => d.type === 'table-rel' ? "" : "url(#arrowhead)");

            const node = g.append("g").selectAll("g").data(nodes).enter().append("g")
                .attr("class", d => "node " + d.type);

            node.each(function(d) {
                const el = d3.select(this);
                if (d.type === 'module' || d.type === 'gateway') {
                    el.append("rect").attr("class", "bg").attr("width", d.w).attr("height", d.h).attr("x", -d.w/2).attr("y", -d.h/2);
                    const title = el.append("text").attr("class", "title").text(d.module.name || d.id);
                    if (d.type === 'gateway') {
                        title.attr("x", 0).attr("y", 0).attr("text-anchor", "middle").attr("dominant-baseline", "middle");
                    } else {
                        title.attr("x", -d.w/2 + 10).attr("y", -d.h/2 + 16);
                    }

                    // In database view, show table list inside module node
                    if (currentView === 'database' && d.module.tables && d.module.tables.length > 0) {
                        const tableList = el.append("g").attr("class", "table-list").attr("transform", `+"`"+`translate(${-d.w/2 + 10}, ${-d.h/2 + 30})`+"`"+`);
                        d.module.tables.forEach((tbl, i) => {
                            // Filter out TYPE entries for cleaner display
                            const displayName = tbl.replace(" (TYPE)", "");
                            const isType = tbl.includes("(TYPE)");
                            tableList.append("text")
                                .attr("y", i * 14)
                                .attr("x", 0)
                                .style("font-size", "10px")
                                .style("fill", isType ? "#888" : "#333")
                                .style("font-style", isType ? "italic" : "normal")
                                .text((isType ? "⬡ " : "⬢ ") + displayName);
                        });
                    }
                    if (currentView === 'service') {
                        if (d.type === 'gateway') {
                             return; // No stats for API Clients
                        }
                        const stats = el.append("g").attr("class", "stats").attr("transform", `+"`"+`translate(${-d.w/2 + 10}, ${-d.h/2 + 32})`+"`"+`);
                        const sCount = d.module.services?.length || 0;
                        const pubCount = sCount; // Total exposed services

                        // Calculate auth count: Services that have at least one non-public method
                        let authCount = 0;
                        const publicMethods = d.module.public_methods || [];
                        const serviceMethods = d.module.service_methods || {};

                        if (d.module.services) {
                            d.module.services.forEach(sName => {
                                const methods = serviceMethods[sName] || [];
                                // If no methods detected, assume secure (auth required)
                                if (methods.length === 0) {
                                    authCount++;
                                    return;
                                }

                                // Check if ALL methods are public
                                const ispublic = methods.every(mName => {
                                    return publicMethods.some(pm => pm.endsWith("/" + sName + "/" + mName));
                                });

                                if (!ispublic) {
                                    authCount++;
                                }
                            });
                        }

                        let xOff = 0;
                        const addStat = (label, val, icon) => {
                            const gStat = stats.append("g").attr("transform", `+"`"+`translate(${xOff}, 0)`+"`"+`);
                            gStat.append("text").text(icon).style("font-size", "10px");
                            gStat.append("text").text(val).attr("class", "stats-val").attr("dx", 12);
                            gStat.append("text").text(label).attr("dx", 24).style("font-weight", "400").style("fill", "#666");
                            xOff += 45;
                        };

                        addStat("pub", pubCount, "→");
                        addStat("auth", authCount, "🔐");
                        addStat("priv", "0", "🔒");

                        if (d.module.tables && d.module.tables.length > 0) {
                            const db = el.append("g").attr("class", "stats").attr("transform", `+"`"+`translate(${-d.w/2 + 10}, ${-d.h/2 + 46})`+"`"+`);
                            db.append("path")
                                .attr("d", "M3 5V19C3 21.2 7 23 12 23C17 23 21 21.2 21 19V5M3 5C3 7.2 7 9 12 9C17 9 21 7.2 21 5M3 5C3 2.8 7 1 12 1C17 1 21 2.8 21 5M3 12C3 14.2 7 16 12 16C17 16 21 14.2 21 12")
                                .attr("transform", "scale(0.5)")
                                .style("fill", "none")
                                .style("stroke", "#000")
                                .style("stroke-width", "2");
                            db.append("text").text(d.id).attr("dx", 16).attr("dy", 11).style("font-weight", "600");
                        }
                    }
                } else if (d.type === 'event') {
                    el.append("rect").attr("class", "bg").attr("width", d.w).attr("height", d.h).attr("x", -d.w/2).attr("y", -d.h/2).attr("rx", 4);
                    el.append("text").attr("y", 4).text(d.name);
                } else if (d.type === 'table') {
                    el.append("rect").attr("class", "bg").attr("width", d.w).attr("height", d.h).attr("x", -d.w/2).attr("y", -d.h/2).style("stroke", "#4a90e2");
                    el.append("text").attr("text-anchor", "middle").attr("y", 5).text(d.name).style("font-size", "11px");
                }
            });

            node.on("mouseover", function(event, d) {
                // Dim everything initially
                link.classed("dimmed", true);
                node.classed("dimmed", true);

                // Undim hovered node
                d3.select(this).classed("dimmed", false);

                // Find connected links and nodes
                const connectedLinks = link.filter(l => l.source.id === d.id || l.target.id === d.id);
                connectedLinks.classed("dimmed", false);

                connectedLinks.each(l => {
                    const peerId = l.source.id === d.id ? l.target.id : l.source.id;
                    node.filter(n => n.id === peerId).classed("dimmed", false);
                });

                if (d.type === 'module') {
                    const m = d.module;
                    let h = `+"`"+`<strong>${m.name}</strong><br/>Services: ${m.services?.length||0}<br/>Events: ${m.events?.length||0}<br/>Tables: ${m.tables?.length||0}`+"`"+`;
                    tooltip.style("display", "block").html(h).style("left", (event.pageX + 10) + "px").style("top", (event.pageY - 10) + "px");
                }
            }).on("mouseout", () => {
                link.classed("dimmed", false);
                node.classed("dimmed", false);
                tooltip.style("display", "none");
            });

            if (currentView === 'service') {
                link.attr("d", pathFn);
                node.attr("transform", d => "translate(" + d.x + "," + d.y + ")");
                setTimeout(() => resetZoom(), 100);
            } else {
                simulation = d3.forceSimulation(nodes)
                    .force("link", d3.forceLink(links).id(d => d.id).distance(40))
                    .force("charge", d3.forceManyBody().strength(-200))
                    .force("center", d3.forceCenter(width / 2, height / 2))
                    .force("collision", d3.forceCollide().radius(d => Math.max(d.w, d.h) / 2 + 5));

                simulation.on("tick", () => {
                    link.attr("d", d => pathFn(d));
                    node.attr("transform", d => "translate(" + d.x + "," + d.y + ")");
                });

                node.call(d3.drag()
                    .on("start", (event, d) => { if (!event.active) simulation.alphaTarget(0.3).restart(); d.fx = d.x; d.fy = d.y; })
                    .on("drag", (event, d) => { d.fx = event.x; d.fy = event.y; })
                    .on("end", (event, d) => { if (!event.active) simulation.alphaTarget(0); d.fx = null; d.fy = null; }));
            }
        }

        const container = document.querySelector('.container');
        window.addEventListener('resize', () => { updateDimensions(); renderGraph(); });
        updateDimensions();
        renderLegend();
        renderGraph();
    </script>
</body>
</html>`, projectName, projectName)
}
