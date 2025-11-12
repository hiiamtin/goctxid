package gin

import (
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

// GetCorrelationID retrieves the correlation ID from the Gin context.
// Returns the correlation ID or an empty string if not found.
// This is a convenience function equivalent to MustFromContext(c.Request.Context()).
func GetCorrelationID(c *gin.Context) string {
	return MustFromContext(c.Request.Context())
}
