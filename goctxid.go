// Package goctxid provides middleware for managing request/correlation IDs
// through context.Context in Go HTTP applications.
package goctxid

import (
	"context"

	"github.com/google/uuid"
)

type correlationIDKey string

const (
	// DefaultHeaderKey is the default header key used to store the correlation ID
	DefaultHeaderKey = "X-Correlation-ID"

	// ctxKey is the key used to store the correlation ID in the context
	ctxKey correlationIDKey = "goctxid_key"
)

// Config struct let user customize the behavior
type Config struct {
	// HeaderKey is the HTTP header key used to store the correlation ID
	HeaderKey string

	// Generator is the function used to generate a new correlation ID
	// Must be thread-safe as it will be called concurrently by multiple requests
	// (Default: UUID v4)
	Generator func() string
}

// DefaultGenerator is the default UUID v4 generator
// Exported so adapters can use it as a fallback
func DefaultGenerator() string {
	return uuid.NewString()
}

// FromContext returns the correlation ID from the context
// This function is used by User in their Handler
func FromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(ctxKey).(string)
	return id, ok
}

// MustFromContext returns the correlation ID or empty string if not found
func MustFromContext(ctx context.Context) string {
	id, _ := FromContext(ctx)
	return id
}

// NewContext creates a new context with the correlation ID.
//
// This function is primarily intended for use by middleware adapters and
// custom middleware implementations. Most users should not need to call this
// directly - instead, use the provided framework adapters (fiber, echo, gin)
// or the standard net/http middleware pattern.
//
// Use cases for calling NewContext directly:
//   - Creating custom middleware for unsupported frameworks
//   - Implementing custom middleware patterns with net/http
//   - Testing scenarios where you need to manually inject a correlation ID
//
// Example (custom middleware):
//
//	func customMiddleware(next http.Handler) http.Handler {
//	    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//	        id := goctxid.DefaultGenerator()
//	        ctx := goctxid.NewContext(r.Context(), id)
//	        r = r.WithContext(ctx)
//	        next.ServeHTTP(w, r)
//	    })
//	}
func NewContext(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKey, id)
}
