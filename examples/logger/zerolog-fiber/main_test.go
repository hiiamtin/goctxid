package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v2"
	goctxid_fiber "github.com/hiiamtin/goctxid/adapters/fiber"
)

// ⚠️ WARNING: THESE BENCHMARKS ARE UNRELIABLE AND DO NOT MEASURE REAL PERFORMANCE ⚠️
//
// Problem: Fiber uses app.Test() which creates a full HTTP connection for each iteration,
// while Gin/Echo use ServeHTTP() which is a direct handler call. This makes Fiber appear
// artificially slow in benchmarks (~24,000 ns/op) when real-world load testing with k6
// shows Fiber is actually FASTER than Gin (~28% faster in production scenarios).
//
// Why this happens:
// - app.Test() creates HTTP server + TCP connection + network stack overhead
// - ServeHTTP() (Gin/Echo) calls handler directly with minimal overhead
// - This creates an unfair comparison that doesn't reflect real-world performance
//
// Real-world performance (k6 load testing):
// - Fiber: 7,512 RPS (Database I/O) - FASTEST
// - Gin:   5,877 RPS (Database I/O)
// - Fiber is ~28% faster in production
//
// Conclusion: DO NOT use these benchmarks to compare frameworks.
// Use real load testing tools (k6, wrk, ab) instead.
//
// Reference: https://github.com/hiiamtin/go-vs-java/blob/main/COMPARISON_REPORT.md

func setupApp() *fiber.App {
	SetupBenchmarkLogger() // Configure logger to write to discard for benchmarks

	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	app.Use(goctxid_fiber.New())
	app.Use(zerologMiddleware())

	app.Get("/health", healthCheck)
	app.Get("/users/:id", getUser)
	app.Post("/users", createUser)

	return app
}

func BenchmarkHealthCheck(b *testing.B) {
	app := setupApp()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("GET", "/health", nil)
		app.Test(req, -1)
	}
}

func BenchmarkGetUser(b *testing.B) {
	app := setupApp()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("GET", "/users/123", nil)
		app.Test(req, -1)
	}
}

func BenchmarkCreateUser(b *testing.B) {
	app := setupApp()

	body := []byte(`{"name":"John Doe"}`)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("POST", "/users", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		app.Test(req, -1)
	}
}

func BenchmarkWithCorrelationID(b *testing.B) {
	app := setupApp()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("GET", "/health", nil)
		req.Header.Set("X-Correlation-ID", "test-correlation-id-123")
		app.Test(req, -1)
	}
}

func BenchmarkConcurrent(b *testing.B) {
	app := setupApp()

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req, _ := http.NewRequest("GET", "/users/123", nil)
			app.Test(req, -1)
		}
	})
}

// Test to ensure everything works
func TestGetUser(t *testing.T) {
	app := setupApp()

	req, _ := http.NewRequest("GET", "/users/123", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var response map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Errorf("Failed to parse response: %v", err)
	}

	if response["id"] != "123" {
		t.Errorf("Expected id 123, got %s", response["id"])
	}
}
