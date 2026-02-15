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

	if len(room.Players) < 5 {
		return ErrNotEnoughPlayers
	}

	// Assign roles
	assignRoles(room)

	// Start game
	now := time.Now()
	room.StartedAt = &now
	room.Phase = models.PhaseDay // เริ่มที่เช้าเลย
	room.Round = 1               // เริ่มรอบ 1
	endTime := now.Add(2 * time.Minute)
	room.PhaseEndTime = &endTime // ตั้งเวลา 2 นาทีสำหรับเฟสกลางวัน

	// Initialize night actions tracking
	for _, player := range room.Players {
		player.HasActedThisNight = false
	}
	room.NightActionsCompleted = make(map[string]bool)

	return nil
}

// assignRoles randomly assigns roles to players
func assignRoles(room *models.GameRoom) {
	playerCount := len(room.Players)

	// Calculate role distribution based on player count
	// 5 คน: เสือ 1, ชาวบ้าน 2, พราน 1, หมอผี 1
	// 6 คน: เสือ 1, ชาวบ้าน 3, พราน 1, หมอผี 1
	// 7+ คน: เสือ 1, พญาสมิง 1, ชาวบ้าน (เหลือ), พราน 1, หมอผี 1

	roles := make([]models.Role, 0, playerCount)

	if playerCount >= 7 {
		// 7+ คน: มีพญาสมิง
		roles = append(roles, models.RoleAlphaTiger) // พญาสมิง
		roles = append(roles, models.RoleTiger)      // เสือสมิง
	} else {
		// 5-6 คน: มีแค่เสือสมิง (ไม่มีพญาสมิง)
		roles = append(roles, models.RoleTiger) // เสือสมิง
	}

	// เพิ่มบทบาทพิเศษ (มีเสมอ)
	roles = append(roles, models.RoleHunter) // นายพราน
	roles = append(roles, models.RoleShaman) // หมอผี

	// เติมที่เหลือด้วยชาวบ้าน
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

// SkipPhase allows host to skip current phase
func (gm *GameManager) SkipPhase(code, playerID string) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	code = strings.ToUpper(code)
	room, exists := gm.Rooms[code]
	if !exists {
		return ErrRoomNotFound
	}

	if room.HostID != playerID {
		return &GameError{"only host can skip phase"}
	}

	// Clear phase end time
	room.PhaseEndTime = nil

	return nil
}

// MarkNightActionComplete marks a player as having completed their night action
func (gm *GameManager) MarkNightActionComplete(code, playerID string) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	code = strings.ToUpper(code)
	room, exists := gm.Rooms[code]
	if !exists {
		return ErrRoomNotFound
	}

	player := room.Players[playerID]
	if player == nil {
		return &GameError{"player not found"}
	}

	// Mark as acted
	player.HasActedThisNight = true

	// Initialize map if nil
	if room.NightActionsCompleted == nil {
		room.NightActionsCompleted = make(map[string]bool)
	}

	room.NightActionsCompleted[playerID] = true

	return nil
}

// CheckAllNightActionsComplete checks if all required night actions are complete
func (gm *GameManager) CheckAllNightActionsComplete(code string) bool {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	code = strings.ToUpper(code)
	room, exists := gm.Rooms[code]
	if !exists {
		return false
	}

	// Count players with night actions
	required := 0
	for _, player := range room.Players {
		if !player.IsAlive {
			continue
		}
		// Only count players with night abilities
		if player.Role == models.RoleTiger || player.Role == models.RoleAlphaTiger ||
			player.Role == models.RoleHunter || player.Role == models.RoleShaman {
			required++
		}
	}

	room.NightActionsRequired = required

	// Check if all have acted
	completed := len(room.NightActionsCompleted)
	return completed >= required
}

// StartDayPhase sets up the day phase with 2-minute timer
func (gm *GameManager) StartDayPhase(code string) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	code = strings.ToUpper(code)
	room, exists := gm.Rooms[code]
	if !exists {
		return ErrRoomNotFound
	}

	room.Phase = models.PhaseDay
	endTime := time.Now().Add(2 * time.Minute)
	room.PhaseEndTime = &endTime

	// Reset night actions tracking
	for _, player := range room.Players {
		player.HasActedThisNight = false
	}
	room.NightActionsCompleted = make(map[string]bool)

	return nil
}

// StartNightPhase sets up the night phase (no timer)
func (gm *GameManager) StartNightPhase(code string) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	code = strings.ToUpper(code)
	room, exists := gm.Rooms[code]
	if !exists {
		return ErrRoomNotFound
	}

	room.Phase = models.PhaseNight
	room.PhaseEndTime = nil // No timer for night phase

	// Reset night actions tracking
	for _, player := range room.Players {
		player.HasActedThisNight = false
	}
	room.NightActionsCompleted = make(map[string]bool)

	return nil
}

// MoveToNextPhase transitions the game to the next phase
func (gm *GameManager) MoveToNextPhase(code string) (*NightResult, error) {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	code = strings.ToUpper(code)
	room, exists := gm.Rooms[code]
	if !exists {
		return nil, ErrRoomNotFound
	}

	var nightResult *NightResult

	switch room.Phase {
	case models.PhaseNight:
		// Process night actions before moving to day
		result, err := gm.ProcessNightPhase(code)
		if err != nil {
			return nil, err
		}
		nightResult = result

		// Night -> Day
		room.Phase = models.PhaseDay
		endTime := time.Now().Add(2 * time.Minute)
		room.PhaseEndTime = &endTime
		room.Round++ // Increment round when day starts

		// Reset night actions tracking
		for _, player := range room.Players {
			player.HasActedThisNight = false
		}
		room.NightActionsCompleted = make(map[string]bool)

	case models.PhaseDay:
		// Day -> Voting
		room.Phase = models.PhaseVoting
		room.PhaseEndTime = nil

	case models.PhaseVoting:
		// Voting -> Night
		room.Phase = models.PhaseNight
		room.PhaseEndTime = nil

		// Reset night actions tracking
		for _, player := range room.Players {
			player.HasActedThisNight = false
		}
		room.NightActionsCompleted = make(map[string]bool)

	default:
		return nil, &GameError{"invalid phase transition"}
	}

	return nightResult, nil
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
