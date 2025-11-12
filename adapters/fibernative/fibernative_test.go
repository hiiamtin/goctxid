package fibernative

import (
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
		config             []Config
		requestHeader      string
		requestHeaderValue string
		expectedInLocals   string
		expectedInResponse string
		checkResponseKey   string
	}{
		{
			name:               "generates new ID when header not present",
			config:             nil,
			requestHeader:      "",
			requestHeaderValue: "",
			expectedInLocals:   "", // Will be generated, just check it exists
			expectedInResponse: "", // Will be generated, just check it exists
			checkResponseKey:   goctxid.DefaultHeaderKey,
		},
		{
			name:               "uses existing ID from request header",
			config:             nil,
			requestHeader:      goctxid.DefaultHeaderKey,
			requestHeaderValue: "existing-correlation-id",
			expectedInLocals:   "existing-correlation-id",
			expectedInResponse: "existing-correlation-id",
			checkResponseKey:   goctxid.DefaultHeaderKey,
		},
		{
			name: "uses custom header key",
			config: []Config{
				{
					Config: goctxid.Config{
						HeaderKey: "X-Custom-ID",
					},
				},
			},
			requestHeader:      "X-Custom-ID",
			requestHeaderValue: "custom-id-123",
			expectedInLocals:   "custom-id-123",
			expectedInResponse: "custom-id-123",
			checkResponseKey:   "X-Custom-ID",
		},
		{
			name: "uses custom generator",
			config: []Config{
				{
					Config: goctxid.Config{
						Generator: func() string {
							return "custom-generated-id"
						},
					},
				},
			},
			requestHeader:      "",
			requestHeaderValue: "",
			expectedInLocals:   "custom-generated-id",
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

			// Test handler that checks Locals
			var localsID string
			app.Get("/test", func(c *fiber.Ctx) error {
				id, exists := FromLocals(c)
				if !exists {
					t.Error("Correlation ID not found in Locals")
				}
				localsID = id
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
			defer func() { _ = resp.Body.Close() }()

			// Check response header
			responseID := resp.Header.Get(tt.checkResponseKey)
			if responseID == "" {
				t.Error("Response header does not contain correlation ID")
			}

			// Check expected values
			if tt.expectedInLocals != "" {
				if localsID != tt.expectedInLocals {
					t.Errorf("Locals ID = %v, want %v", localsID, tt.expectedInLocals)
				}
			} else {
				// Just verify it's not empty
				if localsID == "" {
					t.Error("Locals ID is empty")
				}
			}

			if tt.expectedInResponse != "" {
				if responseID != tt.expectedInResponse {
					t.Errorf("Response header ID = %v, want %v", responseID, tt.expectedInResponse)
				}
			}

			// Verify Locals and response have same ID
			if localsID != responseID {
				t.Errorf("Locals ID (%v) != Response ID (%v)", localsID, responseID)
			}
		})
	}
}

func TestFromLocals(t *testing.T) {
	tests := []struct {
		name       string
		setupFunc  func(*fiber.Ctx)
		expectedID string
		expectedOK bool
	}{
		{
			name: "returns ID when present",
			setupFunc: func(c *fiber.Ctx) {
				c.Locals(DefaultLocalsKey, "test-id-123")
			},
			expectedID: "test-id-123",
			expectedOK: true,
		},
		{
			name: "returns empty when not present",
			setupFunc: func(c *fiber.Ctx) {
				// Don't set anything
			},
			expectedID: "",
			expectedOK: false,
		},
		{
			name: "returns empty when wrong type",
			setupFunc: func(c *fiber.Ctx) {
				c.Locals(DefaultLocalsKey, 12345) // Wrong type
			},
			expectedID: "",
			expectedOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()

			app.Get("/test", func(c *fiber.Ctx) error {
				tt.setupFunc(c)

				id, ok := FromLocals(c)

				if id != tt.expectedID {
					t.Errorf("FromLocals() id = %v, want %v", id, tt.expectedID)
				}
				if ok != tt.expectedOK {
					t.Errorf("FromLocals() ok = %v, want %v", ok, tt.expectedOK)
				}

				return c.SendString("OK")
			})

			req := httptest.NewRequest("GET", "/test", nil)
			resp, _ := app.Test(req)
			_ = resp.Body.Close()
		})
	}
}

func TestMustFromLocals(t *testing.T) {
	tests := []struct {
		name       string
		setupFunc  func(*fiber.Ctx)
		expectedID string
	}{
		{
			name: "returns ID when present",
			setupFunc: func(c *fiber.Ctx) {
				c.Locals(DefaultLocalsKey, "test-id-456")
			},
			expectedID: "test-id-456",
		},
		{
			name: "returns empty string when not present",
			setupFunc: func(c *fiber.Ctx) {
				// Don't set anything
			},
			expectedID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()

			app.Get("/test", func(c *fiber.Ctx) error {
				tt.setupFunc(c)

				id := MustFromLocals(c)

				if id != tt.expectedID {
					t.Errorf("MustFromLocals() = %v, want %v", id, tt.expectedID)
				}

				return c.SendString("OK")
			})

			req := httptest.NewRequest("GET", "/test", nil)
			resp, _ := app.Test(req)
			_ = resp.Body.Close()
		})
	}
}

