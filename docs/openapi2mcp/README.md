# openapi2mcp

The `openapi2mcp` command converts an OpenAPI 3.x specification into a Kong declarative configuration
with the `ai-mcp-proxy` plugin for [Model Context Protocol (MCP)](https://modelcontextprotocol.io/) tool definitions.

This allows you to expose your existing REST APIs as MCP tools that can be consumed by AI agents and LLM applications.

## Usage

```sh
deck file openapi2mcp --help
```

Basic usage:

```sh
deck file openapi2mcp --spec <input-oas-file> --output-file <output-deck-file>
```

## Command Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--spec`, `-s` | Path to the OpenAPI specification file (required) | - |
| `--output-file`, `-o` | Output file path. Use `-` for stdout | `-` |
| `--format` | Output format: `yaml` or `json` | `yaml` |
| `--select-tag` | Tags to select from the spec (can be repeated) | - |
| `--uuid-base` | Base UUID namespace for deterministic ID generation | - |
| `--no-id` | Skip generating UUIDs for entities | `false` |
| `--mode` | MCP proxy mode: `conversion-listener` or `conversion` | `conversion-listener` |
| `--path-prefix` | Custom path prefix for the MCP route | `/<service-name>-mcp` |
| `--include-direct-route` | Include direct routes to original API paths | `false` |
| `--ignore-security-errors` | Ignore errors for unsupported security schemes when ACL generation is active | `false` |

## How It Works

The `openapi2mcp` command:

1. Parses the OpenAPI specification
2. Creates a Kong Service pointing to the upstream API server(s)
3. Creates an MCP route with the `ai-mcp-proxy` plugin
4. Converts each OpenAPI operation into an MCP tool definition

### MCP Tool Generation

Each OpenAPI operation is converted to an MCP tool with:

- **name**: Derived from `x-kong-mcp-tool-name` extension, or `operationId` converted to kebab-case
- **description**: Derived from `x-kong-mcp-tool-description` extension, operation `description`, or `summary` (in that priority order)
- **method**: The HTTP method (GET, POST, PUT, DELETE, etc.)
- **path**: The operation path with parameter placeholders
- **parameters**: Query, path, and header parameters with simplified schemas
- **request_body**: Request body schema (for POST, PUT, PATCH operations)
- **annotations.title**: The operation `summary`

### Schema Simplification

Schemas are simplified to include only essential properties for MCP tool definitions:
- `type`
- `properties`
- `required`
- `items` (for arrays)

Other schema properties like `format`, `pattern`, `minLength`, `maxLength`, etc. are filtered out.

## MCP-Specific Extensions

### `x-kong-mcp-exclude`

Exclude an operation from MCP tool generation.

```yaml
paths:
  /internal/health:
    get:
      x-kong-mcp-exclude: true
      operationId: health-check
      summary: Health check endpoint
```

### `x-kong-mcp-tool-name`

Override the generated tool name.

```yaml
paths:
  /flights:
    get:
      x-kong-mcp-tool-name: list-all-flights
      operationId: getFlights
      summary: Get flights
```

### `x-kong-mcp-tool-description`

Override the tool description (takes priority over `description` and `summary` fields).

```yaml
paths:
  /flights:
    get:
      x-kong-mcp-tool-description: Retrieve all scheduled flights for a specific date range
      operationId: getFlights
      summary: Get flights
      description: Returns a list of flights
```

## Security & ACL Generation

When your OpenAPI spec uses `oauth2` security schemes with the `x-kong-mcp-acl` extension,
the converter automatically generates per-tool ACL entries and plugin-level ACL configuration
in the `ai-mcp-proxy` plugin config.

### How It Works

1. **Auto-detection**: ACL generation activates when any `oauth2` security scheme in
   `components/securitySchemes` has the `x-kong-mcp-acl` extension. No opt-in flag is needed.
2. **Per-tool ACL**: Each operation's `security` scopes are converted to an `acl.allow` list
   on the corresponding MCP tool.
3. **Plugin-level config**: The `acl_attribute_type` and `access_token_claim_field` values
   from `x-kong-mcp-acl` are added to the plugin config. An optional `default_acl` is read
   from the document-level `x-kong-mcp-default-acl` extension.
4. **Security inheritance**: Operations without an explicit `security` field inherit from the
   document-level `security` array (standard OpenAPI semantics).
5. **Opting out**: An operation with `security: []` (empty array) explicitly opts out —
   no ACL entry is generated for that tool.

### Supported Schemes

Only `oauth2` security schemes are supported for ACL generation. Other scheme types
(`apiKey`, `http`, `openIdConnect`) are ignored when no `x-kong-mcp-acl` extension is present.

When ACL generation is active (at least one scheme has `x-kong-mcp-acl`) and an operation
references a non-`oauth2` scheme or a scheme missing the extension, an error is returned.
Use `--ignore-security-errors` to skip these operations instead of failing.

Multiple security requirements (OR logic) and compound requirements (AND logic with
multiple schemes in one requirement) also produce errors unless `--ignore-security-errors`
is set.

### Extensions

#### `x-kong-mcp-acl` (on a security scheme)

Placed on an `oauth2` security scheme in `components/securitySchemes`. Activates ACL
generation and provides plugin-level configuration.

| Field | Description |
|-------|-------------|
| `acl_attribute_type` | The type of attribute used for ACL (e.g., `oauth_access_token`) |
| `access_token_claim_field` | The JWT/token claim field that contains the scopes (e.g., `scp`) |

```yaml
components:
  securitySchemes:
    my_oauth:
      type: oauth2
      x-kong-mcp-acl:
        acl_attribute_type: oauth_access_token
        access_token_claim_field: scp
      flows:
        authorizationCode:
          authorizationUrl: https://auth.example.com/authorize
          tokenUrl: https://auth.example.com/token
          scopes:
            read: Read access
            write: Write access
```

#### `x-kong-mcp-default-acl` (document-level)

Placed at the root level of the OpenAPI document. Defines a default ACL that applies
at the plugin level (not per-tool).

```yaml
x-kong-mcp-default-acl:
  - scope: tools
    allow:
      - "read"
```

### Example

**Input:**
```yaml
openapi: 3.0.0
info:
  title: My API
  version: 1.0.0

x-kong-mcp-default-acl:
  - scope: tools
    allow:
      - "data:read"

servers:
  - url: https://api.example.com

security:
  - my_oauth:
      - data:read

paths:
  /items:
    get:
      operationId: listItems
      summary: List items
      security:
        - my_oauth:
            - data:read
    post:
      operationId: createItem
      summary: Create item
      security:
        - my_oauth:
            - data:write
  /public:
    get:
      operationId: getPublicInfo
      summary: Public info
      security: []  # Opts out of security - no ACL generated

components:
  securitySchemes:
    my_oauth:
      type: oauth2
      x-kong-mcp-acl:
        acl_attribute_type: oauth_access_token
        access_token_claim_field: scp
      flows:
        clientCredentials:
          tokenUrl: https://auth.example.com/token
          scopes:
            data:read: Read data
            data:write: Write data
```

**Output (relevant plugin config section):**
```yaml
plugins:
- name: ai-mcp-proxy
  config:
    mode: conversion-listener
    acl_attribute_type: oauth_access_token
    access_token_claim_field: scp
    default_acl:
    - scope: tools
      allow:
      - "data:read"
    tools:
    - name: list-items
      description: List items
      method: GET
      path: /items
      acl:
        allow:
        - "data:read"
    - name: create-item
      description: Create item
      method: POST
      path: /items
      acl:
        allow:
        - "data:write"
    - name: get-public-info
      description: Public info
      method: GET
      path: /public
      # No acl — security: [] opted out
```

## Kong Extensions Support

The `openapi2mcp` command also supports standard Kong extensions:

| Extension | Level | Description |
|-----------|-------|-------------|
| `x-kong-name` | Document | Override the service name |
| `x-kong-tags` | Document | Tags to apply to all generated entities |
| `x-kong-service-defaults` | Document, Path, Operation | Default values for Kong Service entities |
| `x-kong-route-defaults` | Document, Path, Operation | Default values for Kong Route entities |
| `x-kong-upstream-defaults` | Document | Default values for Kong Upstream entities |
| `x-kong-plugin-<name>` | Document, Path, Operation | Add Kong plugins to generated entities |

## Examples

### Basic Conversion

**Input: flights-api.yaml**
```yaml
openapi: 3.0.0
info:
  title: Flights Service
servers:
  - url: https://api.example.com/v1
paths:
  /flights:
    get:
      operationId: getFlights
      summary: Get all flights
      description: Retrieves a list of all available flights with optional filtering
      parameters:
        - name: date
          in: query
          description: Filter by departure date
          required: false
          schema:
            type: string
            format: date
  /flights/{flightId}:
    get:
      operationId: getFlightById
      summary: Get flight details
      parameters:
        - name: flightId
          in: path
          required: true
          schema:
            type: string
```

**Command:**
```sh
deck file openapi2mcp -s flights-api.yaml -o kong.yaml --no-id
```

**Output: kong.yaml**
```yaml
_format_version: "3.0"
services:
- host: api.example.com
  name: flights-service
  path: /v1
  port: 443
  protocol: https
  plugins: []
  routes:
  - name: flights-service-mcp
    paths:
    - /flights-service-mcp
    plugins:
    - name: ai-mcp-proxy
      config:
        mode: conversion-listener
        tools:
        - name: get-flights
          description: Retrieves a list of all available flights with optional filtering
          method: GET
          path: /flights
          annotations:
            title: Get all flights
          parameters:
          - name: date
            in: query
            required: false
            description: Filter by departure date
            schema:
              type: string
        - name: get-flight-by-id
          description: Get flight details
          method: GET
          path: /flights/{flightId}
          annotations:
            title: Get flight details
          parameters:
          - name: flightId
            in: path
            required: true
            schema:
              type: string
```

### With MCP Extensions

```yaml
openapi: 3.0.0
info:
  title: Booking API
servers:
  - url: https://api.example.com
paths:
  /internal/metrics:
    get:
      x-kong-mcp-exclude: true  # Exclude from MCP tools
      operationId: getMetrics
      summary: Internal metrics endpoint
  /bookings:
    post:
      x-kong-mcp-tool-name: create-new-booking
      x-kong-mcp-tool-description: Create a new booking reservation for a customer
      operationId: createBooking
      summary: Create booking
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              properties:
                customer_id:
                  type: string
                flight_id:
                  type: string
              required:
                - customer_id
                - flight_id
```

### Custom Path Prefix

```sh
deck file openapi2mcp -s api.yaml --path-prefix /ai/tools/flights
```

This generates an MCP route at `/ai/tools/flights` instead of the default `/<service-name>-mcp`.

### Include Direct Routes

```sh
deck file openapi2mcp -s api.yaml --include-direct-route
```

This generates both:
1. The MCP route with the `ai-mcp-proxy` plugin
2. Direct routes to the original API endpoints (without the MCP plugin)

### Conversion Mode

```sh
deck file openapi2mcp -s api.yaml --mode conversion
```

The `--mode` flag controls the `ai-mcp-proxy` plugin mode:
- `conversion-listener` (default): The plugin listens for MCP tool calls and converts them to HTTP requests
- `conversion`: The plugin only performs conversion without the listener functionality

### Multiple Servers (Upstream with Load Balancing)

When your OpenAPI spec has multiple servers, an Upstream entity is created with multiple targets:

```yaml
openapi: 3.0.0
info:
  title: Flights Service
servers:
  - url: https://api1.example.com/v1
  - url: https://api2.example.com/v1
  - url: https://api3.example.com/v1
paths:
  /flights:
    get:
      operationId: getFlights
      summary: Get flights
```

This generates:
- A Kong Upstream with 3 targets for load balancing
- A Kong Service pointing to the Upstream
- MCP route with tools

## Tool Name Normalization

Tool names are automatically converted to kebab-case:

| operationId | Generated Tool Name |
|-------------|---------------------|
| `getFlights` | `get-flights` |
| `GetFlights` | `get-flights` |
| `get_flights` | `get-flights` |
| `listAllUsers` | `list-all-users` |
| `CreateNewItem` | `create-new-item` |

## See Also

- [openapi2kong](../README.md#openapi2kong) - Convert OpenAPI to Kong configuration (without MCP)
- [Kong AI MCP Proxy Plugin](https://docs.konghq.com/hub/kong-inc/ai-mcp-proxy/) - Plugin documentation
- [Model Context Protocol](https://modelcontextprotocol.io/) - MCP specification
