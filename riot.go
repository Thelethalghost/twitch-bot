package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

// ---------- Config & Globals ----------
var (
	once               sync.Once
	riotToken          string
	platformStr        string
	regionStr          string
	httpClient         *http.Client
	playerCacheFile    = "players.json"
	playerCacheLock    sync.Mutex
	championsCacheFile = "champions.json"
	championsMu        sync.Mutex
	championsMap       map[int]string
	streamCache        = map[string]StreamStatsCacheEntry{}
	streamCacheMu      sync.Mutex
)

// ---------- Types ----------
type PlayerCacheEntry struct {
	GameName   string `json:"gameName"`
	TagLine    string `json:"tagLine"`
	PUUID      string `json:"puuid"`
	SummonerID string `json:"summonerId"`
	CachedAt   int64  `json:"cachedAt"`
}

type PlayerCache map[string]PlayerCacheEntry

type LeagueEntry struct {
	QueueType    string `json:"queueType"`
	Tier         string `json:"tier"`
	Rank         string `json:"rank"`
	LeaguePoints int    `json:"leaguePoints"`
	Wins         int    `json:"wins"`
	Losses       int    `json:"losses"`
}

type spectatorResponse struct {
	BannedChampions []struct {
		ChampionID int `json:"championId"`
		PickTurn   int `json:"pickTurn"`
		TeamID     int `json:"teamId"`
	} `json:"bannedChampions"`
}

type summonerV4Resp struct {
	ID    string `json:"id"`
	Puuid string `json:"puuid"`
	Name  string `json:"name"`
}

type StreamStatsCacheEntry struct {
	Wins     int
	Losses   int
	Winrate  float64
	LPStart  map[string]int
	LPEnd    map[string]int
	CachedAt int64
}

// ---------- Initialization ----------
func initEnv() {
	once.Do(func() {
		_ = godotenv.Load()
		riotToken = os.Getenv("RIOT_TOKEN")
		platformStr = os.Getenv("RIOT_PLATFORM")
		regionStr = os.Getenv("RIOT_REGION")
		if platformStr == "" {
			platformStr = "na1"
		}
		if regionStr == "" {
			regionStr = "americas"
		}
		httpClient = &http.Client{Timeout: 15 * time.Second}
	})
}

// ---------- Networking ----------
func makeRequest(hostType string, path string) ([]byte, error) {
	initEnv()
	if riotToken == "" {
		return nil, errors.New("RIOT_TOKEN not set")
	}

	var host string
	switch hostType {
	case "platform":
		host = platformStr + ".api.riotgames.com"
	case "regional":
		host = regionStr + ".api.riotgames.com"
	default:
		return nil, fmt.Errorf("invalid hostType: %s", hostType)
	}
	url := fmt.Sprintf("https://%s%s", host, path)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("X-Riot-Token", riotToken)
	req.Header.Set("Accept", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("request failed %d: %s", resp.StatusCode, string(b))
	}
	return b, nil
}

// ---------- Player caching ----------
func GetOrCachePlayer(gameName, tagLine string) (puuid string, err error) {
	playerCacheLock.Lock()
	defer playerCacheLock.Unlock()

	cache := PlayerCache{}
	if _, err := os.Stat(playerCacheFile); err == nil {
		data, _ := os.ReadFile(playerCacheFile)
		_ = json.Unmarshal(data, &cache)
	}

	key := fmt.Sprintf("%s#%s", gameName, tagLine)
	if p, ok := cache[key]; ok {
		return p.PUUID, nil
	}

	// Use Account V1 endpoint instead of Summoner V4
	path := fmt.Sprintf("/riot/account/v1/accounts/by-riot-id/%s/%s", urlEscape(gameName), urlEscape(tagLine))
	data, err := makeRequest("regional", path) // Use "regional" not "platform"
	if err != nil {
		return "", err
	}

	var accountResp struct {
		PUUID    string `json:"puuid"`
		GameName string `json:"gameName"`
		TagLine  string `json:"tagLine"`
	}
	if err := json.Unmarshal(data, &accountResp); err != nil {
		return "", err
	}

	// Now get summoner ID using PUUID
	summonerPath := fmt.Sprintf("/lol/summoner/v4/summoners/by-puuid/%s", accountResp.PUUID)
	summonerData, err := makeRequest("platform", summonerPath)
	if err != nil {
		return "", err
	}

	var s summonerV4Resp
	if err := json.Unmarshal(summonerData, &s); err != nil {
		return "", err
	}

	cache[key] = PlayerCacheEntry{
		GameName:   accountResp.GameName,
		TagLine:    accountResp.TagLine,
		PUUID:      accountResp.PUUID,
		SummonerID: s.ID,
		CachedAt:   time.Now().Unix(),
	}
	b, _ := json.MarshalIndent(cache, "", "  ")
	_ = os.WriteFile(playerCacheFile, b, 0644)
	return accountResp.PUUID, nil
}

