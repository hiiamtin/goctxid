package main

import (
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/hiiamtin/goctxid"
	goctxid_fiber "github.com/hiiamtin/goctxid/adapters/fiber"
)

var requestCounter uint64

// customIDGenerator creates sequential IDs with timestamp
// Format: REQ-{timestamp}-{counter}
// NOTE: This generator is thread-safe (uses atomic counter)
func customIDGenerator() string {
	counter := atomic.AddUint64(&requestCounter, 1)
	timestamp := time.Now().Unix()
	return fmt.Sprintf("REQ-%d-%06d", timestamp, counter)
}

// prefixedUUIDGenerator adds a custom prefix to UUIDs
// Format: APP-{uuid}
func prefixedUUIDGenerator(prefix string) func() string {
	return func() string {
		// Use the default UUID generator from goctxid
		// In real implementation, you'd import "github.com/google/uuid"
		return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	}
}

func main() {
	app := fiber.New()

	// Example 1: Using custom sequential ID generator
	app.Use("/api/v1/*", goctxid_fiber.New(goctxid.Config{
		Generator: customIDGenerator,
	}))

	// Example 2: Using prefixed UUID generator
	app.Use("/api/v2/*", goctxid_fiber.New(goctxid.Config{
		Generator: prefixedUUIDGenerator("MYAPP"),
	}))

	// Example 3: Custom header key
	app.Use("/api/v3/*", goctxid_fiber.New(goctxid.Config{
		HeaderKey: "X-Request-ID", // Different header name
		Generator: customIDGenerator,
	}))

	// Routes for testing different configurations
	app.Get("/api/v1/test", func(c *fiber.Ctx) error {
		correlationID, _ := goctxid.FromContext(c.UserContext())
		return c.JSON(fiber.Map{
			"version":        "v1",
			"generator":      "sequential",
			"correlation_id": correlationID,
		})
	})

	app.Get("/api/v2/test", func(c *fiber.Ctx) error {
		correlationID, _ := goctxid.FromContext(c.UserContext())
		return c.JSON(fiber.Map{
			"version":        "v2",
			"generator":      "prefixed-uuid",
			"correlation_id": correlationID,
		})
	})

	app.Get("/api/v3/test", func(c *fiber.Ctx) error {
		correlationID, _ := goctxid.FromContext(c.UserContext())
		return c.JSON(fiber.Map{
			"version":        "v3",
			"generator":      "sequential",
			"header":         "X-Request-ID",
			"correlation_id": correlationID,
		})
	})

	log.Println("Server starting on :3000")
	log.Println("\nTry these examples:")
	log.Println("  # Sequential ID generator:")
	log.Println("  curl http://localhost:3000/api/v1/test")
	log.Println("\n  # Prefixed UUID generator:")
	log.Println("  curl http://localhost:3000/api/v2/test")
	log.Println("\n  # Custom header key:")
	log.Println("  curl -v http://localhost:3000/api/v3/test")
	log.Println("\n  # Provide your own ID:")
	log.Println("  curl -H 'X-Correlation-ID: my-custom-id' http://localhost:3000/api/v1/test")

	log.Fatal(app.Listen(":3000"))
}
