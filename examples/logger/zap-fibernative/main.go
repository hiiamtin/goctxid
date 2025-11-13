package main

import (
	"io"
	"time"

	"github.com/gofiber/fiber/v2"
	goctxid_fibernative "github.com/hiiamtin/goctxid/adapters/fibernative"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var logger *zap.Logger

func main() {
	// Initialize Zap logger
	config := zap.NewProductionConfig()
	config.EncoderConfig.TimeKey = "time"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	var err error
	logger, err = config.Build()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	// Middleware stack
	app.Use(goctxid_fibernative.New()) // 1. Correlation ID (Fibernative)
	app.Use(zapMiddleware())           // 2. HTTP access logs

	// Routes
	app.Get("/health", healthCheck)
	app.Get("/users/:id", getUser)
	app.Post("/users", createUser)

	logger.Info("Server starting on :3000")
	app.Listen(":3000")
}

// HTTP Access Logger Middleware
func zapMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		err := c.Next()

		// Get correlation ID from Locals (Fibernative approach)
		correlationID := goctxid_fibernative.MustFromLocals(c)
		latency := time.Since(start)

		// HTTP Access Log
		fields := []zap.Field{
			zap.String("type", "http_access"),
			zap.String("correlation_id", correlationID),
			zap.String("method", c.Method()),
			zap.String("path", c.Path()),
			zap.String("query", string(c.Request().URI().QueryString())),
			zap.Int("status", c.Response().StatusCode()),
			zap.Duration("latency", latency),
			zap.String("ip", c.IP()),
			zap.String("user_agent", c.Get("User-Agent")),
			zap.Int("body_size", len(c.Response().Body())),
		}

		if c.Response().StatusCode() >= 500 {
			logger.Error("HTTP Request", fields...)
		} else if c.Response().StatusCode() >= 400 {
			logger.Warn("HTTP Request", fields...)
		} else {
			logger.Info("HTTP Request", fields...)
		}

		return err
	}
}

// Helper to get logger with correlation ID
func getLogger(c *fiber.Ctx) *zap.Logger {
	// Get correlation ID from Locals (Fibernative approach)
	correlationID := goctxid_fibernative.MustFromLocals(c)
	return logger.With(
		zap.String("correlation_id", correlationID),
		zap.String("type", "application"),
	)
}

func healthCheck(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "ok"})
}

func getUser(c *fiber.Ctx) error {
	log := getLogger(c)
	userID := c.Params("id")

	log.Info("Fetching user from database", zap.String("user_id", userID))

	// Simulate database call
	time.Sleep(10 * time.Millisecond)
	user := fiber.Map{
		"id":   userID,
		"name": "John Doe",
	}

	log.Info("User fetched successfully", zap.String("user_id", userID))
	return c.JSON(user)
}

func createUser(c *fiber.Ctx) error {
	log := getLogger(c)

	var input struct {
		Name string `json:"name"`
	}

	if err := c.BodyParser(&input); err != nil {
		log.Error("Invalid request body", zap.Error(err))
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
	}

	log.Info("Creating new user", zap.String("name", input.Name))

	// Simulate database insert
	time.Sleep(20 * time.Millisecond)
	newUserID := 123

	log.Info("User created successfully", zap.Int("user_id", newUserID))
	return c.Status(201).JSON(fiber.Map{"id": newUserID})
}

// SetupBenchmarkLogger configures logger to write to discard for benchmarks
func SetupBenchmarkLogger() {
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(io.Discard),
		zapcore.InfoLevel,
	)
	logger = zap.New(core)
}
