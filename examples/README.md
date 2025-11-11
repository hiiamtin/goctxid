# goctxid Examples

This directory contains practical examples demonstrating how to use the goctxid library in different scenarios.

## Examples Overview

| Example | Framework | Description | Key Features |
|---------|-----------|-------------|--------------|
| [basic](./basic) | Fiber | Simple usage with default configuration (context-based) | Default middleware setup, accessing correlation IDs |
| [fiber-native](./fiber-native) | Fiber | Fiber-native approach using c.Locals() | Better performance, Fiber-native storage, FromLocals() API |
| [echo-basic](./echo-basic) | Echo | Simple usage with Echo framework | Echo middleware, context operations |
| [gin-basic](./gin-basic) | Gin | Simple usage with Gin framework | Gin middleware, context operations |
| [standard-http](./standard-http) | net/http | Using with standard library | Framework-agnostic usage, custom middleware |
| [custom-generator](./custom-generator) | Fiber | Custom ID generation strategies | Sequential IDs, prefixed UUIDs, custom headers |
| [logging](./logging) | Fiber | Integration with logging systems | Structured logging, service layer integration, request tracing |

## Running the Examples

### Prerequisites

```bash
# Make sure you're in the project root
cd /path/to/goctxid

# Install dependencies
go mod download
```

### Run an Example

```bash
# Fiber basic example (context-based)
cd examples/basic
go run main.go

# Fiber native example (c.Locals() - better performance)
cd examples/fiber-native
go run main.go

# Echo basic example
cd examples/echo-basic
go run main.go

# Gin basic example
cd examples/gin-basic
go run main.go

# Standard net/http example
cd examples/standard-http
go run main.go

# Custom generator example
cd examples/custom-generator
go run main.go

# Logging example
cd examples/logging
go run main.go
```

All examples start a server on `http://localhost:3000`. Each example includes curl commands in its output showing how to test it.

## Example Details

### 1. Basic Usage (Fiber - Context-Based)

**Location:** `examples/basic/`

Demonstrates the simplest way to use goctxid with Fiber using context-based storage:

- Adding middleware with default configuration
- Accessing correlation IDs in handlers via context
- Automatic ID generation
- Using existing IDs from request headers

**Import:**

```go
import goctxid_fiber "github.com/hiiamtin/goctxid/adapters/fiber"
```

**Try it:**

```bash
cd examples/basic
go run main.go

# In another terminal:
curl http://localhost:3000/
curl -H "X-Correlation-ID: my-custom-id" http://localhost:3000/
```

---

### 2. Fiber Native (c.Locals() - Better Performance)

**Location:** `examples/fiber-native/`

Demonstrates the Fiber-native approach using `c.Locals()` for better performance:

- Uses `c.Locals()` instead of context (Fiber-native way)
- Better performance - no context allocation overhead
- Simpler API with `FromLocals()` and `MustFromLocals()`
- Same features as context-based adapter

**Import:**

```go
import goctxid_fibernative "github.com/hiiamtin/goctxid/adapters/fibernative"
```

**Key Differences:**

```go
// Context-based (adapters/fiber)
app.Use(goctxid_fiber.New())
correlationID := goctxid.MustFromContext(c.UserContext())

// Fiber-native (adapters/fibernative) - More performant!
app.Use(goctxid_fibernative.New())
correlationID := goctxid_fibernative.MustFromLocals(c)
```

**Try it:**

```bash
cd examples/fiber-native
go run main.go

# In another terminal:
curl http://localhost:3000/
curl -H "X-Correlation-ID: my-custom-id" http://localhost:3000/
curl http://localhost:3000/user/123
```

**Performance Benefits:**

- 17% faster with existing IDs
- 1 fewer allocation per request
- ~50 bytes less memory per request

---

### 4. Echo Basic Usage

**Location:** `examples/echo-basic/`

Demonstrates using goctxid with Echo framework:

- Echo middleware integration
- Accessing correlation IDs from Echo context
- Service layer integration
- Route parameters with correlation tracking

**Import:**

```go
import goctxid_echo "github.com/hiiamtin/goctxid/adapters/echo"
```

**Try it:**

```bash
cd examples/echo-basic
go run main.go

# In another terminal:
curl http://localhost:3000/
curl -H "X-Correlation-ID: my-custom-id" http://localhost:3000/
curl http://localhost:3000/user/123
```

---

### 5. Gin Basic Usage

**Location:** `examples/gin-basic/`

Demonstrates using goctxid with Gin framework:

