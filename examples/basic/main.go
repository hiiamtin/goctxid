package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/hiiamtin/goctxid"
)

func main() {
	app := fiber.New()

	// Add goctxid middleware
	app.Use(goctxid.New())

	// Example route
	app.Get("/", func(c *fiber.Ctx) error {
		// Get correlation ID from context
		correlationID, exists := goctxid.FromContext(c.UserContext())
		if !exists {
			return c.Status(500).SendString("Correlation ID not found")
		}

		return c.JSON(fiber.Map{
			"message":        "Hello, World!",
			"correlation_id": correlationID,
		})
	})

	// Example route with logging
	app.Get("/user/:id", func(c *fiber.Ctx) error {
		correlationID, _ := goctxid.FromContext(c.UserContext())
		userID := c.Params("id")

		// Use correlation ID in logs
		log.Printf("[%s] Fetching user: %s", correlationID, userID)

		return c.JSON(fiber.Map{
			"user_id":        userID,
			"correlation_id": correlationID,
		})
	})

	log.Println("Server starting on :3000")
	log.Println("Try:")
	log.Println("  curl http://localhost:3000/")
	log.Println("  curl http://localhost:3000/user/123")
	log.Println("  curl -H 'X-Correlation-ID: my-custom-id' http://localhost:3000/")

	log.Fatal(app.Listen(":3000"))
}

