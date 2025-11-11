package fiber

import (
	"github.com/gofiber/fiber/v2"
	"github.com/hiiamtin/goctxid"
)

// configDefault is a helper function that merges the provided config with the default config
func configDefault(config ...goctxid.Config) goctxid.Config {

	cfg := goctxid.Config{}

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
func New(config ...goctxid.Config) fiber.Handler {

	// 1. Merge the provided config with the default config
	cfg := configDefault(config...)

	// 2. Return the middleware function
	return func(c *fiber.Ctx) error {
		// 3. Extract the correlation ID from the request header
		correlationID := c.Get(cfg.HeaderKey)

		// 4. If not found, generate a new one
		if correlationID == "" {
			correlationID = cfg.Generator()
		}

		// 5. Set the response header (send back to the client)
		c.Set(cfg.HeaderKey, correlationID)

		// 6. Get the current user context
		ctx := c.UserContext()

		// 7. Create a new context with our ID (using helper from goctxid.go)
		newCtx := goctxid.NewContext(ctx, correlationID)

		// 8. Set the new context back into Fiber
		c.SetUserContext(newCtx)

		// 9. Continue to the next handler
		return c.Next()
	}
}

