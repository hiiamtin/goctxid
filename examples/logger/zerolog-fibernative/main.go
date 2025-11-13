package main

import (
	"io"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	goctxid_fibernative "github.com/hiiamtin/goctxid/adapters/fibernative"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Initialize Zerolog
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()

	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	// Middleware stack
	app.Use(goctxid_fibernative.New()) // 1. Correlation ID (using c.Locals)
	app.Use(zerologMiddleware())       // 2. HTTP access logs

	// Routes
	app.Get("/health", healthCheck)
	app.Get("/users/:id", getUser)
	app.Post("/users", createUser)

	log.Info().Msg("Server starting on :3000")
	app.Listen(":3000")
}

// HTTP Access Logger Middleware
func zerologMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		err := c.Next()

		correlationID := goctxid_fibernative.MustFromLocals(c)
		latency := time.Since(start)

		// HTTP Access Log
		event := log.Info()
		if c.Response().StatusCode() >= 500 {
			event = log.Error()
		} else if c.Response().StatusCode() >= 400 {
			event = log.Warn()
		}

		event.
			Str("type", "http_access").
			Str("correlation_id", correlationID).
			Str("method", c.Method()).
			Str("path", c.Path()).
			Str("query", string(c.Request().URI().QueryString())).
			Int("status", c.Response().StatusCode()).
			Dur("latency", latency).
			Str("ip", c.IP()).
			Str("user_agent", c.Get("User-Agent")).
			Int("body_size", len(c.Response().Body())).
			Msg("HTTP Request")

		return err
	}
}

// Helper to get logger with correlation ID
func getLogger(c *fiber.Ctx) zerolog.Logger {
	correlationID := goctxid_fibernative.MustFromLocals(c)
	return log.With().
		Str("correlation_id", correlationID).
		Str("type", "application").
		Logger()
}

func healthCheck(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "ok"})
}

func getUser(c *fiber.Ctx) error {
	logger := getLogger(c)
	userID := c.Params("id")

	logger.Info().Str("user_id", userID).Msg("Fetching user from database")

	// Simulate database call
	time.Sleep(10 * time.Millisecond)
	user := fiber.Map{
		"id":   userID,
		"name": "John Doe",
	}

	logger.Info().Str("user_id", userID).Msg("User fetched successfully")
	return c.JSON(user)
}

func createUser(c *fiber.Ctx) error {
	logger := getLogger(c)

	var input struct {
		Name string `json:"name"`
	}

	if err := c.BodyParser(&input); err != nil {
		logger.Error().Err(err).Msg("Invalid request body")
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
	}

	logger.Info().Str("name", input.Name).Msg("Creating new user")

	// Simulate database insert
	time.Sleep(20 * time.Millisecond)
	newUserID := 123

	logger.Info().Int("user_id", newUserID).Msg("User created successfully")
	return c.Status(201).JSON(fiber.Map{"id": newUserID})
}

// SetupBenchmarkLogger configures logger to write to discard for benchmarks
func SetupBenchmarkLogger() {
	log.Logger = zerolog.New(io.Discard).With().Timestamp().Logger()
}
