# Framework Adapters Guide

The `goctxid` library is **framework-agnostic** at its core. The core package (`goctxid.go`) works with standard `context.Context`, making it compatible with any Go HTTP framework.

## Architecture

```text
┌─────────────────────────────────────────────────────────┐
│                    Your Application                      │
│  (Uses goctxid.FromContext, goctxid.MustFromContext)    │
└─────────────────────────────────────────────────────────┘
                            ▲
                            │
┌─────────────────────────────────────────────────────────┐
│              Framework-Specific Adapter                  │
│  (Fiber, Echo, Gin, net/http, Chi, etc.)                │
│  - Extracts header from request                          │
│  - Generates ID if missing                               │
│  - Sets response header                                  │
│  - Injects ID into context                               │
└─────────────────────────────────────────────────────────┘
                            ▲
                            │
┌─────────────────────────────────────────────────────────┐
│                  Core Package (goctxid)                  │
│  - Context operations (NewContext, FromContext)          │
│  - Default generator (UUID v4)                           │
│  - Configuration struct                                  │
└─────────────────────────────────────────────────────────┘
```

## Available Adapters

### 1. Fiber (Context-Based)

**Import:**

```go
import goctxid_fiber "github.com/hiiamtin/goctxid/adapters/fiber"
```

**Usage:**

```go
app := fiber.New()
app.Use(goctxid_fiber.New())

// Access ID using convenience function (recommended)
correlationID := goctxid_fiber.GetCorrelationID(c)

// Or access from context directly
correlationID := goctxid.MustFromContext(c.UserContext())
```

**API:**

- `GetCorrelationID(c *fiber.Ctx) string` - Convenience function (recommended)
- `FromContext(ctx context.Context) (string, bool)` - Get with existence check
- `MustFromContext(ctx context.Context) string` - Get or empty string
- `NewContext(ctx context.Context, id string) context.Context` - Create context with ID
- `DefaultGenerator() string` - UUID v4 generator
- `FastGenerator() string` - Fast UUID generator (17% faster)

**Configuration:**

```go
type Config struct {
    goctxid.Config

    // Next defines a function to skip this middleware when returned true
    Next func(c *fiber.Ctx) bool
}
```

**Location:** `adapters/fiber/`

**Use Case:** Standard approach, compatible with other middleware that uses context

**Features:**

- ✅ 100% test coverage
- ✅ Thread-safe for concurrent requests
- ✅ Conditional middleware execution with `Next` function
- ✅ Re-exported core functions for convenience

---

### 2. Fiber Native (c.Locals() - Better Performance)

**Import:**

```go
import goctxid_fibernative "github.com/hiiamtin/goctxid/adapters/fibernative"
```

**Usage:**

```go
app := fiber.New()
app.Use(goctxid_fibernative.New())

// Access ID from Locals (Fiber-native way)
correlationID := goctxid_fibernative.MustFromLocals(c)
```

**Location:** `adapters/fibernative/`

**Use Case:** Fiber-native approach for maximum performance

**Performance Benefits:**

- 17% faster with existing IDs
- 1 fewer allocation per request
- ~50 bytes less memory per request

**API:**

- `GetCorrelationID(c *fiber.Ctx) string` - Convenience function (recommended)
- `FromLocals(c *fiber.Ctx) (string, bool)` - Get ID from Locals
- `MustFromLocals(c *fiber.Ctx) string` - Get ID or empty string
- `FromLocalsWithKey(c *fiber.Ctx, key string) (string, bool)` - Get ID with custom key
- `MustFromLocalsWithKey(c *fiber.Ctx, key string) string` - Get ID with custom key or empty
- `DefaultLocalsKey = "goctxid"` - The default key used in c.Locals()

**Configuration:**

```go
type Config struct {
    goctxid.Config

    // Next defines a function to skip this middleware when returned true
    Next func(c *fiber.Ctx) bool

    // LocalsKey is the key used to store the correlation ID in c.Locals()
    // Default: "goctxid"
    LocalsKey string
}
```

**Features:**

