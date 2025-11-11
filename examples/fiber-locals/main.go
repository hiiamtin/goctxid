package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
	goctxid_fiberlocals "github.com/hiiamtin/goctxid/adapters/fiberlocals"
)

func main() {
	app := fiber.New()

	// Add goctxid middleware using Fiber's c.Locals() (Fiber-native way)
	// This is more performant than the context-based adapter
	app.Use(goctxid_fiberlocals.New())

	// Example route - basic usage
	app.Get("/", func(c *fiber.Ctx) error {
		// Get correlation ID from Fiber's Locals (Fiber-native way)
		correlationID, exists := goctxid_fiberlocals.FromLocals(c)
		if !exists {
			return c.Status(500).SendString("Correlation ID not found")
		}

		return c.JSON(fiber.Map{
			"message":        "Hello, World!",
			"correlation_id": correlationID,
		})
	})

	// Example route - using MustFromLocals
	app.Get("/user/:id", func(c *fiber.Ctx) error {
		// MustFromLocals returns empty string if not found (no need to check exists)
		correlationID := goctxid_fiberlocals.MustFromLocals(c)
		userID := c.Params("id")

		// Use correlation ID in logs
		log.Printf("[%s] Fetching user: %s", correlationID, userID)

		return c.JSON(fiber.Map{
			"user_id":        userID,
			"correlation_id": correlationID,
		})
	})

	// Example route - custom configuration
	app.Get("/custom", func(c *fiber.Ctx) error {
		correlationID := goctxid_fiberlocals.MustFromLocals(c)

		return c.JSON(fiber.Map{
			"message":        "Custom configuration example",
			"correlation_id": correlationID,
		})
	})

	// Example route - service layer integration
	app.Get("/order/:id", func(c *fiber.Ctx) error {
		correlationID := goctxid_fiberlocals.MustFromLocals(c)
		orderID := c.Params("id")

		// Pass correlation ID to service layer
		order := processOrder(correlationID, orderID)

		return c.JSON(order)
	})

	log.Println("🚀 Server starting on http://localhost:3000")
	log.Println("")
	log.Println("Try these commands:")
	log.Println("  curl http://localhost:3000/")
	log.Println("  curl http://localhost:3000/user/123")
	log.Println("  curl -H \"X-Correlation-ID: my-custom-id\" http://localhost:3000/")
	log.Println("")
	log.Println("Note: This adapter uses c.Locals() which is more performant than context-based storage")
	log.Println("")

	log.Fatal(app.Listen(":3000"))
}

// processOrder simulates a service layer function that uses correlation ID
func processOrder(correlationID, orderID string) fiber.Map {
	log.Printf("[%s] Processing order: %s", correlationID, orderID)

	// Simulate some processing
	log.Printf("[%s] Validating order: %s", correlationID, orderID)
	log.Printf("[%s] Calculating total for order: %s", correlationID, orderID)
	log.Printf("[%s] Order processed successfully: %s", correlationID, orderID)

	return fiber.Map{
		"order_id":       orderID,
		"status":         "processed",
		"correlation_id": correlationID,
	}
}
