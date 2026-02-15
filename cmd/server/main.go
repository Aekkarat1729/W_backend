package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/werewolf-game/backend/internal/game"
	"github.com/werewolf-game/backend/internal/handlers"
)

func main() {
	// Initialize game manager
	gameManager := game.NewGameManager()

	// Setup Gin router
	router := gin.Default()

	// CORS middleware
	router.Use(func(c *gin.Context) {
		// Get allowed origin from environment variable, default to "*" for development
		allowedOrigin := os.Getenv("ALLOWED_ORIGIN")
		if allowedOrigin == "" {
			allowedOrigin = "*"
		}

		c.Writer.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// API routes
	api := router.Group("/api")
	{
		api.POST("/rooms", handlers.CreateRoom(gameManager))
		api.GET("/rooms/:code", handlers.GetRoom(gameManager))
		api.POST("/rooms/:code/join", handlers.JoinRoom(gameManager))
	}

	// WebSocket endpoint
	router.GET("/ws", handlers.HandleWebSocket(gameManager))

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("🎮 Werewolf Game Server starting on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
