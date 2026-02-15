package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/werewolf-game/backend/internal/game"
	"github.com/werewolf-game/backend/internal/models"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in development
	},
}

type Client struct {
	ID       string
	RoomCode string
	Conn     *websocket.Conn
	Send     chan []byte
}

type Hub struct {
	Clients    map[string]*Client
	Broadcast  chan *BroadcastMessage
	Register   chan *Client
	Unregister chan *Client
	mu         sync.RWMutex
}

type BroadcastMessage struct {
	RoomCode string
	Message  []byte
}

var hub = &Hub{
	Clients:    make(map[string]*Client),
	Broadcast:  make(chan *BroadcastMessage),
	Register:   make(chan *Client),
	Unregister: make(chan *Client),
}

func init() {
	go hub.Run()
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			h.Clients[client.ID] = client
			h.mu.Unlock()
			log.Printf("Client registered: %s in room %s", client.ID, client.RoomCode)

		case client := <-h.Unregister:
			h.mu.Lock()
			if _, ok := h.Clients[client.ID]; ok {
				delete(h.Clients, client.ID)
				close(client.Send)
				log.Printf("Client unregistered: %s", client.ID)
			}
			h.mu.Unlock()

		case message := <-h.Broadcast:
			h.mu.RLock()
			for _, client := range h.Clients {
				if client.RoomCode == message.RoomCode {
					select {
					case client.Send <- message.Message:
					default:
						close(client.Send)
						delete(h.Clients, client.ID)
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

// HandleWebSocket handles WebSocket connections
func HandleWebSocket(gm *game.GameManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("WebSocket upgrade error: %v", err)
			return
		}

		playerID := c.Query("playerId")
		roomCode := c.Query("roomCode")

		if playerID == "" || roomCode == "" {
			conn.Close()
			return
		}

		client := &Client{
			ID:       playerID,
			RoomCode: roomCode,
			Conn:     conn,
			Send:     make(chan []byte, 256),
		}

		hub.Register <- client

		// Send current room state to the newly connected client
		room, exists := gm.GetRoom(roomCode)
		if exists {
			sendToClient(client, models.EventGameStateUpdate, room)
			
			// Broadcast player joined event to all clients in the room
			broadcastToRoom(roomCode, models.EventPlayerJoined, room)
		}

		go client.WritePump()
		go client.ReadPump(gm)
	}
}

func (c *Client) ReadPump(gm *game.GameManager) {
	defer func() {
		hub.Unregister <- c
		c.Conn.Close()
	}()

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		var wsMsg models.WSMessage
		if err := json.Unmarshal(message, &wsMsg); err != nil {
			log.Printf("JSON unmarshal error: %v", err)
			continue
		}

		handleWebSocketMessage(c, gm, &wsMsg)
	}
}

func (c *Client) WritePump() {
	defer c.Conn.Close()

	for message := range c.Send {
		if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
			log.Printf("Write error: %v", err)
			return
		}
	}
}

func handleWebSocketMessage(client *Client, gm *game.GameManager, msg *models.WSMessage) {
	switch msg.Type {
	case models.EventStartGame:
		if err := gm.StartGame(client.RoomCode); err != nil {
			sendError(client, err.Error())
			return
		}
		
		room, _ := gm.GetRoom(client.RoomCode)
		broadcastToRoom(client.RoomCode, models.EventGameStarted, room)

	case models.EventChatMessage:
		broadcastToRoom(client.RoomCode, models.EventChatMessage, msg.Payload)

	case models.EventVote:
		// Handle voting logic
		broadcastToRoom(client.RoomCode, models.EventVote, msg.Payload)

	case models.EventNightAction:
		// Handle night action logic
		broadcastToRoom(client.RoomCode, models.EventNightAction, msg.Payload)
	}
}

func broadcastToRoom(roomCode, eventType string, payload interface{}) {
	msg := models.WSMessage{
		Type:    eventType,
		Payload: payload,
	}
	
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("JSON marshal error: %v", err)
		return
	}

	hub.Broadcast <- &BroadcastMessage{
		RoomCode: roomCode,
		Message:  data,
	}
}

func sendError(client *Client, errMsg string) {
	msg := models.WSMessage{
		Type:    models.EventError,
		Payload: map[string]string{"error": errMsg},
	}
	
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("JSON marshal error: %v", err)
		return
	}
	
	client.Send <- data
}

func sendToClient(client *Client, eventType string, payload interface{}) {
	msg := models.WSMessage{
		Type:    eventType,
		Payload: payload,
	}
	
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("JSON marshal error: %v", err)
		return
	}
	
	client.Send <- data
}
	
	data, _ := json.Marshal(msg)
	client.Send <- data
}