- ✅ 100% test coverage
- ✅ 17% faster than context-based approach
- ✅ 1 fewer allocation per request
- ✅ ~50 bytes less memory per request
- ✅ Customizable Locals key to avoid collisions
- ✅ Conditional middleware execution with `Next` function

**⚠️ Goroutine Safety Warning:**

The `fibernative` adapter uses `c.Locals()` which is **NOT safe** to use directly in goroutines because Fiber recycles the context after the handler completes.

```go
// ❌ WRONG - Don't do this:
app.Get("/", func(c *fiber.Ctx) error {
    go func() {
        // ⚠️ DANGER: c may be recycled!
        id := goctxid_fibernative.MustFromLocals(c)
        log.Println(id)
    }()
    return c.SendString("OK")
})

// ✅ CORRECT - Copy the value first:
app.Get("/", func(c *fiber.Ctx) error {
    correlationID := goctxid_fibernative.MustFromLocals(c)

    go func() {
        // Safe to use the copied value
        log.Println(correlationID)
    }()
    return c.SendString("OK")
})
```

**When to use:**

- ✅ Use `fibernative` when you need maximum performance and don't use goroutines
- ✅ Use `fiber` (context-based) when you need to pass IDs to goroutines frequently

---

### 3. Standard net/http

**No adapter needed!** Use the middleware pattern directly.

**Usage:**

```go
import "github.com/hiiamtin/goctxid"

func correlationIDMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        correlationID := r.Header.Get(goctxid.DefaultHeaderKey)
        if correlationID == "" {
            correlationID = goctxid.DefaultGenerator()
        }
        
        w.Header().Set(goctxid.DefaultHeaderKey, correlationID)
        ctx := goctxid.NewContext(r.Context(), correlationID)
        r = r.WithContext(ctx)
        
        next.ServeHTTP(w, r)
    })
}

// Use it
mux := http.NewServeMux()
handler := correlationIDMiddleware(mux)
http.ListenAndServe(":3000", handler)
```

**Example:** See `examples/standard-http/`

---

### 4. Echo

**Import:**

```go
import goctxid_echo "github.com/hiiamtin/goctxid/adapters/echo"
```

**Usage:**

```go
e := echo.New()
e.Use(goctxid_echo.New())

// Access ID using convenience function (recommended)
correlationID := goctxid_echo.GetCorrelationID(c)

// Or access from context directly
correlationID := goctxid.MustFromContext(c.Request().Context())
```

**API:**

- `GetCorrelationID(c echo.Context) string` - Convenience function (recommended)
- `FromContext(ctx context.Context) (string, bool)` - Get with existence check
- `MustFromContext(ctx context.Context) string` - Get or empty string
- `NewContext(ctx context.Context, id string) context.Context` - Create context with ID
- `DefaultGenerator() string` - UUID v4 generator
- `FastGenerator() string` - Fast UUID generator (17% faster)

**Configuration:**

```go
type Config struct {
    goctxid.Config

    // Next defines a function to skip this middleware when returned true
    Next func(c echo.Context) bool
}
```

**Location:** `adapters/echo/`

**Features:**

- ✅ 100% test coverage
- ✅ Thread-safe for concurrent requests
- ✅ Conditional middleware execution with `Next` function
- ✅ Re-exported core functions for convenience

---

### 5. Gin

**Import:**

```go
import goctxid_gin "github.com/hiiamtin/goctxid/adapters/gin"
```

**Usage:**

```go
r := gin.Default()
r.Use(goctxid_gin.New())

// Access ID using convenience function (recommended)
correlationID := goctxid_gin.GetCorrelationID(c)

// Or access from context directly
correlationID := goctxid.MustFromContext(c.Request.Context())
```

**API:**

- `GetCorrelationID(c *gin.Context) string` - Convenience function (recommended)
- `FromContext(ctx context.Context) (string, bool)` - Get with existence check
- `MustFromContext(ctx context.Context) string` - Get or empty string
- `NewContext(ctx context.Context, id string) context.Context` - Create context with ID
- `DefaultGenerator() string` - UUID v4 generator
- `FastGenerator() string` - Fast UUID generator (17% faster)

**Configuration:**

```go
type Config struct {
    goctxid.Config

    // Next defines a function to skip this middleware when returned true
    Next func(c *gin.Context) bool
}
```

