package main

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	goctxid_echo "github.com/hiiamtin/goctxid/adapters/echo"
	"github.com/labstack/echo/v4"
)

// ⚠️ WARNING: THESE BENCHMARKS ARE UNRELIABLE FOR FRAMEWORK COMPARISON ⚠️
//
// Problem: These micro-benchmarks use ServeHTTP() which directly calls handlers
// without real HTTP overhead (no network, no connection pooling, no TCP handshake).
// This makes Gin/Echo appear faster than Fiber in benchmarks, but real-world load
// testing shows the opposite is true.
//
// Why this happens:
// - ServeHTTP() is a direct function call with minimal overhead
// - Real HTTP servers have network stack, connection pooling, and concurrency overhead
// - Micro-benchmarks don't reflect production performance characteristics
//
// Real-world performance (k6 load testing):
// - Fiber: 7,512 RPS (Database I/O) - FASTEST
// - Gin:   5,877 RPS (Database I/O)
// - Fiber is ~28% faster in production
//
// Conclusion: These benchmarks only measure handler execution speed, NOT framework
// performance. Use real load testing tools (k6, wrk, ab) for accurate comparisons.
//
// Reference: https://github.com/hiiamtin/go-vs-java/blob/main/COMPARISON_REPORT.md

func setupEcho() *echo.Echo {
	SetupBenchmarkLogger() // Configure logger to write to discard for benchmarks

	e := echo.New()
	e.HideBanner = true

	e.Use(goctxid_echo.New())
	e.Use(zapMiddleware())

	e.GET("/health", healthCheck)
	e.GET("/users/:id", getUser)
	e.POST("/users", createUser)

	return e
}

func BenchmarkHealthCheck(b *testing.B) {
	e := setupEcho()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/health", nil)
		req.Header.Set("X-Correlation-ID", "test-correlation-id")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
	}
}

func BenchmarkGetUser(b *testing.B) {
	e := setupEcho()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/users/123", nil)
		req.Header.Set("X-Correlation-ID", "test-correlation-id")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
	}
}

func BenchmarkCreateUser(b *testing.B) {
	e := setupEcho()

	body := []byte(`{"name":"John Doe"}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/users", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Correlation-ID", "test-correlation-id")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
	}
}

func BenchmarkWithCorrelationID(b *testing.B) {
	e := setupEcho()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/health", nil)
		req.Header.Set("X-Correlation-ID", "benchmark-correlation-id")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
	}
}

func BenchmarkConcurrent(b *testing.B) {
	e := setupEcho()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/users/456", nil)
			req.Header.Set("X-Correlation-ID", "concurrent-test-id")
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)
		}
	})
}

func TestGetUser(t *testing.T) {
	e := setupEcho()

	req := httptest.NewRequest("GET", "/users/123", nil)
	req.Header.Set("X-Correlation-ID", "test-correlation-id")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result["id"] != "123" {
		t.Errorf("Expected user id 123, got %v", result["id"])
	}
}
