# Performance Benchmarks

## Middleware Overhead Comparison

These benchmarks compare the performance impact of using the goctxid middleware.

### Results (Apple M1)

| Benchmark | Time/op | Memory/op | Allocs/op | vs Baseline |
|-----------|---------|-----------|-----------|-------------|
| **Baseline** (no middleware) | 4,610 ns | 5,854 B | 24 | - |
| **With Middleware** (new ID) | 5,973 ns | 6,111 B | 30 | +29.6% time, +6 allocs |
| **With Middleware** (existing ID) | 5,861 ns | 6,147 B | 29 | +27.1% time, +5 allocs |
| **With Context Access** | 5,758 ns | 6,115 B | 30 | +24.9% time, +6 allocs |

### Analysis

**Overhead Summary:**

- **Time overhead**: ~1,200-1,400 ns per request (~25-30% increase)
- **Memory overhead**: ~250-300 bytes per request
- **Allocation overhead**: +5-6 allocations per request

**Key Insights:**

1. **Minimal overhead**: The middleware adds only ~1.3 microseconds per request
2. **Existing ID is faster**: Using an existing correlation ID (5,861 ns) is slightly faster than generating a new one (5,973 ns)
3. **Context access is cheap**: Reading from context adds negligible overhead (~5 ns)
4. **Production ready**: At 200,000+ requests/second throughput, the overhead is acceptable for most applications

### Core Operations Performance

| Operation | Time/op | Memory/op | Allocs/op |
|-----------|---------|-----------|-----------|
| `FromContext()` | 4.7 ns | 0 B | 0 |
| `NewContext()` | 21.9 ns | 48 B | 1 |
| `defaultGenerator()` (UUID) | 348.6 ns | 64 B | 2 |

### Recommendations

- ✅ **Use this middleware** if you need request tracing (overhead is minimal)
- ✅ **Pass correlation IDs** from upstream services when possible (saves ~100 ns)
- ✅ **Access context freely** - `FromContext()` has zero allocations
- ⚠️ **Custom generators** should be fast - UUID generation takes ~350 ns

## Running Benchmarks

```bash
# Run all benchmarks
go test -bench=. -benchmem

# Run only middleware benchmarks
go test -bench=BenchmarkMiddleware -benchmem

# Compare with baseline
go test -bench=Benchmark -benchmem -run=^$
```
