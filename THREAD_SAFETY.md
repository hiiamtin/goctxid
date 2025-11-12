# Thread-Safety Requirements

This document outlines the thread-safety requirements and best practices for using `goctxid` in concurrent environments.

## Table of Contents

- [Overview](#overview)
- [Custom Generator Requirements](#custom-generator-requirements)
- [Adapter-Specific Thread Safety](#adapter-specific-thread-safety)
- [Goroutine Safety Patterns](#goroutine-safety-patterns)
- [Testing Thread Safety](#testing-thread-safety)
- [Common Pitfalls](#common-pitfalls)

---

## Overview

The `goctxid` library is designed to work safely in highly concurrent web applications where:

- Multiple HTTP requests are handled simultaneously
- Each request may spawn goroutines for async processing
- Correlation IDs must remain consistent throughout the request lifecycle
- No race conditions or data corruption should occur

**Key Principle:** All components that can be accessed concurrently MUST be thread-safe.

---

## Custom Generator Requirements

### Requirement

**Custom ID generators MUST be thread-safe** because they are called concurrently by multiple HTTP requests.

### Why This Matters

When your web server handles multiple requests simultaneously, the middleware calls the generator function from different goroutines at the same time. If the generator is not thread-safe, you may encounter:

- Race conditions
- Duplicate IDs
- Panics
- Data corruption

### ✅ Thread-Safe Generator Examples

#### Example 1: Using UUID (Recommended)

```go
import "github.com/google/uuid"

app.Use(goctxid_fiber.New(goctxid.Config{
    Generator: func() string {
        return uuid.NewString() // Thread-safe
    },
}))
```

**Why it's safe:** The `uuid.NewString()` function uses cryptographically secure random number generation which is thread-safe.

#### Example 2: Atomic Counter with Prefix

```go
import (
    "fmt"
    "sync/atomic"
)

var counter uint64

app.Use(goctxid_fiber.New(goctxid.Config{
    Generator: func() string {
        id := atomic.AddUint64(&counter, 1)
        return fmt.Sprintf("REQ-%d", id)
    },
}))
```

**Why it's safe:** `atomic.AddUint64` provides atomic increment operations that are safe for concurrent use.

#### Example 3: Using Mutex for Complex Logic

```go
import (
    "fmt"
    "sync"
    "time"
)

var (
    mu      sync.Mutex
    counter int
)

app.Use(goctxid_fiber.New(goctxid.Config{
    Generator: func() string {
        mu.Lock()
        defer mu.Unlock()
        
        counter++
        timestamp := time.Now().Unix()
        return fmt.Sprintf("%d-%d", timestamp, counter)
    },
}))
```

**Why it's safe:** The mutex ensures only one goroutine can execute the generator logic at a time.

### ❌ NOT Thread-Safe Examples

#### Example 1: Unprotected Global Variable

```go
var counter int // ❌ WRONG: Race condition!

app.Use(goctxid_fiber.New(goctxid.Config{
    Generator: func() string {
        counter++ // ❌ Multiple goroutines can increment simultaneously
        return fmt.Sprintf("REQ-%d", counter)
    },
}))
```

**Problem:** Multiple goroutines can read and write `counter` simultaneously, causing race conditions.

#### Example 2: Non-Thread-Safe Random

```go
import "math/rand"

app.Use(goctxid_fiber.New(goctxid.Config{
    Generator: func() string {
        // ❌ WRONG: rand.Int() without proper seeding/locking is not thread-safe
        return fmt.Sprintf("REQ-%d", rand.Int())
    },
}))
```

**Problem:** `math/rand` without proper initialization is not safe for concurrent use.

### Testing Your Custom Generator

Always test your custom generator for thread safety:

```go
func TestCustomGeneratorThreadSafety(t *testing.T) {
    generator := yourCustomGenerator()
    
    var wg sync.WaitGroup
    ids := make(map[string]bool)
    var mu sync.Mutex
    
    // Generate IDs concurrently
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            id := generator()
            
            mu.Lock()
            if ids[id] {
                t.Errorf("Duplicate ID generated: %s", id)
            }
            ids[id] = true
            mu.Unlock()
        }()
    }
    
    wg.Wait()
    
    // Verify we got 100 unique IDs
    if len(ids) != 100 {
        t.Errorf("Expected 100 unique IDs, got %d", len(ids))
    }
}
```

---

## Adapter-Specific Thread Safety

### Context-Based Adapters (fiber, echo, gin)

**Status:** ✅ **Thread-safe for goroutines**

These adapters store correlation IDs in `context.Context`, which is immutable and safe to pass to goroutines.

```go
// ✅ SAFE: Context is immutable
app.Get("/", func(c *fiber.Ctx) error {
    ctx := c.UserContext()
    
    go func() {
        // Safe to use ctx in goroutine
        id := goctxid.MustFromContext(ctx)
        log.Println(id)
    }()
    
    return c.SendString("OK")
})
```

**Why it's safe:**

- `context.Context` is designed to be immutable
- Values stored in context cannot be modified
- Safe to pass across goroutine boundaries

### Fiber Native Adapter (fibernative)

**Status:** ⚠️ **NOT safe to access `c.Locals()` in goroutines**

The `fibernative` adapter uses Fiber's `c.Locals()` for storage. Fiber recycles context objects after handlers complete, making them unsafe for goroutine access.

#### ❌ WRONG: Accessing c.Locals() in Goroutine

```go
app.Get("/", func(c *fiber.Ctx) error {
    go func() {
        // ❌ DANGER: c may be recycled!
        id := goctxid_fibernative.MustFromLocals(c)
        log.Println(id) // May panic, return wrong value, or cause race
    }()
    return c.SendString("OK")
})
```

**Problems:**

- Fiber recycles `c` after handler completes
- The goroutine may access recycled/reused context
- May cause panics, wrong values, or race conditions

#### ✅ CORRECT: Copy Value Before Goroutine

```go
app.Get("/", func(c *fiber.Ctx) error {
    // Copy the value BEFORE spawning goroutine
    correlationID := goctxid_fibernative.MustFromLocals(c)
    
    go func() {
        // Safe to use the copied value
        log.Println(correlationID)
    }()
    
    return c.SendString("OK")
})
```

**Why it's safe:**

- The string value is copied before the goroutine starts
- The goroutine uses the copied value, not the Fiber context
- No access to `c` after handler completes

#### When to Use Which Adapter

| Scenario | Recommended Adapter | Reason |
|----------|-------------------|---------|
| Frequent goroutine usage | `adapters/fiber` | Context is safe for goroutines |
| Maximum performance, no goroutines | `adapters/fibernative` | 17% faster, no goroutine overhead |
| Mixed usage | `adapters/fiber` | Safer default, good performance |

---

## Goroutine Safety Patterns

### Pattern 1: Pass Context to Goroutines (Context-Based Adapters)

```go
func handler(c *fiber.Ctx) error {
    ctx := c.UserContext()
    
    // ✅ Pass context to goroutine
    go processAsync(ctx)
    
    return c.SendString("OK")
}

func processAsync(ctx context.Context) {
    id := goctxid.MustFromContext(ctx)
    log.Printf("[%s] Processing...", id)
}
```

### Pattern 2: Copy Values Before Goroutines (Fiber Native)

```go
func handler(c *fiber.Ctx) error {
    // ✅ Copy value before goroutine
    correlationID := goctxid_fibernative.MustFromLocals(c)
    userID := c.Params("id")
    
    go func() {
        // Use copied values
        log.Printf("[%s] Processing user %s", correlationID, userID)
    }()
    
    return c.SendString("OK")
}
```

### Pattern 3: Service Layer with Context

```go
type UserService struct {
    logger *Logger
}

func (s *UserService) GetUser(ctx context.Context, id string) (*User, error) {
    // ✅ Context is passed through service layers
    correlationID := goctxid.MustFromContext(ctx)
    s.logger.Info(ctx, "Fetching user", "user_id", id, "correlation_id", correlationID)
    
    // Can spawn goroutines safely
    go s.sendAnalytics(ctx, id)
    
    return s.repo.FindByID(ctx, id)
}
```

---

## Testing Thread Safety

### Test 1: Concurrent Request Handling

Verify that multiple concurrent requests each get unique correlation IDs:

```go
func TestConcurrentRequests(t *testing.T) {
    app := fiber.New()
    app.Use(goctxid_fiber.New())
    
    var wg sync.WaitGroup
    ids := make(map[string]bool)
    var mu sync.Mutex
    
    for i := 0; i < 50; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            
            req := httptest.NewRequest("GET", "/", nil)
            resp, _ := app.Test(req)
            
            id := resp.Header.Get("X-Correlation-ID")
            
            mu.Lock()
            if ids[id] {
                t.Errorf("Duplicate ID: %s", id)
            }
            ids[id] = true
            mu.Unlock()
        }()
    }
    
    wg.Wait()
}
```

### Test 2: Goroutine Safety

Verify that correlation IDs remain correct when accessed from goroutines:

```go
func TestGoroutineSafety(t *testing.T) {
    app := fiber.New()
    app.Use(goctxid_fiber.New())
    
    var wg sync.WaitGroup
    
    app.Get("/test", func(c *fiber.Ctx) error {
        ctx := c.UserContext()
        expectedID := goctxid.MustFromContext(ctx)
        
        wg.Add(1)
        go func() {
            defer wg.Done()
            time.Sleep(50 * time.Millisecond)
            
            capturedID := goctxid.MustFromContext(ctx)
            if capturedID != expectedID {
                t.Errorf("ID mismatch: expected %s, got %s", expectedID, capturedID)
            }
        }()
        
        return c.SendString("OK")
    })
    
    // Make request and wait for goroutine
    req := httptest.NewRequest("GET", "/test", nil)
    req.Header.Set("X-Correlation-ID", "test-123")
    app.Test(req)
    wg.Wait()
}
```

---

## Common Pitfalls

### Pitfall 1: Accessing Fiber Context in Goroutines

```go
// ❌ WRONG
app.Get("/", func(c *fiber.Ctx) error {
    go func() {
        id := c.Get("X-Correlation-ID") // ❌ c may be recycled
    }()
    return nil
})

// ✅ CORRECT
app.Get("/", func(c *fiber.Ctx) error {
    id := c.Get("X-Correlation-ID")
    go func() {
        log.Println(id) // ✅ Use copied value
    }()
    return nil
})
```

### Pitfall 2: Non-Thread-Safe Generator

```go
// ❌ WRONG
var counter int
Generator: func() string {
    counter++ // ❌ Race condition
    return fmt.Sprintf("ID-%d", counter)
}

// ✅ CORRECT
var counter uint64
Generator: func() string {
    id := atomic.AddUint64(&counter, 1)
    return fmt.Sprintf("ID-%d", id)
}
```

### Pitfall 3: Modifying Context Values

```go
// ❌ WRONG - Context values are immutable
ctx := c.UserContext()
// Cannot modify correlation ID after it's set
// This won't work and shouldn't be attempted

// ✅ CORRECT - Set once in middleware, read everywhere
id := goctxid.MustFromContext(ctx) // Read-only access
```

---

## Summary

### Requirements Checklist

- ✅ Custom generators MUST be thread-safe
- ✅ Use `context.Context` for goroutine-safe access (context-based adapters)
- ✅ Copy values before goroutines when using `fibernative` adapter
- ✅ Test concurrent request handling
- ✅ Test goroutine safety
- ✅ Never modify correlation IDs after they're set
- ✅ Never access Fiber context (`c`) in goroutines with `fibernative`

### Quick Reference

| Component | Thread-Safe? | Notes |
|-----------|--------------|-------|
| `DefaultGenerator()` | ✅ Yes | Uses UUID v4 |
| `FromContext()` | ✅ Yes | Context is immutable |
| `MustFromContext()` | ✅ Yes | Context is immutable |
| `NewContext()` | ✅ Yes | Creates new context |
| Context-based adapters | ✅ Yes | Safe for goroutines |
| `fibernative` adapter | ⚠️ Partial | Must copy values before goroutines |
| Custom generators | ⚠️ Your responsibility | Must implement thread-safety |

---

## Additional Resources

- [Go Context Package](https://pkg.go.dev/context)
- [Go Concurrency Patterns](https://go.dev/blog/pipelines)
- [Fiber Context Pooling](https://docs.gofiber.io/guide/faster-fiber#custom-json-encoder-decoder)
- [Testing Concurrent Code](https://go.dev/blog/race-detector)
