package main

import (
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/hiiamtin/goctxid"
	goctxid_fiber "github.com/hiiamtin/goctxid/adapters/fiber"
)

var logger *slog.Logger

func main() {
	// Initialize slog logger
	logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	// Middleware stack
	app.Use(goctxid_fiber.New()) // 1. Correlation ID
	app.Use(slogMiddleware())    // 2. HTTP access logs

	// Routes
	app.Get("/health", healthCheck)
	app.Get("/users/:id", getUser)
	app.Post("/users", createUser)

	logger.Info("Server starting on :3000")
	app.Listen(":3000")
}

// HTTP Access Logger Middleware
func slogMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		err := c.Next()

		correlationID := goctxid.MustFromContext(c.UserContext())
		latency := time.Since(start)

		// HTTP Access Log
		status := c.Response().StatusCode()
		logFunc := logger.Info
		if status >= 500 {
			logFunc = logger.Error
		} else if status >= 400 {
			logFunc = logger.Warn
		}

		logFunc("HTTP Request",
			"type", "http_access",
			"correlation_id", correlationID,
			"method", c.Method(),
			"path", c.Path(),
			"query", string(c.Request().URI().QueryString()),
			"status", status,
			"latency", latency,
			"ip", c.IP(),
			"user_agent", c.Get("User-Agent"),
			"body_size", len(c.Response().Body()),
		)

		return err
	}
}

// Helper to get logger with correlation ID
func getLogger(c *fiber.Ctx) *slog.Logger {
	correlationID := goctxid.MustFromContext(c.UserContext())
	return logger.With(
		"correlation_id", correlationID,
		"type", "application",
	)
}

func healthCheck(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "ok"})
}

func getUser(c *fiber.Ctx) error {
	log := getLogger(c)
	userID := c.Params("id")

	log.Info("Fetching user from database", "user_id", userID)

	// Simulate database call
	time.Sleep(10 * time.Millisecond)
	user := fiber.Map{
		"id":   userID,
		"name": "John Doe",
	}

	log.Info("User fetched successfully", "user_id", userID)
	return c.JSON(user)
}

func createUser(c *fiber.Ctx) error {
	log := getLogger(c)

	var input struct {
		Name string `json:"name"`
	}

	if err := c.BodyParser(&input); err != nil {
		log.Error("Invalid request body", "error", err)
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
	}

	log.Info("Creating new user", "name", input.Name)

	// Simulate database insert
	time.Sleep(20 * time.Millisecond)
	newUserID := 123

	log.Info("User created successfully", "user_id", newUserID)
	return c.Status(201).JSON(fiber.Map{"id": newUserID})
}

// SetupBenchmarkLogger configures logger to write to discard for benchmarks
func SetupBenchmarkLogger() {
	logger = slog.New(slog.NewJSONHandler(io.Discard, nil))
}

