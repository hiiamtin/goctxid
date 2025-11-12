package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/hiiamtin/goctxid"
	goctxid_fiber "github.com/hiiamtin/goctxid/adapters/fiber"
	"github.com/hiiamtin/goctxid/adapters/fibernative"
)

func main() {
	app := fiber.New()

	// Example 1: Using Next function to skip middleware for health checks
	// This saves ~400-500 ns per skipped request
	app.Use("/api/*", goctxid_fiber.New(goctxid_fiber.Config{
		Next: func(c *fiber.Ctx) bool {
			// Skip middleware for health and metrics endpoints
			path := c.Path()
			return path == "/api/health" || path == "/api/metrics"
		},
	}))

	// Example 2: Using FastGenerator for high-throughput endpoints
	// ⚠️ WARNING: FastGenerator exposes request count - use only when acceptable
	app.Use("/api/fast/*", goctxid_fiber.New(goctxid_fiber.Config{
		Config: goctxid.Config{
			Generator: goctxid.FastGenerator, // ~33% faster, but exposes request count
		},
	}))

	// Example 3: Using fibernative with custom LocalsKey
	// Prevents collisions if you're already using c.Locals("goctxid")
	app.Use("/api/native/*", fibernative.New(fibernative.Config{
		LocalsKey: "my_correlation_id", // Custom key to avoid collisions
	}))

	// Example 4: Combining Next function with FastGenerator
	app.Use("/api/optimized/*", goctxid_fiber.New(goctxid_fiber.Config{
		Config: goctxid.Config{
			Generator: goctxid.FastGenerator,
		},
		Next: func(c *fiber.Ctx) bool {
			// Skip for static files
			return c.Path() == "/api/optimized/static"
		},
	}))

	// Routes demonstrating different configurations

	// Health check - middleware is skipped (no correlation ID)
	app.Get("/api/health", func(c *fiber.Ctx) error {
		id, exists := goctxid.FromContext(c.UserContext())
		return c.JSON(fiber.Map{
			"status":         "healthy",
			"correlation_id": id,
			"has_id":         exists, // Will be false - middleware was skipped
		})
	})

	// Metrics - middleware is skipped (no correlation ID)
	app.Get("/api/metrics", func(c *fiber.Ctx) error {
		id, exists := goctxid.FromContext(c.UserContext())
		return c.JSON(fiber.Map{
			"metrics":        "data",
			"correlation_id": id,
			"has_id":         exists, // Will be false - middleware was skipped
		})
	})

	// Normal API endpoint - uses default UUID v4 generator
	app.Get("/api/users", func(c *fiber.Ctx) error {
		id := goctxid.MustFromContext(c.UserContext())
		return c.JSON(fiber.Map{
			"message":        "List of users",
			"correlation_id": id,
			"generator":      "UUID v4 (secure)",
		})
	})

	// Fast endpoint - uses FastGenerator
	app.Get("/api/fast/data", func(c *fiber.Ctx) error {
		id := goctxid.MustFromContext(c.UserContext())
		return c.JSON(fiber.Map{
			"message":        "Fast data",
			"correlation_id": id,
			"generator":      "FastGenerator (exposes count)",
		})
	})

	// Native endpoint - uses fibernative with custom LocalsKey
	app.Get("/api/native/info", func(c *fiber.Ctx) error {
		// Use custom key to retrieve ID
		id := fibernative.MustFromLocalsWithKey(c, "my_correlation_id")
		return c.JSON(fiber.Map{
			"message":        "Native info",
			"correlation_id": id,
			"storage":        "c.Locals() with custom key",
		})
	})

	// Optimized endpoint - combines FastGenerator + Next function
	app.Get("/api/optimized/process", func(c *fiber.Ctx) error {
		id := goctxid.MustFromContext(c.UserContext())
		return c.JSON(fiber.Map{
			"message":        "Optimized processing",
			"correlation_id": id,
			"optimizations":  "FastGenerator + Next function",
		})
	})

	// Static endpoint - middleware is skipped
	app.Get("/api/optimized/static", func(c *fiber.Ctx) error {
		id, exists := goctxid.FromContext(c.UserContext())
		return c.JSON(fiber.Map{
			"message":        "Static content",
			"correlation_id": id,
			"has_id":         exists, // Will be false - middleware was skipped
		})
	})

	log.Println("🚀 Server starting on :3000")
	log.Println("\n📚 Try these examples:")
	log.Println("\n  # Health check (middleware skipped):")
	log.Println("  curl http://localhost:3000/api/health")
	log.Println("\n  # Normal endpoint (UUID v4):")
	log.Println("  curl http://localhost:3000/api/users")
	log.Println("\n  # Fast endpoint (FastGenerator):")
	log.Println("  curl http://localhost:3000/api/fast/data")
	log.Println("\n  # Native endpoint (custom LocalsKey):")
	log.Println("  curl http://localhost:3000/api/native/info")
	log.Println("\n  # Optimized endpoint (FastGenerator + Next):")
	log.Println("  curl http://localhost:3000/api/optimized/process")
	log.Println("\n  # Static endpoint (middleware skipped):")
	log.Println("  curl http://localhost:3000/api/optimized/static")
	log.Println("\n  # Check response headers:")
	log.Println("  curl -v http://localhost:3000/api/users")

	log.Fatal(app.Listen(":3000"))
}
