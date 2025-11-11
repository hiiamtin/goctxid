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

// NewContext create a new context with the correlation ID
// This function is used by the middleware
func NewContext(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKey, id)
}
