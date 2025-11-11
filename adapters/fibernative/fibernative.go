package fibernative

import (
	"github.com/gofiber/fiber/v2"
	"github.com/hiiamtin/goctxid"
)

const (
	// LocalsKey is the key used to store the correlation ID in c.Locals()
	LocalsKey = "goctxid"
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

// New creates a Fiber middleware that uses c.Locals() for storage (Fiber-native way)
// This is more performant than using context as it avoids context allocation overhead
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

		// 6. Store in Fiber's Locals (Fiber-native way - no context overhead)
		c.Locals(LocalsKey, correlationID)

		// 7. Continue to the next handler
		return c.Next()
	}
}

// FromLocals retrieves the correlation ID from Fiber's c.Locals()
// This is the Fiber-native way to access the correlation ID
func FromLocals(c *fiber.Ctx) (string, bool) {
	id := c.Locals(LocalsKey)
	if id == nil {
		return "", false
	}

	idStr, ok := id.(string)
	return idStr, ok
}

// MustFromLocals retrieves the correlation ID from c.Locals() or returns empty string
func MustFromLocals(c *fiber.Ctx) string {
	id, _ := FromLocals(c)
	return id
}
