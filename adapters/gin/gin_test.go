package gin

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

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
	r.Use(New(goctxid.Config{Generator: generator}))

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
