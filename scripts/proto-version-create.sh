#!/bin/bash

# Script to create a new API version for a module
# Usage: ./scripts/proto-version-create.sh <module_name> <version>
# Example: ./scripts/proto-version-create.sh auth v2

set -e

if [ -z "$1" ] || [ -z "$2" ]; then
    echo "Usage: $0 <module_name> <version>"
    echo "Example: $0 auth v2"
    exit 1
fi

MODULE_NAME=$1
VERSION=$2

# Validate version format (should be v1, v2, v3, etc.)
if [[ ! "$VERSION" =~ ^v[0-9]+$ ]]; then
    echo "Error: Version must be in format v1, v2, v3, etc. (got: $VERSION)"
    exit 1
fi

# Find the latest existing version
LATEST_VERSION="v1"
PROTO_BASE_DIR="proto/${MODULE_NAME}"

if [ ! -d "$PROTO_BASE_DIR" ]; then
    echo "Error: Module '$MODULE_NAME' not found in proto/"
    exit 1
fi

# Find the highest version number
for dir in "$PROTO_BASE_DIR"/v*; do
    if [ -d "$dir" ]; then
        dir_version=$(basename "$dir")
        if [[ "$dir_version" =~ ^v([0-9]+)$ ]]; then
            dir_num="${BASH_REMATCH[1]}"
            if [ -z "$LATEST_NUM" ] || [ "$dir_num" -gt "$LATEST_NUM" ]; then
                LATEST_NUM=$dir_num
                LATEST_VERSION=$dir_version
            fi
        fi
    fi
done

# Check if version already exists
NEW_VERSION_DIR="$PROTO_BASE_DIR/$VERSION"
if [ -d "$NEW_VERSION_DIR" ]; then
    echo "Error: Version $VERSION already exists for module $MODULE_NAME"
    exit 1
fi

# Extract version number for comparison
if [[ "$VERSION" =~ ^v([0-9]+)$ ]]; then
    NEW_VERSION_NUM="${BASH_REMATCH[1]}"
else
    echo "Error: Invalid version format: $VERSION"
    exit 1
fi

if [[ "$LATEST_VERSION" =~ ^v([0-9]+)$ ]]; then
    LATEST_VERSION_NUM="${BASH_REMATCH[1]}"
else
    LATEST_VERSION_NUM=1
fi

# Validate that new version is greater than latest
if [ "$NEW_VERSION_NUM" -le "$LATEST_VERSION_NUM" ]; then
    echo "Error: New version ($VERSION) must be greater than latest version ($LATEST_VERSION)"
    exit 1
fi

# Find the source proto file from the latest version
SOURCE_PROTO=""
for proto_file in "$PROTO_BASE_DIR/$LATEST_VERSION"/*.proto; do
    if [ -f "$proto_file" ]; then
        SOURCE_PROTO="$proto_file"
        break
    fi
done

if [ -z "$SOURCE_PROTO" ]; then
    echo "Error: No .proto file found in $PROTO_BASE_DIR/$LATEST_VERSION/"
    exit 1
fi

PROTO_FILENAME=$(basename "$SOURCE_PROTO")
PROJECT_NAME="github.com/cmelgarejo/go-modulith-template"

echo "📦 Creating new API version: $MODULE_NAME/$VERSION"
echo "   Source: $SOURCE_PROTO"
echo "   Target: $NEW_VERSION_DIR/$PROTO_FILENAME"
echo ""

# Create new version directory
mkdir -p "$NEW_VERSION_DIR"

# Copy proto file
cp "$SOURCE_PROTO" "$NEW_VERSION_DIR/$PROTO_FILENAME"

# Update package name and paths
# Extract version numbers (remove 'v' prefix)
LATEST_VERSION_NUM=${LATEST_VERSION#v}
NEW_VERSION_NUM=${VERSION#v}

# Use different sed command based on OS (macOS uses -i '', Linux uses -i)
if [[ "$OSTYPE" == "darwin"* ]]; then
    SED_IN_PLACE="sed -i ''"
else
    SED_IN_PLACE="sed -i"
fi

# Update package name
$SED_IN_PLACE "s/package ${MODULE_NAME}\.${LATEST_VERSION_NUM};/package ${MODULE_NAME}.${NEW_VERSION_NUM};/g" "$NEW_VERSION_DIR/$PROTO_FILENAME"

# Update go_package option (more flexible pattern)
$SED_IN_PLACE "s|proto/${MODULE_NAME}/${LATEST_VERSION}|proto/${MODULE_NAME}/${VERSION}|g" "$NEW_VERSION_DIR/$PROTO_FILENAME"
$SED_IN_PLACE "s|${MODULE_NAME}${LATEST_VERSION_NUM}|${MODULE_NAME}${NEW_VERSION_NUM}|g" "$NEW_VERSION_DIR/$PROTO_FILENAME"

# Update REST paths in HTTP annotations
$SED_IN_PLACE "s|/${LATEST_VERSION_NUM}/|/${NEW_VERSION_NUM}/|g" "$NEW_VERSION_DIR/$PROTO_FILENAME"

# No backup file needed with our sed approach

echo "✅ Created $NEW_VERSION_DIR/$PROTO_FILENAME"
echo ""
echo "📝 Next steps:"
echo "   1. Review and modify $NEW_VERSION_DIR/$PROTO_FILENAME"
echo "   2. Update REST paths from /${LATEST_VERSION#v}/ to /${VERSION#v}/ if needed"
echo "   3. Make your breaking changes"
echo "   4. Run 'make proto' to generate code"
echo "   5. Implement service handlers for the new version"
echo ""

