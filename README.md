# CS Match Summary Bot

A Discord bot with webhook functionality for CS match summaries, written in Go. Includes comprehensive data models for managing Discord guilds, Steam users, and CS match data with PostgreSQL integration.

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
WEBHOOK_HOST=localhost
WEBHOOK_PORT=8080
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
- **Rich Commands**: Admin and user commands for managing matches and users
- **Permission System**: Admin commands require proper Discord permissions
- **Smart Channel Detection**: Automatically finds suitable channels for notifications
- **Startup Recovery**: Registers existing guilds when bot restarts

### Webhooks & API
- **Demo Processing**: `/demoReady` POST endpoint for automatic match processing
- **REST API**: Query endpoints for matches, users, and guilds
- **Discord Integration**: Automatic notifications when matches are processed
- **Error Handling**: Robust error handling with meaningful responses
- **Configurable host and port**: Via environment variables

### Data Models
- **Guild Management**: Store Discord guild information with associated users and games
- **User Management**: Track Steam users with authentication codes and game history
- **Game Management**: Store CS match data with share codes, demo files, and player information
- **PostgreSQL Integration**: Robust database schema with UUID primary keys and JSONB arrays

## Configuration

The bot uses environment variables for configuration:

- `DISCORD_BOT_TOKEN` - Your Discord bot token (required)
- `WEBHOOK_HOST` - Host for webhook server (default: localhost)
- `WEBHOOK_PORT` - Port for webhook server (default: 8080)
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
├── examples.go        # Usage examples
├── DATA_MODELS.md     # Detailed documentation
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
- Fields: UUID, steam_id, auth_code, game_ids[]

### Game
- Represents CS matches with demo files and player data
- Fields: UUID, share_code, demo_name, steam_ids[]

## Database Operations

Comprehensive CRUD operations are available for all models:

```go
// Create entities
guild, err := createGuild("discord_guild_id", "discord_channel_id")
user, err := createUser("steam_id", "auth_code")
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

The bot provides comprehensive Discord commands for managing matches and users:

### General Commands
```
!cs help          # Show command help
!cs ping          # Test bot responsiveness
```

### Admin Commands (Requires Admin/Manage Server permissions)
```
!cs setchannel [#channel]                    # Set notification channel
!cs stats                                    # Show guild statistics  
!cs register <steam_id> <auth_code>          # Register a Steam user
!cs addmatch <share_code> <demo_name> [steam_ids...]  # Add a match manually
!cs listusers                                # List registered users
!cs listgames                                # List tracked games
```

## Webhook Integration

### Demo Ready Webhook
Process matches automatically via HTTP webhook:

```bash
POST /demoReady
Content-Type: application/json

{
    "share_code": "CSGO-XXXXX-XXXXX-XXXXX-XXXXX",
    "demo_name": "match_2024_01_15_001.dem", 
    "guild_id": "123456789012345678",
    "steam_ids": ["76561198000000001", "76561198000000002"],
    "channel_id": "987654321098765432"  // Optional
}
```

### API Endpoints
```
GET /api/v1/match/{shareCode}     # Get match information
GET /api/v1/user/{steamID}        # Get user information  
GET /api/v1/guild/{guildID}       # Get guild information
```

## Guild Integration

The bot automatically:
- **Registers new guilds** when invited to Discord servers
- **Finds suitable channels** for notifications
- **Sends welcome messages** with setup instructions
- **Preserves data** when temporarily removed from servers
- **Recovers missing guilds** on bot restart

See `GUILD_INTEGRATION.md` for detailed integration documentation.

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