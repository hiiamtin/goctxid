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

// Access ID from context
correlationID := goctxid.MustFromContext(c.UserContext())
```

**Location:** `adapters/fiber/`

**Use Case:** Standard approach, compatible with other middleware that uses context

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

- `FromLocals(c *fiber.Ctx) (string, bool)` - Get ID from Locals
- `MustFromLocals(c *fiber.Ctx) string` - Get ID or empty string
- `LocalsKey = "goctxid"` - The key used in c.Locals()

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
```

**Location:** `adapters/echo/`

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
```

**Location:** `adapters/gin/`

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

### `goctxid.FromContext(ctx context.Context) (string, bool)`

Retrieves the correlation ID from context.

### `goctxid.MustFromContext(ctx context.Context) string`

Retrieves the correlation ID or returns empty string.

### `goctxid.DefaultGenerator() string`

Default UUID v4 generator (thread-safe).

### `goctxid.DefaultHeaderKey`

Default header key: `"X-Correlation-ID"`

## Configuration

All adapters accept the same configuration:

```go
type Config struct {
    // HeaderKey is the HTTP header key
    // Default: "X-Correlation-ID"
    HeaderKey string

    // Generator is the ID generation function
    // Must be thread-safe!
    // Default: UUID v4
    Generator func() string
}
```

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

## Testing Your Adapter

```go
func TestYourAdapter(t *testing.T) {
    // Test 1: Generates new ID when header not present
    // Test 2: Uses existing ID from request header
    // Test 3: Uses custom header key
    // Test 4: Uses custom generator
    // Test 5: Thread safety with concurrent requests
}
```

See `fiber_test.go` for a complete example.

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
