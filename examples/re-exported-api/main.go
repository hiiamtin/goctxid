package main

import (
	"context"
	"fmt"
	"log"

	"github.com/gofiber/fiber/v2"
	goctxid_fiber "github.com/hiiamtin/goctxid/adapters/fiber"
)

func main() {
	app := fiber.New()

	// Add correlation ID middleware
	app.Use(goctxid_fiber.New())

	// Example 1: Using re-exported FromContext (no need to import goctxid)
	app.Get("/api/user/:id", func(c *fiber.Ctx) error {
		// ✅ OLD WAY: Had to import "github.com/hiiamtin/goctxid"
		// import "github.com/hiiamtin/goctxid"
		// correlationID := goctxid.MustFromContext(c.UserContext())

		// ✅ NEW WAY: Use re-exported function directly from adapter
		correlationID := goctxid_fiber.MustFromContext(c.UserContext())

		userID := c.Params("id")

		return c.JSON(fiber.Map{
			"message":        "User retrieved successfully",
			"user_id":        userID,
			"correlation_id": correlationID,
		})
	})

	// Example 2: Using re-exported constants
	app.Get("/api/config", func(c *fiber.Ctx) error {
		// ✅ Use re-exported constant
		headerKey := goctxid_fiber.DefaultHeaderKey

		return c.JSON(fiber.Map{
			"header_key": headerKey,
			"message":    fmt.Sprintf("Correlation ID is stored in '%s' header", headerKey),
		})
	})

	// Example 3: Using re-exported generators
	app.Get("/api/custom-id", func(c *fiber.Ctx) error {
		// ✅ Use re-exported generators
		uuidID := goctxid_fiber.DefaultGenerator()
		fastID := goctxid_fiber.FastGenerator()

		return c.JSON(fiber.Map{
			"uuid_v4_id": uuidID,
			"fast_id":    fastID,
			"note":       "FastGenerator is faster but exposes request count",
		})
	})

	// Example 4: Creating new context with correlation ID
	app.Post("/api/process", func(c *fiber.Ctx) error {
		ctx := c.UserContext()

		// ✅ Use re-exported NewContext
		newCtx := goctxid_fiber.NewContext(ctx, "custom-process-id-123")

		// Retrieve it back
		processID := goctxid_fiber.MustFromContext(newCtx)

		return c.JSON(fiber.Map{
			"process_id": processID,
			"message":    "Process started with custom ID",
		})
	})

	// Example 5: Complete example with service layer
	app.Get("/api/order/:id", func(c *fiber.Ctx) error {
		ctx := c.UserContext()
		orderID := c.Params("id")

		// Pass context to service layer
		order := processOrder(ctx, orderID)

		return c.JSON(order)
	})

	fmt.Println("🚀 Server started on http://localhost:3000")
	fmt.Println("\n📝 Try these commands:")
	fmt.Println("  curl http://localhost:3000/api/user/123")
	fmt.Println("  curl http://localhost:3000/api/config")
	fmt.Println("  curl http://localhost:3000/api/custom-id")
	fmt.Println("  curl -X POST http://localhost:3000/api/process")
	fmt.Println("  curl http://localhost:3000/api/order/456")
	fmt.Println("\n✨ Notice: No need to import 'github.com/hiiamtin/goctxid' package!")
	fmt.Println("   All functions are re-exported from goctxid_fiber adapter")

	log.Fatal(app.Listen(":3000"))
}

// Service layer function that uses correlation ID
func processOrder(ctx context.Context, orderID string) map[string]interface{} {
	// ✅ Use re-exported function in service layer too!
	correlationID := goctxid_fiber.MustFromContext(ctx)

	// Simulate processing
	log.Printf("[%s] Processing order: %s", correlationID, orderID)

	return map[string]interface{}{
		"order_id":       orderID,
		"status":         "processing",
		"correlation_id": correlationID,
	}
}
