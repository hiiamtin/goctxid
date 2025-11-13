package main

import (
	"io"
	"os"
	"time"

	"github.com/hiiamtin/goctxid"
	goctxid_echo "github.com/hiiamtin/goctxid/adapters/echo"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Initialize Zerolog
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()

	e := echo.New()
	e.HideBanner = true

	// Middleware stack
	e.Use(goctxid_echo.New())  // 1. Correlation ID
	e.Use(zerologMiddleware()) // 2. HTTP access logs

	// Routes
	e.GET("/health", healthCheck)
	e.GET("/users/:id", getUser)
	e.POST("/users", createUser)

	log.Info().Msg("Server starting on :3000")
	e.Start(":3000")
}

// HTTP Access Logger Middleware
func zerologMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			err := next(c)

			correlationID := goctxid.MustFromContext(c.Request().Context())
			latency := time.Since(start)

			// HTTP Access Log
			status := c.Response().Status
			event := log.Info()
			if status >= 500 {
				event = log.Error()
			} else if status >= 400 {
				event = log.Warn()
			}

			event.
				Str("type", "http_access").
				Str("correlation_id", correlationID).
				Str("method", c.Request().Method).
				Str("path", c.Path()).
				Str("query", c.QueryString()).
				Int("status", status).
				Dur("latency", latency).
				Str("ip", c.RealIP()).
				Str("user_agent", c.Request().UserAgent()).
				Int64("body_size", c.Response().Size).
				Msg("HTTP Request")

			return err
		}
	}
}

// Helper to get logger with correlation ID
func getLogger(c echo.Context) zerolog.Logger {
	correlationID := goctxid.MustFromContext(c.Request().Context())
	return log.With().
		Str("correlation_id", correlationID).
		Str("type", "application").
		Logger()
}

func healthCheck(c echo.Context) error {
	return c.JSON(200, map[string]string{"status": "ok"})
}

func getUser(c echo.Context) error {
	logger := getLogger(c)
	userID := c.Param("id")

	logger.Info().Str("user_id", userID).Msg("Fetching user from database")

	// Simulate database call
	time.Sleep(10 * time.Millisecond)
	user := map[string]string{
		"id":   userID,
		"name": "John Doe",
	}

	logger.Info().Str("user_id", userID).Msg("User fetched successfully")
	return c.JSON(200, user)
}

func createUser(c echo.Context) error {
	logger := getLogger(c)

	var input struct {
		Name string `json:"name"`
	}

	if err := c.Bind(&input); err != nil {
		logger.Error().Err(err).Msg("Invalid request body")
		return c.JSON(400, map[string]string{"error": "Invalid request"})
	}

	logger.Info().Str("name", input.Name).Msg("Creating new user")

	// Simulate database insert
	time.Sleep(20 * time.Millisecond)
	newUserID := 123

	logger.Info().Int("user_id", newUserID).Msg("User created successfully")
	return c.JSON(201, map[string]int{"id": newUserID})
}

// SetupBenchmarkLogger configures logger to write to discard for benchmarks
func SetupBenchmarkLogger() {
	log.Logger = zerolog.New(io.Discard).With().Timestamp().Logger()
}
