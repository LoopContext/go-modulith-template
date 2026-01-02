# GraphQL Auto-Generation from Proto/OpenAPI

This document explains how to automatically generate GraphQL schemas from Protocol Buffer (proto) definitions via OpenAPI/Swagger files.

## Overview

The project includes a tool that automatically generates GraphQL schema files from your existing proto definitions. This eliminates the need to manually write GraphQL schemas, ensuring consistency between your gRPC API and GraphQL API.

## How It Works

1. **Proto → OpenAPI**: When you run `make proto`, `buf` generates OpenAPI/Swagger JSON files from your proto definitions
2. **OpenAPI → GraphQL**: The `graphql-from-proto` tool parses these OpenAPI files and generates GraphQL schema files
3. **GraphQL Code Generation**: Run `make graphql-generate-all` to generate resolver code from the schemas

## Usage

### Generate Schemas for All Modules

```bash
# First, ensure proto files are compiled to OpenAPI
make proto

# Then generate GraphQL schemas from OpenAPI
make graphql-from-proto
```

This will:
- Discover all modules with OpenAPI definitions in `gen/openapiv2/proto/`
- Generate GraphQL schema files in `internal/graphql/schema/`
- Create one schema file per module (e.g., `auth.graphql`, `order.graphql`)

### Generate Schema for a Specific Module

```bash
make graphql-from-proto-module MODULE_NAME=auth
```

### Complete Workflow

```bash
# 1. Update your proto files
# Edit proto/auth/v1/auth.proto

# 2. Generate gRPC and OpenAPI code
make proto

# 3. Generate GraphQL schemas from OpenAPI
make graphql-from-proto

# 4. Review and customize schemas (optional)
# Edit internal/graphql/schema/auth.graphql if needed

# 5. Generate GraphQL resolver code
make graphql-generate-all

# 6. Implement resolvers
# Edit internal/graphql/resolver/auth.go
```

## What Gets Generated

### Types

All proto message types are converted to GraphQL types:

**Proto:**
```protobuf
message User {
  string id = 1;
  string email = 2;
  google.protobuf.Timestamp created_at = 6;
}
```

**Generated GraphQL:**
```graphql
type User {
  id: String!
  email: String!
  createdAt: String!
}
```

### Operations

gRPC service methods are converted to GraphQL queries and mutations:

- **GET operations** → GraphQL Queries
- **POST/PUT/DELETE operations** → GraphQL Mutations

**Proto:**
```protobuf
service AuthService {
  rpc GetProfile(GetProfileRequest) returns (GetProfileResponse);
  rpc UpdateProfile(UpdateProfileRequest) returns (UpdateProfileResponse);
}
```

**Generated GraphQL:**
```graphql
extend type Query {
  getProfile(input: GetProfileRequest): GetProfileResponse
}

extend type Mutation {
  updateProfile(input: UpdateProfileRequest): UpdateProfileResponse
}
```

## Type Mappings

| Proto Type | GraphQL Type |
|------------|--------------|
| `string` | `String` |
| `int32`, `int64` | `Int` |
| `float`, `double` | `Float` |
| `bool` | `Boolean` |
| `bytes` | `String` (Base64) |
| `google.protobuf.Timestamp` | `String` |
| `repeated T` | `[T!]` (array) |
| Message types | Custom type (e.g., `User`) |

## Customization

### After Generation

The generated schemas are meant to be a starting point. You can:

1. **Edit the schema files** to:
   - Add descriptions
   - Adjust field types
   - Add custom scalars
   - Remove unwanted operations

2. **Customize resolvers** to:
   - Add business logic
   - Transform data
   - Add authentication/authorization
   - Handle errors

### Regeneration

⚠️ **Important**: The generated schema files have a header indicating they're auto-generated. If you make manual changes, they will be overwritten when you regenerate.

**Best Practice**:
- Keep generated schemas as-is for basic CRUD operations
- Create separate custom schema files for complex operations
- Or use gqlgen's `# +genql` directives to preserve custom code

## Integration with Module Scaffolding

When you create a new module with `make new-module <name>`, if GraphQL is initialized, it will:

1. Create a basic GraphQL schema template
2. After you define your proto file and run `make proto`
3. You can then run `make graphql-from-proto-module MODULE_NAME=<name>` to generate the full schema

## Troubleshooting

### "No OpenAPI files found"

**Solution**: Run `make proto` first to generate OpenAPI definitions from your proto files.

### "GraphQL not initialized"

**Solution**: Run `make add-graphql` first to set up GraphQL infrastructure.

### Schema validation errors

**Solution**:
1. Check that your proto files are valid
2. Ensure OpenAPI generation succeeded (`make proto`)
3. Review the generated schema for any issues
4. You may need to manually fix type mappings for complex types

### Type mismatches

Some proto types don't map perfectly to GraphQL. Common issues:

- **Timestamps**: Converted to `String` - consider using a custom scalar
- **Enums**: Converted to `String` - consider defining GraphQL enums manually
- **Oneof fields**: Not directly supported - may need manual schema definition

## Advanced Usage

### Custom Type Mappings

You can extend the `graphql-from-proto` tool to handle custom type mappings. Edit `scripts/graphql-from-proto/main.go` and modify the `openAPITypeToGraphQL` function.

### Preserving Custom Code

Use gqlgen's model generation features to preserve custom resolver implementations. See [gqlgen documentation](https://gqlgen.com/reference/model-generation/) for details.

## See Also

- [GraphQL Integration Guide](./GRAPHQL_INTEGRATION.md)
- [Modulith Architecture](./MODULITH_ARCHITECTURE.md)
- [gqlgen Documentation](https://gqlgen.com/)

