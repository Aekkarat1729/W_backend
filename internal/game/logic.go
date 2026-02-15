package game

import (
	"strings"

	"github.com/werewolf-game/backend/internal/models"
)

// ProcessNightPhase processes all night actions
func (gm *GameManager) ProcessNightPhase(code string) (*NightResult, error) {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	code = strings.ToUpper(code)
	room, exists := gm.Rooms[code]
	if !exists {
		return nil, ErrRoomNotFound
	}

	result := &NightResult{
		Killed:       "",
		Protected:    false,
		ShamanSaved:  false,
		ShamanVision: "",
		VisionResult: "",
	}

	// 1. Check if hunter protected the tiger's target
	if room.TigerTarget != "" {
		if room.HunterProtection == room.TigerTarget {
			result.Protected = true
			result.Killed = ""
		} else {
			// Check if victim is shaman who saw alpha tiger
			victim := room.Players[room.TigerTarget]
			if victim != nil && victim.Role == models.RoleShaman && room.ShamanVision != "" {
				// Check if shaman saw alpha tiger tonight
				seen := room.Players[room.ShamanVision]
				if seen != nil && seen.Role == models.RoleAlphaTiger && !seen.HasUsedCurse {
					// Shaman survives (ดวงแข็ง)
					result.ShamanSaved = true
					result.Killed = ""
				} else {
					// Shaman dies
					victim.IsAlive = false
					result.Killed = room.TigerTarget
				}
			} else {
				// Normal death
				victim.IsAlive = false
				result.Killed = room.TigerTarget
			}
		}
	}

	// 2. Process shaman's vision
	if room.ShamanVision != "" {
		target := room.Players[room.ShamanVision]
		if target != nil {
			// Check if target is cursed
			if target.IsCursed {
				result.VisionResult = "tiger"
			} else if target.Role == models.RoleAlphaTiger {
				// Alpha tiger can hide unless curse was used
				if target.HasUsedCurse {
					result.VisionResult = "tiger"
				} else {
					result.VisionResult = "human"
				}
			} else if target.Role == models.RoleTiger {
				result.VisionResult = "tiger"
			} else {
				result.VisionResult = "human"
			}
			result.ShamanVision = target.Username
		}
	}

	// Reset night actions
	room.TigerTarget = ""
	room.HunterProtection = ""
	room.ShamanVision = ""

	return result, nil
}

// SetAlphaTigerCurse sets a curse on a player
func (gm *GameManager) SetAlphaTigerCurse(code, alphaTigerID, targetID string) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	code = strings.ToUpper(code)
	room, exists := gm.Rooms[code]
	if !exists {
		return ErrRoomNotFound
	}

	alphaTiger := room.Players[alphaTigerID]
	if alphaTiger == nil || alphaTiger.Role != models.RoleAlphaTiger {
		return &GameError{"not alpha tiger"}
	}

	if alphaTiger.HasUsedCurse {
		return &GameError{"curse already used"}
	}

	target := room.Players[targetID]
	if target == nil {
		return &GameError{"target not found"}
	}

	// Set curse
	target.IsCursed = true
	alphaTiger.HasUsedCurse = true
	room.CursedPlayer = targetID

	return nil
}

// SetTigerTarget sets the tiger's kill target
func (gm *GameManager) SetTigerTarget(code, targetID string) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	code = strings.ToUpper(code)
	room, exists := gm.Rooms[code]
	if !exists {
		return ErrRoomNotFound
	}

	room.TigerTarget = targetID
	return nil
}

// SetHunterProtection sets the hunter's protection target
func (gm *GameManager) SetHunterProtection(code, hunterID, targetID string) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	code = strings.ToUpper(code)
	room, exists := gm.Rooms[code]
	if !exists {
		return ErrRoomNotFound
	}

	hunter := room.Players[hunterID]
	if hunter == nil || hunter.Role != models.RoleHunter {
		return &GameError{"not hunter"}
	}

	// Can't protect same person two nights in a row
	if hunter.LastProtected == targetID {
		return &GameError{"cannot protect same person twice in a row"}
	}

	room.HunterProtection = targetID
	hunter.LastProtected = targetID

	return nil
}

// SetShamanVision sets the shaman's vision target
func (gm *GameManager) SetShamanVision(code, targetID string) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	code = strings.ToUpper(code)
	room, exists := gm.Rooms[code]
	if !exists {
		return ErrRoomNotFound
	}

	room.ShamanVision = targetID
	return nil
}

// HunterShoot allows hunter to shoot when dying
func (gm *GameManager) HunterShoot(code, hunterID, targetID string) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	code = strings.ToUpper(code)
	room, exists := gm.Rooms[code]
	if !exists {
		return ErrRoomNotFound
	}

	hunter := room.Players[hunterID]
	if hunter == nil || hunter.Role != models.RoleHunter || hunter.IsAlive {
		return &GameError{"invalid hunter shoot"}
	}

	target := room.Players[targetID]
	if target == nil {
		return &GameError{"target not found"}
	}

	target.IsAlive = false
	return nil
}

// ProcessVoting processes voting results
func (gm *GameManager) ProcessVoting(code string, votes map[string]string) (string, error) {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	code = strings.ToUpper(code)
	room, exists := gm.Rooms[code]
	if !exists {
		return "", ErrRoomNotFound
	}

	// Count votes
	voteCount := make(map[string]int)
	for _, targetID := range votes {
		voteCount[targetID]++
	}

	// Find player with most votes
	maxVotes := 0
	var eliminated string
	for playerID, count := range voteCount {
		if count > maxVotes {
			maxVotes = count
			eliminated = playerID
		}
	}

	// Eliminate player
	if eliminated != "" {
		player := room.Players[eliminated]
		if player != nil {
			player.IsAlive = false

			// If hunter, allow shooting
			if player.Role == models.RoleHunter {
				player.CanShoot = true
			}
		}
	}

	room.VoteResults = voteCount
	return eliminated, nil
}

// CheckGameEnd checks if the game has ended
func (gm *GameManager) CheckGameEnd(code string) (bool, string, error) {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	code = strings.ToUpper(code)
	room, exists := gm.Rooms[code]
	if !exists {
		return false, "", ErrRoomNotFound
	}

	tigerCount := 0
	humanCount := 0

	for _, player := range room.Players {
		if player.IsAlive {
			if player.Role == models.RoleAlphaTiger || player.Role == models.RoleTiger {
				tigerCount++
			} else {
				humanCount++
			}
		}
	}

	// Tigers win if they equal or outnumber humans
	if tigerCount >= humanCount && tigerCount > 0 {
		return true, "tigers", nil
	}

	// Humans win if all tigers are dead
	if tigerCount == 0 {
		return true, "humans", nil
	}

	return false, "", nil
}

// NightResult represents the result of night actions
type NightResult struct {
	Killed       string `json:"killed"`       // ID of killed player
	Protected    bool   `json:"protected"`    // Was target protected
	ShamanSaved  bool   `json:"shamanSaved"`  // Shaman saved by luck
	ShamanVision string `json:"shamanVision"` // Who shaman saw
	VisionResult string `json:"visionResult"` // "tiger" or "human"
}