// ---------- Champion cache ----------
// ---------- Champion cache ----------
func LoadChampionMap() error {
	championsMu.Lock()
	defer championsMu.Unlock()

	if championsMap != nil {
		return nil // Already loaded
	}

	// Load from static file
	data, err := os.ReadFile(championsCacheFile)
	if err != nil {
		return fmt.Errorf("failed to read champions.json: %w", err)
	}

	// Parse JSON where keys are string IDs
	var championsStrMap map[string]string
	if err := json.Unmarshal(data, &championsStrMap); err != nil {
		return fmt.Errorf("failed to parse champions.json: %w", err)
	}

	// Convert string keys to int keys
	championsMap = make(map[int]string)
	for idStr, name := range championsStrMap {
		id, err := strconv.Atoi(idStr)
		if err != nil {
			continue // Skip invalid entries
		}
		championsMap[id] = name
	}

	log.Printf("Loaded %d champions from %s", len(championsMap), championsCacheFile)
	return nil
}

func GetChampionName(id int) string {
	championsMu.Lock()
	defer championsMu.Unlock()

	if championsMap == nil {
		if err := LoadChampionMap(); err != nil {
			log.Printf("Error loading champions: %v", err)
			return fmt.Sprintf("Unknown(%d)", id)
		}
	}

	if name, ok := championsMap[id]; ok {
		return name
	}
	return fmt.Sprintf("Unknown(%d)", id)
}

// ---------- Current rank ----------
func GetCurrentRank(puuid string) ([]LeagueEntry, error) {
	path := fmt.Sprintf("/lol/league/v4/entries/by-puuid/%s", puuid)
	data, err := makeRequest("platform", path)
	if err != nil {
		return nil, err
	}
	var ranks []LeagueEntry
	_ = json.Unmarshal(data, &ranks)
	return ranks, nil
}

// ---------- Active game bans ----------
func GetActiveMatchBans(puuid string) ([]string, error) {
	path := fmt.Sprintf("/lol/spectator/v5/active-games/by-summoner/%s", puuid)
	data, err := makeRequest("platform", path)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return []string{}, nil
		}
		return nil, err
	}
	var resp spectatorResponse
	_ = json.Unmarshal(data, &resp)
	bans := []string{}
	for _, b := range resp.BannedChampions {
		bans = append(bans, GetChampionName(b.ChampionID))
	}
	return bans, nil
}

// ---------- Stream stats ----------
func GetStreamStats(puuid string, startTime int64) (StreamStatsCacheEntry, error) {
	// End time is always now
	endTime := time.Now().Unix()
	key := fmt.Sprintf("%s_%d", puuid, startTime)

	streamCacheMu.Lock()
	if val, ok := streamCache[key]; ok {
		streamCacheMu.Unlock()
		return val, nil
	}
	streamCacheMu.Unlock()

	path := fmt.Sprintf("/lol/match/v5/matches/by-puuid/%s/ids?startTime=%d&endTime=%d", puuid, startTime, endTime)
	data, err := makeRequest("regional", path)
	if err != nil {
		return StreamStatsCacheEntry{}, err
	}
	var matchIDs []string
	_ = json.Unmarshal(data, &matchIDs)

	wins, losses := 0, 0
	for _, matchID := range matchIDs {
		matchPath := fmt.Sprintf("/lol/match/v5/matches/%s", matchID)
		matchData, err := makeRequest("regional", matchPath)
		if err != nil {
			continue
		}
		var matchJSON map[string]interface{}
		_ = json.Unmarshal(matchData, &matchJSON)
		info, ok := matchJSON["info"].(map[string]interface{})
		if !ok {
			continue
		}
		participants, ok := info["participants"].([]interface{})
		if !ok {
			continue
		}
		for _, p := range participants {
			participant, ok := p.(map[string]interface{})
			if !ok {
				continue
			}
			if participant["puuid"] == puuid {
				if win, ok := participant["win"].(bool); ok && win {
					wins++
				} else {
					losses++
				}
				break
			}
		}
	}

	total := wins + losses
	winrate := 0.0
	if total > 0 {
		winrate = float64(wins) / float64(total) * 100
	}

	ranks, _ := GetCurrentRank(puuid)
	LPStart := map[string]int{}
	LPEnd := map[string]int{}
	for _, r := range ranks {
		LPStart[r.QueueType] = r.LeaguePoints - (wins - losses) // approx start LP
		LPEnd[r.QueueType] = r.LeaguePoints
	}

	entry := StreamStatsCacheEntry{
		Wins:     wins,
		Losses:   losses,
		Winrate:  winrate,
		LPStart:  LPStart,
		LPEnd:    LPEnd,
		CachedAt: time.Now().Unix(),
	}

	streamCacheMu.Lock()
	streamCache[key] = entry
	streamCacheMu.Unlock()

	return entry, nil
}

// ---------- Helpers ----------
func urlEscape(s string) string {
	return strings.ReplaceAll(s, " ", "%20")
}
