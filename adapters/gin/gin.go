package gin

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/hiiamtin/goctxid"
)

// Config extends goctxid.Config with Gin-specific options
type Config struct {
	goctxid.Config

	// Next defines a function to skip this middleware when returned true.
	//
	// Optional. Default: nil
	Next func(c *gin.Context) bool
}

// configDefault is a helper function that merges the provided config with the default config
func configDefault(config ...Config) Config {

	var cfg Config

	// If a config is provided, use it
	if len(config) > 0 {
		cfg = config[0]
	}

	// Check and fill in default values
	if cfg.HeaderKey == "" {
		cfg.HeaderKey = goctxid.DefaultHeaderKey
	}
	// Generator must be thread-safe as middleware runs concurrently for multiple requests
	if cfg.Generator == nil {
		cfg.Generator = goctxid.DefaultGenerator
	}

	return cfg
}

// New creates a new Gin middleware for correlation ID management
func New(config ...Config) gin.HandlerFunc {

	// 1. Merge the provided config with the default config
	cfg := configDefault(config...)

	// 2. Return the middleware function
	return func(c *gin.Context) {
		// 3. Check if we should skip this middleware
		if cfg.Next != nil && cfg.Next(c) {
			c.Next()
			return
		}

		// 4. Extract the correlation ID from the request header
		correlationID := c.GetHeader(cfg.HeaderKey)

		// 5. If not found, generate a new one
		if correlationID == "" {
			correlationID = cfg.Generator()
		}

		// 6. Set the response header (send back to the client)
		c.Header(cfg.HeaderKey, correlationID)

		// 7. Get the current request context
		ctx := c.Request.Context()

		// 8. Create a new context with our ID
		newCtx := goctxid.NewContext(ctx, correlationID)

		// 9. Set the new context back into the request
		c.Request = c.Request.WithContext(newCtx)

		// 10. Continue to the next handler
		c.Next()
	}
}

// Re-exported constants from goctxid package for convenience
const (
	// DefaultHeaderKey is the default HTTP header key for correlation ID
	DefaultHeaderKey = goctxid.DefaultHeaderKey
)

// Re-exported generator functions from goctxid package for convenience
var (
	// DefaultGenerator is the default UUID v4 generator (cryptographically secure)
	DefaultGenerator = goctxid.DefaultGenerator

	// FastGenerator is a high-performance generator using atomic counter
	// ⚠️ WARNING: Exposes request count. Use only when performance is critical.
	FastGenerator = goctxid.FastGenerator
)

// Re-exported functions from goctxid package for convenience
// This allows users to call goctxid_gin.FromContext() instead of importing goctxid separately

// FromContext retrieves the correlation ID from the context.
// Returns the correlation ID and a boolean indicating if it was found.
func FromContext(ctx context.Context) (string, bool) {
	return goctxid.FromContext(ctx)
}

// MustFromContext retrieves the correlation ID from the context.
// Returns the correlation ID or an empty string if not found.
func MustFromContext(ctx context.Context) string {
	return goctxid.MustFromContext(ctx)
}

// NewContext creates a new context with the correlation ID.
func NewContext(ctx context.Context, correlationID string) context.Context {
	return goctxid.NewContext(ctx, correlationID)
}
