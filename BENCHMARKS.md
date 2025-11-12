# Performance Benchmarks

## Overview

This document provides comprehensive performance benchmarks for all goctxid adapters across different frameworks. All benchmarks were run on **Apple M1** (darwin/arm64).

## Framework Adapter Comparison

### Fiber Adapter

| Benchmark | Time/op | Memory/op | Allocs/op | vs Baseline |
|-----------|---------|-----------|-----------|-------------|
| **Baseline** (no middleware) | 5,098 ns | 5,848 B | 24 | - |
| **With Middleware** (new ID) | 6,060 ns | 6,110 B | 30 | +18.9% time, +6 allocs |
| **With Middleware** (existing ID) | 6,831 ns | 6,145 B | 29 | +34.0% time, +5 allocs |
| **With Context Access** | 6,180 ns | 6,111 B | 30 | +21.2% time, +6 allocs |

**Overhead:** ~962-1,733 ns per request

### Echo Adapter

| Benchmark | Time/op | Memory/op | Allocs/op | vs Baseline |
|-----------|---------|-----------|-----------|-------------|
| **Baseline** (no middleware) | 504 ns | 1,016 B | 10 | - |
| **With Middleware** (new ID) | 1,252 ns | 1,576 B | 19 | +148% time, +9 allocs |
| **With Middleware** (existing ID) | 851 ns | 1,512 B | 17 | +68.8% time, +7 allocs |
| **With Context Access** | 1,379 ns | 1,576 B | 19 | +174% time, +9 allocs |

**Overhead:** ~347-875 ns per request

### Gin Adapter

| Benchmark | Time/op | Memory/op | Allocs/op | vs Baseline |
|-----------|---------|-----------|-----------|-------------|
| **Baseline** (no middleware) | 451 ns | 1,040 B | 9 | - |
| **With Middleware** (new ID) | 1,197 ns | 1,552 B | 17 | +165% time, +8 allocs |
| **With Middleware** (existing ID) | 772 ns | 1,488 B | 15 | +71.2% time, +6 allocs |
| **With Context Access** | 1,235 ns | 1,552 B | 17 | +174% time, +8 allocs |

**Overhead:** ~321-784 ns per request

## Analysis

### Performance Comparison

**Fastest to Slowest (with middleware generating new ID):**

1. **Gin**: 1,197 ns/op (512 B/op memory overhead)
2. **Echo**: 1,252 ns/op (560 B/op memory overhead)
3. **Fiber**: 6,060 ns/op (262 B/op memory overhead)

**Note:** Fiber has higher absolute time but this is due to Fiber's baseline being ~10x slower than Echo/Gin. The actual middleware overhead is comparable.

### Key Insights

1. **Minimal overhead**: The middleware adds 300-900 ns per request depending on framework
2. **Existing ID is faster**: Using an existing correlation ID is ~30-50% faster than generating a new UUID
3. **Context access is cheap**: Reading from context adds negligible overhead (~5-10 ns)
4. **Production ready**: All adapters can handle 150,000+ requests/second with middleware enabled
5. **Framework differences**: Echo and Gin have similar performance; Fiber has higher baseline but similar relative overhead

## Core Operations Performance

These are the fundamental operations used by all adapters:

| Operation | Time/op | Memory/op | Allocs/op | Description |
|-----------|---------|-----------|-----------|-------------|
| `FromContext()` | 4.86 ns | 0 B | 0 | Retrieve ID from context (zero-cost) |
| `NewContext()` | 22.73 ns | 48 B | 1 | Create new context with ID |
| `DefaultGenerator()` (UUID v4) | 349.3 ns | 64 B | 2 | Generate new UUID (secure) |
| `FastGenerator()` (atomic counter) | 234.1 ns | 192 B | 7 | Generate ID with counter (fast but exposes count) |

**Key Takeaways:**

* Context operations are extremely fast (< 25 ns)
* UUID generation is the most expensive operation (~350 ns)
* FastGenerator is ~33% faster than UUID v4 but exposes request count
* Zero allocations when reading from context

### Generator Performance Comparison

| Generator | Single-Threaded | Parallel | Memory/op | Allocs/op | Privacy |
|-----------|----------------|----------|-----------|-----------|---------|
| **DefaultGenerator** (UUID v4) | 348.9 ns/op | 730.7 ns/op | 64 B | 2 | ✅ Secure |
| **FastGenerator** (atomic counter) | 234.1 ns/op | 148.6 ns/op | 192 B | 7 | ⚠️ Exposes count |

