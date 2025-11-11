package fiber

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/hiiamtin/goctxid"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name               string
		config             []goctxid.Config
		requestHeader      string
		requestHeaderValue string
		expectedInContext  string
		expectedInResponse string
		checkResponseKey   string
	}{
		{
			name:               "generates new ID when header not present",
			config:             nil,
			requestHeader:      "",
			requestHeaderValue: "",
			expectedInContext:  "", // Will be generated, just check it exists
			expectedInResponse: "", // Will be generated, just check it exists
			checkResponseKey:   goctxid.DefaultHeaderKey,
		},
		{
			name:               "uses existing ID from request header",
			config:             nil,
			requestHeader:      goctxid.DefaultHeaderKey,
			requestHeaderValue: "existing-correlation-id",
			expectedInContext:  "existing-correlation-id",
			expectedInResponse: "existing-correlation-id",
			checkResponseKey:   goctxid.DefaultHeaderKey,
		},
		{
			name: "uses custom header key",
			config: []goctxid.Config{
				{
					HeaderKey: "X-Custom-ID",
				},
			},
			requestHeader:      "X-Custom-ID",
			requestHeaderValue: "custom-id-123",
			expectedInContext:  "custom-id-123",
			expectedInResponse: "custom-id-123",
			checkResponseKey:   "X-Custom-ID",
		},
		{
			name: "uses custom generator",
			config: []goctxid.Config{
				{
					Generator: func() string {
						return "custom-generated-id"
					},
				},
			},
			requestHeader:      "",
			requestHeaderValue: "",
			expectedInContext:  "custom-generated-id",
			expectedInResponse: "custom-generated-id",
			checkResponseKey:   goctxid.DefaultHeaderKey,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()

			// Apply middleware
			if tt.config != nil {
				app.Use(New(tt.config...))
			} else {
				app.Use(New())
			}

			// Test handler that checks context
			var contextID string
			app.Get("/test", func(c *fiber.Ctx) error {
				id, exists := goctxid.FromContext(c.UserContext())
				if !exists {
					t.Error("Correlation ID not found in context")
				}
				contextID = id
				return c.SendString("OK")
			})

			// Create request
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.requestHeader != "" {
				req.Header.Set(tt.requestHeader, tt.requestHeaderValue)
			}

			// Execute request
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			defer resp.Body.Close()

			// Check response header
			responseID := resp.Header.Get(tt.checkResponseKey)
			if responseID == "" {
				t.Error("Response header does not contain correlation ID")
			}

			// Check expected values
			if tt.expectedInContext != "" {
				if contextID != tt.expectedInContext {
					t.Errorf("Context ID = %v, want %v", contextID, tt.expectedInContext)
				}
			} else {
				// Just verify it's not empty
				if contextID == "" {
					t.Error("Context ID is empty")
				}
			}

			if tt.expectedInResponse != "" {
				if responseID != tt.expectedInResponse {
					t.Errorf("Response header ID = %v, want %v", responseID, tt.expectedInResponse)
				}
			}

			// Verify context and response have same ID
			if contextID != responseID {
				t.Errorf("Context ID (%v) != Response ID (%v)", contextID, responseID)
			}
		})
	}
}

func TestConfigDefault(t *testing.T) {
	tests := []struct {
		name              string
		config            []goctxid.Config
		expectedHeaderKey string
		testGenerator     bool
	}{
		{
			name:              "uses defaults when no config provided",
			config:            nil,
			expectedHeaderKey: goctxid.DefaultHeaderKey,
			testGenerator:     true,
		},
		{
			name:              "uses defaults when empty config provided",
			config:            []goctxid.Config{{}},
			expectedHeaderKey: goctxid.DefaultHeaderKey,
			testGenerator:     true,
		},
		{
			name: "uses custom header key",
			config: []goctxid.Config{
				{HeaderKey: "X-Request-ID"},
			},
			expectedHeaderKey: "X-Request-ID",
			testGenerator:     true,
		},
		{
			name: "uses custom generator",
			config: []goctxid.Config{
				{
					Generator: func() string { return "test" },
				},
			},
			expectedHeaderKey: goctxid.DefaultHeaderKey,
			testGenerator:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := configDefault(tt.config...)

			if cfg.HeaderKey != tt.expectedHeaderKey {
				t.Errorf("HeaderKey = %v, want %v", cfg.HeaderKey, tt.expectedHeaderKey)
			}

			if cfg.Generator == nil {
				t.Error("Generator is nil")
			}

			if tt.testGenerator {
				// Test that default generator works
				id := cfg.Generator()
				if id == "" {
					t.Error("Generator returned empty string")
				}
			}
		})
	}
}

