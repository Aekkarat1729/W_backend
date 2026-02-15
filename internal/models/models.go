package models

import "time"

// GamePhase represents the current phase of the game
type GamePhase string

const (
	PhaseWaiting GamePhase = "waiting"
	PhaseNight   GamePhase = "night"
	PhaseDay     GamePhase = "day"
	PhaseVoting  GamePhase = "voting"
	PhaseEnded   GamePhase = "ended"
)

// Role represents player roles in the game
type Role string

const (
	RoleAlphaTiger Role = "alpha_tiger" // พญาสมิง
	RoleTiger      Role = "tiger"       // เสือสมิง
	RoleShaman     Role = "shaman"      // หมอผี
	RoleHunter     Role = "hunter"      // นายพราน
	RoleVillager   Role = "villager"    // ชาวบ้าน
)

// Player represents a player in the game
type Player struct {
	ID                string    `json:"id"`
	Username          string    `json:"username"`
	Role              Role      `json:"role,omitempty"` // Hidden from other players
	IsAlive           bool      `json:"isAlive"`
	IsReady           bool      `json:"isReady"`
	IsCursed          bool      `json:"isCursed,omitempty"`          // ถูกสาปโดยพญาสมิง
	HasUsedCurse      bool      `json:"hasUsedCurse,omitempty"`      // พญาสมิงใช้สาปแล้ว
	CanShoot          bool      `json:"canShoot,omitempty"`          // นายพรานสามารถยิงได้
	LastProtected     string    `json:"lastProtected,omitempty"`     // ID ของคนที่กันไปคืนก่อน
	HasActedThisNight bool      `json:"hasActedThisNight,omitempty"` // ใช้ความสามารถในคืนนี้แล้ว
	VotedFor          string    `json:"votedFor,omitempty"`          // ID ของคนที่โหวต (ใน voting phase)
	RoomCode          string    `json:"roomCode"`
	JoinedAt          time.Time `json:"joinedAt"`
}

// GameRoom represents a game room
type GameRoom struct {
	Code                  string             `json:"code"`
	HostID                string             `json:"hostId"`
	Players               map[string]*Player `json:"players"`
	Phase                 GamePhase          `json:"phase"`
	Round                 int                `json:"round"`
	MaxPlayers            int                `json:"maxPlayers"`
	CreatedAt             time.Time          `json:"createdAt"`
	StartedAt             *time.Time         `json:"startedAt,omitempty"`
	VoteResults           map[string]int     `json:"voteResults,omitempty"`
	HunterProtection      string             `json:"hunterProtection,omitempty"`      // ID ของคนที่นายพรานกัน
	TigerTarget           string             `json:"tigerTarget,omitempty"`           // ID ของเหยื่อที่เสือเลือก
	ShamanVision          string             `json:"shamanVision,omitempty"`          // ID ของคนที่หมอผีส่อง
	KilledTonight         string             `json:"killedTonight,omitempty"`         // ID ของคนที่ตายคืนนี้
	CursedPlayer          string             `json:"cursedPlayer,omitempty"`          // ID ของคนที่ถูกสาป
	PhaseEndTime          *time.Time         `json:"phaseEndTime,omitempty"`          // เวลาสิ้นสุดเฟส
	NightActionsCompleted map[string]bool    `json:"nightActionsCompleted,omitempty"` // ผู้เล่นที่ใช้พลังหรือข้ามแล้วในคืนนี้
	NightActionsRequired  int                `json:"nightActionsRequired,omitempty"`  // จำนวนผู้เล่นที่ต้องใช้พลังในคืนนี้
	CurrentNightRole      Role               `json:"currentNightRole,omitempty"`      // Role ที่กำลัง action ในคืนนี้
	NightActionOrder      []Role             `json:"nightActionOrder,omitempty"`      // ลำดับการ action ในคืน
	WaitingHunterShoot    bool               `json:"waitingHunterShoot,omitempty"`    // รอนายพรานยิงหรือไม่
	DeadHunterID          string             `json:"deadHunterID,omitempty"`          // ID ของนายพรานที่ตายและรอยิง
	WinningTeam           string             `json:"winningTeam,omitempty"`           // "human" หรือ "tiger"
}

// Message represents a chat message
type Message struct {
	ID        string    `json:"id"`
	RoomCode  string    `json:"roomCode"`
	PlayerID  string    `json:"playerId"`
	Username  string    `json:"username"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"` // "chat", "system", "werewolf"
}

// WSMessage represents a WebSocket message
type WSMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// Event types
const (
	EventJoinRoom        = "join_room"
	EventLeaveRoom       = "leave_room"
	EventStartGame       = "start_game"
	EventPlayerJoined    = "player_joined"
	EventPlayerLeft      = "player_left"
	EventGameStarted     = "game_started"
	EventPhaseChanged    = "phase_changed"
	EventNightAction     = "night_action"
	EventSkipAction      = "skip_action" // ข้ามการใช้พลัง
	EventSkipPhase       = "skip_phase"  // ข้ามเฟส (host only)
	EventVote            = "vote"
	EventVoteUpdate      = "vote_update" // real-time vote update
	EventVoteResult      = "vote_result"
	EventPlayerDied      = "player_died"
	EventGameEnded       = "game_ended"
	EventChatMessage     = "chat_message"
	EventGameStateUpdate = "game_state_update"
	EventNightRoleChange = "night_role_change" // เปลี่ยน role ที่กำลัง action
	EventHunterShoot     = "hunter_shoot"      // นายพรานยิงเมื่อตาย
	EventCurseAction     = "curse_action"      // พญาสมิงสาป
	EventError           = "error"
)
