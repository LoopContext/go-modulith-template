// Package main provides a tool to convert OpenAPI/Swagger JSON files to GraphQL schemas.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// OpenAPI2 represents the Swagger/OpenAPI 2.0 structure
type OpenAPI2 struct {
	Swagger string                 `json:"swagger"`
	Info    map[string]interface{} `json:"info"`
	Paths   map[string]PathItem    `json:"paths"`
	Defs    map[string]Schema      `json:"definitions"`
}

type PathItem struct {
	Get    *Operation `json:"get,omitempty"`
	Post   *Operation `json:"post,omitempty"`
	Put    *Operation `json:"put,omitempty"`
	Delete *Operation `json:"delete,omitempty"`
	Patch  *Operation `json:"patch,omitempty"`
}

type Operation struct {
	Summary     string              `json:"summary"`
	OperationID string              `json:"operationId"`
	Parameters  []Parameter         `json:"parameters"`
	Responses   map[string]Response `json:"responses"`
	Tags        []string            `json:"tags"`
}

type Parameter struct {
	Name     string     `json:"name"`
	In       string     `json:"in"`
	Required bool       `json:"required"`
	Schema   *SchemaRef `json:"schema,omitempty"`
	Type     string     `json:"type,omitempty"`
}

type Response struct {
	Description string     `json:"description"`
	Schema      *SchemaRef `json:"schema,omitempty"`
}

type SchemaRef struct {
	Ref string `json:"$ref"`
}

type Schema struct {
	Type       string              `json:"type"`
	Properties map[string]Property `json:"properties"`
	Required   []string            `json:"required"`
}

type Property struct {
	Type   string     `json:"type"`
	Format string     `json:"format,omitempty"`
	Items  *SchemaRef `json:"items,omitempty"`
	Ref    string     `json:"$ref,omitempty"`
}

const graphQLStringType = "String"

func main() {
	moduleName := flag.String("module", "", "Module name (e.g., auth)")
	openAPIPath := flag.String("openapi", "", "Path to OpenAPI/Swagger JSON file")
	outputPath := flag.String("output", "", "Output path for GraphQL schema file")

	flag.Parse()

	if *moduleName == "" && *openAPIPath == "" {
		fmt.Fprintf(os.Stderr, "Usage: %s -module <name> OR -openapi <path>\n", os.Args[0])
		os.Exit(1)
	}

	swaggerPath := determineSwaggerPath(*openAPIPath, *moduleName)
	output := determineOutputPath(*outputPath, *moduleName)

	openAPI := loadOpenAPIFile(swaggerPath)
	schema := generateGraphQLSchema(openAPI, *moduleName)
	writeSchemaFile(output, schema)

	fmt.Printf("✅ Generated GraphQL schema: %s\n", output)
}

func determineSwaggerPath(openAPIPath, moduleName string) string {
	switch {
	case openAPIPath != "":
		return openAPIPath
	case moduleName != "":
		return fmt.Sprintf("gen/openapiv2/proto/%s/v1/%s.swagger.json", moduleName, moduleName)
	default:
		fmt.Fprintf(os.Stderr, "Error: must specify either -module or -openapi\n")
		os.Exit(1)
	}

	return ""
}

func determineOutputPath(outputPath, moduleName string) string {
	switch {
	case outputPath != "":
		return outputPath
	case moduleName != "":
		return fmt.Sprintf("internal/graphql/schema/%s.graphql", moduleName)
	default:
		fmt.Fprintf(os.Stderr, "Error: must specify -output when using -openapi\n")
		os.Exit(1)
	}

	return ""
}

func loadOpenAPIFile(swaggerPath string) OpenAPI2 {
	// #nosec G304 -- file path is controlled by user input or module name
	data, err := os.ReadFile(swaggerPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading OpenAPI file: %v\n", err)
		os.Exit(1)
	}

	var openAPI OpenAPI2
	if err := json.Unmarshal(data, &openAPI); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing OpenAPI JSON: %v\n", err)
		os.Exit(1)
	}

	return openAPI
}

