package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand/v2"
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

func GetRandomDuckJoke() (string, error) {
	jokes := []string{
		"What did the duck say when he bought lipstick? Put it on my bill!",
		"What time do ducks wake up? At the quack of dawn!",
		"What do you call a clever duck? A wise quacker!",
		"Why did the duckling almost fall? She tripped on a quack!",
		"What is a duck’s favorite sea monster? The Quacken!",
		"Why are ducks bad drivers? Their windshields are always quacked!",
		"What do you call a duck that breaks into houses? A robber ducky!",
		"Why was the duck put into the basketball game? To make a fowl shot!",
		"What do you call a bird that can fix anything? Duck Tape!",
		"How do you get down off a horse? You don't, you get down off a duck!",
		"What's a duck's favorite taco topping? Quackamole!",
		"Why do ducks fly south for the winter? It's too far to waddle!",
		"What did the duck say to the waiter? Quack!",
		"What do you call a duck that loves fireworks? A fire-quacker!",
		"Where do ducks go when they are sick? To the duck-tor!",
		"What did the detective duck say to his partner? Let's quack this case!",
		"What do you call a duck with a soul? A quack-vocalist!",
		"Why did the duck cross the playground? To get to the other slide!",
		"What do you call a prehistoric duck? A Quack-a-dactyl!",
		"Why did the duck get an A on his test? He was a real egg-head!",
		"what is the ducks favourite scientific field? Kvaktum physics",
	}

	// Returns a random one from the 20 codes above
	return jokes[rand.IntN(len(jokes))], nil
}
