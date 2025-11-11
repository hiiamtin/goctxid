package fiberlocals

import (
	"io"
	"net/http/httptest"
	"sync"
	"testing"

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
			config: []goctxid.Config{
				{
					HeaderKey: "X-Custom-ID",
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
			config: []goctxid.Config{
				{
					Generator: func() string {
						return "custom-generated-id"
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
			defer resp.Body.Close()

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
				c.Locals(LocalsKey, "test-id-123")
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
				c.Locals(LocalsKey, 12345) // Wrong type
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
			resp.Body.Close()
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
				c.Locals(LocalsKey, "test-id-456")
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
			resp.Body.Close()
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
	resp.Body.Close()

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
		resp.Body.Close()
	}
}
