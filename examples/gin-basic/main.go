package main

import (
	"context"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hiiamtin/goctxid"
	goctxid_gin "github.com/hiiamtin/goctxid/adapters/gin"
)

func main() {
	r := gin.Default()

	// Add goctxid middleware
	r.Use(goctxid_gin.New())

	// Basic route
	r.GET("/", func(c *gin.Context) {
		// Get correlation ID from context
		correlationID, exists := goctxid.FromContext(c.Request.Context())

		c.JSON(http.StatusOK, gin.H{
			"message":        "Hello from Gin!",
			"correlation_id": correlationID,
			"id_exists":      exists,
		})
	})

	// Route with custom header
	r.GET("/custom", func(c *gin.Context) {
		correlationID := goctxid.MustFromContext(c.Request.Context())

		c.JSON(http.StatusOK, gin.H{
			"message":        "Custom route",
			"correlation_id": correlationID,
		})
	})

	// Route demonstrating service layer usage
	r.GET("/user/:id", func(c *gin.Context) {
		ctx := c.Request.Context()
		userID := c.Param("id")

		// Simulate service layer call
		user := getUserFromService(ctx, userID)

		c.JSON(http.StatusOK, user)
	})

	log.Println("Gin server starting on :3000")
	log.Println("Try:")
	log.Println("  curl http://localhost:3000/")
	log.Println("  curl -H 'X-Correlation-ID: my-custom-id' http://localhost:3000/")
	log.Println("  curl http://localhost:3000/user/123")

	if err := r.Run(":3000"); err != nil {
		log.Fatal(err)
	}
}

// getUserFromService simulates a service layer that uses the correlation ID
func getUserFromService(ctx context.Context, userID string) gin.H {
	correlationID := goctxid.MustFromContext(ctx)

	// In a real app, you'd use the correlation ID for logging
	log.Printf("[%s] Fetching user: %s", correlationID, userID)

	return gin.H{
		"id":             userID,
		"name":           "John Doe",
		"correlation_id": correlationID,
	}
}
