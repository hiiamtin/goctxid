# goctxid

**A lightweight Go middleware for managing and propagating request/correlation IDs through `context.Context`.**

`goctxid` provides a simple way to ensure every request has a unique identifier, making your services observable and traceable. It's built on the standard `context.Context` package, making it compatible with any Go HTTP framework (with adapters included for popular frameworks like **Fiber**).

## 🚀 Features

* **Framework Agnostic:** Core logic is built on standard `context.Context`.
* **Multiple Framework Support:**
  * ✅ [Fiber](https://gofiber.io/) - Two adapters available:
    * `adapters/fiber` - Context-based (standard approach)
    * `adapters/fibernative` - Fiber-native using c.Locals() (better performance)
  * ✅ Standard `net/http` (no adapter needed - use core package directly)
  * ✅ [Echo](https://echo.labstack.com/) (adapter in `adapters/echo`)
  * ✅ [Gin](https://gin-gonic.com/) (adapter in `adapters/gin`)
  * 🔧 Easy to create adapters for other frameworks (Chi, Gorilla, etc.)
* **Extract or Generate:** Automatically extracts an existing ID from request headers (e.g., `X-Correlation-ID`) or generates a new one if not found.
* **Propagation:**
      - Injects the ID into the `context.Context` (via `c.UserContext()` in Fiber) for use in your application logic (logging, downstream API calls).
      - Adds the ID to the response headers so clients (like web frontends or mobile apps) can also use it for debugging.
* **Customizable:**
      - Easily change the default header key (e.g., use `X-Request-ID`, `X-Trace-ID`).
      - Provide your own custom ID generator function (e.g., UUID, nanoid).

## 📦 Installation

```bash
go get github.com/hiiamtin/goctxid
```

## 🎯 Quick Start

### Basic Usage with Fiber

```go
package main

import (
    "log"
    "github.com/gofiber/fiber/v2"
    "github.com/hiiamtin/goctxid"
    goctxid_fiber "github.com/hiiamtin/goctxid/adapters/fiber"
)

func main() {
    app := fiber.New()

    // Add goctxid middleware
    app.Use(goctxid_fiber.New())

    app.Get("/", func(c *fiber.Ctx) error {
        // Get correlation ID from context
        correlationID, _ := goctxid.FromContext(c.UserContext())

        return c.JSON(fiber.Map{
            "message": "Hello, World!",
            "correlation_id": correlationID,
        })
    })

    log.Fatal(app.Listen(":3000"))
}
```

### Custom Configuration

```go
app.Use(goctxid_fiber.New(goctxid_fiber.Config{
    Config: goctxid.Config{
        HeaderKey: "X-Request-ID",  // Custom header name
        Generator: func() string {   // Custom ID generator
            return "REQ-" + uuid.NewString()
        },
    },
}))
```

## ⚡ Advanced Features

### Skip Middleware for Specific Requests (Next Function)

Save ~400-500 ns per request by skipping middleware for health checks, metrics, or static files:

```go
app.Use(goctxid_fiber.New(goctxid_fiber.Config{
    Next: func(c *fiber.Ctx) bool {
        // Skip middleware for these paths
        path := c.Path()
        return path == "/health" || path == "/metrics"
    },
}))
```

**Available for all adapters:**

* **Fiber**: `Next func(c *fiber.Ctx) bool`
* **Echo**: `Next func(c echo.Context) bool`
* **Gin**: `Next func(c *gin.Context) bool`

### High-Performance ID Generation (FastGenerator)

For high-throughput systems, use `FastGenerator` for ~33% faster ID generation:

```go
app.Use(goctxid_fiber.New(goctxid_fiber.Config{
    Config: goctxid.Config{
        Generator: goctxid.FastGenerator,  // Fast but exposes request count
    },
}))
```

**⚠️ Privacy Warning:** `FastGenerator` uses an atomic counter and **exposes your request count**. Use only when:

* Performance is critical (high-throughput systems)
* Request count exposure is acceptable
* IDs are used only for internal tracing (not exposed to clients)

**Performance Comparison:**

```text
FastGenerator:    234 ns/op (single-threaded), 149 ns/op (parallel)
DefaultGenerator: 349 ns/op (single-threaded), 731 ns/op (parallel)
```

For most applications, use `DefaultGenerator` (UUID v4) for better privacy/security.

### Custom LocalsKey (Fiber Native Only)

Prevent collisions when using `fibernative` adapter:

```go
app.Use(fibernative.New(fibernative.Config{
    LocalsKey: "my_correlation_id",  // Custom key to avoid collisions
}))

// Retrieve with custom key
id := fibernative.MustFromLocalsWithKey(c, "my_correlation_id")
```

See [examples/advanced-features](./examples/advanced-features) for complete examples.

## 🔌 Framework Support

### Using with Different Frameworks

#### Fiber (Context-Based)

```go
package main

import (
    "github.com/gofiber/fiber/v2"
    "github.com/hiiamtin/goctxid"
    goctxid_fiber "github.com/hiiamtin/goctxid/adapters/fiber"
)

func main() {
    app := fiber.New()

    // Add middleware (context-based)
    app.Use(goctxid_fiber.New())

    app.Get("/", func(c *fiber.Ctx) error {
        correlationID := goctxid.MustFromContext(c.UserContext())
        return c.SendString("Correlation ID: " + correlationID)
    })

    app.Listen(":3000")
}
```

#### Fiber Native (c.Locals() - Better Performance)

```go
package main

import (
    "github.com/gofiber/fiber/v2"
    goctxid_fibernative "github.com/hiiamtin/goctxid/adapters/fibernative"
)

func main() {
    app := fiber.New()

    // Add middleware (Fiber-native using c.Locals())
    app.Use(goctxid_fibernative.New())

    app.Get("/", func(c *fiber.Ctx) error {
        // Access ID directly from Locals (more performant!)
        correlationID := goctxid_fibernative.MustFromLocals(c)
        return c.SendString("Correlation ID: " + correlationID)
    })

    app.Listen(":3000")
}
```

**Which Fiber adapter should I use?**

| Adapter | Storage | Use Case | Performance | Goroutine Safety |
|---------|---------|----------|-------------|------------------|
| `adapters/fiber` | `context.Context` | Standard patterns, compatibility with other middleware | Good | ✅ Safe (context is immutable) |
| `adapters/fibernative` | `c.Locals()` | Fiber-native, maximum performance | Better (17% faster) | ⚠️ **Must copy values before goroutines** |

See complete example: [examples/fiber-native](./examples/fiber-native)

**⚠️ Important: Goroutine Safety with `fibernative`**

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

If you need to use correlation IDs in goroutines frequently, consider using the context-based adapter (`adapters/fiber`) instead.

#### Standard net/http

```go
package main

import (
    "net/http"
    "github.com/hiiamtin/goctxid"
)

func correlationIDMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Get or generate correlation ID
        correlationID := r.Header.Get(goctxid.DefaultHeaderKey)
        if correlationID == "" {
            correlationID = goctxid.DefaultGenerator()
        }

        // Set response header
        w.Header().Set(goctxid.DefaultHeaderKey, correlationID)

        // Add to context
        ctx := goctxid.NewContext(r.Context(), correlationID)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

func main() {
    mux := http.NewServeMux()

    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        correlationID := goctxid.MustFromContext(r.Context())
        w.Write([]byte("Correlation ID: " + correlationID))
    })

    handler := correlationIDMiddleware(mux)
    http.ListenAndServe(":3000", handler)
}
```

#### Echo

See complete example: [examples/echo-basic](./examples/echo-basic)

```go
import goctxid_echo "github.com/hiiamtin/goctxid/adapters/echo"
e.Use(goctxid_echo.New())
```

#### Gin

See complete example: [examples/gin-basic](./examples/gin-basic)

```go
import goctxid_gin "github.com/hiiamtin/goctxid/adapters/gin"
r.Use(goctxid_gin.New())
```

**Other frameworks?** See [adapters/README.md](./adapters/README.md) for a guide on creating your own adapter.

## 📚 Examples

Check out the [examples/](./examples) directory for complete, runnable examples:

| Example | Framework | Description |
|---------|-----------|-------------|
| **[basic](./examples/basic)** | Fiber | Simple usage with default configuration (context-based) |
| **[advanced-features](./examples/advanced-features)** | Fiber | Next function, FastGenerator, and custom LocalsKey |
| **[fiber-native](./examples/fiber-native)** | Fiber | Fiber-native approach using c.Locals() (better performance) |
| **[standard-http](./examples/standard-http)** | net/http | Framework-agnostic usage with standard library |
| **[custom-generator](./examples/custom-generator)** | Fiber | Custom ID generation strategies (sequential, prefixed) |
| **[logging](./examples/logging)** | Fiber | Integration with logging systems and service layers |

### Running Examples

```bash
# Run any example
cd examples/basic
go run main.go

# Test it
curl http://localhost:3000/
curl -H "X-Correlation-ID: my-custom-id" http://localhost:3000/
```

## 🔧 API Reference

### Middleware

Each framework adapter provides a `New()` function that creates middleware for that framework:

#### Fiber (Context-Based): `goctxid_fiber.New(config ...fiber.Config)`

```go
import goctxid_fiber "github.com/hiiamtin/goctxid/adapters/fiber"

// Default configuration
app.Use(goctxid_fiber.New())

// Custom configuration
app.Use(goctxid_fiber.New(goctxid_fiber.Config{
    Config: goctxid.Config{
        HeaderKey: "X-Request-ID",
        Generator: goctxid.FastGenerator,
    },
    Next: func(c *fiber.Ctx) bool {
        return c.Path() == "/health"
    },
}))
```

#### Fiber Native (c.Locals()): `goctxid_fibernative.New(config ...fibernative.Config)`

```go
import goctxid_fibernative "github.com/hiiamtin/goctxid/adapters/fibernative"

// Default configuration
app.Use(goctxid_fibernative.New())

// Custom configuration
app.Use(goctxid_fibernative.New(goctxid_fibernative.Config{
    Config: goctxid.Config{
        HeaderKey: "X-Request-ID",
    },
    LocalsKey: "my_correlation_id",
    Next: func(c *fiber.Ctx) bool {
        return c.Path() == "/health"
    },
}))

// Access ID from Locals
correlationID := goctxid_fibernative.MustFromLocals(c)
// Or with custom key
correlationID := goctxid_fibernative.MustFromLocalsWithKey(c, "my_correlation_id")
```

#### Echo: `goctxid_echo.New(config ...echo.Config)`

```go
import goctxid_echo "github.com/hiiamtin/goctxid/adapters/echo"

// Default configuration
e.Use(goctxid_echo.New())

// Custom configuration
e.Use(goctxid_echo.New(goctxid_echo.Config{
    Config: goctxid.Config{
        HeaderKey: "X-Request-ID",
    },
    Next: func(c echo.Context) bool {
        return c.Path() == "/health"
    },
}))
```

#### Gin: `goctxid_gin.New(config ...gin.Config)`

```go
import goctxid_gin "github.com/hiiamtin/goctxid/adapters/gin"

// Default configuration
r.Use(goctxid_gin.New())

// Custom configuration
r.Use(goctxid_gin.New(goctxid_gin.Config{
    Config: goctxid.Config{
        HeaderKey: "X-Request-ID",
    },
    Next: func(c *gin.Context) bool {
        return c.Request.URL.Path == "/health"
    },
}))
```

### Configuration

#### Base `goctxid.Config`

All adapters embed this base configuration:

```go
type Config struct {
    // HeaderKey is the HTTP header key used to store the correlation ID
    // Default: "X-Correlation-ID"
    HeaderKey string

    // Generator is the function used to generate a new correlation ID
    // Must be thread-safe as it will be called concurrently by multiple requests
    // Default: UUID v4 (goctxid.DefaultGenerator)
    // Alternative: goctxid.FastGenerator (faster but exposes request count)
    Generator func() string
}
```

**⚠️ Important:** Custom generators MUST be thread-safe. See [Thread-Safety Requirements](./THREAD_SAFETY.md) for details and examples.

#### Adapter-Specific Configs

Each adapter extends the base config with framework-specific options:

**Fiber (Context-Based):**

```go
type Config struct {
    goctxid.Config
    // Next defines a function to skip middleware
    Next func(c *fiber.Ctx) bool
}
```

**Fiber Native (c.Locals()):**

```go
type Config struct {
    goctxid.Config
    // LocalsKey is the key used to store the correlation ID in c.Locals()
    // Default: "goctxid"
    LocalsKey string
    // Next defines a function to skip middleware
    Next func(c *fiber.Ctx) bool
}
```

**Echo:**

```go
type Config struct {
    goctxid.Config
    // Next defines a function to skip middleware
    Next func(c echo.Context) bool
}
```

**Gin:**

```go
type Config struct {
    goctxid.Config
    // Next defines a function to skip middleware
    Next func(c *gin.Context) bool
}
```

**Custom Configuration Example:**

```go
import (
    "github.com/hiiamtin/goctxid"
    goctxid_fiber "github.com/hiiamtin/goctxid/adapters/fiber"
)

// Custom configuration
app.Use(goctxid_fiber.New(goctxid_fiber.Config{
    Config: goctxid.Config{
        HeaderKey: "X-Request-ID",  // Use different header
        Generator: func() string {   // Custom ID generator
            return "REQ-" + uuid.NewString()
        },
    },
    Next: func(c *fiber.Ctx) bool {
        // Skip middleware for health checks
        return c.Path() == "/health"
    },
}))
```

### Context Operations

#### `FromContext(ctx context.Context) (string, bool)`

Retrieves the correlation ID from the context.

**Returns:**

* `string`: The correlation ID
* `bool`: `true` if the ID exists, `false` otherwise

**Example:**

```go
correlationID, exists := goctxid.FromContext(c.UserContext())
if !exists {
    // Handle missing ID
}
```

#### `MustFromContext(ctx context.Context) string`

Retrieves the correlation ID from the context, returning an empty string if not found.

**Returns:**

* `string`: The correlation ID or empty string

**Example:**

```go
correlationID := goctxid.MustFromContext(c.UserContext())
log.Printf("[%s] Processing request", correlationID)
```

#### `NewContext(ctx context.Context, id string) context.Context`

Creates a new context with the correlation ID.

**⚠️ Note:** This function is primarily intended for middleware adapters and custom middleware implementations. Most users should use the provided framework adapters instead of calling this directly.

**When to use:**

* ✅ Creating custom middleware for unsupported frameworks
* ✅ Implementing custom middleware patterns with `net/http`
* ✅ Testing scenarios where you need to manually inject a correlation ID
* ❌ Regular application code (use `FromContext` or `MustFromContext` instead)

**Parameters:**

* `ctx`: The parent context
* `id`: The correlation ID to store

**Returns:**

* `context.Context`: New context with the correlation ID

**Example (custom middleware):**

```go
func customMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        id := goctxid.DefaultGenerator()
        ctx := goctxid.NewContext(r.Context(), id)
        r = r.WithContext(ctx)
        next.ServeHTTP(w, r)
    })
}
```

## 🎨 Common Patterns

### Pattern 1: Logging with Correlation ID

```go
func logWithCorrelation(ctx context.Context, message string) {
    correlationID := goctxid.MustFromContext(ctx)
    log.Printf("[%s] %s", correlationID, message)
}

app.Get("/user/:id", func(c *fiber.Ctx) error {
    ctx := c.UserContext()
    logWithCorrelation(ctx, "Fetching user")
    // ... your logic
})
```

### Pattern 2: Propagating to Downstream Services

```go
func callExternalAPI(ctx context.Context, url string) error {
    correlationID := goctxid.MustFromContext(ctx)

    req, _ := http.NewRequest("GET", url, nil)
    req.Header.Set("X-Correlation-ID", correlationID)

    // Make request...
}
```

### Pattern 3: Service Layer Integration

```go
type UserService struct {
    logger *Logger
}

func (s *UserService) GetUser(ctx context.Context, userID string) (*User, error) {
    correlationID := goctxid.MustFromContext(ctx)
    s.logger.Info(correlationID, "Fetching user", userID)

    // ... your logic
}
```

## ⚡ Performance

The middleware has minimal overhead:

* **Time overhead**: ~1.3 microseconds per request (~25-30% increase)
* **Memory overhead**: ~250-300 bytes per request
* **Throughput**: 200,000+ requests/second

See [BENCHMARKS.md](./BENCHMARKS.md) for detailed performance analysis.

## 🧪 Testing

```bash
# Run all tests
go test -v

# Run tests with coverage
go test -cover

# Run benchmarks
go test -bench=. -benchmem
```

**Test Coverage:** 100%

## 📚 Documentation

* **[Thread-Safety Requirements](./THREAD_SAFETY.md)** - Comprehensive guide on thread-safety requirements for custom generators and goroutine usage
* **[Adapter Documentation](./adapters/README.md)** - Detailed information about framework adapters
* **[Examples](./examples/README.md)** - Complete working examples for all supported frameworks
* **[Benchmarks](./BENCHMARKS.md)** - Performance comparisons between adapters

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 📄 License

MIT License - see LICENSE file for details

## 🙏 Acknowledgments

Built with support for:

* [Fiber](https://gofiber.io/) - Express-inspired web framework
* [Echo](https://echo.labstack.com/) - High performance, minimalist Go web framework
* [Gin](https://gin-gonic.com/) - HTTP web framework written in Go
* [google/uuid](https://github.com/google/uuid) - UUID generation
