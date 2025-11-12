package fibernative

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/hiiamtin/goctxid"
)

const (
	// DefaultLocalsKey is the default key used to store the correlation ID in c.Locals()
	DefaultLocalsKey = "goctxid"
)

// Config extends goctxid.Config with Fiber-native specific options
type Config struct {
	goctxid.Config

	// Next defines a function to skip this middleware when returned true.
	//
	// Optional. Default: nil
	Next func(c *fiber.Ctx) bool

	// LocalsKey is the key used to store the correlation ID in c.Locals()
	// This allows customization to avoid collisions with existing code.
	//
	// Optional. Default: "goctxid"
	LocalsKey string
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
	// LocalsKey default
	if cfg.LocalsKey == "" {
		cfg.LocalsKey = DefaultLocalsKey
	}

	return cfg
}

// New creates a Fiber middleware that uses c.Locals() for storage (Fiber-native way)
// This is more performant than using context as it avoids context allocation overhead
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

		// 7. Store in Fiber's Locals (Fiber-native way - no context overhead)
		c.Locals(cfg.LocalsKey, correlationID)

		// 8. Continue to the next handler
		return c.Next()
	}
}

// FromLocals retrieves the correlation ID from Fiber's c.Locals() using the default key.
// This is the Fiber-native way to access the correlation ID.
func FromLocals(c *fiber.Ctx) (string, bool) {
	return FromLocalsWithKey(c, DefaultLocalsKey)
}

// FromLocalsWithKey retrieves the correlation ID from c.Locals() using a custom key.
// Use this if you configured a custom LocalsKey in the middleware.
func FromLocalsWithKey(c *fiber.Ctx, key string) (string, bool) {
	id := c.Locals(key)
	if id == nil {
		return "", false
	}

	idStr, ok := id.(string)
	return idStr, ok
}

// MustFromLocals retrieves the correlation ID from c.Locals() or returns empty string.
// Uses the default key.
func MustFromLocals(c *fiber.Ctx) string {
	id, _ := FromLocals(c)
	return id
}

// MustFromLocalsWithKey retrieves the correlation ID from c.Locals() using a custom key,
// or returns empty string if not found.
func MustFromLocalsWithKey(c *fiber.Ctx, key string) string {
	id, _ := FromLocalsWithKey(c, key)
	return id
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
// This allows users to call goctxid_fibernative.FromContext() instead of importing goctxid separately
// Note: For Fiber-native usage, prefer FromLocals() which uses c.Locals() instead of context

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
