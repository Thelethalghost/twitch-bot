package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"sync"
	"time"

	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

var (
	messageCache   []string
	lastFetch      time.Time
	messageCacheMu sync.Mutex
)

// LoadMessages fetches messages from Google Sheet and caches them
func LoadMessages() error {
	messageCacheMu.Lock()
	defer messageCacheMu.Unlock()

	ctx := context.Background()
	srv, err := sheets.NewService(ctx, option.WithCredentialsFile("credentials.json"))
	if err != nil {
		return fmt.Errorf("unable to create sheets client: %v", err)
	}

	spreadsheetId := os.Getenv("GOOGLE_SHEET_ID")
	if spreadsheetId == "" {
		return fmt.Errorf("GOOGLE_SHEET_ID not set in .env")
	}

	// Only read column A (messages), skip header row
	resp, err := srv.Spreadsheets.Values.Get(spreadsheetId, "Sheet1!A2:A").Do()
	if err != nil {
		return fmt.Errorf("unable to retrieve data from sheet: %v", err)
	}

	messageCache = []string{}
	for _, row := range resp.Values {
		if len(row) >= 1 && row[0] != "" {
			// Just the message text
			message := fmt.Sprintf("%v", row[0])
			messageCache = append(messageCache, message)
		}
	}

	lastFetch = time.Now()
	log.Printf("Loaded %d messages from Google Sheets", len(messageCache))
	return nil
}

// GetRandomMessage returns a random message from cache
// Refreshes cache if older than 2 minutes
func GetRandomMessage() (string, error) {
	messageCacheMu.Lock()
	defer messageCacheMu.Unlock()

	// Refresh if cache is old
	if time.Since(lastFetch) > 2*time.Minute {
		messageCacheMu.Unlock()
		if err := LoadMessages(); err != nil {
			messageCacheMu.Lock()
			return "", err
		}
		messageCacheMu.Lock()
	}

	if len(messageCache) == 0 {
		return "No messages yet! Submit one using the form.", nil
	}

	randomIndex := rand.Intn(len(messageCache))
	return messageCache[randomIndex], nil
}

// StartMessageRefresher auto-refreshes messages every 2 minutes
func StartMessageRefresher() {
	// Initial load
	if err := LoadMessages(); err != nil {
		log.Printf("Warning: Failed to load messages initially: %v", err)
	}

	ticker := time.NewTicker(2 * time.Minute)
	go func() {
		for range ticker.C {
			if err := LoadMessages(); err != nil {
				log.Printf("Error refreshing messages: %v", err)
			}
		}
	}()
}
