package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/hiiamtin/goctxid"
)

// Middleware for standard net/http
func correlationIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Extract correlation ID from request header
		correlationID := r.Header.Get(goctxid.DefaultHeaderKey)

		// 2. If not found, generate a new one
		if correlationID == "" {
			correlationID = goctxid.DefaultGenerator()
		}

		// 3. Set response header
		w.Header().Set(goctxid.DefaultHeaderKey, correlationID)

		// 4. Create new context with correlation ID
		ctx := goctxid.NewContext(r.Context(), correlationID)

		// 5. Create new request with updated context
		r = r.WithContext(ctx)

		// 6. Continue to next handler
		next.ServeHTTP(w, r)
	})
}

// Custom middleware with configuration
func correlationIDMiddlewareWithConfig(config goctxid.Config) func(http.Handler) http.Handler {
	// Set defaults
	if config.HeaderKey == "" {
		config.HeaderKey = goctxid.DefaultHeaderKey
	}
	if config.Generator == nil {
		config.Generator = goctxid.DefaultGenerator
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			correlationID := r.Header.Get(config.HeaderKey)

			if correlationID == "" {
				correlationID = config.Generator()
			}

			w.Header().Set(config.HeaderKey, correlationID)
			ctx := goctxid.NewContext(r.Context(), correlationID)
			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)
		})
	}
}

// Example handler
func helloHandler(w http.ResponseWriter, r *http.Request) {
	correlationID, exists := goctxid.FromContext(r.Context())
	if !exists {
		http.Error(w, "Correlation ID not found", http.StatusInternalServerError)
		return
	}

	response := map[string]string{
		"message":        "Hello, World!",
		"correlation_id": correlationID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Example handler with logging
func userHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	correlationID := goctxid.MustFromContext(ctx)

	userID := r.URL.Query().Get("id")
	log.Printf("[%s] Fetching user: %s", correlationID, userID)

	// Simulate calling a service
	user := getUserFromService(ctx, userID)

	response := map[string]interface{}{
		"user":           user,
		"correlation_id": correlationID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Simulated service function
func getUserFromService(ctx context.Context, userID string) map[string]string {
	correlationID := goctxid.MustFromContext(ctx)
	log.Printf("[%s] Service: Getting user %s from database", correlationID, userID)

	return map[string]string{
		"id":   userID,
		"name": "John Doe",
	}
}

func main() {
	mux := http.NewServeMux()

	// Register handlers
	mux.HandleFunc("/", helloHandler)
	mux.HandleFunc("/user", userHandler)

	// Wrap with correlation ID middleware
	handler := correlationIDMiddleware(mux)

	log.Println("Server starting on :3000")
	log.Println("\nTry these examples:")
	log.Println("  # Basic request:")
	log.Println("  curl http://localhost:3000/")
	log.Println("\n  # User request:")
	log.Println("  curl http://localhost:3000/user?id=123")
	log.Println("\n  # With custom correlation ID:")
	log.Println("  curl -H 'X-Correlation-ID: my-custom-id' http://localhost:3000/")
	log.Println("\n  # Check response headers:")
	log.Println("  curl -v http://localhost:3000/")

	log.Fatal(http.ListenAndServe(":3000", handler))
}
