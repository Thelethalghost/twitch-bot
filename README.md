# Twitch Chat Bot for League of Legends Players

A Twitch chat bot built in Go that automatically responds to chat commands with real-time League of Legends player statistics and stream information. Perfect for streamers who want to share their rank, stream stats, and match information with their viewers.

## Features

- **Stream Information Commands**: Display current stream title and game being played
- **Rank Display**: Show your current League of Legends rank, tier, and LP in chat
- **Live Stream Statistics**: Track and display wins, losses, winrate, and LP gains during your stream session
- **Active Match Information**: Show which champions are banned in your current game
- **Quick Static Responses**: Customizable welcome messages and help text
- **Anti-Spam Protection**: Built-in cooldown system to prevent command abuse
- **Automatic Token Management**: Handles Twitch OAuth token refresh automatically

## Available Chat Commands

Viewers can use these commands in your Twitch chat:

- `!hello` - Get a welcome message
- `!help` - See all available commands
- `!title` - Check what game you're streaming and the stream title
- `!elo` or `!rank` - See your current League rank and LP points
- `!stats` - View your performance during this stream (wins, losses, winrate, LP changes)
- `!bans` - See which champions are banned in your current match

## What You Need Before Installing

To run this bot, you'll need to set up credentials from three services:

1. **Twitch Developer Account**
   - Bot account username and OAuth token (chat access)
   - Client ID and Client Secret (API access)

2. **Riot API Key**
   - API key from the Riot Developer Portal
   - Your summoner name and tag (the #XXXX part)

3. **Go 1.25.1 or later**
   - Download from golang.org

## Installation & Setup

### Step 1: Clone and Install Dependencies
```bash
go mod download
```

### Step 2: Create Your `.env` File

Create a `.env` file in the project root directory with your credentials:
```env
# Twitch Bot Configuration
TWITCH_BOT_USERNAME=your_bot_account_name
TWITCH_OAUTH_TOKEN=oauth:your_oauth_token_here
TWITCH_CHANNEL=your_twitch_channel_name
TWITCH_CLIENT_ID=your_twitch_client_id
TWITCH_CLIENT_SECRET=your_twitch_client_secret

# League of Legends Configuration
RIOT_TOKEN=your_riot_api_token
RIOT_PLATFORM=na1
RIOT_REGION=americas
SUMMONER_NAME=YourSummonerName
SUMMONER_TAG=NA1
```

### Step 3: Run the Bot
```bash
go run main.go riot.go twitch.go
```

You should see output confirming the bot connected to Twitch IRC and loaded all commands.

## Customizing Commands

Commands are defined in `commands.json`. You can add, remove, or modify commands by editing this file.

### Static Command Example

A static command just returns the same response every time:
```json
{
  "!hello": {
    "type": "static",
    "response": "Hello there, welcome to the stream!",
    "cooldown": 2
  }
}
```

### API Command Example

An API command fetches live data and returns dynamic responses:
```json
{
  "!elo": {
    "type": "api",
    "endpoint": "riot_rank_info",
    "cooldown": 2
  }
}
```

Available API endpoints include:
- `twitch_stream_info` - Current stream title and game
- `riot_rank_info` - Your current rank and LP
- `stream_stats_info` - Session wins, losses, and winrate
- `current_bans_info` - Banned champions in active match

The `cooldown` value is in seconds—this prevents viewers from spamming commands.

## How It Works Behind the Scenes

1. Bot connects to Twitch IRC chat using your OAuth token
2. Monitors all chat messages for commands (starting with `!`)
3. Normalizes commands (converts to lowercase, removes special characters)
4. Checks if enough time has passed since the last use (cooldown)
5. Executes either a static response or fetches live data from APIs
6. Sends response to chat with a mention of the user who used the command

## Data Caching

The bot automatically caches data locally to reduce API calls:

- **`players.json`** - Stores your summoner PUUID and ID (so it doesn't have to look it up every time)
- **`champions.json`** - Maps champion IDs to names (used for the bans command)

These files are created automatically on first run.

## API Integrations

### Twitch Helix API
Fetches your stream status, title, and game information. The bot automatically refreshes authentication tokens every 50 minutes.

### Riot API
Retrieves your League of Legends data including:
- Current rank, tier, and LP
- Match history during stream session
- Active game information (banned champions)

## Troubleshooting

**"TWITCH_BOT_USERNAME not set" error**
- Make sure your `.env` file exists and has all required variables

**Bot doesn't respond to commands**
- Check that the bot account is actually in your channel
- Verify command names are exactly as typed (case-insensitive matching is built in)
- Check the cooldown hasn't triggered

**"Stream is offline" when using !stats**
- The stats command only works while you're actively streaming
- Make sure your stream is live when testing

