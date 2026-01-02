package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type CommandConfig struct {
	Type     string `json:"type"`
	Response string `json:"response,omitempty"`
	Endpoint string `json:"endpoint,omitempty"`
	Cooldown int    `json:"cooldown"`
}

type Joke struct {
	ID   string `json:"id"`
	Joke string `json:"joke"`
}

func getDadJoke() (Joke, error) {
	req, err := http.NewRequest("GET", "https://icanhazdadjoke.com/", nil)
	if err != nil {
		return Joke{}, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Joke{}, err
	}
	defer resp.Body.Close()

	var j Joke
	if err := json.NewDecoder(resp.Body).Decode(&j); err != nil {
		return Joke{}, err
	}

	return j, nil
}

func loadCommands(path string) map[string]CommandConfig {
	file, err := os.ReadFile(path)
	if err != nil {
		log.Fatal("Error reading commands.json:", err)
	}

	var commands map[string]CommandConfig
	if err := json.Unmarshal(file, &commands); err != nil {
		log.Fatal("Error parsing commands.json:", err)
	}

	normalizedCommands := make(map[string]CommandConfig)
	for k, v := range commands {
		// lowercase + trim spaces + remove non-ASCII characters
		cleanKey := strings.ToLower(strings.TrimSpace(k))
		cleanKey = strings.Map(func(r rune) rune {
			if r > 127 { // remove non-ASCII
				return -1
			}
			return r
		}, cleanKey)
		normalizedCommands[cleanKey] = v
	}

	fmt.Println("Loaded commands:")
	for k := range normalizedCommands {
		fmt.Printf("[%q]\n", k)
	}

	commands = normalizedCommands

	return commands
}

