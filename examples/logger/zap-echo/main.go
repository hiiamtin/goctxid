package main

import (
	"io"
	"time"

	"github.com/hiiamtin/goctxid"
	goctxid_echo "github.com/hiiamtin/goctxid/adapters/echo"
	"github.com/labstack/echo/v4"
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

	e := echo.New()
	e.HideBanner = true

	// Middleware stack
	e.Use(goctxid_echo.New())  // 1. Correlation ID
	e.Use(zapMiddleware())     // 2. HTTP access logs

	// Routes
	e.GET("/health", healthCheck)
	e.GET("/users/:id", getUser)
	e.POST("/users", createUser)

	logger.Info("Server starting on :3000")
	e.Start(":3000")
}

// HTTP Access Logger Middleware
func zapMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			err := next(c)

			correlationID := goctxid.MustFromContext(c.Request().Context())
			latency := time.Since(start)

			// HTTP Access Log
			status := c.Response().Status
			fields := []zap.Field{
				zap.String("type", "http_access"),
				zap.String("correlation_id", correlationID),
				zap.String("method", c.Request().Method),
				zap.String("path", c.Path()),
				zap.String("query", c.QueryString()),
				zap.Int("status", status),
				zap.Duration("latency", latency),
				zap.String("ip", c.RealIP()),
				zap.String("user_agent", c.Request().UserAgent()),
				zap.Int64("body_size", c.Response().Size),
			}

			if status >= 500 {
				logger.Error("HTTP Request", fields...)
			} else if status >= 400 {
				logger.Warn("HTTP Request", fields...)
			} else {
				logger.Info("HTTP Request", fields...)
			}

			return err
		}
	}
}

// Helper to get logger with correlation ID
func getLogger(c echo.Context) *zap.Logger {
	correlationID := goctxid.MustFromContext(c.Request().Context())
	return logger.With(
		zap.String("correlation_id", correlationID),
		zap.String("type", "application"),
	)
}

func healthCheck(c echo.Context) error {
	return c.JSON(200, map[string]string{"status": "ok"})
}

func getUser(c echo.Context) error {
	log := getLogger(c)
	userID := c.Param("id")

	log.Info("Fetching user from database", zap.String("user_id", userID))

	// Simulate database call
	time.Sleep(10 * time.Millisecond)
	user := map[string]interface{}{
		"id":   userID,
		"name": "John Doe",
	}

	log.Info("User fetched successfully", zap.String("user_id", userID))
	return c.JSON(200, user)
}

func createUser(c echo.Context) error {
	log := getLogger(c)

	var input struct {
		Name string `json:"name"`
	}

	if err := c.Bind(&input); err != nil {
		log.Error("Invalid request body", zap.Error(err))
		return c.JSON(400, map[string]string{"error": "Invalid request"})
	}

	log.Info("Creating new user", zap.String("name", input.Name))

	// Simulate database insert
	time.Sleep(20 * time.Millisecond)
	newUserID := 123

	log.Info("User created successfully", zap.Int("user_id", newUserID))
	return c.JSON(201, map[string]interface{}{"id": newUserID})
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

