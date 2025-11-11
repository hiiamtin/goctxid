package gin

import (
	"github.com/gin-gonic/gin"
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

// New creates a new Gin middleware for correlation ID management
func New(config ...goctxid.Config) gin.HandlerFunc {

	// 1. Merge the provided config with the default config
	cfg := configDefault(config...)

	// 2. Return the middleware function
	return func(c *gin.Context) {
		// 3. Extract the correlation ID from the request header
		correlationID := c.GetHeader(cfg.HeaderKey)

		// 4. If not found, generate a new one
		if correlationID == "" {
			correlationID = cfg.Generator()
		}

		// 5. Set the response header (send back to the client)
		c.Header(cfg.HeaderKey, correlationID)

		// 6. Get the current request context
		ctx := c.Request.Context()

		// 7. Create a new context with our ID
		newCtx := goctxid.NewContext(ctx, correlationID)

		// 8. Set the new context back into the request
		c.Request = c.Request.WithContext(newCtx)

		// 9. Continue to the next handler
		c.Next()
	}
}

