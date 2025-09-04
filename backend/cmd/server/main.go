package main

import (
	"fmt"
	"log"

	"SYS_DESIGN_PLAYGROUND/internal/api"
	"SYS_DESIGN_PLAYGROUND/internal/registry"
	_ "SYS_DESIGN_PLAYGROUND/scenarios/cache_inconsistency" // Import for side-effect of registration

	"github.com/gin-gonic/gin"
)

func main() {
	fmt.Println("Starting Backend Problem Playground server...")

	// Initialize all registered scenarios
	if err := registry.InitializeAll(); err != nil {
		log.Fatalf("Failed to initialize scenarios: %v", err)
	}

	// Initialize Gin router
	router := gin.Default()

	// Setup API routes
	api.SetupRouter(router)

	// Simple health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "UP",
		})
	})

	// Start the server
	port := "8080"
	log.Printf("Server listening on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
