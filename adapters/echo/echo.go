package echo

import (
	"github.com/hiiamtin/goctxid"
	"github.com/labstack/echo/v4"
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

// New creates a new Echo middleware for correlation ID management
func New(config ...goctxid.Config) echo.MiddlewareFunc {

	// 1. Merge the provided config with the default config
	cfg := configDefault(config...)

	// 2. Return the middleware function
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// 3. Extract the correlation ID from the request header
			correlationID := c.Request().Header.Get(cfg.HeaderKey)

			// 4. If not found, generate a new one
			if correlationID == "" {
				correlationID = cfg.Generator()
			}

			// 5. Set the response header (send back to the client)
			c.Response().Header().Set(cfg.HeaderKey, correlationID)

			// 6. Get the current request context
			ctx := c.Request().Context()

			// 7. Create a new context with our ID
			newCtx := goctxid.NewContext(ctx, correlationID)

			// 8. Set the new context back into the request
			c.SetRequest(c.Request().WithContext(newCtx))

			// 9. Continue to the next handler
			return next(c)
		}
	}
}

