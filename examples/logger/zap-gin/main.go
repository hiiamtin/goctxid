package main

import (
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hiiamtin/goctxid"
	goctxid_gin "github.com/hiiamtin/goctxid/adapters/gin"
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

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	// Middleware stack
	r.Use(goctxid_gin.New()) // 1. Correlation ID
	r.Use(zapMiddleware())   // 2. HTTP access logs
	r.Use(gin.Recovery())    // 3. Panic recovery

	// Routes
	r.GET("/health", healthCheck)
	r.GET("/users/:id", getUser)
	r.POST("/users", createUser)

	logger.Info("Server starting on :3000")
	r.Run(":3000")
}

// HTTP Access Logger Middleware
func zapMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		correlationID := goctxid.MustFromContext(c.Request.Context())
		latency := time.Since(start)

		// HTTP Access Log
		fields := []zap.Field{
			zap.String("type", "http_access"),
			zap.String("correlation_id", correlationID),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", latency),
			zap.String("ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
			zap.Int("body_size", c.Writer.Size()),
		}

		if c.Writer.Status() >= 500 {
			logger.Error("HTTP Request", fields...)
		} else if c.Writer.Status() >= 400 {
			logger.Warn("HTTP Request", fields...)
		} else {
			logger.Info("HTTP Request", fields...)
		}
	}
}

// Helper to get logger with correlation ID
func getLogger(c *gin.Context) *zap.Logger {
	correlationID := goctxid.MustFromContext(c.Request.Context())
	return logger.With(
		zap.String("correlation_id", correlationID),
		zap.String("type", "application"),
	)
}

func healthCheck(c *gin.Context) {
	c.JSON(200, gin.H{"status": "ok"})
}

func getUser(c *gin.Context) {
	log := getLogger(c)
	userID := c.Param("id")

	log.Info("Fetching user from database", zap.String("user_id", userID))

	// Simulate database call
	time.Sleep(10 * time.Millisecond)
	user := gin.H{
		"id":   userID,
		"name": "John Doe",
	}

	log.Info("User fetched successfully", zap.String("user_id", userID))
	c.JSON(200, user)
}

func createUser(c *gin.Context) {
	log := getLogger(c)

	var input struct {
		Name string `json:"name"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		log.Error("Invalid request body", zap.Error(err))
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}

	log.Info("Creating new user", zap.String("name", input.Name))

	// Simulate database insert
	time.Sleep(20 * time.Millisecond)
	newUserID := 123

	log.Info("User created successfully", zap.Int("user_id", newUserID))
	c.JSON(201, gin.H{"id": newUserID})
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