func TestMiddlewareChaining(t *testing.T) {
	app := fiber.New()
	app.Use(New())

	var firstHandlerID, secondHandlerID string

	app.Use(func(c *fiber.Ctx) error {
		id, _ := goctxid.FromContext(c.UserContext())
		firstHandlerID = id
		return c.Next()
	})

	app.Get("/test", func(c *fiber.Ctx) error {
		id, _ := goctxid.FromContext(c.UserContext())
		secondHandlerID = id
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if firstHandlerID == "" || secondHandlerID == "" {
		t.Error("Correlation ID not propagated through middleware chain")
	}

	if firstHandlerID != secondHandlerID {
		t.Errorf("Correlation ID changed in middleware chain: %v != %v", firstHandlerID, secondHandlerID)
	}
}

func TestConcurrentRequests(t *testing.T) {
	app := fiber.New()

	var mu sync.Mutex
	seenIDs := make(map[string]bool)

	app.Use(New())

	app.Get("/test", func(c *fiber.Ctx) error {
		id, exists := goctxid.FromContext(c.UserContext())
		if !exists {
			t.Error("Correlation ID not found in context")
		}

		mu.Lock()
		seenIDs[id] = true
		mu.Unlock()

		return c.SendString(id)
	})

	// Make multiple concurrent requests
	var wg sync.WaitGroup
	numRequests := 50

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest("GET", "/test", nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Errorf("Request failed: %v", err)
				return
			}
			defer resp.Body.Close()

			body, _ := io.ReadAll(resp.Body)
			responseID := string(body)

			if responseID == "" {
				t.Error("Empty correlation ID in response")
			}
		}()
	}

	wg.Wait()

	// Verify we got unique IDs for each request
	mu.Lock()
	uniqueCount := len(seenIDs)
	mu.Unlock()

	if uniqueCount != numRequests {
		t.Errorf("Expected %d unique IDs, got %d", numRequests, uniqueCount)
	}
}

func TestGeneratorThreadSafety(t *testing.T) {
	// Test that custom generator is called safely from multiple goroutines
	var callCount int
	var mu sync.Mutex

	generator := func() string {
		mu.Lock()
		callCount++
		mu.Unlock()
		return uuid.NewString() // Use a different ID for each call
	}

	app := fiber.New()
	app.Use(New(goctxid.Config{Generator: generator}))

	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	var wg sync.WaitGroup
	numRequests := 20

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest("GET", "/test", nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Errorf("Request failed: %v", err)
				return
			}
			resp.Body.Close()
		}()
	}

	wg.Wait()

	mu.Lock()
	finalCount := callCount
	mu.Unlock()

	if finalCount != numRequests {
		t.Errorf("Generator called %d times, expected %d", finalCount, numRequests)
	}
}

func BenchmarkBaseline(b *testing.B) {
	// Baseline: Fiber app WITHOUT goctxid middleware
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, _ := app.Test(req)
		resp.Body.Close()
	}
}

func BenchmarkMiddleware(b *testing.B) {
	// With goctxid middleware - generates new ID
	app := fiber.New()
	app.Use(New())
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, _ := app.Test(req)
		resp.Body.Close()
	}
}

func BenchmarkMiddlewareWithExistingID(b *testing.B) {
	// With goctxid middleware - uses existing ID from header
	app := fiber.New()
	app.Use(New())
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set(goctxid.DefaultHeaderKey, "existing-id-123")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, _ := app.Test(req)
		resp.Body.Close()
	}
}

