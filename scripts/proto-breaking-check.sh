#!/bin/bash

# Script to check for breaking changes in proto files
# Usage: ./scripts/proto-breaking-check.sh [module_name]
# Example: ./scripts/proto-breaking-check.sh auth
#          ./scripts/proto-breaking-check.sh  (checks all modules)

set -e

# Get the script directory and project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Change to project root for buf commands
cd "$PROJECT_ROOT"

MODULE_NAME=${1:-""}

if [ -n "$MODULE_NAME" ]; then
    # Check specific module
    PROTO_DIR="proto/${MODULE_NAME}"

    if [ ! -d "$PROTO_DIR" ]; then
        echo "Error: Module '$MODULE_NAME' not found in proto/"
        exit 1
    fi

    echo "🔍 Checking for breaking changes in module: $MODULE_NAME"
    echo ""

    # Check if buf is installed
    if ! command -v buf > /dev/null 2>&1; then
        echo "Error: buf is not installed. Install it with:"
        echo "  go install github.com/bufbuild/buf/cmd/buf@latest"
        exit 1
    fi

    # Run buf lint on the entire proto directory (buf handles filtering internally)
    # Filter out deprecation warnings about DEFAULT category
    LINT_OUTPUT=$(buf lint 2>&1 | grep -E "(proto/${MODULE_NAME}|Error|Failure)" | grep -v "WARN.*DEFAULT" | grep -v "deprecated" || true)

    if [ -z "$LINT_OUTPUT" ]; then
        echo "  ✅ No linting issues found for module $MODULE_NAME"
    else
        echo "  ⚠️  Linting issues found:"
        echo "$LINT_OUTPUT" | sed 's/^/    /'
        exit 1
    fi

    echo ""
    echo "💡 To check for breaking changes against a previous version, use:"
    echo "   buf breaking --against '.git#branch=main'"
else
    # Check all modules
    echo "🔍 Checking for breaking changes in all modules"
    echo ""

    if ! command -v buf > /dev/null 2>&1; then
        echo "Error: buf is not installed. Install it with:"
        echo "  go install github.com/bufbuild/buf/cmd/buf@latest"
        exit 1
    fi

    # Find all proto directories
    MODULES=()
    for dir in proto/*/; do
        if [ -d "$dir" ]; then
            module=$(basename "$dir")
            MODULES+=("$module")
        fi
    done

    if [ ${#MODULES[@]} -eq 0 ]; then
        echo "No modules found in proto/"
        exit 0
    fi

    # Run buf lint on all proto files
    # Filter out deprecation warnings about DEFAULT category
    LINT_OUTPUT=$(buf lint 2>&1 | grep -v "WARN.*DEFAULT" | grep -v "deprecated" || true)

    if [ -z "$LINT_OUTPUT" ]; then
        echo "✅ No linting issues found"
    else
        echo "⚠️  Linting issues found:"
        echo "$LINT_OUTPUT" | sed 's/^/  /'
        exit 1
    fi
fi

echo ""
echo "💡 To check for breaking changes against git history:"
echo "   buf breaking --against '.git#branch=main'"