func TestConfigDefault(t *testing.T) {
	tests := []struct {
		name              string
		config            []Config
		expectedHeaderKey string
		expectedLocalsKey string
		testGenerator     bool
	}{
		{
			name:              "uses defaults when no config provided",
			config:            nil,
			expectedHeaderKey: goctxid.DefaultHeaderKey,
			expectedLocalsKey: DefaultLocalsKey,
			testGenerator:     true,
		},
		{
			name:              "uses defaults when empty config provided",
			config:            []Config{{}},
			expectedHeaderKey: goctxid.DefaultHeaderKey,
			expectedLocalsKey: DefaultLocalsKey,
			testGenerator:     true,
		},
		{
			name: "uses custom header key",
			config: []Config{
				{Config: goctxid.Config{HeaderKey: "X-Request-ID"}},
			},
			expectedHeaderKey: "X-Request-ID",
			expectedLocalsKey: DefaultLocalsKey,
			testGenerator:     true,
		},
		{
			name: "uses custom generator",
			config: []Config{
				{
					Config: goctxid.Config{
						Generator: func() string { return "test" },
					},
				},
			},
			expectedHeaderKey: goctxid.DefaultHeaderKey,
			expectedLocalsKey: DefaultLocalsKey,
			testGenerator:     false,
		},
		{
			name: "uses custom locals key",
			config: []Config{
				{LocalsKey: "my_custom_key"},
			},
			expectedHeaderKey: goctxid.DefaultHeaderKey,
			expectedLocalsKey: "my_custom_key",
			testGenerator:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := configDefault(tt.config...)

			if cfg.HeaderKey != tt.expectedHeaderKey {
				t.Errorf("HeaderKey = %v, want %v", cfg.HeaderKey, tt.expectedHeaderKey)
			}

			if cfg.LocalsKey != tt.expectedLocalsKey {
				t.Errorf("LocalsKey = %v, want %v", cfg.LocalsKey, tt.expectedLocalsKey)
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
		id, _ := FromLocals(c)
		firstHandlerID = id
		return c.Next()
	})

	app.Get("/test", func(c *fiber.Ctx) error {
		id, _ := FromLocals(c)
		secondHandlerID = id
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	_ = resp.Body.Close()

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
		id, exists := FromLocals(c)
		if !exists {
			t.Error("Correlation ID not found in Locals")
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
			defer func() { _ = resp.Body.Close() }()

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
	app.Use(New(Config{Config: goctxid.Config{Generator: generator}}))

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
			_ = resp.Body.Close()
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
		_ = resp.Body.Close()
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
		_ = resp.Body.Close()
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
		_ = resp.Body.Close()
	}
}

func BenchmarkMiddlewareWithLocalsAccess(b *testing.B) {
	// With goctxid middleware - accessing ID from Locals in handler
	app := fiber.New()
	app.Use(New())
	app.Get("/test", func(c *fiber.Ctx) error {
		// Simulate real-world usage: accessing the correlation ID
		id := MustFromLocals(c)
		_ = id // Use the ID
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, _ := app.Test(req)
		_ = resp.Body.Close()
	}
}

// TestGoroutineSafety tests that correlation ID must be copied before using in goroutines
// This test demonstrates the CORRECT way to use fibernative with goroutines
func TestGoroutineSafety(t *testing.T) {
	app := fiber.New()
	app.Use(New())

	var wg sync.WaitGroup
	capturedIDs := make([]string, 0)
	var mu sync.Mutex

	// ✅ CORRECT: Copy the value before using in goroutine
	app.Get("/correct", func(c *fiber.Ctx) error {
		// Copy the correlation ID before spawning goroutine
		correlationID := MustFromLocals(c)

		wg.Add(1)
		go func() {
			defer wg.Done()
			// Small delay to ensure handler completes first
			time.Sleep(10 * time.Millisecond)

			// Use the copied value - this is safe
			mu.Lock()
			capturedIDs = append(capturedIDs, correlationID)
			mu.Unlock()
		}()

		return c.SendString("OK")
	})

	// Make request
	req := httptest.NewRequest("GET", "/correct", nil)
	req.Header.Set(goctxid.DefaultHeaderKey, "test-id-123")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	_ = resp.Body.Close()

	// Wait for goroutine to complete
	wg.Wait()

	// Verify the goroutine captured the correct ID
	if len(capturedIDs) != 1 {
		t.Fatalf("Expected 1 captured ID, got %d", len(capturedIDs))
	}

	if capturedIDs[0] != "test-id-123" {
		t.Errorf("Expected captured ID to be 'test-id-123', got '%s'", capturedIDs[0])
	}
}

// TestGoroutineUnsafe demonstrates the WRONG way (for documentation purposes)
// This test shows what happens when you try to access c.Locals() after handler completes
func TestGoroutineUnsafe(t *testing.T) {
	t.Skip("This test demonstrates unsafe behavior - skipped by default")

	app := fiber.New()
	app.Use(New())

	var wg sync.WaitGroup
	var capturedID string
	var mu sync.Mutex

	// ❌ WRONG: Don't access c.Locals() in goroutine
	app.Get("/unsafe", func(c *fiber.Ctx) error {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Wait to ensure handler completes and context is recycled
			time.Sleep(50 * time.Millisecond)

			// ⚠️ DANGER: Accessing c after handler completes
			// This may panic, return wrong value, or cause race conditions
			id := MustFromLocals(c)

			mu.Lock()
			capturedID = id
			mu.Unlock()
		}()

		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/unsafe", nil)
	req.Header.Set(goctxid.DefaultHeaderKey, "test-id-456")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	_ = resp.Body.Close()

	wg.Wait()

	// The captured ID may be empty, wrong, or cause a panic
	t.Logf("Captured ID (may be wrong): %s", capturedID)
}

// TestConcurrentRequestsWithGoroutinesCopyValue tests that copying values before goroutines
// works correctly with TRUE concurrent requests using a real HTTP server.
func TestConcurrentRequestsWithGoroutinesCopyValue(t *testing.T) {
	app := fiber.New()
	app.Use(New())

	type result struct {
		requestID  string
		capturedID string
	}

	results := make([]result, 0)
	var resultsMu sync.Mutex
	var handlerWg sync.WaitGroup

	// ✅ CORRECT: Copy the value before using in goroutine
	app.Get("/test", func(c *fiber.Ctx) error {
		// Copy the correlation ID before spawning goroutine
		correlationID := MustFromLocals(c)

		// Uncomment to debug:
		// t.Logf("Handler: correlationID = %s", correlationID)

		handlerWg.Add(1)
		go func(copiedID string) {
			defer handlerWg.Done()
			// Delay to simulate async processing and ensure handler completes first
			time.Sleep(50 * time.Millisecond)

			// Use the copied value - this is safe
			// Uncomment to debug:
			// t.Logf("Goroutine: copiedID = %s", copiedID)

			resultsMu.Lock()
			results = append(results, result{
				requestID:  copiedID,
				capturedID: copiedID,
			})
			resultsMu.Unlock()
		}(correlationID)

		return c.SendString("OK")
	})

	// Find an available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to find available port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()

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
			defer func() { _ = resp.Body.Close() }()
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

	// Uncomment to debug:
	// t.Logf("=== Results ===")
	// for i, r := range results {
	// 	t.Logf("[%d] requestID: %s, capturedID: %s", i, r.requestID, r.capturedID)
	// }

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
			t.Errorf("Duplicate ID found: %s - This means values got mixed up!", r.requestID)
		}
		seenIDs[r.requestID] = true
	}

	if len(seenIDs) != numRequests {
		t.Errorf("Expected %d unique IDs, got %d - Values got mixed up!", numRequests, len(seenIDs))
	}

	// Uncomment to debug:
	// t.Logf("✅ Test passed! All %d requests had unique IDs and goroutines captured correct values", numRequests)
}

// TestConcurrentRequestsWithGoroutinesUnsafe demonstrates the WRONG way
// This test shows what happens when you access c.Locals() in goroutines
// The IDs will likely get mixed up or cause race conditions
func TestConcurrentRequestsWithGoroutinesUnsafe(t *testing.T) {
	t.Skip("This test demonstrates unsafe behavior - skipped by default")

	app := fiber.New()
	app.Use(New())

	type result struct {
		requestID  string
		capturedID string
	}

	results := make([]result, 0)
	var resultsMu sync.Mutex
	var wg sync.WaitGroup

	// ❌ WRONG: Don't access c.Locals() in goroutine
	app.Get("/unsafe", func(c *fiber.Ctx) error {
		// Get the ID in the handler (this is safe)
		requestID := MustFromLocals(c)

		wg.Add(1)
		go func(expectedID string) {
			defer wg.Done()
			// Wait to ensure handler completes and context is recycled
			time.Sleep(50 * time.Millisecond)

			// ⚠️ DANGER: Accessing c after handler completes
			// This may panic, return wrong value, or cause race conditions
			capturedID := MustFromLocals(c)

			resultsMu.Lock()
			results = append(results, result{
				requestID:  expectedID,
				capturedID: capturedID,
			})
			resultsMu.Unlock()
		}(requestID)

		return c.SendString("OK")
	})

	// Send multiple requests - each will spawn a goroutine
	numRequests := 5

	for i := 0; i < numRequests; i++ {
		req := httptest.NewRequest("GET", "/unsafe", nil)
		req.Header.Set(goctxid.DefaultHeaderKey, fmt.Sprintf("request-%d", i))

		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Request %d failed: %v", i, err)
		}
		_ = resp.Body.Close()
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Log results - they will likely show ID mismatches
	t.Logf("Total results: %d (expected %d)", len(results), numRequests)

	mismatches := 0
	for _, r := range results {
		if r.requestID != r.capturedID {
			mismatches++
			t.Logf("ID mismatch! Request had '%s' but goroutine captured '%s'",
				r.requestID, r.capturedID)
		}
	}

	if mismatches > 0 {
		t.Logf("⚠️ Found %d ID mismatches out of %d results - this demonstrates the problem!", mismatches, len(results))
	}
}
