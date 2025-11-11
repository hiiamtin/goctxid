package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/hiiamtin/goctxid"
)

// Logger is a simple structured logger that includes correlation ID
type Logger struct {
	prefix string
}

// NewLogger creates a new logger instance
func NewLogger(prefix string) *Logger {
	return &Logger{prefix: prefix}
}

// Info logs an info message with correlation ID from context
func (l *Logger) Info(ctx context.Context, format string, args ...interface{}) {
	correlationID := goctxid.MustFromContext(ctx)
	message := fmt.Sprintf(format, args...)
	log.Printf("[INFO] [%s] [%s] %s", l.prefix, correlationID, message)
}

// Error logs an error message with correlation ID from context
func (l *Logger) Error(ctx context.Context, format string, args ...interface{}) {
	correlationID := goctxid.MustFromContext(ctx)
	message := fmt.Sprintf(format, args...)
	log.Printf("[ERROR] [%s] [%s] %s", l.prefix, correlationID, message)
}

// Warn logs a warning message with correlation ID from context
func (l *Logger) Warn(ctx context.Context, format string, args ...interface{}) {
	correlationID := goctxid.MustFromContext(ctx)
	message := fmt.Sprintf(format, args...)
	log.Printf("[WARN] [%s] [%s] %s", l.prefix, correlationID, message)
}

// UserService simulates a service layer with logging
type UserService struct {
	logger *Logger
}

func NewUserService() *UserService {
	return &UserService{
		logger: NewLogger("UserService"),
	}
}

func (s *UserService) GetUser(ctx context.Context, userID string) (map[string]interface{}, error) {
	s.logger.Info(ctx, "Fetching user: %s", userID)

	// Simulate database query
	time.Sleep(10 * time.Millisecond)

	if userID == "error" {
		s.logger.Error(ctx, "Failed to fetch user: %s", userID)
		return nil, fmt.Errorf("user not found")
	}

	s.logger.Info(ctx, "Successfully fetched user: %s", userID)
	return map[string]interface{}{
		"id":   userID,
		"name": "John Doe",
	}, nil
}

// OrderService simulates another service that calls UserService
type OrderService struct {
	logger      *Logger
	userService *UserService
}

func NewOrderService(userService *UserService) *OrderService {
	return &OrderService{
		logger:      NewLogger("OrderService"),
		userService: userService,
	}
}

func (s *OrderService) CreateOrder(ctx context.Context, userID string, items []string) error {
	s.logger.Info(ctx, "Creating order for user: %s with %d items", userID, len(items))

	// Fetch user (correlation ID is automatically propagated through context)
	user, err := s.userService.GetUser(ctx, userID)
	if err != nil {
		s.logger.Error(ctx, "Failed to create order: user not found")
		return err
	}

	s.logger.Info(ctx, "Order created for user: %v", user["name"])
	return nil
}

// Custom logging middleware that logs request/response
func loggingMiddleware(logger *Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		ctx := c.UserContext()

		logger.Info(ctx, "Request: %s %s", c.Method(), c.Path())

		// Process request
		err := c.Next()

		duration := time.Since(start)
		status := c.Response().StatusCode()

		if err != nil {
			logger.Error(ctx, "Request failed: %s %s - Status: %d - Duration: %v - Error: %v",
				c.Method(), c.Path(), status, duration, err)
		} else {
			logger.Info(ctx, "Request completed: %s %s - Status: %d - Duration: %v",
				c.Method(), c.Path(), status, duration)
		}

		return err
	}
}

func main() {
	// Set up logging format
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	log.SetOutput(os.Stdout)

	app := fiber.New()

	// Initialize services
	userService := NewUserService()
	orderService := NewOrderService(userService)
	requestLogger := NewLogger("HTTP")

	// Add goctxid middleware first
	app.Use(goctxid.New())

	// Add custom logging middleware
	app.Use(loggingMiddleware(requestLogger))

	// Routes
	app.Get("/user/:id", func(c *fiber.Ctx) error {
		ctx := c.UserContext()
		userID := c.Params("id")

		user, err := userService.GetUser(ctx, userID)
		if err != nil {
			return c.Status(404).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.JSON(user)
	})

	app.Post("/order", func(c *fiber.Ctx) error {
		ctx := c.UserContext()

		var body struct {
			UserID string   `json:"user_id"`
			Items  []string `json:"items"`
		}

		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error": "Invalid request body",
			})
		}

		err := orderService.CreateOrder(ctx, body.UserID, body.Items)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.JSON(fiber.Map{
			"message": "Order created successfully",
		})
	})

	log.Println("Server starting on :3000")
	log.Println("\nTry these examples:")
	log.Println("  # Get user (success):")
	log.Println("  curl http://localhost:3000/user/123")
	log.Println("\n  # Get user (error):")
	log.Println("  curl http://localhost:3000/user/error")
	log.Println("\n  # Create order:")
	log.Println(`  curl -X POST http://localhost:3000/order -H "Content-Type: application/json" -d '{"user_id":"123","items":["item1","item2"]}'`)
	log.Println("\n  # With custom correlation ID:")
	log.Println(`  curl -H "X-Correlation-ID: my-trace-id" http://localhost:3000/user/123`)
	log.Println("\nNotice how the same correlation ID appears in all log entries for a single request!")

	log.Fatal(app.Listen(":3000"))
}