**Location:** `adapters/gin/`

**Features:**

- ✅ 100% test coverage
- ✅ Thread-safe for concurrent requests
- ✅ Conditional middleware execution with `Next` function
- ✅ Re-exported core functions for convenience

---

## Creating Your Own Adapter

If you're using a different framework (Chi, Gorilla Mux, etc.), creating an adapter is simple:

### Template

```go
package yourframework

import (
    "github.com/hiiamtin/goctxid"
    "github.com/yourframework/framework"
)

func New(config ...goctxid.Config) framework.MiddlewareType {
    // 1. Set defaults
    cfg := goctxid.Config{}
    if len(config) > 0 {
        cfg = config[0]
    }
    if cfg.HeaderKey == "" {
        cfg.HeaderKey = goctxid.DefaultHeaderKey
    }
    if cfg.Generator == nil {
        cfg.Generator = goctxid.DefaultGenerator
    }

    // 2. Return middleware
    return func(/* framework-specific signature */) {
        // 3. Extract correlation ID from request header
        correlationID := /* get header using framework API */
        
        // 4. Generate if missing
        if correlationID == "" {
            correlationID = cfg.Generator()
        }
        
        // 5. Set response header
        /* set header using framework API */
        
        // 6. Get request context
        ctx := /* get context using framework API */
        
        // 7. Create new context with correlation ID
        newCtx := goctxid.NewContext(ctx, correlationID)
        
        // 8. Set context back to request
        /* set context using framework API */
        
        // 9. Continue to next handler
        /* call next handler using framework API */
    }
}
```

### Example: Chi Router

```go
package chi

import (
    "net/http"
    "github.com/hiiamtin/goctxid"
)

func New(config ...goctxid.Config) func(http.Handler) http.Handler {
    cfg := goctxid.Config{}
    if len(config) > 0 {
        cfg = config[0]
    }
    if cfg.HeaderKey == "" {
        cfg.HeaderKey = goctxid.DefaultHeaderKey
    }
    if cfg.Generator == nil {
        cfg.Generator = goctxid.DefaultGenerator
    }

    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            correlationID := r.Header.Get(cfg.HeaderKey)
            if correlationID == "" {
                correlationID = cfg.Generator()
            }
            
            w.Header().Set(cfg.HeaderKey, correlationID)
            ctx := goctxid.NewContext(r.Context(), correlationID)
            r = r.WithContext(ctx)
            
            next.ServeHTTP(w, r)
        })
    }
}
```

## Framework Comparison

| Framework | Adapter Location | Import Path | Middleware Type |
|-----------|-----------------|-------------|-----------------|
| **Fiber (Context)** | `adapters/fiber/` | `github.com/hiiamtin/goctxid/adapters/fiber` | `fiber.Handler` |
| **Fiber (Native)** | `adapters/fibernative/` | `github.com/hiiamtin/goctxid/adapters/fibernative` | `fiber.Handler` |
| **net/http** | No adapter needed | `github.com/hiiamtin/goctxid` | `func(http.Handler) http.Handler` |
| **Echo** | `adapters/echo/` | `github.com/hiiamtin/goctxid/adapters/echo` | `echo.MiddlewareFunc` |
| **Gin** | `adapters/gin/` | `github.com/hiiamtin/goctxid/adapters/gin` | `gin.HandlerFunc` |
| **Chi** | DIY (see template) | - | `func(http.Handler) http.Handler` |

## Core Package API

All adapters use these core functions:

### `goctxid.NewContext(ctx context.Context, id string) context.Context`

Creates a new context with the correlation ID.

**Intended for:** Middleware adapters and custom middleware implementations.

**Usage in adapters:**

```go
// Inside your custom adapter
newCtx := goctxid.NewContext(ctx, correlationID)
```

### `goctxid.FromContext(ctx context.Context) (string, bool)`

Retrieves the correlation ID from context.

**Intended for:** Application code to access the correlation ID.

**Usage:**

```go
id, exists := goctxid.FromContext(ctx)
if exists {
    log.Printf("Request ID: %s", id)
}
```

### `goctxid.MustFromContext(ctx context.Context) string`

