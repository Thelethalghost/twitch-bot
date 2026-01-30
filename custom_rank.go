package main

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type CustomGameSession struct {
	Players    []PlayerEntry
	StartTime  time.Time
	IsActive   bool
	MaxPlayers int
	mu         sync.Mutex
}

type PlayerEntry struct {
	Username  string
	RankInput string // What they typed (e.g., "gold2", "plat4")
	RankValue float64
}

var (
	activeSession      *CustomGameSession
	activeSessionLock  sync.Mutex
	lastGeneratedTeams string // Store the last generated teams
	lastTeamsLock      sync.Mutex
)

// Rank values matching League of Flex system
var rankMap = map[string]float64{
	"iron":            1,
	"iron1":           1,
	"iron2":           1,
	"iron3":           1,
	"iron4":           1,
	"ironbronze":      2,
	"bronze":          3,
	"bronze1":         3,
	"bronze2":         3,
	"bronze3":         3,
	"bronze4":         3,
	"bronzesilver":    4,
	"silver":          5,
	"silver1":         5,
	"silver2":         5,
	"silver3":         5,
	"silver4":         5,
	"silvergold":      6.5,
	"gold":            8,
	"gold1":           8,
	"gold2":           8,
	"gold3":           8,
	"gold4":           8,
	"goldplatinum":    9.5,
	"goldplat":        9.5,
	"platinum":        11,
	"plat":            11,
	"plat1":           11,
	"plat2":           11,
	"plat3":           11,
	"plat4":           11,
	"platinum1":       11,
	"platinum2":       11,
	"platinum3":       11,
	"platinum4":       11,
	"platinumemerald": 13,
	"platemerald":     13,
	"emerald":         15,
	"em":              15,
	"em1":             15,
	"em2":             15,
	"em3":             15,
	"em4":             15,
	"emerald1":        15,
	"emerald2":        15,
	"emerald3":        15,
	"emerald4":        15,
	"emeralddiamond":  17,
	"emdiamond":       17,
	"diamond":         19,
	"dia":             19,
	"d":               19,
	"d1":              19,
	"d2":              19,
	"d3":              19,
	"d4":              19,
	"diamond1":        19,
	"diamond2":        19,
	"diamond3":        19,
	"diamond4":        19,
	"diamondmaster":   21.5,
	"diamaster":       21.5,
	"master":          24,
	"m":               24,
	"grandmaster":     27,
	"gm":              27,
	"challenger":      30,
	"chall":           30,
	"c":               30,
}

// Parse rank input (e.g., "gold2", "plat", "d1")
func parseRank(input string) (float64, bool) {
	// Normalize input
	input = strings.ToLower(strings.TrimSpace(input))
	input = strings.ReplaceAll(input, " ", "")
	input = strings.ReplaceAll(input, "-", "")

	if val, ok := rankMap[input]; ok {
		return val, true
	}
	return 0, false
}

// Start a new custom game session

func StartCustomSession() *CustomGameSession {
	activeSessionLock.Lock()
	defer activeSessionLock.Unlock()

	// Close any existing session
	if activeSession != nil && activeSession.IsActive {
		activeSession.IsActive = false
		fmt.Println("🔄 Closed previous session")
	}

	activeSession = &CustomGameSession{
		Players:    make([]PlayerEntry, 0, 10),
		StartTime:  time.Now(),
		IsActive:   true,
		MaxPlayers: 10,
	}

	fmt.Printf("✅✅✅ NEW SESSION CREATED - Active: %v, Time: %v\n",
		activeSession.IsActive, activeSession.StartTime)

	// Auto-close after 5 minutes
	sessionToClose := activeSession // Capture the current session
	go func() {
		time.Sleep(60 * time.Minute)
		activeSessionLock.Lock()
		defer activeSessionLock.Unlock()

		// Only close if it's still the same session
		if activeSession == sessionToClose && activeSession != nil {
			activeSession.IsActive = false
			fmt.Println("⏰ Session auto-closed after 5 minutes")
		}
	}()

	return activeSession
}

