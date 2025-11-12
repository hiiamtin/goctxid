package fiber

import (
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

// GetCorrelationID retrieves the correlation ID from the Fiber context.
// Returns the correlation ID or an empty string if not found.
// This is a convenience function equivalent to MustFromContext(c.UserContext()).
func GetCorrelationID(c *fiber.Ctx) string {
	return MustFromContext(c.UserContext())
}