Retrieves the correlation ID or returns empty string.

**Intended for:** Application code when you don't need to check if ID exists.

**Usage:**

```go
id := goctxid.MustFromContext(ctx)
log.Printf("[%s] Processing request", id)
```

### `goctxid.DefaultGenerator() string`

Default UUID v4 generator (thread-safe).

**Intended for:** Adapters as fallback when no custom generator is provided.

### `goctxid.DefaultHeaderKey`

Default header key: `"X-Correlation-ID"`

**Intended for:** Adapters as fallback when no custom header key is provided.

## Configuration

All adapters accept the same base configuration:

```go
type Config struct {
    // HeaderKey is the HTTP header key
    // Default: "X-Correlation-ID"
    HeaderKey string

    // Generator is the ID generation function
    // Must be thread-safe!
    // Default: UUID v4 (goctxid.DefaultGenerator)
    Generator func() string
}
```

### Available Generators

#### 1. DefaultGenerator (UUID v4)

Standard UUID v4 generator using `google/uuid` library.

```go
app.Use(goctxid_fiber.New(goctxid_fiber.Config{
    Generator: goctxid.DefaultGenerator, // or omit for default
}))
```

**Characteristics:**

- ✅ Cryptographically secure random UUIDs
- ✅ RFC 4122 compliant
- ✅ Thread-safe
- ⚠️ Slightly slower due to crypto/rand usage

#### 2. FastGenerator (Fast UUID)

High-performance UUID generator optimized for speed.

```go
app.Use(goctxid_fiber.New(goctxid_fiber.Config{
    Generator: goctxid.FastGenerator,
}))
```

**Characteristics:**

- ✅ 17% faster than DefaultGenerator
- ✅ Thread-safe with sync.Pool
- ✅ Still produces valid UUIDs
- ✅ Good for high-throughput applications
- ⚠️ Uses math/rand (not cryptographically secure)

**Performance Comparison:**

```text
BenchmarkDefaultGenerator-8    1000000    1234 ns/op    48 B/op    1 allocs/op
BenchmarkFastGenerator-8       1200000    1024 ns/op    48 B/op    1 allocs/op
```

#### 3. Custom Generator

You can provide your own generator function:

```go
func customGenerator() string {
    return fmt.Sprintf("REQ-%d", time.Now().UnixNano())
}

app.Use(goctxid_fiber.New(goctxid_fiber.Config{
    Generator: customGenerator,
}))
```

**Requirements:**

- ✅ Must be thread-safe (called concurrently)
- ✅ Must return unique IDs
- ✅ Should be fast (called on every request)

### Adapter-Specific Configuration

Some adapters extend the base configuration with additional options:

#### Fiber, Echo, Gin

```go
type Config struct {
    goctxid.Config

    // Next defines a function to skip this middleware when returned true
    Next func(c *FrameworkContext) bool
}
```

**Example - Skip middleware for health checks:**

```go
app.Use(goctxid_fiber.New(goctxid_fiber.Config{
    Next: func(c *fiber.Ctx) bool {
        return c.Path() == "/health" || c.Path() == "/metrics"
    },
}))
```

#### Fibernative

```go
type Config struct {
    goctxid.Config

    // Next defines a function to skip this middleware when returned true
    Next func(c *fiber.Ctx) bool

    // LocalsKey is the key used to store the correlation ID in c.Locals()
    // Default: "goctxid"
    LocalsKey string
}
```

**Example - Custom Locals key:**

```go
app.Use(goctxid_fibernative.New(goctxid_fibernative.Config{
    LocalsKey: "request-id", // Avoid collision with existing code
}))
```

## Quality & Testing

All adapters maintain **100% test coverage** and are tested across multiple platforms:

### Test Coverage

| Package | Coverage | Tests |
|---------|----------|-------|
| Core (`goctxid`) | 100% | ✅ |
| `adapters/echo` | 100% | ✅ |
| `adapters/fiber` | 100% | ✅ |
| `adapters/fibernative` | 100% | ✅ |
| `adapters/gin` | 100% | ✅ |

### CI/CD Pipeline