// Add player to active session
func AddPlayerToSession(username, rankInput string) (int, error) {
	activeSessionLock.Lock()
	defer activeSessionLock.Unlock()

	fmt.Printf("\n=== ADD PLAYER DEBUG ===\n")
	fmt.Printf("Username: %s\n", username)
	fmt.Printf("Rank Input: %s\n", rankInput)
	fmt.Printf("activeSession == nil: %v\n", activeSession == nil)

	if activeSession != nil {
		fmt.Printf("activeSession.IsActive: %v\n", activeSession.IsActive)
		fmt.Printf("Current players: %d\n", len(activeSession.Players))
		fmt.Printf("Start time: %v\n", activeSession.StartTime)
		fmt.Printf("Time since start: %v\n", time.Since(activeSession.StartTime))
	}
	fmt.Printf("======================\n\n")

	if activeSession == nil || !activeSession.IsActive {
		return 0, fmt.Errorf("no active session")
	}

	activeSession.mu.Lock()
	defer activeSession.mu.Unlock()

	// Check if player already joined
	for _, p := range activeSession.Players {
		if p.Username == username {
			return 0, fmt.Errorf("already joined")
		}
	}

	// Check if session is full
	if len(activeSession.Players) >= activeSession.MaxPlayers {
		return 0, fmt.Errorf("session full")
	}

	// Parse rank
	rankValue, ok := parseRank(rankInput)
	if !ok {
		return 0, fmt.Errorf("invalid rank (examples: gold, plat2, d1)")
	}

	player := PlayerEntry{
		Username:  username,
		RankInput: rankInput,
		RankValue: rankValue,
	}

	activeSession.Players = append(activeSession.Players, player)
	fmt.Printf("✅✅✅ PLAYER ADDED: %s (%s) - Total: %d/10\n\n",
		username, rankInput, len(activeSession.Players))

	return len(activeSession.Players), nil
}

// Balance teams using greedy algorithm (same as League of Flex)
func BalanceTeams(players []PlayerEntry) ([]PlayerEntry, []PlayerEntry) {
	// Sort players by rank value (highest first)
	sort.Slice(players, func(i, j int) bool {
		return players[i].RankValue > players[j].RankValue
	})

	team1 := []PlayerEntry{}
	team2 := []PlayerEntry{}
	team1Total := 0.0
	team2Total := 0.0

	// Greedy assignment: assign each player to the team with lower total
	for _, player := range players {
		if team1Total <= team2Total {
			team1 = append(team1, player)
			team1Total += player.RankValue
		} else {
			team2 = append(team2, player)
			team2Total += player.RankValue
		}
	}

	return team1, team2
}

// Generate teams and return formatted message
func GenerateTeams() string {
	activeSessionLock.Lock()
	defer activeSessionLock.Unlock()

	if activeSession == nil || !activeSession.IsActive {
		msg := "No active session. Use !custom start to begin."
		fmt.Printf("⚠️ GenerateTeams called but: %s\n", msg)
		return msg
	}

	activeSession.mu.Lock()
	defer activeSession.mu.Unlock()

	if len(activeSession.Players) < 2 {
		msg := "Not enough players (need at least 2)"
		fmt.Printf("⚠️ GenerateTeams: %s\n", msg)
		return msg
	}

	team1, team2 := BalanceTeams(activeSession.Players)

	// Calculate team averages
	team1Avg := 0.0
	for _, p := range team1 {
		team1Avg += p.RankValue
	}
	team1Avg /= float64(len(team1))

	team2Avg := 0.0
	for _, p := range team2 {
		team2Avg += p.RankValue
	}
	team2Avg /= float64(len(team2))

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Teams Generated (%d players) | ", len(activeSession.Players)))

	// Team 1
	sb.WriteString(fmt.Sprintf("Team 1 (avg %.1f): ", team1Avg))
	for i, p := range team1 {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(fmt.Sprintf("%s(%s)", p.Username, p.RankInput))
	}

	sb.WriteString(fmt.Sprintf(" | Team 2 (avg %.1f): ", team2Avg))
	for i, p := range team2 {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(fmt.Sprintf("%s(%s)", p.Username, p.RankInput))
	}

	result := sb.String()

	// ✅ Store the teams
	lastTeamsLock.Lock()
	lastGeneratedTeams = result
	lastTeamsLock.Unlock()

	fmt.Printf("🎮 Teams Generated:\n%s\n", result)
	fmt.Printf("📏 Message length: %d characters\n", len(result))

	return result
}

// Close the active session
func CloseCustomSession() {
	activeSessionLock.Lock()
	defer activeSessionLock.Unlock()

	if activeSession != nil {
		activeSession.IsActive = false
		fmt.Println("❌❌❌ SESSION CLOSED BY CloseCustomSession()")

		// Print stack trace to see WHO called this
		// This helps debug unexpected closes
	}
}

// Get current session status
func GetSessionStatus() string {
	activeSessionLock.Lock()
	defer activeSessionLock.Unlock()

	if activeSession == nil || !activeSession.IsActive {
		return "No active session. Use !custom start to begin."
	}

	activeSession.mu.Lock()
	defer activeSession.mu.Unlock()

	return fmt.Sprintf("Custom game active | %d/%d players joined",
		len(activeSession.Players), activeSession.MaxPlayers)
}

func GetLastTeams() string {
	lastTeamsLock.Lock()
	defer lastTeamsLock.Unlock()

	if lastGeneratedTeams == "" {
		return "No teams generated yet. Use !custom teams after players join."
	}
	return lastGeneratedTeams
}
