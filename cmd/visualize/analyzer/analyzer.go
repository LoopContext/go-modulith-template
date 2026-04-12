// Package analyzer analyzes the modulith codebase to extract module connections.
//
//nolint:revive,wrapcheck,gosec,gocognit,cyclop,gocritic,gocyclo,nestif,funlen,unparam // This is a dev tool, lenient linting
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
	Name           string              `json:"name"`
	Services       []string            `json:"services"`
	Events         []string            `json:"events"`
	Tables         []string            `json:"tables"`
	PublicMethods  []string            `json:"public_methods"`
	ServiceMethods map[string][]string `json:"service_methods"` // Service Name -> []Method Name
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

func analyzeModule(projectRoot, moduleName, modulePath string) *Module {
	module := &Module{
		Name:           moduleName,
		Services:       []string{},
		Events:         []string{},
		PublicMethods:  []string{},
		ServiceMethods: make(map[string][]string),
	}

	// Analyze public endpoints from module.go
	if publicMethods, err := analyzeModuleAuth(modulePath); err == nil {
		module.PublicMethods = publicMethods
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
				for sName, sMethods := range services {
					module.Services = append(module.Services, sName)
					module.ServiceMethods[sName] = sMethods
				}
			}
		}

		return nil
	}); err != nil {
		// Proto path might not exist, that's okay - ignore the error
		_ = err
	}

	// Analyze database tables
	// Look in modules/{moduleName}/resources/db/migration
	migrationDir := filepath.Join(projectRoot, "modules", moduleName, "resources", "db", "migration")

	tables, err := analyzeDatabase(migrationDir)
	if err == nil {
		module.Tables = tables
	}

	return module
}

func analyzeDatabase(migrationDir string) ([]string, error) {
	var tables []string

	seenTables := make(map[string]bool)

	// Check if directory exists
	if _, err := os.Stat(migrationDir); os.IsNotExist(err) {
		return nil, nil // No migrations for this module
	}

	err := filepath.Walk(migrationDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// Only look at .up.sql files
		if !strings.HasSuffix(path, ".up.sql") {
			return nil
		}

		extractedTables, err := extractTablesFromSQL(path)
		if err != nil {
			return nil // Skip file on error
		}

		for _, table := range extractedTables {
			if !seenTables[table] {
				tables = append(tables, table)
				seenTables[table] = true
			}
		}

		return nil
	})

	return tables, err
}

func extractTablesFromSQL(sqlFile string) ([]string, error) {
	data, err := os.ReadFile(filepath.Clean(sqlFile))
	if err != nil {
		return nil, fmt.Errorf("failed to read SQL file: %w", err)
	}

	var tables []string

	content := string(data)

	// Regex to match CREATE TABLE statements
	// Matches: CREATE TABLE [IF NOT EXISTS] table_name
	// Ignore case
	tableRegex := regexp.MustCompile(`(?i)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?([a-zA-Z0-9_]+)`)
	matches := tableRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) > 1 {
			tables = append(tables, match[1])
		}
	}

	// Also look for CREATE TYPE statements (enums)
	typeRegex := regexp.MustCompile(`(?i)CREATE\s+TYPE\s+([a-zA-Z0-9_]+)`)
	typeMatches := typeRegex.FindAllStringSubmatch(content, -1)

	for _, match := range typeMatches {
		if len(match) > 1 {
			tables = append(tables, match[1]+" (TYPE)")
		}
	}

	return tables, nil
}

