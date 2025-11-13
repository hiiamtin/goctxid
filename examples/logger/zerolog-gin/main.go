package main

import (
	"io"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hiiamtin/goctxid"
	goctxid_gin "github.com/hiiamtin/goctxid/adapters/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Initialize Zerolog
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	// Middleware stack
	r.Use(goctxid_gin.New())   // 1. Correlation ID
	r.Use(zerologMiddleware()) // 2. HTTP access logs
	r.Use(gin.Recovery())      // 3. Panic recovery

	// Routes
	r.GET("/health", healthCheck)
	r.GET("/users/:id", getUser)
	r.POST("/users", createUser)

	log.Info().Msg("Server starting on :3000")
	r.Run(":3000")
}

// HTTP Access Logger Middleware
func zerologMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		correlationID := goctxid.MustFromContext(c.Request.Context())
		latency := time.Since(start)

		// HTTP Access Log
		event := log.Info()
		if c.Writer.Status() >= 500 {
			event = log.Error()
		} else if c.Writer.Status() >= 400 {
			event = log.Warn()
		}

		event.
			Str("type", "http_access").
			Str("correlation_id", correlationID).
			Str("method", c.Request.Method).
			Str("path", path).
			Str("query", query).
			Int("status", c.Writer.Status()).
			Dur("latency", latency).
			Str("ip", c.ClientIP()).
			Str("user_agent", c.Request.UserAgent()).
			Int("body_size", c.Writer.Size()).
			Msg("HTTP Request")
	}
}

// Helper to get logger with correlation ID
func getLogger(c *gin.Context) zerolog.Logger {
	correlationID := goctxid.MustFromContext(c.Request.Context())
	return log.With().
		Str("correlation_id", correlationID).
		Str("type", "application").
		Logger()
}

func healthCheck(c *gin.Context) {
	c.JSON(200, gin.H{"status": "ok"})
}

func getUser(c *gin.Context) {
	logger := getLogger(c)
	userID := c.Param("id")

	logger.Info().Str("user_id", userID).Msg("Fetching user from database")

	// Simulate database call
	time.Sleep(10 * time.Millisecond)
	user := gin.H{
		"id":   userID,
		"name": "John Doe",
	}

	logger.Info().Str("user_id", userID).Msg("User fetched successfully")
	c.JSON(200, user)
}

func createUser(c *gin.Context) {
	logger := getLogger(c)

	var input struct {
		Name string `json:"name"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		logger.Error().Err(err).Msg("Invalid request body")
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}

	logger.Info().Str("name", input.Name).Msg("Creating new user")

	// Simulate database insert
	time.Sleep(20 * time.Millisecond)
	newUserID := 123

	logger.Info().Int("user_id", newUserID).Msg("User created successfully")
	c.JSON(201, gin.H{"id": newUserID})
}

// SetupBenchmarkLogger configures logger to write to discard for benchmarks
func SetupBenchmarkLogger() {
	log.Logger = zerolog.New(io.Discard).With().Timestamp().Logger()
}
