package game

import (
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/werewolf-game/backend/internal/models"
)

// GameManager manages all game rooms
type GameManager struct {
	Rooms map[string]*models.GameRoom
	mu    sync.RWMutex
}

// NewGameManager creates a new game manager
func NewGameManager() *GameManager {
	return &GameManager{
		Rooms: make(map[string]*models.GameRoom),
	}
}

// CreateRoom creates a new game room
func (gm *GameManager) CreateRoom(hostID, hostUsername string) *models.GameRoom {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	code := generateRoomCode()
	room := &models.GameRoom{
		Code:       code,
		HostID:     hostID,
		Players:    make(map[string]*models.Player),
		Phase:      models.PhaseWaiting,
		Round:      0,
		MaxPlayers: 10,
		CreatedAt:  time.Now(),
	}

	// Add host as first player
	room.Players[hostID] = &models.Player{
		ID:       hostID,
		Username: hostUsername,
		IsAlive:  true,
		IsReady:  false,
		RoomCode: code,
		JoinedAt: time.Now(),
	}

	gm.Rooms[code] = room
	return room
}

// GetRoom retrieves a room by code
func (gm *GameManager) GetRoom(code string) (*models.GameRoom, bool) {
	gm.mu.RLock()
	defer gm.mu.RUnlock()
	code = strings.ToUpper(code)
	room, exists := gm.Rooms[code]
	return room, exists
}

// JoinRoom adds a player to a room
func (gm *GameManager) JoinRoom(code, playerID, username string) (*models.GameRoom, error) {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	code = strings.ToUpper(code)
	room, exists := gm.Rooms[code]
	if !exists {
		return nil, ErrRoomNotFound
	}

	if len(room.Players) >= room.MaxPlayers {
		return nil, ErrRoomFull
	}

	if room.Phase != models.PhaseWaiting {
		return nil, ErrGameAlreadyStarted
	}

	room.Players[playerID] = &models.Player{
		ID:       playerID,
		Username: username,
		IsAlive:  true,
		IsReady:  false,
		RoomCode: code,
		JoinedAt: time.Now(),
	}

	return room, nil
}

// RemovePlayer removes a player from a room
func (gm *GameManager) RemovePlayer(code, playerID string) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	code = strings.ToUpper(code)
	room, exists := gm.Rooms[code]
	if !exists {
		return ErrRoomNotFound
	}

	delete(room.Players, playerID)

	// Delete room if empty
	if len(room.Players) == 0 {
		delete(gm.Rooms, code)
	}

	return nil
}

// StartGame assigns roles and starts the game
func (gm *GameManager) StartGame(code string) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	code = strings.ToUpper(code)
	room, exists := gm.Rooms[code]
	if !exists {
		return ErrRoomNotFound
	}

	if len(room.Players) < 4 {
		return ErrNotEnoughPlayers
	}

	// Assign roles
	assignRoles(room)

	// Start game
	now := time.Now()
	room.StartedAt = &now
	room.Phase = models.PhaseNight
	room.Round = 1

	return nil
}

// assignRoles randomly assigns roles to players
func assignRoles(room *models.GameRoom) {
	playerCount := len(room.Players)

	// Calculate role distribution
	// 4-5 players: 1 Alpha Tiger, 1 Tiger
	// 6-7 players: 1 Alpha Tiger, 2 Tigers
	// 8-10 players: 1 Alpha Tiger, 2 Tigers
	tigerCount := 1
	if playerCount >= 6 {
		tigerCount = 2
	}

	roles := make([]models.Role, 0, playerCount)

	// Add Alpha Tiger (พญาสมิง) - always 1
	roles = append(roles, models.RoleAlphaTiger)

	// Add Tigers (เสือสมิง)
	for i := 0; i < tigerCount; i++ {
		roles = append(roles, models.RoleTiger)
	}

	// Add special roles (always have if enough players)
	roles = append(roles, models.RoleShaman) // หมอผี
	roles = append(roles, models.RoleHunter) // นายพราน

	// Fill remaining with villagers
	for len(roles) < playerCount {
		roles = append(roles, models.RoleVillager)
	}

	// Shuffle roles
	rand.Shuffle(len(roles), func(i, j int) {
		roles[i], roles[j] = roles[j], roles[i]
	})

	// Assign to players
	i := 0
	for _, player := range room.Players {
		player.Role = roles[i]
		player.IsCursed = false
		player.HasUsedCurse = false
		player.CanShoot = false
		player.LastProtected = ""
		i++
	}
}

// generateRoomCode generates a random 6-character room code
func generateRoomCode() string {
	code := uuid.New().String()[:6]
	return strings.ToUpper(code)
}

// Custom errors
var (
	ErrRoomNotFound       = &GameError{"room not found"}
	ErrRoomFull           = &GameError{"room is full"}
	ErrGameAlreadyStarted = &GameError{"game already started"}
	ErrNotEnoughPlayers   = &GameError{"not enough players to start"}
)

type GameError struct {
	message string
}

func (e *GameError) Error() string {
	return e.message
}
