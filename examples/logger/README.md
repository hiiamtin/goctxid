# goctxid Logger Examples

Example implementations of different logging libraries (Zerolog, Zap, slog) with different web frameworks (Gin, Fiber, Fibernative, Echo) using the goctxid correlation ID library.

## ⚠️ Important Note About Benchmarks

**The Go benchmarks in these examples are NOT reliable for comparing framework performance.**

The benchmarks use different testing methods:
- **Fiber** uses `app.Test()` which creates full HTTP connections (high overhead)
- **Gin/Echo** use `ServeHTTP()` which directly calls handlers (low overhead)

This makes Fiber appear artificially slow in benchmarks, when **real-world load testing shows Fiber is actually faster than Gin** (~28% faster).

**For accurate performance comparison, use real load testing tools like k6, wrk, or ab.**

See real-world results: [go-vs-java comparison](https://github.com/hiiamtin/go-vs-java/blob/main/COMPARISON_REPORT.md)

## ✅ All 12 Examples Completed!

| Example | Logger | Framework | Status | Test | Benchmark |
|---------|--------|-----------|--------|------|-----------|
| `zerolog-gin` | Zerolog | Gin | ✅ Complete | ✅ Pass | ✅ Pass |
| `zerolog-fiber` | Zerolog | Fiber | ✅ Complete | ✅ Pass | ✅ Pass |
| `zerolog-fibernative` | Zerolog | Fibernative | ✅ Complete | ✅ Pass | ✅ Pass |
| `zerolog-echo` | Zerolog | Echo | ✅ Complete | ✅ Pass | ✅ Pass |
| `zap-gin` | Zap | Gin | ✅ Complete | ✅ Pass | ✅ Pass |
| `zap-fiber` | Zap | Fiber | ✅ Complete | ✅ Pass | ✅ Pass |
| `zap-fibernative` | Zap | Fibernative | ✅ Complete | ✅ Pass | ✅ Pass |
| `zap-echo` | Zap | Echo | ✅ Complete | ✅ Pass | ✅ Pass |
| `slog-gin` | slog | Gin | ✅ Complete | ✅ Pass | ✅ Pass |
| `slog-fiber` | slog | Fiber | ✅ Complete | ✅ Pass | ✅ Pass |
| `slog-fibernative` | slog | Fibernative | ✅ Complete | ✅ Pass | ✅ Pass |
| `slog-echo` | slog | Echo | ✅ Complete | ✅ Pass | ✅ Pass |

## 🚀 Quick Start

### Run an Example

```bash
cd zerolog-gin
go mod tidy
go run main.go
```

### Test an Example

```bash
cd zerolog-gin
go test -v
```

### Benchmark an Example (Not Recommended for Framework Comparison)

```bash
cd zerolog-gin
go test -bench=. -benchmem
```

**Note:** These benchmarks measure code execution but don't reflect real-world HTTP performance. Use k6/wrk/ab for accurate framework comparison.

## 🎯 Key Features

Each example demonstrates:

1. **HTTP Access Logging** - Automatic logging of all HTTP requests/responses
2. **Application Logging** - Business logic logging with correlation IDs
3. **Correlation ID Propagation** - Automatic extraction from headers or generation
4. **Structured Logging** - JSON output for easy parsing
5. **Performance Benchmarks** - Comprehensive performance testing

## 📝 Example Structure

Each example includes:

- `main.go` - Server implementation with middleware and handlers
- `main_test.go` - Tests and benchmarks
- `go.mod` - Module definition with dependencies

## 🔍 Logger Comparison

### Zerolog
- **Performance:** ⚡⚡⚡ Fastest (zero allocations)
- **API:** Chainable (`.Str().Int().Msg()`)
- **Use Case:** High-throughput applications

### Zap
- **Performance:** ⚡⚡ Very Fast (near-zero allocations)
- **API:** Structured (`zap.String(), zap.Int()`)
- **Use Case:** Production applications (battle-tested by Uber)

### slog
- **Performance:** ⚡ Good (standard library)
- **API:** Key-value pairs (`"key", value`)
- **Use Case:** Simple applications, no external dependencies

## 🌐 Framework Comparison

### Gin
- Popular, mature framework
- Large ecosystem
- Good performance in production

### Fiber
- Express-inspired
- **Fastest in real-world load testing** (28% faster than Gin)
- Context-based correlation ID

### Fibernative
- Fiber with c.Locals() for correlation ID
- ~17% faster than context-based Fiber
- Fiber-native approach

### Echo
- Minimalist framework
- Clean API
- Good performance

## 📚 Recommendations

**For Performance-Critical Applications:**
- Use **Fiber** (proven fastest in real-world load testing)
- Choose **Zerolog** or **Zap** for logging

**For Standard Applications:**
- Use **Gin** or **Echo** (mature, well-documented)
- **slog** is fine for standard library preference

**For Accurate Performance Testing:**
- Use k6, wrk, or ab for load testing
- Don't rely on Go micro-benchmarks for framework comparison

## 🤝 Contributing

To add a new example:

1. Create directory: `poc/{logger}-{framework}/`
2. Create `go.mod`, `main.go`, `main_test.go`
3. Implement logger-specific code
4. Test: `go test -v`
5. Benchmark: `go test -bench=. -benchmem`

## 📄 License

Same as the main goctxid project.

