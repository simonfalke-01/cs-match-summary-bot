# CS Match Summary Bot

A Discord bot with webhook functionality for CS match summaries, written in Go. Features automatic Steam API polling, slash commands, and comprehensive data models for managing Discord guilds, Steam users, and CS match data with PostgreSQL integration.

## Setup

1. Install dependencies:
```bash
go mod tidy
```

2. Set up PostgreSQL database and copy the environment file:
```bash
cp .env.example .env
```

3. Edit `.env` with your configuration:
```
DISCORD_BOT_TOKEN=your_bot_token_here
STEAM_API_KEY=your_steam_api_key_here
WEBHOOK_HOST=localhost
WEBHOOK_PORT=8080
WEBHOOK_BASE_URL=https://cs-bot.simonfalke.com
DEMO_PARSE_BASE_URL=https://cs-demo-parsing.simonfalke.com
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=cs
```

4. Run database migrations:
```bash
go build -o migrate cmd/migrate.go
./migrate
```

5. Run the bot:
```bash
go run main.go
```

## Features

### Discord Bot
- **Auto Guild Registration**: Automatically registers when added to new Discord servers
- **Slash Commands**: Modern Discord slash commands for user interaction
- **Steam API Polling**: Automatically detects new matches every 10 seconds
- **Permission System**: Admin commands require proper Discord permissions
- **Smart Channel Detection**: Automatically finds suitable channels for notifications
- **Startup Recovery**: Registers existing guilds when bot restarts

### Webhooks & API
- **Demo Processing**: `/webhooks/demoReady` and `/webhooks/demoParsed` endpoints
- **REST API**: Query endpoints for matches, users, and guilds
- **Discord Integration**: Rich embed notifications when matches are processed
- **External Service Integration**: Connects to demo parsing services
- **Error Handling**: Robust error handling with meaningful responses
- **Configurable endpoints**: Via environment variables

### Data Models
- **Guild Management**: Store Discord guild information with associated users and games
- **User Management**: Track Steam users with authentication codes and game history
- **Game Management**: Store CS match data with share codes, demo files, and player information
- **PostgreSQL Integration**: Robust database schema with UUID primary keys and JSONB arrays

## Configuration

The bot uses environment variables for configuration:

- `DISCORD_BOT_TOKEN` - Your Discord bot token (required)
- `STEAM_API_KEY` - Your Steam API key for polling (required)
- `WEBHOOK_HOST` - Host for webhook server (default: localhost)
- `WEBHOOK_PORT` - Port for webhook server (default: 8080)
- `WEBHOOK_BASE_URL` - Base URL for webhook callbacks (default: https://cs-bot.simonfalke.com)
- `DEMO_PARSE_BASE_URL` - Base URL for demo parsing service (default: https://cs-demo-parsing.simonfalke.com)
- `DB_HOST` - Database host (default: localhost)
- `DB_PORT` - Database port (default: 5432)
- `DB_USER` - Database username (default: postgres)
- `DB_PASSWORD` - Database password (default: postgres)
- `DB_NAME` - Database name (default: cs)

## Project Structure

```
cs-match-summary-bot/
├── webhooks/           # Webhook server package
│   └── server.go      # HTTP server and handlers
├── cmd/               # Command line tools
│   └── migrate.go     # Database migration tool
├── main.go            # Main application entry point
├── db.go              # Database connection management
├── models.go          # Data model definitions
├── database.go        # Database operations (CRUD)
├── slash_commands.go  # Discord slash command handlers
├── steam_poller.go    # Steam API polling system
├── webhook_handlers.go # Webhook processing
├── guild_manager.go   # Guild management functions
├── examples.go        # Usage examples
├── DATA_MODELS.md     # Detailed documentation
├── FEATURES.md        # Complete feature documentation
├── .env.example       # Example environment configuration
└── README.md          # This file
```

## Data Models

The bot includes three main data models:

### Guild
- Represents Discord servers with associated users and games
- Fields: UUID, guild_id, channel_id, user_ids[], game_ids[]

### User  
- Represents Steam users with authentication and game history
- Fields: UUID, steam_id, auth_code, last_share_code, game_ids[]

### Game
- Represents CS matches with demo files and player data
- Fields: UUID, share_code, demo_name, steam_ids[]

## Database Operations

Comprehensive CRUD operations are available for all models:

```go
// Create entities
guild, err := createGuild("discord_guild_id", "discord_channel_id")
user, err := createUser("steam_id", "auth_code", "last_share_code")
game, err := createGame("share_code", "demo.dem", []string{"steam_id1", "steam_id2"})

// Link entities
err = addUserToGuild("guild_id", userUUID)
err = addGameToGuild("guild_id", gameUUID)
err = addGameToUser("steam_id", gameUUID)

// Query data
games, err := getGamesBySteamID("steam_id")
guildGames, err := getGamesForGuild("guild_id")
```

See `DATA_MODELS.md` for complete documentation and `examples.go` for usage examples.

## Discord Commands

The bot provides modern Discord slash commands for managing matches and users:

### Slash Commands
```
/register                   # Register with Steam ID, auth code, and last share code
/remove                     # Remove a user from the system
/users                      # Show list of registered users in the guild
/set_channel               # Set notification channel (Admin only)
```

### Legacy Text Commands (Still Available)
```
!cs help          # Show command help
!cs ping          # Test bot responsiveness
```

## Webhook Integration

### Demo Processing Webhooks

Process demos automatically via HTTP webhooks:

**Demo Ready:**
```bash
POST /webhooks/demoReady
Content-Type: application/json

{
    "success": true,
    "message": "Demo finished downloading.",
    "data": {
        "share_code": "CSGO-XXXXX-XXXXX-XXXXX-XXXXX",
        "demo_path": "/demos/match_001.dem"
    }
}
```

**Demo Parsed:**
```bash
POST /webhooks/demoParsed
Content-Type: application/json

{
    "success": true,
    "message": "Demo parsed.",
    "data": {
        "share_code": "CSGO-XXXXX-XXXXX-XXXXX-XXXXX",
        "demo_path": "/demos/match_001.dem",
        "stats": "placeholder for future stats implementation"
    }
}
```

### API Endpoints
```
GET /api/v1/match/{shareCode}     # Get match information
GET /api/v1/user/{steamID}        # Get user information  
GET /api/v1/guild/{guildID}       # Get guild information
```

## Steam API Integration

The bot automatically polls Steam API every 10 seconds to detect new matches:

- **Automatic Detection**: Monitors all registered users for new matches
- **Duplicate Prevention**: Groups users by share code to avoid duplicate downloads
- **External Integration**: Connects to demo parsing services automatically
- **Rich Notifications**: Sends detailed match summaries to Discord channels

## Guild Integration

The bot automatically:
- **Registers new guilds** when invited to Discord servers
- **Finds suitable channels** for notifications
- **Sends welcome messages** with setup instructions
- **Preserves data** when temporarily removed from servers
- **Recovers missing guilds** on bot restart

See `GUILD_INTEGRATION.md` and `FEATURES.md` for detailed documentation.

## Database Migration

Use the migration tool to set up or reset the database:

```bash
# Create tables
./migrate

# Drop and recreate all tables
./migrate -reset

# Just drop tables
./migrate -drop
```</edits>

<edits>

<old_text>
├── cmd/               # Command line tools
│   └── migrate.go     # Database migration tool
├── main.go            # Main application entry point
├── db.go              # Database connection management
├── models.go          # Data model definitions
├── database.go        # Database operations (CRUD)
├── examples.go        # Usage examples
├── DATA_MODELS.md     # Detailed documentation