func BenchmarkMiddlewareWithContextAccess(b *testing.B) {
	// With goctxid middleware - accessing ID from context in handler
	app := fiber.New()
	app.Use(New())
	app.Get("/test", func(c *fiber.Ctx) error {
		// Simulate real-world usage: accessing the correlation ID
		id, _ := goctxid.FromContext(c.UserContext())
		_ = id // Use the ID
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, _ := app.Test(req)
		resp.Body.Close()
	}
}

// TestGoroutineSafety tests that context-based approach is safer for goroutines
// Context is immutable and can be safely passed to goroutines
func TestGoroutineSafety(t *testing.T) {
	app := fiber.New()
	app.Use(New())

	var wg sync.WaitGroup
	capturedIDs := make([]string, 0)
	var mu sync.Mutex

	// ✅ Context-based approach is safer for goroutines
	app.Get("/safe", func(c *fiber.Ctx) error {
		// Get the context - it's immutable and safe to pass to goroutines
		ctx := c.UserContext()

		wg.Add(1)
		go func() {
			defer wg.Done()
			// Small delay to ensure handler completes first
			time.Sleep(10 * time.Millisecond)

			// Access correlation ID from context - this is safe
			id := goctxid.MustFromContext(ctx)

			mu.Lock()
			capturedIDs = append(capturedIDs, id)
			mu.Unlock()
		}()

		return c.SendString("OK")
	})

	// Make request
	req := httptest.NewRequest("GET", "/safe", nil)
	req.Header.Set(goctxid.DefaultHeaderKey, "test-id-789")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	resp.Body.Close()

	// Wait for goroutine to complete
	wg.Wait()

	// Verify the goroutine captured the correct ID
	if len(capturedIDs) != 1 {
		t.Fatalf("Expected 1 captured ID, got %d", len(capturedIDs))
	}

	if capturedIDs[0] != "test-id-789" {
		t.Errorf("Expected captured ID to be 'test-id-789', got '%s'", capturedIDs[0])
	}
}

// TestMultipleGoroutines tests that context can be safely shared across multiple goroutines
func TestMultipleGoroutines(t *testing.T) {
	app := fiber.New()
	app.Use(New())

	const numGoroutines = 10
	var wg sync.WaitGroup
	capturedIDs := make([]string, 0, numGoroutines)
	var mu sync.Mutex

	app.Get("/multi", func(c *fiber.Ctx) error {
		ctx := c.UserContext()

		// Spawn multiple goroutines
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				time.Sleep(time.Duration(index) * time.Millisecond)

				// All goroutines should get the same ID
				id := goctxid.MustFromContext(ctx)

				mu.Lock()
				capturedIDs = append(capturedIDs, id)
				mu.Unlock()
			}(i)
		}

		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/multi", nil)
	req.Header.Set(goctxid.DefaultHeaderKey, "multi-test-id")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	resp.Body.Close()

	wg.Wait()

	// Verify all goroutines captured the same ID
	if len(capturedIDs) != numGoroutines {
		t.Fatalf("Expected %d captured IDs, got %d", numGoroutines, len(capturedIDs))
	}

	for i, id := range capturedIDs {
		if id != "multi-test-id" {
			t.Errorf("Goroutine %d: expected 'multi-test-id', got '%s'", i, id)
		}
	}
}

// TestConcurrentRequestsWithGoroutines tests that correlation IDs are preserved correctly
// when passed to goroutines with TRUE concurrent requests using a real HTTP server.
func TestConcurrentRequestsWithGoroutines(t *testing.T) {
	app := fiber.New()
	app.Use(New())

	type result struct {
		requestID  string
		capturedID string
	}

	results := make([]result, 0)
	var resultsMu sync.Mutex
	var handlerWg sync.WaitGroup

	app.Get("/test", func(c *fiber.Ctx) error {
		// Get the context - it's immutable and safe to pass to goroutines
		ctx := c.UserContext()
		requestID := goctxid.MustFromContext(ctx)

		handlerWg.Add(1)
		go func(capturedCtx context.Context, expectedID string) {
			defer handlerWg.Done()
			// Delay to simulate async processing and ensure handler completes first
			time.Sleep(50 * time.Millisecond)

			// Access ID from goroutine - should still be the correct ID
			capturedID := goctxid.MustFromContext(capturedCtx)

			resultsMu.Lock()
			results = append(results, result{
				requestID:  expectedID,
				capturedID: capturedID,
			})
			resultsMu.Unlock()
		}(ctx, requestID)

		return c.SendString("OK")
	})

	// Find an available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to find available port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	// Start a real HTTP server in a goroutine
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	go func() {
		if err := app.Listen(addr); err != nil {
			t.Logf("Server error: %v", err)
		}
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Shutdown server after test
	defer func() {
		if err := app.Shutdown(); err != nil {
			t.Logf("Server shutdown error: %v", err)
		}
	}()

	// Send TRUE concurrent requests using goroutines
	numRequests := 10
	var clientWg sync.WaitGroup

	for i := 0; i < numRequests; i++ {
		clientWg.Add(1)
		go func(requestNum int) {
			defer clientWg.Done()

			url := fmt.Sprintf("http://%s/test", addr)
			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				t.Errorf("Failed to create request %d: %v", requestNum, err)
				return
			}
			req.Header.Set(goctxid.DefaultHeaderKey, fmt.Sprintf("request-%d", requestNum))

			client := &http.Client{Timeout: 5 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				t.Errorf("Request %d failed: %v", requestNum, err)
				return
			}
			defer resp.Body.Close()
		}(i)
	}

	// Wait for all client requests to complete
	clientWg.Wait()

	// Wait for all handler goroutines to complete
	handlerWg.Wait()

	// Verify: Each goroutine should capture the correct ID
	if len(results) != numRequests {
		t.Fatalf("Expected %d results, got %d", numRequests, len(results))
	}

	// Check that no IDs got mixed up
	for _, r := range results {
		if r.requestID != r.capturedID {
			t.Errorf("ID mismatch! Request had '%s' but goroutine captured '%s'",
				r.requestID, r.capturedID)
		}
	}

	// Verify all IDs are unique (this is the real test!)
	seenIDs := make(map[string]bool)
	for _, r := range results {
		if seenIDs[r.requestID] {
			t.Errorf("Duplicate ID found: %s - This means contexts got mixed up!", r.requestID)
		}
		seenIDs[r.requestID] = true
	}

	if len(seenIDs) != numRequests {
		t.Errorf("Expected %d unique IDs, got %d - Contexts got mixed up!", numRequests, len(seenIDs))
	}
}
