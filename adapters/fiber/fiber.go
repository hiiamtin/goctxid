package fiber

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/hiiamtin/goctxid"
)

// Config extends goctxid.Config with Fiber-specific options
type Config struct {
	goctxid.Config

	// Next defines a function to skip this middleware when returned true.
	//
	// Optional. Default: nil
	Next func(c *fiber.Ctx) bool
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

// New is the main function that users will call
// It returns a fiber.Handler (Middleware)
func New(config ...Config) fiber.Handler {

	// 1. Merge the provided config with the default config
	cfg := configDefault(config...)

	// 2. Return the middleware function
	return func(c *fiber.Ctx) error {
		// 3. Check if we should skip this middleware
		if cfg.Next != nil && cfg.Next(c) {
			return c.Next()
		}

		// 4. Extract the correlation ID from the request header
		correlationID := c.Get(cfg.HeaderKey)

		// 5. If not found, generate a new one
		if correlationID == "" {
			correlationID = cfg.Generator()
		}

		// 6. Set the response header (send back to the client)
		c.Set(cfg.HeaderKey, correlationID)

		// 7. Get the current user context
		ctx := c.UserContext()

		// 8. Create a new context with our ID (using helper from goctxid.go)
		newCtx := goctxid.NewContext(ctx, correlationID)

		// 9. Set the new context back into Fiber
		c.SetUserContext(newCtx)

		// 10. Continue to the next handler
		return c.Next()
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
// This allows users to call goctxid_fiber.FromContext() instead of importing goctxid separately

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
