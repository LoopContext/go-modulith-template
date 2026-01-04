// Package analyzer analyzes the modulith codebase to extract module connections.
package analyzer

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Graph represents the complete module dependency graph.
type Graph struct {
	Modules     []Module     `json:"modules"`
	Connections []Connection `json:"connections"`
}

// Module represents a single module in the system.
type Module struct {
	Name     string   `json:"name"`
	Services []string `json:"services"`
	Events   []string `json:"events"`
}

// Connection represents a connection between modules.
type Connection struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Type      string `json:"type"` // "grpc" or "event"
	Service   string `json:"service,omitempty"`
	Event     string `json:"event,omitempty"`
	Direction string `json:"direction,omitempty"` // "inbound" or "outbound"
}

// Analyze scans the codebase and builds the module graph.
func Analyze(projectRoot string) (*Graph, error) {
	graph := &Graph{
		Modules:     []Module{},
		Connections: []Connection{},
	}

	// Step 1: Discover modules
	modulesDir := filepath.Join(projectRoot, "modules")

	modules, err := discoverModules(modulesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to discover modules: %w", err)
	}

	// Step 2: Analyze each module
	for _, moduleName := range modules {
		modulePath := filepath.Join(modulesDir, moduleName)

		module := analyzeModule(projectRoot, moduleName, modulePath)

		graph.Modules = append(graph.Modules, *module)
	}

	// Step 3: Find gRPC connections by scanning proto files
	protoDir := filepath.Join(projectRoot, "proto")
	if err := analyzeProtoConnections(projectRoot, protoDir, graph); err != nil {
		return nil, fmt.Errorf("failed to analyze proto connections: %w", err)
	}

	// Step 4: Find event connections by scanning Go code
	if err := analyzeEventConnections(projectRoot, modulesDir, graph); err != nil {
		return nil, fmt.Errorf("failed to analyze event connections: %w", err)
	}

	return graph, nil
}

func discoverModules(modulesDir string) ([]string, error) {
	entries, err := os.ReadDir(modulesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read modules directory: %w", err)
	}

	var modules []string

	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			// Check if it has a module.go file
			moduleGo := filepath.Join(modulesDir, entry.Name(), "module.go")
			if _, err := os.Stat(moduleGo); err == nil {
				modules = append(modules, entry.Name())
			}
		}
	}

	return modules, nil
}

func analyzeModule(projectRoot, moduleName, _ string) *Module {
	module := &Module{
		Name:     moduleName,
		Services: []string{},
		Events:   []string{},
	}

	// Find proto files for this module (check subdirectories like v1/, v2/, etc.)
	protoPath := filepath.Join(projectRoot, "proto", moduleName)

	if err := filepath.Walk(protoPath, func(path string, _ os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if strings.HasSuffix(path, ".proto") {
			services, err := extractServicesFromProto(path)
			if err == nil {
				module.Services = append(module.Services, services...)
			}
		}

		return nil
	}); err != nil {
		// Proto path might not exist, that's okay - ignore the error
		_ = err
	}

	return module
}

func extractServicesFromProto(protoFile string) ([]string, error) {
	// Validate file path to prevent directory traversal
	if !filepath.IsAbs(protoFile) {
		absPath, err := filepath.Abs(protoFile)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve absolute path: %w", err)
		}

		protoFile = absPath
	}

	data, err := os.ReadFile(filepath.Clean(protoFile))
	if err != nil {
		return nil, fmt.Errorf("failed to read proto file: %w", err)
	}

	var services []string

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "service ") {
			// Extract service name: "service AuthService {"
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				serviceName := strings.TrimSuffix(parts[1], "{")
				services = append(services, serviceName)
			}
		}
	}

	return services, nil
}

