# Data Models Documentation

This document describes the data models and database operations for the CS Match Summary Bot.

## Overview

The bot stores information about Discord guilds, Steam users, and CS matches (games) with the following relationships:

- **Guilds** can have multiple users and games
- **Users** can belong to multiple guilds and participate in multiple games  
- **Games** can include multiple users and be associated with multiple guilds

## Data Models

### Guild

Represents a Discord server/guild where the bot operates.

**Fields:**
- `uuid` (UUID) - Primary key, auto-generated
- `guild_id` (string) - Discord guild ID (unique)
- `channel_id` (string) - Discord channel ID for bot messages
- `user_ids` ([]string) - Array of user UUIDs associated with this guild
- `game_ids` ([]string) - Array of game UUIDs associated with this guild
- `created_at` (timestamp) - Auto-generated creation time
- `updated_at` (timestamp) - Auto-updated modification time

**Example:**
```go
guild := &Guild{
    UUID:      uuid.New(),
    GuildID:   "123456789012345678",
    ChannelID: "987654321098765432",
    UserIDs:   []string{"user-uuid-1", "user-uuid-2"},
    GameIDs:   []string{"game-uuid-1", "game-uuid-2"},
}
```

### User

Represents a Steam user who can participate in CS matches.

**Fields:**
- `uuid` (UUID) - Primary key, auto-generated
- `steam_id` (string) - Steam ID (unique)
- `auth_code` (string) - Authentication code for Steam API access
- `game_ids` ([]string) - Array of game UUIDs the user participated in
- `created_at` (timestamp) - Auto-generated creation time
- `updated_at` (timestamp) - Auto-updated modification time

**Example:**
```go
user := &User{
    UUID:     uuid.New(),
    SteamID:  "76561198000000001",
    AuthCode: "auth_code_123",
    GameIDs:  []string{"game-uuid-1", "game-uuid-2"},
}
```

### Game

Represents a CS match with demo file information.

**Fields:**
- `uuid` (UUID) - Primary key, auto-generated
- `share_code` (string) - CS match share code (unique)
- `demo_name` (string) - Path/name of the demo file
- `steam_ids` ([]string) - Array of Steam IDs of players in this match
- `created_at` (timestamp) - Auto-generated creation time
- `updated_at` (timestamp) - Auto-updated modification time

**Example:**
```go
game := &Game{
    UUID:      uuid.New(),
    ShareCode: "CSGO-XXXXX-XXXXX-XXXXX-XXXXX",
    DemoName:  "match_2024_01_15_001.dem",
    SteamIDs:  []string{"76561198000000001", "76561198000000002"},
}
```

## Database Operations

### Guild Operations

#### Create Guild
```go
guild, err := createGuild("discord_guild_id", "discord_channel_id")
```

#### Get Guild by Discord ID
```go
guild, err := getGuildByGuildID("discord_guild_id")
```

#### Update Guild
```go
guild.ChannelID = "new_channel_id"
err := updateGuild(guild)
```

#### Add User to Guild
```go
err := addUserToGuild("discord_guild_id", userUUID)
```

#### Add Game to Guild
```go
err := addGameToGuild("discord_guild_id", gameUUID)
```

### User Operations

#### Create User
```go
user, err := createUser("steam_id", "auth_code")
```

#### Get User by Steam ID
```go
user, err := getUserBySteamID("steam_id")
```

#### Get User by UUID
```go
user, err := getUserByUUID(userUUID)
```

#### Update User
```go
user.AuthCode = "new_auth_code"
err := updateUser(user)
```

#### Add Game to User
```go
err := addGameToUser("steam_id", gameUUID)
```

### Game Operations

#### Create Game
```go
steamIDs := []string{"steam_id_1", "steam_id_2"}
game, err := createGame("share_code", "demo_name.dem", steamIDs)
```

#### Get Game by Share Code
```go
game, err := getGameByShareCode("CSGO-XXXXX-XXXXX-XXXXX-XXXXX")
```

#### Get Game by UUID
```go
game, err := getGameByUUID(gameUUID)
```

#### Update Game
```go
game.DemoName = "processed_demo.dem"
err := updateGame(game)
```

#### Get Games by Steam ID
```go
games, err := getGamesBySteamID("steam_id")
```

#### Get Games for Guild
```go
games, err := getGamesForGuild("discord_guild_id")
```

## Database Schema

The PostgreSQL database includes the following tables:

### guilds
```sql
CREATE TABLE guilds (
    uuid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    guild_id VARCHAR(255) UNIQUE NOT NULL,
    channel_id VARCHAR(255) NOT NULL,
    user_ids JSONB DEFAULT '[]',
    game_ids JSONB DEFAULT '[]',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
```

### users
```sql
CREATE TABLE users (
    uuid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    steam_id VARCHAR(255) UNIQUE NOT NULL,
    auth_code VARCHAR(255) NOT NULL,
    game_ids JSONB DEFAULT '[]',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
```

### games
```sql
CREATE TABLE games (
    uuid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    share_code VARCHAR(255) UNIQUE NOT NULL,
    demo_name VARCHAR(255) NOT NULL,
    steam_ids JSONB DEFAULT '[]',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
```

## Indexes

For optimal performance, the following indexes are created:
- `idx_guilds_guild_id` on `guilds(guild_id)`
- `idx_users_steam_id` on `users(steam_id)`
- `idx_games_share_code` on `games(share_code)`

## Triggers

Automatic `updated_at` timestamp updates are handled by PostgreSQL triggers:
- `update_guilds_updated_at`
- `update_users_updated_at` 
- `update_games_updated_at`

## Usage Examples

See `examples.go` for comprehensive usage examples including:
- Basic CRUD operations
- Linking entities together
- Batch operations
- Error handling patterns

## Environment Variables

Required database configuration:
- `DB_HOST` - Database host (default: localhost)
- `DB_PORT` - Database port (default: 5432)
- `DB_USER` - Database username (default: postgres)
- `DB_PASSWORD` - Database password (default: postgres)
- `DB_NAME` - Database name (default: cs)

## Error Handling

All database operations return errors that should be handled appropriately:

```go
guild, err := getGuildByGuildID("guild_id")
if err != nil {
    if err == sql.ErrNoRows {
        // Handle case where guild doesn't exist
        log.Printf("Guild not found: %v", err)
    } else {
        // Handle other database errors
        log.Printf("Database error: %v", err)
    }
    return
}
```

## Best Practices

1. Always check for errors when calling database operations
2. Use transactions for operations that modify multiple tables
3. Use the UUID fields for internal references between entities
4. Use the human-readable IDs (guild_id, steam_id, share_code) for external API interactions
5. The JSONB arrays automatically prevent duplicate entries when using the provided helper functions