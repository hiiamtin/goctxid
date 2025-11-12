// Package goctxid provides middleware for managing request/correlation IDs
// through context.Context in Go HTTP applications.
package goctxid

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"sync"
	"sync/atomic"

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

var (
	// fastGenSeed stores random bytes for the fast generator
	fastGenSeed [24]byte
	// fastGenCounter is an atomic counter for the fast generator
	fastGenCounter uint64
	// fastGenOnce ensures initialization happens only once
	fastGenOnce sync.Once
)

// initFastGenerator initializes the fast generator with random seed
func initFastGenerator() {
	// Read random bytes for the seed using crypto/rand for security
	_, _ = rand.Read(fastGenSeed[:])
	// Initialize counter with first 8 bytes of seed
	fastGenCounter = binary.LittleEndian.Uint64(fastGenSeed[:8])
}

// FastGenerator generates correlation IDs using an atomic counter.
//
// ⚠️ PRIVACY WARNING: This generator is ~250-300ns faster than UUID v4,
// but it EXPOSES REQUEST COUNT and traffic patterns because it uses a
// sequential atomic counter embedded in the ID.
//
// Security implications:
//   - Attackers can calculate total request count by comparing IDs
//   - Traffic patterns and request rates can be inferred
//   - Server restarts are detectable (counter resets)
//   - Not suitable for privacy-sensitive applications
//
// Performance vs Security trade-off:
//   - FastGenerator: ~50-100 ns/op (fast, but exposes request count)
//   - DefaultGenerator: ~350 ns/op (secure, cryptographically random)
//
// Use this generator ONLY when:
//   - Performance is critical (high-throughput systems)
//   - Request count exposure is acceptable
//   - IDs are used only for internal tracing (not exposed to clients)
//
// For most applications, use DefaultGenerator (UUID v4) instead.
//
// Example usage:
//
//	app.Use(fiber.New(fiber.Config{
//	    Config: goctxid.Config{
//	        Generator: goctxid.FastGenerator,  // Opt-in to fast but less private
//	    },
//	}))
func FastGenerator() string {
	fastGenOnce.Do(initFastGenerator)

	// Atomically increment counter
	x := atomic.AddUint64(&fastGenCounter, 1)

	// Create a copy of the seed
	var id [24]byte
	copy(id[:], fastGenSeed[:])

	// Embed counter in first 8 bytes
	binary.LittleEndian.PutUint64(id[:8], x)

	// Format as UUID-like string (8-4-4-4-12 format)
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		id[0:4],
		id[4:6],
		id[6:8],
		id[8:10],
		id[10:16])
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