- Gin middleware integration
- Accessing correlation IDs from Gin context
- Service layer integration
- Route parameters with correlation tracking

**Import:**

```go
import goctxid_gin "github.com/hiiamtin/goctxid/adapters/gin"
```

**Try it:**

```bash
cd examples/gin-basic
go run main.go

# In another terminal:
curl http://localhost:3000/
curl -H "X-Correlation-ID: my-custom-id" http://localhost:3000/
curl http://localhost:3000/user/123
```

---

### 6. Standard net/http

**Location:** `examples/standard-http/`

Demonstrates framework-agnostic usage with standard library:

- Custom middleware implementation
- Using core goctxid functions directly
- No framework dependencies
- Works with any router/framework

**Try it:**

```bash
cd examples/standard-http
go run main.go

# In another terminal:
curl http://localhost:3000/
curl -H "X-Correlation-ID: my-custom-id" http://localhost:3000/
```

---

### 7. Custom Generator

**Location:** `examples/custom-generator/`

Shows how to customize ID generation:

- Sequential ID generator (REQ-{timestamp}-{counter})
- Prefixed UUID generator
- Custom header keys
- Multiple middleware configurations for different routes

**Try it:**

```bash
cd examples/custom-generator
go run main.go

# In another terminal:
curl http://localhost:3000/api/v1/test  # Sequential IDs
curl http://localhost:3000/api/v2/test  # Prefixed UUIDs
curl http://localhost:3000/api/v3/test  # Custom header
```

**Important:** Custom generators must be **thread-safe** as they're called concurrently!

---

### 8. Logging Integration

**Location:** `examples/logging/`

Demonstrates real-world usage with logging:

- Custom structured logger that includes correlation IDs
- Service layer integration (UserService, OrderService)
- Automatic correlation ID propagation through context
- Request/response logging middleware
- Multi-service request tracing

**Try it:**

```bash
cd examples/logging
go run main.go

# In another terminal:
curl http://localhost:3000/user/123
curl -X POST http://localhost:3000/order \
  -H "Content-Type: application/json" \
  -d '{"user_id":"123","items":["item1","item2"]}'
```

**Watch the logs** - notice how the same correlation ID appears in all log entries for a single request, even across multiple service calls!

## Common Patterns

### Pattern 1: Accessing Correlation ID

```go
// Method 1: Check if exists
correlationID, exists := goctxid.FromContext(c.UserContext())
if !exists {
    // Handle missing ID
}

// Method 2: Get or empty string
correlationID := goctxid.MustFromContext(c.UserContext())
```

### Pattern 2: Passing Context to Services

```go
app.Get("/user/:id", func(c *fiber.Ctx) error {
    ctx := c.UserContext()  // Get context with correlation ID
    
    // Pass to service - correlation ID is automatically included
    user, err := userService.GetUser(ctx, c.Params("id"))
    
    return c.JSON(user)
})
```

### Pattern 3: Custom Logging

```go
func logWithCorrelation(ctx context.Context, message string) {
    correlationID := goctxid.MustFromContext(ctx)
    log.Printf("[%s] %s", correlationID, message)
}
```

### Pattern 4: Propagating to Downstream Services

```go
func callDownstreamAPI(ctx context.Context, url string) error {
    correlationID := goctxid.MustFromContext(ctx)
    
    req, _ := http.NewRequest("GET", url, nil)
    req.Header.Set("X-Correlation-ID", correlationID)
    
    // Make request...
}
```

## Best Practices

1. **Add middleware early** - Place `goctxid.New()` before other middleware that might need the correlation ID

2. **Always pass context** - Pass `c.UserContext()` to service layers to maintain correlation ID

3. **Use in logs** - Include correlation ID in all log messages for request tracing

4. **Propagate downstream** - Send correlation ID to external services/APIs

5. **Thread-safe generators** - Ensure custom ID generators are safe for concurrent use

6. **Don't modify IDs** - Once set, correlation IDs should remain constant throughout the request lifecycle

## Testing Examples

You can also use the examples as integration tests:

```bash
# Start the server
cd examples/basic
go run main.go &
SERVER_PID=$!

# Test it
curl -s http://localhost:3000/ | jq .

# Cleanup
kill $SERVER_PID
```

## Need Help?

- Check the main [README](../README.md) for library documentation
- See [BENCHMARKS.md](../BENCHMARKS.md) for performance information
- Review the test files for more usage patterns

## Contributing Examples

Have a useful example? Feel free to contribute:

1. Create a new directory under `examples/`
2. Add a `main.go` with clear comments
3. Update this README with your example
4. Submit a pull request!
