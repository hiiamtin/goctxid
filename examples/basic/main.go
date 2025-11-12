package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/hiiamtin/goctxid"
	goctxid_fiber "github.com/hiiamtin/goctxid/adapters/fiber"
)

func main() {
	app := fiber.New()

	// Add goctxid middleware
	app.Use(goctxid_fiber.New())

	// Example route - using convenience function
	app.Get("/", func(c *fiber.Ctx) error {
		// Get correlation ID using the convenience function
		correlationID := goctxid_fiber.GetCorrelationID(c)

		return c.JSON(fiber.Map{
			"message":        "Hello, World!",
			"correlation_id": correlationID,
		})
	})

	// Example route with logging - alternative method
	app.Get("/user/:id", func(c *fiber.Ctx) error {
		// Alternative: Get from context directly
		correlationID := goctxid.MustFromContext(c.UserContext())
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