- ✅ **Multi-platform testing:** Ubuntu, macOS, Windows
- ✅ **Race detection:** Enabled on Unix platforms
- ✅ **Linting:** golangci-lint with strict rules
- ✅ **Security scanning:** CodeQL analysis
- ✅ **Benchmarking:** Performance regression detection
- ✅ **Code generation:** Automated re-export generation

### Test Categories

Each adapter includes comprehensive tests for:

1. **Basic functionality**
   - ID generation when header not present
   - ID extraction from request header
   - Custom header key configuration
   - Custom generator configuration

2. **Concurrency & thread safety**
   - Concurrent request handling
   - Generator thread safety
   - Race condition detection

3. **Convenience functions**
   - `GetCorrelationID` function
   - Re-exported core functions

4. **Advanced features**
   - `Next` function for conditional execution
   - Custom configuration options
   - Edge cases and error handling

## Best Practices

1. **Keep core package clean** - Don't add framework-specific code to `goctxid.go`

2. **Thread-safe generators** - Custom generators MUST be safe for concurrent use

3. **Consistent behavior** - All adapters should:
   - Extract ID from request header
   - Generate if missing
   - Set response header
   - Inject into context
   - Continue to next handler

4. **Use standard context** - Always use `context.Context` for propagation

5. **Export defaults** - Make `DefaultGenerator` and `DefaultHeaderKey` available

6. **100% test coverage** - All adapters must maintain complete test coverage

7. **Document goroutine safety** - Clearly document any goroutine safety concerns

## Testing Your Adapter

All adapters should include comprehensive tests covering:

### Required Test Cases

```go
// 1. Basic Functionality
func TestNew(t *testing.T) {
    t.Run("generates_new_ID_when_header_not_present", func(t *testing.T) { /* ... */ })
    t.Run("uses_existing_ID_from_request_header", func(t *testing.T) { /* ... */ })
    t.Run("uses_custom_header_key", func(t *testing.T) { /* ... */ })
    t.Run("uses_custom_generator", func(t *testing.T) { /* ... */ })
}

// 2. Configuration
func TestConfigDefault(t *testing.T) {
    t.Run("uses_defaults_when_no_config_provided", func(t *testing.T) { /* ... */ })
    t.Run("uses_defaults_when_empty_config_provided", func(t *testing.T) { /* ... */ })
}

// 3. Concurrency & Thread Safety
func TestConcurrentRequests(t *testing.T) { /* ... */ }
func TestGeneratorThreadSafety(t *testing.T) { /* ... */ }

// 4. Convenience Functions
func TestGetCorrelationID(t *testing.T) { /* ... */ }

// 5. Advanced Features
func TestNextFunction(t *testing.T) {
    t.Run("middleware_runs_when_Next_is_nil", func(t *testing.T) { /* ... */ })
    t.Run("middleware_skips_when_Next_returns_true", func(t *testing.T) { /* ... */ })
}

// 6. Re-exported Functions (if applicable)
func TestReExportedFunctions(t *testing.T) { /* ... */ }
```

### Running Tests

```bash
# Run tests with coverage
go test -v -coverprofile=coverage.out ./adapters/yourframework

# View coverage report
go tool cover -func=coverage.out

# View detailed HTML coverage report
go tool cover -html=coverage.out

# Run with race detection
go test -v -race ./adapters/yourframework
```

### Coverage Requirements

- **Target:** 100% code coverage
- **Minimum:** 90% code coverage
- All public functions must be tested
- All configuration options must be tested
- Concurrent behavior must be tested

See existing adapter tests for complete examples:

- `adapters/fiber/fiber_test.go` - Context-based adapter
- `adapters/fibernative/fibernative_test.go` - Locals-based adapter with goroutine safety tests
- `adapters/echo/echo_test.go` - Echo framework adapter
- `adapters/gin/gin_test.go` - Gin framework adapter

## Contributing Adapters

Want to contribute an adapter for a popular framework?

1. Create `adapters/{framework}/` directory
2. Implement the adapter following the template
3. Add tests
4. Update this document
5. Submit a pull request!

## Questions?

- Check the [main README](./README.md) for general usage
- See [examples/](./examples/) for practical examples
- Review existing adapters for reference
