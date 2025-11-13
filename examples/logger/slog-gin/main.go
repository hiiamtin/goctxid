package main

import (
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hiiamtin/goctxid"
	goctxid_gin "github.com/hiiamtin/goctxid/adapters/gin"
)

var logger *slog.Logger

func main() {
	// Initialize slog logger
	logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	// Middleware stack
	r.Use(goctxid_gin.New())   // 1. Correlation ID
	r.Use(slogMiddleware())    // 2. HTTP access logs
	r.Use(gin.Recovery())

	// Routes
	r.GET("/health", healthCheck)
	r.GET("/users/:id", getUser)
	r.POST("/users", createUser)

	logger.Info("Server starting on :3000")
	r.Run(":3000")
}

// HTTP Access Logger Middleware
func slogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		correlationID := goctxid.MustFromContext(c.Request.Context())
		latency := time.Since(start)

		// HTTP Access Log
		status := c.Writer.Status()
		logFunc := logger.Info
		if status >= 500 {
			logFunc = logger.Error
		} else if status >= 400 {
			logFunc = logger.Warn
		}

		logFunc("HTTP Request",
			"type", "http_access",
			"correlation_id", correlationID,
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"query", c.Request.URL.RawQuery,
			"status", status,
			"latency", latency,
			"ip", c.ClientIP(),
			"user_agent", c.Request.UserAgent(),
			"body_size", c.Writer.Size(),
		)
	}
}

// Helper to get logger with correlation ID
func getLogger(c *gin.Context) *slog.Logger {
	correlationID := goctxid.MustFromContext(c.Request.Context())
	return logger.With(
		"correlation_id", correlationID,
		"type", "application",
	)
}

func healthCheck(c *gin.Context) {
	c.JSON(200, gin.H{"status": "ok"})
}

func getUser(c *gin.Context) {
	log := getLogger(c)
	userID := c.Param("id")

	log.Info("Fetching user from database", "user_id", userID)

	// Simulate database call
	time.Sleep(10 * time.Millisecond)
	user := gin.H{
		"id":   userID,
		"name": "John Doe",
	}

	log.Info("User fetched successfully", "user_id", userID)
	c.JSON(200, user)
}

func createUser(c *gin.Context) {
	log := getLogger(c)

	var input struct {
		Name string `json:"name"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		log.Error("Invalid request body", "error", err)
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}

	log.Info("Creating new user", "name", input.Name)

	// Simulate database insert
	time.Sleep(20 * time.Millisecond)
	newUserID := 123

	log.Info("User created successfully", "user_id", newUserID)
	c.JSON(201, gin.H{"id": newUserID})
}

// SetupBenchmarkLogger configures logger to write to discard for benchmarks
func SetupBenchmarkLogger() {
	logger = slog.New(slog.NewJSONHandler(io.Discard, nil))
}

