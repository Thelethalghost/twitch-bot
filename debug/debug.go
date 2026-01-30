package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		fmt.Println("⚠️  No .env file found")
	}

	// Get environment variables
	riotToken := os.Getenv("RIOT_TOKEN")
	summonerName := os.Getenv("SUMMONER_NAME")
	summonerTag := os.Getenv("SUMMONER_TAG")
	region := os.Getenv("RIOT_REGION")

	if region == "" {
		region = "americas"
	}

	fmt.Println("=== Environment Variables ===")
	fmt.Printf("RIOT_TOKEN: '%s'\n", riotToken)
	fmt.Printf("RIOT_TOKEN length: %d\n", len(riotToken))
	fmt.Printf("RIOT_TOKEN has spaces: %v\n", strings.Contains(riotToken, " "))
	fmt.Printf("RIOT_TOKEN starts with 'RGAPI-': %v\n", strings.HasPrefix(riotToken, "RGAPI-"))
	fmt.Printf("SUMMONER_NAME: '%s'\n", summonerName)
	fmt.Printf("SUMMONER_TAG: '%s'\n", summonerTag)
	fmt.Printf("RIOT_REGION: '%s'\n", region)
	fmt.Println()

	if riotToken == "" {
		fmt.Println("❌ RIOT_TOKEN is empty!")
		return
	}

	// Test API call
	url := fmt.Sprintf("https://%s.api.riotgames.com/riot/account/v1/accounts/by-riot-id/%s/%s",
		region, summonerName, summonerTag)

	fmt.Println("=== Testing API Call ===")
	fmt.Printf("URL: %s\n", url)
	fmt.Println()

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Printf("❌ Error creating request: %v\n", err)
		return
	}

	req.Header.Set("X-Riot-Token", riotToken)
	req.Header.Set("Accept", "application/json")

	fmt.Println("Request Headers:")
	fmt.Printf("  X-Riot-Token: '%s'\n", req.Header.Get("X-Riot-Token"))
	fmt.Printf("  Accept: '%s'\n", req.Header.Get("Accept"))
	fmt.Println()

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("❌ Error making request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	fmt.Printf("Response Status: %d\n", resp.StatusCode)
	fmt.Printf("Response Body: %s\n", string(body))
	fmt.Println()

	if resp.StatusCode == 200 {
		fmt.Println("✅ API key is working correctly!")
	} else if resp.StatusCode == 401 {
		fmt.Println("❌ 401 Unauthorized - Possible issues:")
		fmt.Println("   1. API key has expired (dev keys expire after 24 hours)")
		fmt.Println("   2. API key has extra spaces or invisible characters")
		fmt.Println("   3. API key is incorrect")
		fmt.Println("   4. Copy-paste issue from Riot Developer Portal")
		fmt.Println()
		fmt.Println("Solutions:")
		fmt.Println("   1. Get a fresh API key from https://developer.riotgames.com/")
		fmt.Println("   2. Copy ONLY the key (starts with RGAPI-)")
		fmt.Println("   3. Update .env with: RIOT_TOKEN=RGAPI-your-key-here")
		fmt.Println("   4. No quotes, no spaces before/after the equals sign")
	} else if resp.StatusCode == 403 {
		fmt.Println("❌ 403 Forbidden - API key is valid but lacks permissions")
	} else if resp.StatusCode == 404 {
		fmt.Println("❌ 404 Not Found - Check summoner name and tag")
	}
}
