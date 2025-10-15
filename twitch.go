package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

type StreamResponse struct {
	Data []struct {
		UserName string `json:"user_name"`
		Title    string `json:"title"`
		GameName string `json:"game_name"`
	} `json:"data"`
}

type AppTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

var TwitchAppToken string

// Refresh Twitch App Token
func RefreshAppToken() {
	clientID := os.Getenv("TWITCH_CLIENT_ID")
	clientSecret := os.Getenv("TWITCH_CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		log.Fatal("TWITCH_CLIENT_ID or TWITCH_CLIENT_SECRET not set")
	}

	url := fmt.Sprintf(
		"https://id.twitch.tv/oauth2/token?client_id=%s&client_secret=%s&grant_type=client_credentials",
		clientID, clientSecret,
	)

	res, err := http.Post(url, "application/json", nil)
	if err != nil {
		log.Println("Error refreshing Twitch App Token:", err)
		return
	}
	defer res.Body.Close()

	var tokenResp AppTokenResponse
	if err := json.NewDecoder(res.Body).Decode(&tokenResp); err != nil {
		log.Println("Error decoding Twitch App Token:", err)
		return
	}

	TwitchAppToken = tokenResp.AccessToken
	log.Println("Twitch App Token refreshed successfully!")
}

// Start automatic token refresh
func StartAppTokenRefresher() {
	RefreshAppToken() // initial refresh
	ticker := time.NewTicker(50 * time.Minute)
	go func() {
		for range ticker.C {
			RefreshAppToken()
		}
	}()
}

// Get stream info (title + game)
func GetTwitchStreamInfo(channel string) (string, string, error) {
	clientID := os.Getenv("TWITCH_CLIENT_ID")
	if clientID == "" || TwitchAppToken == "" {
		return "", "", fmt.Errorf("Twitch App Token not set")
	}

	req, _ := http.NewRequest("GET",
		"https://api.twitch.tv/helix/streams?user_login="+channel, nil)
	req.Header.Set("Client-Id", clientID)
	req.Header.Set("Authorization", "Bearer "+TwitchAppToken)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer res.Body.Close()

	var stream StreamResponse
	if err := json.NewDecoder(res.Body).Decode(&stream); err != nil {
		return "", "", err
	}

	if len(stream.Data) == 0 {
		return "Offline", "", nil
	}

	return stream.Data[0].Title, stream.Data[0].GameName, nil
}

func GetTwitchStreamStart(channel string) (int64, error) {
	clientID := os.Getenv("TWITCH_CLIENT_ID")
	if clientID == "" || TwitchAppToken == "" {
		return int64(0), fmt.Errorf("Twitch App Token not set")
	}
	url := fmt.Sprintf("https://api.twitch.tv/helix/streams?user_login=%s", channel)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Client-ID", clientID)
	req.Header.Set("Authorization", "Bearer "+TwitchAppToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	var res struct {
		Data []struct {
			StartedAt string `json:"started_at"`
		} `json:"data"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&res)
	if len(res.Data) == 0 {
		return 0, fmt.Errorf("stream not live")
	}

	t, _ := time.Parse(time.RFC3339, res.Data[0].StartedAt)
	return t.Unix(), nil
}