func say(conn net.Conn, channel, msg string) {
	fmt.Fprintf(conn, "PRIVMSG #%s :%s\r\n", channel, msg)
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on system env vars")
	}
	rand.Seed(time.Now().UnixNano())

	username := os.Getenv("TWITCH_BOT_USERNAME")
	oauth := os.Getenv("TWITCH_OAUTH_TOKEN")
	channel := os.Getenv("TWITCH_CHANNEL")
	summoner := os.Getenv("SUMMONER_NAME")
	tag := os.Getenv("SUMMONER_TAG")

	if username == "" || oauth == "" || channel == "" || summoner == "" {
		log.Fatal("Set TWITCH_BOT_USERNAME, TWITCH_OAUTH_TOKEN, TWITCH_CHANNEL, SUMMONER_NAME")
	}

	puuid, err := GetOrCachePlayer(summoner, tag)
	if err != nil {
		log.Fatalf("Error fetching player: %v", err)
	}

	commands := loadCommands("commands.json")
	lastUsed := make(map[string]time.Time)

	StartAppTokenRefresher()
	LoadChampionMap()
	StartMessageRefresher()

	conn, err := net.Dial("tcp", "irc.chat.twitch.tv:6667")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	fmt.Fprintf(conn, "PASS %s\r\n", oauth)
	fmt.Fprintf(conn, "NICK %s\r\n", username)
	fmt.Fprintf(conn, "JOIN #%s\r\n", channel)

	log.Println("Connected to Twitch IRC as", username)

	reader := bufio.NewReader(conn)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			log.Println("Read error:", err)
			return
		}
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "PING") {
			fmt.Fprintf(conn, "PONG :tmi.twitch.tv\r\n")
			continue
		}

		if strings.Contains(line, "PRIVMSG") {
			parts := strings.Split(line, "PRIVMSG")
			if len(parts) < 2 {
				continue
			}
			rawUser := strings.Split(parts[0], "!")[0]
			user := strings.TrimPrefix(rawUser, ":")
			msg := strings.SplitN(parts[1], ":", 2)[1]
			command := strings.ToLower(strings.TrimSpace(msg))
			command = strings.Map(func(r rune) rune {
				if r > 127 { // remove non-ASCII
					return -1
				}
				return r
			}, command)
			cfg, ok := commands[command]
			fmt.Printf("Received: [%q]\n", msg)
			if !ok {
				fmt.Println("User:", user, "Message:", msg, "Command key found:", ok)
				continue
			}

			if t, ok := lastUsed[command]; ok {
				if time.Since(t) < time.Duration(cfg.Cooldown)*time.Second {
					continue
				}
			}

			switch cfg.Type {
			case "static":
				say(conn, channel, fmt.Sprintf("@%s %s", user, cfg.Response))
			case "api":
				switch cfg.Endpoint {
				case "twitch_stream_info":
					title, game, err := GetTwitchStreamInfo(channel)
					if err != nil {
						say(conn, channel, fmt.Sprintf("@%s Error fetching stream info.", user))
					} else if title == "Offline" {
						say(conn, channel, fmt.Sprintf("@%s Stream is offline.", user))
					} else {
						say(conn, channel, fmt.Sprintf("@%s Title: %s | Game: %s", user, title, game))
					}
				case "riot_rank_info":
					rank, err := GetCurrentRank(puuid)
					if err != nil {
						log.Printf("Rank error: %v", err)
					}
					say(conn, channel, fmt.Sprintf("@%s Current Rank: %s %s %d", user, rank[0].Tier, rank[0].Rank, rank[0].LeaguePoints))
				case "stream_stats_info":
					start, err := GetTwitchStreamStart(channel)
					if err != nil {
						say(conn, channel, fmt.Sprintf("@%s Error fetching stream info.", user))
					}
					stats, err := GetStreamStats(puuid, start)
					if err != nil {
						say(conn, channel, fmt.Sprintf("@%s Error Fetching stream stats.", user))
					} else {
						say(conn, channel, fmt.Sprintf("@%s Wins: %d | Loss: %d | Winrate: %.2f%% ", user, stats.Wins, stats.Losses, stats.Winrate))
					}
				case "current_bans_info":
					bans, err := GetActiveMatchBans(puuid)
					if err != nil {
						say(conn, channel, fmt.Sprintf("@%s Not in an Active Match", user))
					} else {
						banString := strings.Join(bans, ", ")
						say(conn, channel, fmt.Sprintf("@%s Banned Champions: %s", user, banString))
					}
				case "random_magnet_facts":
					moreMagnetFacts := []string{
						"MRI machines use powerful superconducting magnets to scan the human body.",
						"The strongest magnetic field ever created in a lab exceeded 1,200 teslas—over a million times stronger than Earth's field.",
						"Magnets can lose their magnetism over time, especially if dropped or heated.",
						"Jupiter has the strongest magnetic field of any planet in our solar system.",
						"There are three main types of magnets: permanent, temporary, and electromagnets.",
						"Even some bacteria contain magnetic particles and align themselves with Earth's magnetic field.",
						"Electromagnets only work when electric current flows through them.",
						"A magnet’s strength is measured in units called gauss or tesla.",
						"Refrigerator magnets are usually made from a flexible rubber-plastic material filled with ferrite powder.",
						"Sunspots are areas of intense magnetic activity on the Sun’s surface.",
						"Magnets always have two poles: a north and a south. You can't isolate just one pole.",
						"The Earth's core acts like a giant magnet, creating the planet’s magnetic field.",
						"Magnetic fields can pass through most materials, including glass, plastic, and even your hand.",
						"Lodestone is a naturally occurring magnetic mineral that ancient people used to make the first compasses.",
						"Strong neodymium magnets are made from an alloy of neodymium, iron, and boron.",
						"Magnetic levitation trains use powerful magnets to float above tracks, reducing friction.",
						"Some animals like pigeons and sea turtles can sense Earth’s magnetic field to navigate.",
						"Heat can weaken a magnet’s strength, and extremely high temperatures can demagnetize it completely.",
						"Opposite magnetic poles attract each other, while like poles repel.",
						"Magnetic hard drives store data by magnetizing tiny sections to represent bits.",
					}
					randomIndex := rand.Intn(len(moreMagnetFacts))
					say(conn, channel, fmt.Sprintf("nawi %s", moreMagnetFacts[randomIndex]))
				case "random_dad_joke":
					joke, err := getDadJoke()
					if err != nil {
						log.Fatal(err)
					}
					say(conn, channel, fmt.Sprintf("@nawi %s", joke.Joke))
				case "random_message":
					msg, err := GetRandomMessage()
					if err != nil {
						say(conn, channel, fmt.Sprintf("@%s Error fetching messages.", user))
						log.Printf("Message fetch error: %v", err)
					} else {
						say(conn, channel, fmt.Sprintf("@%s %s", user, msg))
					}
				}
			}

			lastUsed[command] = time.Now()
		}
	}
}
