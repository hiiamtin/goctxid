package gin

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hiiamtin/goctxid"
)

func init() {
	// Set Gin to test mode to reduce noise in test output
	gin.SetMode(gin.TestMode)
}

func TestNew(t *testing.T) {
	tests := []struct {
		name               string
		config             []Config
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
			config: []Config{
				{
					Config: goctxid.Config{
						HeaderKey: "X-Custom-ID",
					},
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
			expectedInContext:  "custom-generated-id",
			expectedInResponse: "custom-generated-id",
			checkResponseKey:   goctxid.DefaultHeaderKey,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()

			// Apply middleware
			if tt.config != nil {
				r.Use(New(tt.config...))
			} else {
				r.Use(New())
			}

			// Test handler that checks context
			var contextID string
			r.GET("/test", func(c *gin.Context) {
				id, exists := goctxid.FromContext(c.Request.Context())
				if !exists {
					t.Error("Correlation ID not found in context")
				}
				contextID = id
				c.String(http.StatusOK, "OK")
			})

			// Create request
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.requestHeader != "" {
				req.Header.Set(tt.requestHeader, tt.requestHeaderValue)
			}
			rec := httptest.NewRecorder()

			// Execute request
			r.ServeHTTP(rec, req)

			// Check response header
			responseID := rec.Header().Get(tt.checkResponseKey)
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
		config            []Config
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
			config:            []Config{{}},
			expectedHeaderKey: goctxid.DefaultHeaderKey,
			testGenerator:     true,
		},
		{
			name: "uses custom header key",
			config: []Config{
				{Config: goctxid.Config{HeaderKey: "X-Request-ID"}},
			},
			expectedHeaderKey: "X-Request-ID",
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
	r := gin.New()
	r.Use(New())

	var firstHandlerID, secondHandlerID string

	r.Use(func(c *gin.Context) {
		id, _ := goctxid.FromContext(c.Request.Context())
		firstHandlerID = id
		c.Next()
	})

	r.GET("/test", func(c *gin.Context) {
		id, _ := goctxid.FromContext(c.Request.Context())
		secondHandlerID = id
		c.String(http.StatusOK, "OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if firstHandlerID == "" || secondHandlerID == "" {
		t.Error("Correlation ID not propagated through middleware chain")
	}

	if firstHandlerID != secondHandlerID {
		t.Errorf("Correlation ID changed in middleware chain: %v != %v", firstHandlerID, secondHandlerID)
	}
}

func TestConcurrentRequests(t *testing.T) {
	r := gin.New()

	var mu sync.Mutex
	seenIDs := make(map[string]bool)

	r.Use(New())

	r.GET("/test", func(c *gin.Context) {
		id, exists := goctxid.FromContext(c.Request.Context())
		if !exists {
			t.Error("Correlation ID not found in context")
		}

		mu.Lock()
		seenIDs[id] = true
		mu.Unlock()

		c.String(http.StatusOK, id)
	})

	// Make multiple concurrent requests
	var wg sync.WaitGroup
	numRequests := 50

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest("GET", "/test", nil)
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			body, _ := io.ReadAll(rec.Body)
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

	r := gin.New()
	r.Use(New(Config{Config: goctxid.Config{Generator: generator}}))

	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	var wg sync.WaitGroup
	numRequests := 20

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest("GET", "/test", nil)
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)
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
	// Baseline: Gin app WITHOUT goctxid middleware
	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
	}
}

func BenchmarkMiddleware(b *testing.B) {
	// With goctxid middleware - generates new ID
	r := gin.New()
	r.Use(New())
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
	}
}

func BenchmarkMiddlewareWithExistingID(b *testing.B) {
	// With goctxid middleware - uses existing ID from header
	r := gin.New()
	r.Use(New())
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set(goctxid.DefaultHeaderKey, "existing-id-123")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
	}
}

func BenchmarkMiddlewareWithContextAccess(b *testing.B) {
	// With goctxid middleware - accessing ID from context in handler
	r := gin.New()
	r.Use(New())
	r.GET("/test", func(c *gin.Context) {
		// Simulate real-world usage: accessing the correlation ID
		id, _ := goctxid.FromContext(c.Request.Context())
		_ = id // Use the ID
		c.String(http.StatusOK, "OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
	}
}

// TestGoroutineSafety tests that context-based approach is safe for goroutines
func TestGoroutineSafety(t *testing.T) {
	r := gin.New()
	r.Use(New())

	var wg sync.WaitGroup
	capturedIDs := make([]string, 0)
	var mu sync.Mutex

	r.GET("/safe", func(c *gin.Context) {
		// Get the context - it's immutable and safe to pass to goroutines
		ctx := c.Request.Context()

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

		c.String(http.StatusOK, "OK")
	})

	// Make request
	req := httptest.NewRequest("GET", "/safe", nil)
	req.Header.Set(goctxid.DefaultHeaderKey, "gin-test-id")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	// Wait for goroutine to complete
	wg.Wait()

	// Verify the goroutine captured the correct ID
	if len(capturedIDs) != 1 {
		t.Fatalf("Expected 1 captured ID, got %d", len(capturedIDs))
	}

	if capturedIDs[0] != "gin-test-id" {
		t.Errorf("Expected captured ID to be 'gin-test-id', got '%s'", capturedIDs[0])
	}
}

// TestMultipleGoroutines tests that context can be safely shared across multiple goroutines
func TestMultipleGoroutines(t *testing.T) {
	r := gin.New()
	r.Use(New())

	const numGoroutines = 10
	var wg sync.WaitGroup
	capturedIDs := make([]string, 0, numGoroutines)
	var mu sync.Mutex

	r.GET("/multi", func(c *gin.Context) {
		ctx := c.Request.Context()

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

		c.String(http.StatusOK, "OK")
	})

	req := httptest.NewRequest("GET", "/multi", nil)
	req.Header.Set(goctxid.DefaultHeaderKey, "gin-multi-id")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	wg.Wait()

	// Verify all goroutines captured the same ID
	if len(capturedIDs) != numGoroutines {
		t.Fatalf("Expected %d captured IDs, got %d", numGoroutines, len(capturedIDs))
	}

	for i, id := range capturedIDs {
		if id != "gin-multi-id" {
			t.Errorf("Goroutine %d: expected 'gin-multi-id', got '%s'", i, id)
		}
	}
}

// TestConcurrentRequestsWithGoroutines tests that correlation IDs don't get mixed up
// when multiple concurrent requests each spawn goroutines that access the ID
func TestConcurrentRequestsWithGoroutines(t *testing.T) {
	r := gin.New()
	r.Use(New())

	type result struct {
		requestID  string
		capturedID string
	}

	results := make([]result, 0)
	var mu sync.Mutex
	var wg sync.WaitGroup

	r.GET("/test", func(c *gin.Context) {
		// Get the context - it's immutable and safe to pass to goroutines
		ctx := c.Request.Context()
		requestID := goctxid.MustFromContext(ctx)

		// Uncomment to debug:
		// t.Logf("Handler: requestID = %s, ctx = %p", requestID, ctx)

		wg.Add(1)
		go func(capturedCtx context.Context, expectedID string) {
			defer wg.Done()
			// Delay to simulate async processing and ensure handler completes first
			time.Sleep(50 * time.Millisecond)

			// Access ID from goroutine - should still be the correct ID
			capturedID := goctxid.MustFromContext(capturedCtx)

			// Uncomment to debug:
			// t.Logf("Goroutine: expectedID = %s, capturedID = %s, ctx = %p", expectedID, capturedID, capturedCtx)

			mu.Lock()
			results = append(results, result{
				requestID:  expectedID,
				capturedID: capturedID,
			})
			mu.Unlock()
		}(ctx, requestID)

		c.String(http.StatusOK, "OK")
	})

	// Send multiple concurrent requests with different IDs
	numRequests := 20
	requestWg := sync.WaitGroup{}

	for i := 0; i < numRequests; i++ {
		requestWg.Add(1)
		go func(index int) {
			defer requestWg.Done()

			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set(goctxid.DefaultHeaderKey, fmt.Sprintf("request-%d", index))
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)
		}(i)
	}

	// Wait for all requests to complete
	requestWg.Wait()

	// Wait for all goroutines to complete
	wg.Wait()

	// Verify: Each goroutine should capture the SAME ID as its request
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

	// Verify all IDs are unique (no duplicates)
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

	// Uncomment to debug:
	// t.Logf("✅ Test passed! All %d requests had unique IDs and goroutines captured correct values", numRequests)
}

// TestReExportedFunctions tests that re-exported functions work correctly
// This allows users to use goctxid_gin.FromContext() instead of importing goctxid
func TestReExportedFunctions(t *testing.T) {
	ctx := context.Background()

	// Test NewContext
	testID := "test-correlation-id-123"
	newCtx := NewContext(ctx, testID)

	// Test FromContext
	retrievedID, exists := FromContext(newCtx)
	if !exists {
		t.Error("FromContext should return true for existing ID")
	}
	if retrievedID != testID {
		t.Errorf("FromContext returned wrong ID: got %s, want %s", retrievedID, testID)
	}

	// Test MustFromContext
	mustID := MustFromContext(newCtx)
	if mustID != testID {
		t.Errorf("MustFromContext returned wrong ID: got %s, want %s", mustID, testID)
	}

	// Test MustFromContext with empty context
	emptyID := MustFromContext(ctx)
	if emptyID != "" {
		t.Errorf("MustFromContext should return empty string for context without ID, got %s", emptyID)
	}

	// Test re-exported constants
	if DefaultHeaderKey != "X-Correlation-ID" {
		t.Errorf("DefaultHeaderKey should be X-Correlation-ID, got %s", DefaultHeaderKey)
	}

	// Test re-exported generators
	if DefaultGenerator == nil {
		t.Error("DefaultGenerator should not be nil")
	}
	if FastGenerator == nil {
		t.Error("FastGenerator should not be nil")
	}

	// Test that generators actually work
	id1 := DefaultGenerator()
	if id1 == "" {
		t.Error("DefaultGenerator should return non-empty ID")
	}

	id2 := FastGenerator()
	if id2 == "" {
		t.Error("FastGenerator should return non-empty ID")
	}
}

// TestGetCorrelationID tests the GetCorrelationID convenience function
func TestGetCorrelationID(t *testing.T) {
	r := gin.New()
	r.Use(New())

	r.GET("/test", func(c *gin.Context) {
		// Test GetCorrelationID function
		id := GetCorrelationID(c)
		if id == "" {
			t.Error("GetCorrelationID should return non-empty ID")
		}

		// Verify it matches what's in the context
		idFromContext := MustFromContext(c.Request.Context())
		if id != idFromContext {
			t.Errorf("GetCorrelationID (%s) should match MustFromContext (%s)", id, idFromContext)
		}

		c.String(200, id)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
}

// TestNextFunction tests the Next configuration option
func TestNextFunction(t *testing.T) {
	r := gin.New()

	// Configure middleware to skip requests to /skip path
	r.Use(New(Config{
		Next: func(c *gin.Context) bool {
			return c.Request.URL.Path == "/skip"
		},
	}))

	r.GET("/skip", func(c *gin.Context) {
		// This should NOT have a correlation ID because middleware was skipped
		id := GetCorrelationID(c)
		if id != "" {
			t.Errorf("Expected empty ID for skipped path, got %s", id)
		}
		c.String(200, "skipped")
	})

	r.GET("/process", func(c *gin.Context) {
		// This SHOULD have a correlation ID
		id := GetCorrelationID(c)
		if id == "" {
			t.Error("Expected non-empty ID for processed path")
		}
		c.String(200, id)
	})

	// Test skipped path
	req := httptest.NewRequest("GET", "/skip", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("Expected status 200 for /skip, got %d", rec.Code)
	}

	// Verify no correlation ID header was set
	if rec.Header().Get(DefaultHeaderKey) != "" {
		t.Errorf("Expected no correlation ID header for skipped path, got %s", rec.Header().Get(DefaultHeaderKey))
	}

	// Test processed path
	req = httptest.NewRequest("GET", "/process", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("Expected status 200 for /process, got %d", rec.Code)
	}

	// Verify correlation ID header was set
	if rec.Header().Get(DefaultHeaderKey) == "" {
		t.Error("Expected correlation ID header for processed path")
	}
}
