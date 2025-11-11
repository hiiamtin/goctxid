package main

import (
	"context"
	"log"

	"github.com/hiiamtin/goctxid"
	goctxid_echo "github.com/hiiamtin/goctxid/adapters/echo"
	"github.com/labstack/echo/v4"
)

func main() {
	e := echo.New()

	// Add goctxid middleware
	e.Use(goctxid_echo.New())

	// Basic route
	e.GET("/", func(c echo.Context) error {
		// Get correlation ID from context
		correlationID, exists := goctxid.FromContext(c.Request().Context())

		return c.JSON(200, map[string]interface{}{
			"message":        "Hello from Echo!",
			"correlation_id": correlationID,
			"id_exists":      exists,
		})
	})

	// Route with custom header
	e.GET("/custom", func(c echo.Context) error {
		correlationID := goctxid.MustFromContext(c.Request().Context())

		return c.JSON(200, map[string]string{
			"message":        "Custom route",
			"correlation_id": correlationID,
		})
	})

	// Route demonstrating service layer usage
	e.GET("/user/:id", func(c echo.Context) error {
		ctx := c.Request().Context()
		userID := c.Param("id")

		// Simulate service layer call
		user := getUserFromService(ctx, userID)

		return c.JSON(200, user)
	})

	log.Println("Echo server starting on :3000")
	log.Println("Try:")
	log.Println("  curl http://localhost:3000/")
	log.Println("  curl -H 'X-Correlation-ID: my-custom-id' http://localhost:3000/")
	log.Println("  curl http://localhost:3000/user/123")

	if err := e.Start(":3000"); err != nil {
		log.Fatal(err)
	}
}

// getUserFromService simulates a service layer that uses the correlation ID
func getUserFromService(ctx context.Context, userID string) map[string]interface{} {
	correlationID := goctxid.MustFromContext(ctx)

	// In a real app, you'd use the correlation ID for logging
	log.Printf("[%s] Fetching user: %s", correlationID, userID)

	return map[string]interface{}{
		"id":             userID,
		"name":           "John Doe",
		"correlation_id": correlationID,
	}
}
