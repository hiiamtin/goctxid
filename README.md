# goctxid

**A lightweight Go middleware for managing and propagating request/correlation IDs through `context.Context`.**

`goctxid` provides a simple way to ensure every request has a unique identifier, making your services observable and traceable. It's built on the standard `context.Context` package, making it compatible with any Go HTTP framework (with adapters included for popular frameworks like **Fiber**).

## 🚀 Features

* **Framework Agnostic:** Core logic is built on standard `context.Context`.
* **Multiple Framework Support:**
  * ✅ [Fiber](https://gofiber.io/) (adapter in `adapters/fiber`)
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
app.Use(goctxid_fiber.New(goctxid.Config{
    HeaderKey: "X-Request-ID",  // Custom header name
    Generator: func() string {   // Custom ID generator
        return "REQ-" + uuid.NewString()
    },
}))
```

## 🔌 Framework Support

### Using with Different Frameworks

#### Fiber

```go
package main

import (
    "github.com/gofiber/fiber/v2"
    "github.com/hiiamtin/goctxid"
    goctxid_fiber "github.com/hiiamtin/goctxid/adapters/fiber"
)

func main() {
    app := fiber.New()

    // Add middleware
    app.Use(goctxid_fiber.New())

    app.Get("/", func(c *fiber.Ctx) error {
        correlationID := goctxid.MustFromContext(c.UserContext())
        return c.SendString("Correlation ID: " + correlationID)
    })

    app.Listen(":3000")
}
```

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
| **[basic](./examples/basic)** | Fiber | Simple usage with default configuration |
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

#### Fiber: `goctxid_fiber.New(config ...goctxid.Config)`

```go
import goctxid_fiber "github.com/hiiamtin/goctxid/adapters/fiber"

app.Use(goctxid_fiber.New())
app.Use(goctxid_fiber.New(goctxid.Config{...}))
```

#### Echo: `goctxid_echo.New(config ...goctxid.Config)`

```go
import goctxid_echo "github.com/hiiamtin/goctxid/adapters/echo"

e.Use(goctxid_echo.New())
e.Use(goctxid_echo.New(goctxid.Config{...}))
```

#### Gin: `goctxid_gin.New(config ...goctxid.Config)`

```go
import goctxid_gin "github.com/hiiamtin/goctxid/adapters/gin"

r.Use(goctxid_gin.New())
r.Use(goctxid_gin.New(goctxid.Config{...}))
```

### Configuration

#### `Config`

```go
type Config struct {
    // HeaderKey is the HTTP header key used to store the correlation ID
    // Default: "X-Correlation-ID"
    HeaderKey string

    // Generator is the function used to generate a new correlation ID
    // Must be thread-safe as it will be called concurrently by multiple requests
    // Default: UUID v4
    Generator func() string
}
```

**Custom Configuration Example (works with all adapters):**

```go
import (
    "github.com/hiiamtin/goctxid"
    goctxid_fiber "github.com/hiiamtin/goctxid/adapters/fiber"
)

// Custom configuration
app.Use(goctxid_fiber.New(goctxid.Config{
    HeaderKey: "X-Request-ID",  // Use different header
    Generator: func() string {   // Custom ID generator
        return "REQ-" + uuid.NewString()
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

Creates a new context with the correlation ID. Primarily used internally by the middleware.

**Parameters:**

* `ctx`: The parent context
* `id`: The correlation ID to store

**Returns:**

* `context.Context`: New context with the correlation ID

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
