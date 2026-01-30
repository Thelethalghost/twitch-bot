package main

import (
	"fmt"
)

// Simple test file to verify custom game logic works
// Save as test_custom.go and run: go run test_custom.go custom_rank.go

func main() {
	fmt.Println("=== Testing Custom Game System ===\n")

	// Test 1: Start session
	fmt.Println("1. Starting custom session...")
	session := StartCustomSession()
	if session != nil && session.IsActive {
		fmt.Println("✅ Session started successfully")
	}

	// Test 2: Add players with different ranks
	fmt.Println("\n2. Adding players...")

	testPlayers := []struct {
		username string
		rank     string
	}{
		{"Player1", "gold2"},
		{"Player2", "plat"},
		{"Player3", "diamond"},
		{"Player4", "silver"},
		{"Player5", "gold"},
		{"Player6", "plat2"},
		{"Player7", "emerald"},
		{"Player8", "bronze"},
		{"Player9", "d1"},
		{"Player10", "master"},
	}

	for _, p := range testPlayers {
		count, err := AddPlayerToSession(p.username, p.rank)
		if err != nil {
			fmt.Printf("❌ %s (%s) failed: %s\n", p.username, p.rank, err.Error())
		} else {
			fmt.Printf("✅ %s (%s) joined - Total: %d/10\n", p.username, p.rank, count)
		}
	}

	// Test 3: Generate teams
	fmt.Println("\n3. Generating teams...")
	teams := GenerateTeams()
	fmt.Println(teams)

	// Test 4: Get session status
	fmt.Println("\n4. Session status after teams:")
	status := GetSessionStatus()
	fmt.Println(status)

	// Test 5: Try invalid rank
	fmt.Println("\n5. Testing invalid rank...")
	StartCustomSession()
	_, err := AddPlayerToSession("BadPlayer", "invalidrank123")
	if err != nil {
		fmt.Printf("✅ Correctly rejected invalid rank: %s\n", err.Error())
	}

	// Test 6: Try duplicate join
	fmt.Println("\n6. Testing duplicate join...")
	AddPlayerToSession("TestUser", "gold")
	_, err = AddPlayerToSession("TestUser", "plat")
	if err != nil {
		fmt.Printf("✅ Correctly rejected duplicate: %s\n", err.Error())
	}

	// Test 7: Session full test
	fmt.Println("\n7. Testing full session...")
	StartCustomSession()
	for i := 1; i <= 11; i++ {
		username := fmt.Sprintf("User%d", i)
		count, err := AddPlayerToSession(username, "gold")
		if err != nil {
			fmt.Printf("✅ Correctly rejected 11th player: %s\n", err.Error())
			break
		} else if i == 10 {
			fmt.Printf("✅ Session full at %d players\n", count)
		}
	}

	fmt.Println("\n=== All Tests Complete ===")
}
