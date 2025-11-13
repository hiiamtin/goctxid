package main

import (
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/hiiamtin/goctxid"
	goctxid_echo "github.com/hiiamtin/goctxid/adapters/echo"
	"github.com/labstack/echo/v4"
)

var logger *slog.Logger

func main() {
	// Initialize slog logger
	logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

	e := echo.New()
	e.HideBanner = true

	// Middleware stack
	e.Use(goctxid_echo.New())  // 1. Correlation ID
	e.Use(slogMiddleware())    // 2. HTTP access logs

	// Routes
	e.GET("/health", healthCheck)
	e.GET("/users/:id", getUser)
	e.POST("/users", createUser)

	logger.Info("Server starting on :3000")
	e.Start(":3000")
}

// HTTP Access Logger Middleware
func slogMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			err := next(c)

			correlationID := goctxid.MustFromContext(c.Request().Context())
			latency := time.Since(start)

			// HTTP Access Log
			status := c.Response().Status
			logFunc := logger.Info
			if status >= 500 {
				logFunc = logger.Error
			} else if status >= 400 {
				logFunc = logger.Warn
			}

			logFunc("HTTP Request",
				"type", "http_access",
				"correlation_id", correlationID,
				"method", c.Request().Method,
				"path", c.Path(),
				"query", c.QueryString(),
				"status", status,
				"latency", latency,
				"ip", c.RealIP(),
				"user_agent", c.Request().UserAgent(),
				"body_size", c.Response().Size,
			)

			return err
		}
	}
}

// Helper to get logger with correlation ID
func getLogger(c echo.Context) *slog.Logger {
	correlationID := goctxid.MustFromContext(c.Request().Context())
	return logger.With(
		"correlation_id", correlationID,
		"type", "application",
	)
}

func healthCheck(c echo.Context) error {
	return c.JSON(200, map[string]string{"status": "ok"})
}

func getUser(c echo.Context) error {
	log := getLogger(c)
	userID := c.Param("id")

	log.Info("Fetching user from database", "user_id", userID)

	// Simulate database call
	time.Sleep(10 * time.Millisecond)
	user := map[string]interface{}{
		"id":   userID,
		"name": "John Doe",
	}

	log.Info("User fetched successfully", "user_id", userID)
	return c.JSON(200, user)
}

func createUser(c echo.Context) error {
	log := getLogger(c)

	var input struct {
		Name string `json:"name"`
	}

	if err := c.Bind(&input); err != nil {
		log.Error("Invalid request body", "error", err)
		return c.JSON(400, map[string]string{"error": "Invalid request"})
	}

	log.Info("Creating new user", "name", input.Name)

	// Simulate database insert
	time.Sleep(20 * time.Millisecond)
	newUserID := 123

	log.Info("User created successfully", "user_id", newUserID)
	return c.JSON(201, map[string]interface{}{"id": newUserID})
}

// SetupBenchmarkLogger configures logger to write to discard for benchmarks
func SetupBenchmarkLogger() {
	logger = slog.New(slog.NewJSONHandler(io.Discard, nil))
}

