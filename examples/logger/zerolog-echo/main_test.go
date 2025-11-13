package main

import (
	"bytes"
	"encoding/json"
	"net/http"
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
	e.Use(zerologMiddleware())

	e.GET("/health", healthCheck)
	e.GET("/users/:id", getUser)
	e.POST("/users", createUser)

	return e
}

func BenchmarkHealthCheck(b *testing.B) {
	e := setupEcho()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/health", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
	}
}

func BenchmarkGetUser(b *testing.B) {
	e := setupEcho()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/users/123", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
	}
}

func BenchmarkCreateUser(b *testing.B) {
	e := setupEcho()

	body := []byte(`{"name":"John Doe"}`)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/users", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
	}
}

func BenchmarkWithCorrelationID(b *testing.B) {
	e := setupEcho()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/health", nil)
		req.Header.Set("X-Correlation-ID", "test-correlation-id-123")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
	}
}

func BenchmarkConcurrent(b *testing.B) {
	e := setupEcho()

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/users/123", nil)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)
		}
	})
}

// Test to ensure everything works
func TestGetUser(t *testing.T) {
	e := setupEcho()

	req := httptest.NewRequest("GET", "/users/123", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to parse response: %v", err)
	}

	if response["id"] != "123" {
		t.Errorf("Expected id 123, got %s", response["id"])
	}
}