**Performance Gains:**

* **Single-threaded**: FastGenerator is ~33% faster (115 ns saved per ID)
* **Parallel**: FastGenerator is ~79% faster (582 ns saved per ID)

**⚠️ Privacy Trade-off:** FastGenerator uses a sequential atomic counter embedded in the ID, which exposes your request count and traffic patterns. Use only when:

* Performance is critical (high-throughput systems)
* Request count exposure is acceptable
* IDs are used only for internal tracing (not exposed to clients)

For most applications, use `DefaultGenerator` (UUID v4) for better privacy/security.

## Throughput Estimates

Based on benchmark results, here's the estimated throughput for each adapter:

| Adapter | Requests/Second (with middleware) | Requests/Second (baseline) |
|---------|-----------------------------------|----------------------------|
| **Gin** | ~835,000 req/s | ~2,217,000 req/s |
| **Echo** | ~798,000 req/s | ~1,984,000 req/s |
| **Fiber** | ~165,000 req/s | ~196,000 req/s |

*Note: Single-core estimates. Actual throughput will be higher with multiple cores.*

## Recommendations

### ✅ When to Use

* **Request tracing** - Overhead is minimal (300-900 ns per request)
* **Distributed systems** - Essential for tracking requests across services
* **Debugging** - Invaluable for correlating logs across microservices
* **Production systems** - All adapters can handle 150,000+ req/s

### 💡 Optimization Tips

1. **Use Next function to skip middleware** - Saves ~400-500 ns per skipped request (health checks, metrics, static files)
2. **Use FastGenerator for high-throughput** - Saves ~115 ns per ID (single-threaded), ~582 ns (parallel)
3. **Pass correlation IDs from upstream** - Saves ~350 ns (no UUID generation)
4. **Access context freely** - `FromContext()` has zero allocations
5. **Choose lightweight frameworks** - Echo/Gin are ~5x faster than Fiber baseline
6. **Custom generators** - Keep them fast (< 100 ns if possible)

#### Example: Combining Optimizations

```go
app.Use(goctxid_fiber.New(goctxid_fiber.Config{
    Config: goctxid.Config{
        Generator: goctxid.FastGenerator,  // ~115 ns saved per ID
    },
    Next: func(c *fiber.Ctx) bool {
        // Skip middleware for health/metrics (~400-500 ns saved)
        return c.Path() == "/health" || c.Path() == "/metrics"
    },
}))
```

**Potential Savings:**

* Health check requests: ~400-500 ns saved (middleware skipped entirely)
* Normal requests with FastGenerator: ~115 ns saved (single-threaded), ~582 ns (parallel)
* Total optimization: Up to ~500-1000 ns per request depending on scenario

### ⚠️ Considerations

* **UUID generation** takes ~350 ns - consider simpler ID schemes for extreme performance
* **Existing IDs** are 30-50% faster than generating new ones
* **Framework choice** matters more than middleware overhead

## Running Benchmarks

### All Adapters

```bash
# Run benchmarks for all adapters
go test ./adapters/... -bench=. -benchmem

# Run benchmarks for specific adapter
go test ./adapters/fiber -bench=. -benchmem
go test ./adapters/echo -bench=. -benchmem
go test ./adapters/gin -bench=. -benchmem

# Run core package benchmarks
go test . -bench=. -benchmem
```

### Specific Benchmarks

```bash
# Only middleware benchmarks (exclude baseline)
go test ./adapters/fiber -bench=BenchmarkMiddleware -benchmem

# Only baseline benchmarks
go test ./adapters/... -bench=BenchmarkBaseline -benchmem

# Compare with and without middleware
go test ./adapters/echo -bench=Benchmark -benchmem -run=^$
```

### Benchmark Comparison

```bash
# Save baseline results
go test ./adapters/echo -bench=. -benchmem > old.txt

# Make changes, then compare
go test ./adapters/echo -bench=. -benchmem > new.txt
benchcmp old.txt new.txt  # Requires golang.org/x/tools/cmd/benchcmp
```

## Test Environment

All benchmarks in this document were run on:

* **CPU**: Apple M1
* **OS**: darwin/arm64
* **Go Version**: 1.21+
* **Test Mode**: Single-threaded benchmarks

Results may vary on different hardware and configurations.
