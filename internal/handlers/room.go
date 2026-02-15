package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/werewolf-game/backend/internal/game"
)

type CreateRoomRequest struct {
	Username string `json:"username" binding:"required"`
}

type JoinRoomRequest struct {
	Username string `json:"username" binding:"required"`
}

// CreateRoom creates a new game room
func CreateRoom(gm *game.GameManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateRoomRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		playerID := uuid.New().String()
		room := gm.CreateRoom(playerID, req.Username)

		c.JSON(http.StatusCreated, gin.H{
			"room":     room,
			"playerId": playerID,
		})
	}
}

// GetRoom retrieves room information
func GetRoom(gm *game.GameManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		code := c.Param("code")
		
		room, exists := gm.GetRoom(code)
		if !exists {
			c.JSON(http.StatusNotFound, gin.H{"error": "room not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"room": room})
	}
}

// JoinRoom adds a player to a room
func JoinRoom(gm *game.GameManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		code := c.Param("code")
		
		var req JoinRoomRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		playerID := uuid.New().String()
		room, err := gm.JoinRoom(code, playerID, req.Username)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"room":     room,
			"playerId": playerID,
		})
	}
}