func writeSchemaFile(output, schema string) {
	// #nosec G301 -- 0755 is appropriate for directory permissions
	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	// #nosec G306 -- 0644 is appropriate for schema file permissions
	if err := os.WriteFile(output, []byte(schema), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing schema file: %v\n", err)
		os.Exit(1)
	}
}

func generateGraphQLSchema(openAPI OpenAPI2, _ string) string {
	var sb strings.Builder

	// Header (GraphQL uses # for comments, not //)
	sb.WriteString("# Auto-generated GraphQL schema from OpenAPI/Swagger definition\n")
	sb.WriteString("# DO NOT EDIT - This file is generated from proto definitions\n")
	sb.WriteString("# Run 'make proto' and 'make graphql-from-proto' to regenerate\n\n")

	// Generate types from definitions
	generateTypes(&sb, openAPI.Defs)

	// Generate queries and mutations from paths
	generateOperations(&sb, openAPI.Paths)

	return sb.String()
}

//nolint:cyclop,funlen // Code generation logic has inherent complexity
func generateTypes(sb *strings.Builder, defs map[string]Schema) {
	// Separate request types (inputs) from response types (outputs)
	requestTypes := make(map[string]bool)
	responseTypes := make(map[string]bool)

	// Identify request vs response types by name patterns
	for name := range defs {
		graphQLTypeName := toGraphQLTypeName(name)
		// Request types, Input types, and Body types should be inputs
		if strings.HasSuffix(graphQLTypeName, "Request") ||
			strings.HasSuffix(graphQLTypeName, "Input") ||
			strings.HasSuffix(graphQLTypeName, "Body") {
			requestTypes[name] = true
		} else {
			responseTypes[name] = true
		}
	}

	// Sort definitions for consistent output
	typeNames := make([]string, 0, len(defs))
	for name := range defs {
		typeNames = append(typeNames, name)
	}

	// Generate input types first
	for _, typeName := range typeNames {
		if !requestTypes[typeName] {
			continue
		}

		schema := defs[typeName]
		graphQLTypeName := toGraphQLTypeName(typeName)

		fmt.Fprintf(sb, "input %s {\n", graphQLTypeName)

		if schema.Properties != nil {
			for fieldName, prop := range schema.Properties {
				graphQLFieldName := toGraphQLFieldName(fieldName)
				graphQLType := openAPITypeToGraphQL(prop, defs)

				// Check if required
				required := ""
				if isRequired(fieldName, schema.Required) {
					required = "!"
				}

				fmt.Fprintf(sb, "  %s: %s%s\n", graphQLFieldName, graphQLType, required)
			}
		}

		sb.WriteString("}\n\n")
	}

	// Generate output types
	for _, typeName := range typeNames {
		if requestTypes[typeName] {
			continue
		}

		schema := defs[typeName]
		graphQLTypeName := toGraphQLTypeName(typeName)

		fmt.Fprintf(sb, "type %s {\n", graphQLTypeName)

		if schema.Properties != nil {
			for fieldName, prop := range schema.Properties {
				graphQLFieldName := toGraphQLFieldName(fieldName)
				graphQLType := openAPITypeToGraphQL(prop, defs)

				// Check if required
				required := ""
				if isRequired(fieldName, schema.Required) {
					required = "!"
				}

				fmt.Fprintf(sb, "  %s: %s%s\n", graphQLFieldName, graphQLType, required)
			}
		}

		sb.WriteString("}\n\n")
	}
}

//nolint:gocognit,cyclop // Code generation logic has inherent complexity
func generateOperations(sb *strings.Builder, paths map[string]PathItem) {
	var queries []string

	var mutations []string

	for path, item := range paths {
		// Process GET operations (queries)
		if item.Get != nil {
			op := generateOperation(item.Get, path, "query")
			if op != "" {
				queries = append(queries, op)
			}
		}

		// Process POST/PUT/DELETE/PATCH operations (mutations)
		if item.Post != nil {
			op := generateOperation(item.Post, path, "mutation")
			if op != "" {
				mutations = append(mutations, op)
			}
		}

		if item.Put != nil {
			op := generateOperation(item.Put, path, "mutation")
			if op != "" {
				mutations = append(mutations, op)
			}
		}

		if item.Delete != nil {
			op := generateOperation(item.Delete, path, "mutation")
			if op != "" {
				mutations = append(mutations, op)
			}
		}

		if item.Patch != nil {
			op := generateOperation(item.Patch, path, "mutation")
			if op != "" {
				mutations = append(mutations, op)
			}
		}
	}

	// Generate extend type Query
	if len(queries) > 0 {
		sb.WriteString("extend type Query {\n")

		for _, query := range queries {
			sb.WriteString("  " + query + "\n")
		}

		sb.WriteString("}\n\n")
	}

	// Generate extend type Mutation
	if len(mutations) > 0 {
		sb.WriteString("extend type Mutation {\n")

		for _, mutation := range mutations {
			sb.WriteString("  " + mutation + "\n")
		}

		sb.WriteString("}\n\n")
	}
}

