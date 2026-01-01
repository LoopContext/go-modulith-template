// Package version provides build information for the application.
package version

import (
	"fmt"
	"runtime"
)

// Build information. These variables are set via ldflags at build time.
var (
	// Version is the semantic version of the build.
	Version = "dev"
	// Commit is the git commit hash.
	Commit = "unknown"
	// BuildTime is the build timestamp.
	BuildTime = "unknown"
	// GoVersion is the Go version used to build.
	GoVersion = runtime.Version()
)

// Info returns the full build information.
func Info() string {
	return fmt.Sprintf("Version: %s, Commit: %s, Built: %s, Go: %s",
		Version, Commit, BuildTime, GoVersion)
}

// Short returns a short version string.
func Short() string {
	if Version == "dev" {
		return fmt.Sprintf("dev-%s", Commit[:7])
	}
	return Version
}