//nolint:cyclop // Complex function needed to analyze proto connections
func analyzeProtoConnections(_ string, protoDir string, graph *Graph) error {
	// Walk through proto files and find service definitions
	if err := filepath.Walk(protoDir, func(path string, _ os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(path, ".proto") {
			return nil
		}

		// Extract module name from path: proto/{module}/v1/... or proto/{module}/...
		relPath, err := filepath.Rel(protoDir, path)
		if err != nil {
			return nil
		}

		parts := strings.Split(relPath, string(filepath.Separator))
		if len(parts) < 1 {
			return nil
		}

		moduleName := parts[0]
		if moduleName == "google" {
			return nil // Skip Google proto files
		}

		// Update module's services list
		serviceName := ""

		// Read proto file to find services
		cleanPath := filepath.Clean(path)

		data, err := os.ReadFile(cleanPath)
		if err != nil {
			return nil
		}

		// Simple parsing: find service definitions
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "service ") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					serviceName = strings.TrimSuffix(parts[1], "{")
					// Add connection: module exposes this service
					graph.Connections = append(graph.Connections, Connection{
						From:      "external",
						To:        moduleName,
						Type:      "grpc",
						Service:   serviceName,
						Direction: "inbound",
					})
					// Also add to module's services list
					updateModuleServices(graph, moduleName, serviceName)
				}
			}
		}

		return nil
	}); err != nil {
		return fmt.Errorf("failed to walk proto directory: %w", err)
	}

	return nil
}