func extractServicesFromProto(protoFile string) (map[string][]string, error) {
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

	services := make(map[string][]string)

	lines := strings.Split(string(data), "\n")

	var currentService string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Detect service start
		if strings.HasPrefix(line, "service ") {
			// Extract service name: "service AuthService {"
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				serviceName := strings.TrimSuffix(parts[1], "{")
				currentService = serviceName
				services[currentService] = []string{}
			}
		} else if currentService != "" && strings.Contains(line, "rpc ") && strings.Contains(line, "(") {
			// Very basic RPC detection: rpc Login(LoginRequest) returns (LoginResponse) {}
			// or rpc Login (LoginRequest) returns (LoginResponse);
			parts := strings.Fields(line)
			if len(parts) >= 2 && parts[0] == "rpc" {
				rpcName := parts[1]
				// Handle "rpc Login(" case
				if idx := strings.Index(rpcName, "("); idx != -1 {
					rpcName = rpcName[:idx]
				}

				services[currentService] = append(services[currentService], rpcName)
			}
		} else if strings.Contains(line, "}") && currentService != "" {
			// End of service block (naive but works for standard formatting)
			if strings.HasPrefix(line, "}") {
				currentService = ""
			}
		}
	}

	return services, nil
}

// analyzeModuleAuth scans the module directory for PublicEndpoints method.
func analyzeModuleAuth(moduleDir string) ([]string, error) {
	var publicMethods []string

	// Look for module.go
	moduleGo := filepath.Join(moduleDir, "module.go")
	if _, err := os.Stat(moduleGo); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(moduleGo)
	if err != nil {
		return nil, err
	}

	content := string(data)

	// Regex to find the return []string{ ... } block inside PublicEndpoints
	// This is a bit brittle with regex but sufficient for this specific codebase convention.
	// func (m *Module) PublicEndpoints() []string {
	// 	return []string{
	// 		"/auth.v1.AuthService/RequestLogin",
	//      ...
	// 	}
	// }

	// Find the start of the function
	funcStart := strings.Index(content, "PublicEndpoints() []string")
	if funcStart == -1 {
		return nil, nil // Not implemented
	}

	// Extract the content after function definition
	rest := content[funcStart:]

	// Find strings starting with "/" inside quotes
	// Matches: "/package.Service/Method"
	re := regexp.MustCompile(`"(/[a-zA-Z0-9_./]+)"`)
	matches := re.FindAllStringSubmatch(rest, -1)

	for _, match := range matches {
		if len(match) > 1 {
			// Ensure verify it looks like a gRPC path
			if strings.Count(match[1], "/") >= 2 {
				publicMethods = append(publicMethods, match[1])
			}
		}
	}

	return publicMethods, nil
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
						From:      "gateway",
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

// analyzeEventConnections scans the codebase for event publications and subscriptions.
func analyzeEventConnections(projectRoot, modulesDir string, graph *Graph) error {
	// Map to track event publishers
	eventPublishers := make(map[string]string) // event name -> module name

	internalDir := filepath.Join(projectRoot, "internal")
	scanDirs := []string{modulesDir, internalDir}

	moduleMap := make(map[string]bool)
	for _, m := range graph.Modules {
		moduleMap[m.Name] = true
	}

	// First pass: find all event publications
	for _, dir := range scanDirs {
		_ = filepath.Walk(dir, func(path string, _ os.FileInfo, err error) error {
			if err != nil || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}

			data, err := os.ReadFile(filepath.Clean(path))
			if err != nil {
				return nil
			}

			content := string(data)

			// Find module name from path
			relPath, _ := filepath.Rel(dir, path)

			parts := strings.Split(relPath, string(filepath.Separator))
			if len(parts) < 1 {
				return nil
			}

			moduleName := parts[0]

			// Skip internal packages that don't map to modules
			if dir == internalDir && !moduleMap[moduleName] {
				return nil
			}

			// Better multi-line regexes
			patterns := []*regexp.Regexp{
				// Event{Name: "..."}
				regexp.MustCompile(`(?s)Event\s*\{[^}]*Name:\s*("([^"]+)"|([a-zA-Z0-9_.]+))`),
				// .StoreOutbox(ctx, "...")
				regexp.MustCompile(`(?s)StoreOutbox\s*\(\s*[^,]+\s*,\s*("([^"]+)"|([a-zA-Z0-9_.]+))`),
				// .Publish(ctx, "...")
				regexp.MustCompile(`(?s)Publish\s*\(\s*[^,]+\s*,\s*("([^"]+)"|([a-zA-Z0-9_.]+))`),
				// audit.Log(...) always publishes EventAuditLogCreated
				regexp.MustCompile(`(?s)audit\.Log\s*\(`),
			}

			for _, re := range patterns {
				matches := re.FindAllStringSubmatchIndex(content, -1)
				for _, matchIdx := range matches {
					eventName := ""
					isConstant := false

					// Specific handling for audit.Log
					if strings.Contains(re.String(), "audit\\.Log") {
						eventName = "audit.log.created" // This is the value of EventAuditLogCreated
					} else if matchIdx[4] != -1 {
						eventName = content[matchIdx[4]:matchIdx[5]]
					} else if matchIdx[6] != -1 {
						eventName = content[matchIdx[6]:matchIdx[7]]
						isConstant = true
					}

					start := matchIdx[0]

					lookBackLimit := 0
					if start > 40 {
						lookBackLimit = start - 40
					}

					preceding := content[lookBackLimit:start]
					if strings.Contains(preceding, "func ") || strings.Contains(preceding, "interface {") {
						continue
					}

					if eventName == "" || eventName == "eventName" || eventName == "name" || eventName == "ctx" || eventName == "arg" || eventName == "event" {
						continue
					}

					if isConstant && strings.Contains(eventName, ".") {
						eventName = resolveEventConstant(projectRoot, eventName)
					}

					if eventName != "" {
						eventPublishers[eventName] = moduleName
						updateModuleEvents(graph, moduleName, eventName)
					}
				}
			}

			return nil
		})
	}

	// Second pass: find subscriptions and create connections
	for _, dir := range scanDirs {
		_ = filepath.Walk(dir, func(path string, _ os.FileInfo, err error) error {
			if err != nil || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}

			data, _ := os.ReadFile(filepath.Clean(path))
			content := string(data)

			relPath, _ := filepath.Rel(dir, path)

			parts := strings.Split(relPath, string(filepath.Separator))
			if len(parts) < 1 {
				return nil
			}

			moduleName := parts[0]

			if dir == internalDir && !moduleMap[moduleName] {
				return nil
			}

			re := regexp.MustCompile(`(?s)Subscribe\s*\(\s*("([^"]+)"|([a-zA-Z0-9_.]+))`)
			matches := re.FindAllStringSubmatchIndex(content, -1)

			for _, matchIdx := range matches {
				eventName := ""
				isConstant := false

				if matchIdx[4] != -1 {
					eventName = content[matchIdx[4]:matchIdx[5]]
				} else if matchIdx[6] != -1 {
					eventName = content[matchIdx[6]:matchIdx[7]]
					isConstant = true
				}

				start := matchIdx[0]

				lookBackLimit := 0
				if start > 20 {
					lookBackLimit = start - 20
				}

				if strings.Contains(content[lookBackLimit:start], "func ") {
					continue
				}

				if eventName == "" || eventName == "eventName" || eventName == "name" || eventName == "event" {
					continue
				}

				if isConstant && strings.Contains(eventName, ".") {
					eventName = resolveEventConstant(projectRoot, eventName)
				}

				if eventName != "" {
					if publisher, ok := eventPublishers[eventName]; ok {
						if publisher != moduleName {
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
		})
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
	case strings.HasPrefix(constant, "internalEvents."):
		typesFile := filepath.Join(projectRoot, "internal", "events", "types.go")
		constantName := strings.TrimPrefix(constant, "internalEvents.")

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
	// Trim spaces to handle indentation
	line = strings.TrimSpace(line)

	// Check if line starts with constant name
	if !strings.HasPrefix(line, constantName) {
		return ""
	}

	rest := strings.TrimPrefix(line, constantName)

	// Ensure exact match (next character must be whitespace or =)
	if len(rest) > 0 && rest[0] != ' ' && rest[0] != '\t' && rest[0] != '=' {
		return ""
	}

	// Check if followed by = (allowing for spaces/tabs)
	if !strings.Contains(rest, "=") {
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