func generateOperation(op *Operation, _ string, _ string) string {
	opName := toGraphQLOperationName(op.OperationID)
	if opName == "" {
		return ""
	}

	// Determine input type
	inputType := graphQLStringType // Default

	if len(op.Parameters) > 0 {
		// Find body parameter
		for _, param := range op.Parameters {
			if param.In == "body" && param.Schema != nil {
				inputType = refToGraphQLType(param.Schema.Ref)
				break
			}
		}
	}

	// Determine output type
	outputType := "Boolean" // Default
	if resp, ok := op.Responses["200"]; ok && resp.Schema != nil {
		outputType = refToGraphQLType(resp.Schema.Ref)
	}

	return fmt.Sprintf("%s(input: %s): %s", opName, inputType, outputType)
}

//nolint:cyclop // Type conversion logic has inherent complexity
func openAPITypeToGraphQL(prop Property, _ map[string]Schema) string {
	if prop.Ref != "" {
		return refToGraphQLType(prop.Ref)
	}

	if prop.Items != nil {
		itemType := refToGraphQLType(prop.Items.Ref)
		return fmt.Sprintf("[%s!]", itemType)
	}

	switch prop.Type {
	case "string":
		if prop.Format == "date-time" || prop.Format == "date" {
			return graphQLStringType // Could be custom scalar
		}

		return graphQLStringType
	case "integer", "int32", "int64":
		return "Int"
	case "number", "float", "double":
		return "Float"
	case "boolean":
		return "Boolean"
	case "array":
		if prop.Items != nil {
			itemType := refToGraphQLType(prop.Items.Ref)
			return fmt.Sprintf("[%s!]", itemType)
		}

		return fmt.Sprintf("[%s!]", graphQLStringType)
	default:
		return graphQLStringType
	}
}

func refToGraphQLType(ref string) string {
	// Convert "#/definitions/v1User" -> "User"
	if ref == "" {
		return graphQLStringType
	}

	parts := strings.Split(ref, "/")
	if len(parts) > 0 {
		typeName := parts[len(parts)-1]
		// Remove "v1" prefix if present
		typeName = strings.TrimPrefix(typeName, "v1")

		return toGraphQLTypeName(typeName)
	}

	return graphQLStringType
}

func toGraphQLTypeName(name string) string {
	// Remove common prefixes
	name = strings.TrimPrefix(name, "v1")
	name = strings.TrimPrefix(name, "V1")

	return name
}

func toGraphQLFieldName(name string) string {
	// Handle special cases
	if name == "@type" || name == "type" {
		return "type_" // GraphQL doesn't allow @ in field names
	}

	// Convert snake_case to camelCase
	parts := strings.Split(name, "_")
	result := parts[0]

	// Remove @ prefix if present
	result = strings.TrimPrefix(result, "@")

	for i := 1; i < len(parts); i++ {
		if len(parts[i]) > 0 {
			result += strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}

	return result
}

func toGraphQLOperationName(opID string) string {
	// Convert "AuthService_RequestLogin" -> "requestLogin"
	parts := strings.Split(opID, "_")
	if len(parts) > 1 {
		methodName := parts[len(parts)-1]
		if len(methodName) > 0 {
			return strings.ToLower(methodName[:1]) + methodName[1:]
		}
	}

	return strings.ToLower(opID[:1]) + opID[1:]
}

func isRequired(fieldName string, required []string) bool {
	for _, req := range required {
		if req == fieldName {
			return true
		}
	}

	return false
}