//nolint:gocognit,cyclop,funlen // Complex function needed to analyze event connections
func analyzeEventConnections(projectRoot, modulesDir string, graph *Graph) error {
	// Map to track event publishers
	eventPublishers := make(map[string]string) // event name -> module name

	// First pass: find all event publications
	err := filepath.Walk(modulesDir, func(path string, _ os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		cleanPath := filepath.Clean(path)

		data, err := os.ReadFile(cleanPath)
		if err != nil {
			return nil
		}

		content := string(data)

		// Find module name from path
		relPath, err := filepath.Rel(modulesDir, path)
		if err != nil {
			return nil
		}

		parts := strings.Split(relPath, string(filepath.Separator))
		if len(parts) < 1 {
			return nil
		}

		moduleName := parts[0]

		// Find Publish calls with event names
		// Pattern: bus.Publish(ctx, events.Event{Name: "event.name", ...})
		// Or: events.Event{Name: events.EventUserCreated, ...}
		// Or: events.Event{Name: notifier.EventMagicCodeRequested, ...}
		// Handle multi-line patterns - look for Name: field after Event{
		publishRegex := regexp.MustCompile(`Event\s*\{[^}]*Name:\s*([^,}\n]+)`)

		matches := publishRegex.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			if len(match) > 1 {
				eventName := strings.TrimSpace(match[1])
				eventName = strings.Trim(eventName, `"`)

				// Handle constants from different packages
				if strings.Contains(eventName, ".") {
					// It's a constant like "events.EventUserCreated" or "notifier.EventMagicCodeRequested"
					eventName = resolveEventConstant(projectRoot, eventName)
				}

				if eventName != "" {
					eventPublishers[eventName] = moduleName
					// Also add to module's events list
					updateModuleEvents(graph, moduleName, eventName)
				}
			}
		}

		// Also look for direct string literals in Publish calls (multi-line aware)
		publishStringRegex := regexp.MustCompile(`Name:\s*"([^"]+)"`)

		matches2 := publishStringRegex.FindAllStringSubmatch(content, -1)
		for _, match := range matches2 {
			if len(match) > 1 {
				eventName := match[1]
				eventPublishers[eventName] = moduleName
				updateModuleEvents(graph, moduleName, eventName)
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to walk modules directory: %w", err)
	}

	// Second pass: find subscriptions and create connections
	if err := filepath.Walk(modulesDir, func(path string, _ os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		cleanPath := filepath.Clean(path)
		data, err := os.ReadFile(cleanPath)
		if err != nil {
			return nil
		}

		content := string(data)

		// Find module name from path
		relPath, err := filepath.Rel(modulesDir, path)
		if err != nil {
			return nil
		}

		parts := strings.Split(relPath, string(filepath.Separator))
		if len(parts) < 1 {
			return nil
		}

		moduleName := parts[0]

		// Find Subscribe calls
		// Pattern: bus.Subscribe("event.name", handler)
		// Or: eventBus.Subscribe(events.EventUserCreated, handler)
		subscribeRegex := regexp.MustCompile(`\.Subscribe\(([^,)]+)`)

		matches := subscribeRegex.FindAllStringSubmatch(content, -1)

		for _, match := range matches {
			if len(match) > 1 {
				eventName := strings.Trim(match[1], `" `)
				eventName = strings.Trim(eventName, `"`)

				if strings.HasPrefix(eventName, "events.") {
					eventName = resolveEventConstant(projectRoot, eventName)
				}

				if eventName != "" {
					// Find publisher
					if publisher, ok := eventPublishers[eventName]; ok && publisher != moduleName {
						graph.Connections = append(graph.Connections, Connection{
							From:  publisher,
							To:    moduleName,
							Type:  "event",
							Event: eventName,
						})
					}
				}
			}
		}

		return nil
	}); err != nil {
		return fmt.Errorf("failed to walk modules directory for subscriptions: %w", err)
	}

	return nil
}

func resolveEventConstant(projectRoot, constant string) string {
	typesFile, constantName := determineConstantFile(projectRoot, constant)
	if typesFile == "" || constantName == "" {
		return ""
	}

	return extractConstantValue(typesFile, constantName)
}

func determineConstantFile(projectRoot, constant string) (string, string) {
	switch {
	case strings.HasPrefix(constant, "events."):
		typesFile := filepath.Join(projectRoot, "internal", "events", "types.go")
		constantName := strings.TrimPrefix(constant, "events.")

		return typesFile, constantName
	case strings.HasPrefix(constant, "notifier."):
		typesFile := filepath.Join(projectRoot, "internal", "notifier", "subscriber.go")
		constantName := strings.TrimPrefix(constant, "notifier.")

		return typesFile, constantName
	default:
		return "", ""
	}
}

func extractConstantValue(typesFile, constantName string) string {
	cleanPath := filepath.Clean(typesFile)

	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return ""
	}

	lines := strings.Split(string(data), "\n")

	for i, line := range lines {
		if value := extractFromLine(line, constantName); value != "" {
			return value
		}

		// Check for multi-line definitions
		if strings.Contains(line, constantName+" =") {
			if value := extractFromNextLines(lines, i, constantName); value != "" {
				return value
			}
		}
	}

	return ""
}

func extractFromLine(line, constantName string) string {
	if !strings.Contains(line, constantName+" = ") {
		return ""
	}

	if idx := strings.Index(line, `"`); idx != -1 {
		if endIdx := strings.Index(line[idx+1:], `"`); endIdx != -1 {
			return line[idx+1 : idx+1+endIdx]
		}
	}

	return ""
}

func extractFromNextLines(lines []string, startIdx int, _ string) string {
	for j := startIdx + 1; j < len(lines) && j < startIdx+5; j++ {
		if idx := strings.Index(lines[j], `"`); idx != -1 {
			if endIdx := strings.Index(lines[j][idx+1:], `"`); endIdx != -1 {
				return lines[j][idx+1 : idx+1+endIdx]
			}
		}
	}

	return ""
}

func updateModuleServices(graph *Graph, moduleName, serviceName string) {
	// Find the module and add the service to its list
	for i := range graph.Modules {
		if graph.Modules[i].Name == moduleName {
			// Check if service already exists
			found := false

			for _, s := range graph.Modules[i].Services {
				if s == serviceName {
					found = true

					break
				}
			}

			if !found {
				graph.Modules[i].Services = append(graph.Modules[i].Services, serviceName)
			}

			break
		}
	}
}

func updateModuleEvents(graph *Graph, moduleName, eventName string) {
	// Find the module and add the event to its list
	for i := range graph.Modules {
		if graph.Modules[i].Name == moduleName {
			// Check if event already exists
			found := false

			for _, e := range graph.Modules[i].Events {
				if e == eventName {
					found = true

					break
				}
			}

			if !found {
				graph.Modules[i].Events = append(graph.Modules[i].Events, eventName)
			}

			break
		}
	}
}
