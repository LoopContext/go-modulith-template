// Package swagger provides utilities for serving Swagger UI and OpenAPI specifications.
package swagger

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/cmelgarejo/go-modulith-template/internal/version"
)

const (
	swaggerBasePath   = "gen/openapiv2/proto"
	swaggerUITemplate = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <title>Swagger UI</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5.11.0/swagger-ui.css" />
</head>
<body>
<div id="swagger-ui"></div>
<script src="https://unpkg.com/swagger-ui-dist@5.11.0/swagger-ui-bundle.js"></script>
<script>
window.onload = () => {
  window.ui = SwaggerUIBundle({
    url: '/swagger.json',
    dom_id: '#swagger-ui',
    persistAuthorization: true,
    requestInterceptor: (request) => {
      if (request.headers && request.headers.Authorization) {
        const auth = request.headers.Authorization.trim();
        if (auth && !auth.startsWith('Bearer ')) {
          request.headers.Authorization = 'Bearer ' + auth;
        }
      }
      return request;
    },
  });
};
</script>
</body>
</html>`
)

// Setup registers Swagger UI endpoints on the provided mux.
// It automatically discovers and merges Swagger specs from all modules.
func Setup(mux *http.ServeMux, apiTitle string) {
	slog.Info("Serving Swagger UI", "path", "/swagger-ui/")

	mux.HandleFunc("/swagger.json", func(w http.ResponseWriter, r *http.Request) {
		handleSwaggerJSON(w, r, apiTitle)
	})
	mux.HandleFunc("/swagger-ui/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")

		if _, err := w.Write([]byte(swaggerUITemplate)); err != nil {
			slog.Error("failed to write swagger-ui response", "error", err)
		}
	})
}

// SetupForModule registers Swagger UI endpoints for a single module.
// It loads only the specified module's Swagger spec.
func SetupForModule(mux *http.ServeMux, moduleName, apiTitle string) {
	slog.Info("Serving Swagger UI", "path", "/swagger-ui/", "module", moduleName)

	mux.HandleFunc("/swagger.json", func(w http.ResponseWriter, _ *http.Request) {
		swagger, err := loadModuleSwaggerSpec(moduleName)
		if err != nil {
			slog.Error("failed to load swagger spec", "module", moduleName, "error", err)
			http.Error(w, "Failed to load Swagger specification", http.StatusInternalServerError)

			return
		}

		enhanceSwaggerSpec(swagger, apiTitle)

		jsonData, err := json.Marshal(swagger)
		if err != nil {
			slog.Error("failed to marshal swagger", "error", err)
			http.Error(w, "Failed to generate Swagger specification", http.StatusInternalServerError)

			return
		}

		w.Header().Set("Content-Type", "application/json")

		if _, err := w.Write(jsonData); err != nil {
			slog.Error("failed to write swagger response", "error", err)
		}
	})

	mux.HandleFunc("/swagger-ui/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")

		if _, err := w.Write([]byte(swaggerUITemplate)); err != nil {
			slog.Error("failed to write swagger-ui response", "error", err)
		}
	})
}

func handleSwaggerJSON(w http.ResponseWriter, _ *http.Request, apiTitle string) {
	swagger, err := loadAndMergeSwaggerSpecs(apiTitle)
	if err != nil {
		slog.Error("failed to load swagger specs", "error", err)
		http.Error(w, "Failed to load Swagger specification", http.StatusInternalServerError)

		return
	}

	enhanceSwaggerSpec(swagger, apiTitle)

	jsonData, err := json.Marshal(swagger)
	if err != nil {
		slog.Error("failed to marshal swagger", "error", err)
		http.Error(w, "Failed to generate Swagger specification", http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "application/json")

	if _, err := w.Write(jsonData); err != nil {
		slog.Error("failed to write swagger response", "error", err)
	}
}

func loadAndMergeSwaggerSpecs(apiTitle string) (map[string]interface{}, error) {
	swaggerFiles, err := filepath.Glob(filepath.Join(swaggerBasePath, "*", "v1", "*.swagger.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to discover swagger files: %w", err)
	}

	if len(swaggerFiles) == 0 {
		return nil, fmt.Errorf("no swagger files found in %s", swaggerBasePath)
	}

	merged := make(map[string]interface{})
	allPaths := make(map[string]interface{})
	allDefinitions := make(map[string]interface{})
	seenTags := make(map[string]bool)

	var allTags []interface{}

	for _, file := range swaggerFiles {
		spec, err := loadSwaggerFile(file)
		if err != nil {
			slog.Warn("failed to load swagger file", "file", file, "error", err)
			continue
		}

		initializeMergedSpec(merged, spec, &allPaths, &allDefinitions, apiTitle)
		mergePaths(spec, allPaths)
		mergeDefinitions(spec, allDefinitions)
		mergeTags(spec, seenTags, &allTags)
	}

	merged["paths"] = allPaths
	merged["definitions"] = allDefinitions
	merged["tags"] = allTags

	return merged, nil
}

func loadSwaggerFile(file string) (map[string]interface{}, error) {
	// #nosec G304 -- file path is controlled by filepath.Glob pattern
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var spec map[string]interface{}
	if err := json.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("failed to parse file: %w", err)
	}

	return spec, nil
}

func initializeMergedSpec(merged map[string]interface{}, spec map[string]interface{}, allPaths, allDefinitions *map[string]interface{}, apiTitle string) {
	if merged["swagger"] == nil {
		merged["swagger"] = spec["swagger"]
		merged["consumes"] = spec["consumes"]
		merged["produces"] = spec["produces"]
		merged["info"] = map[string]interface{}{
			"title":   apiTitle,
			"version": "version not set",
		}
		*allPaths = make(map[string]interface{})
		*allDefinitions = make(map[string]interface{})
	}
}

func mergePaths(spec map[string]interface{}, allPaths map[string]interface{}) {
	paths, ok := spec["paths"].(map[string]interface{})
	if !ok {
		return
	}

	for path, pathItem := range paths {
		allPaths[path] = pathItem
	}
}

func mergeDefinitions(spec map[string]interface{}, allDefinitions map[string]interface{}) {
	definitions, ok := spec["definitions"].(map[string]interface{})
	if !ok {
		return
	}

	for defName, defValue := range definitions {
		allDefinitions[defName] = defValue
	}
}

func mergeTags(spec map[string]interface{}, seenTags map[string]bool, allTags *[]interface{}) {
	tags, ok := spec["tags"].([]interface{})
	if !ok {
		return
	}

	for _, tag := range tags {
		tagMap, ok := tag.(map[string]interface{})
		if !ok {
			continue
		}

		name, ok := tagMap["name"].(string)
		if !ok {
			continue
		}

		if !seenTags[name] {
			seenTags[name] = true

			*allTags = append(*allTags, tag)
		}
	}
}

func loadModuleSwaggerSpec(moduleName string) (map[string]interface{}, error) {
	swaggerPath := filepath.Join(swaggerBasePath, moduleName, "v1", fmt.Sprintf("%s.swagger.json", moduleName))

	// #nosec G304 -- file path is constructed from module name, which is controlled
	data, err := os.ReadFile(swaggerPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read swagger file: %w", err)
	}

	var spec map[string]interface{}
	if err := json.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("failed to parse swagger file: %w", err)
	}

	return spec, nil
}

func enhanceSwaggerSpec(swagger map[string]interface{}, apiTitle string) {
	// Update version and title
	if info, ok := swagger["info"].(map[string]interface{}); ok {
		info["version"] = version.Short()
		info["title"] = apiTitle
	}

	// Add Bearer token security definition
	if swagger["securityDefinitions"] == nil {
		swagger["securityDefinitions"] = map[string]interface{}{
			"Bearer": map[string]interface{}{
				"type":        "apiKey",
				"name":        "Authorization",
				"in":          "header",
				"description": "JWT token. Enter your token (Bearer prefix will be added automatically)",
			},
		}
	}

	// Add global security requirement (makes "Authorize" button appear)
	if swagger["security"] == nil {
		swagger["security"] = []interface{}{
			map[string]interface{}{
				"Bearer": []interface{}{},
			},
		}
	}
}
