# CS Match Summary Bot - Features Documentation

This document provides comprehensive documentation for all bot features including slash commands, Steam API polling, and webhook integration.

## Slash Commands

The bot uses Discord slash commands for user interaction. All commands are guild-scoped and provide rich interactive experiences.

### `/register`

Register a new user with Steam integration.

**Parameters:**
- `steam_id` (required) - Your Steam ID (e.g., "76561198000000001")
- `auth_code` (required) - Your Steam authentication code
- `last_share_code` (required) - Your last known CS match share code (e.g., "CSGO-ABCDE-FGHIJ-KLMNO-PQRST")

**Functionality:**
- Creates new user or updates existing user information
- Automatically adds user to the current guild
- Validates share code format (must start with "CSGO-")
- Provides immediate feedback on success/failure

**Usage Example:**
```
/register steam_id:76561198000000001 auth_code:AAAA-BBBBB-CCCC last_share_code:CSGO-ABCDE-FGHIJ-KLMNO-PQRST
```

### `/remove`

Remove a user from the system.

**Parameters:**
- `steam_id` (required) - Steam ID of the user to remove

**Functionality:**
- Removes user from all guilds
- Deletes user record completely
- Provides confirmation of removal

**Usage Example:**
```
/remove steam_id:76561198000000001
```

### `/users`

Display list of registered users in the current guild.

**Parameters:** None

**Functionality:**
- Shows up to 25 users (Discord embed limitations)
- Displays Steam ID and last known share code for each user
- Shows total user count in footer
- Handles empty lists gracefully

**Example Output:**
```
👥 Registered Users

• 76561198000000001 - Last: CSGO-ABCDE-FGHIJ-KLMNO-PQRST
• 76561198000000002 - Last: CSGO-ZYXWV-UTSRQ-PONML-KJIHG

Total: 2 users
```

### `/set_channel`

Set the channel for match summaries (Admin only).

**Parameters:**
- `channel` (optional) - Channel to send match summaries to (defaults to current channel)

**Permissions:** Requires "Manage Server" permission

**Functionality:**
- Updates guild's notification channel
- Validates channel permissions
- Uses current channel if no channel specified
- Provides immediate confirmation

**Usage Example:**
```
/set_channel channel:#match-summaries
```

## Steam API Polling

The bot continuously polls the Steam API to detect new matches for registered users.

### Polling Mechanism

**Frequency:** Every 10 seconds
**API Endpoint:** `https://api.steampowered.com/ICSGOPlayers_730/GetNextMatchSharingCode/v1`

**Parameters:**
- `key` - Steam API key (from `STEAM_API_KEY` environment variable)
- `steamid` - User's Steam ID
- `steamidkey` - User's authentication code
- `knowncode` - User's last known share code

### Response Handling

**New match available:**
```json
{
    "result": {
        "nextcode": "CSGO-XXXXX-XXXXX-XXXXX-XXXXX"
    }
}
```

**No new match:**
```json
{
    "result": {
        "nextcode": "n/a"
    }
}
```

### Duplicate Prevention

- Groups users by share code to avoid duplicate downloads
- Tracks processed codes to prevent reprocessing
- Automatically cleans up processed codes cache (hourly)

### Error Handling

- Continues polling other users if one fails
- Logs errors for monitoring
- Retries failed demo requests by removing from processed cache

## Webhook Integration

The bot provides webhook endpoints for external demo processing services.

### `/webhooks/demoReady`

Receives notifications when demo download is complete.

**Expected Payload:**
```json
{
    "success": true,
    "message": "Demo finished downloading.",
    "data": {
        "share_code": "CSGO-XXXXX-XXXXX-XXXXX-XXXXX",
        "demo_path": "/demos/match_001.dem"
    }
}
```

**Processing:**
1. Validates payload structure
2. Creates or updates game record in database
3. Triggers demo parsing request
4. Returns success confirmation

**Response:**
```json
{
    "status": "success",
    "message": "Demo ready processed successfully"
}
```

### `/webhooks/demoParsed`

Receives notifications when demo parsing is complete.

**Expected Payload:**
```json
{
    "success": true,
    "message": "Demo parsed.",
    "data": {
        "share_code": "CSGO-XXXXX-XXXXX-XXXXX-XXXXX",
        "demo_path": "/demos/match_001.dem",
        "stats": "placeholder, will be a dictionary when implemented"
    }
}
```

**Processing:**
1. Validates payload structure
2. Retrieves game from database
3. Sends match summaries to all relevant guilds
4. Returns success confirmation

**Response:**
```json
{
    "status": "success",
    "message": "Demo parsing completed successfully"
}
```

## Match Processing Workflow

### 1. Detection Phase
- Steam API polling detects new match
- Updates user's last share code in database
- Groups users by share code to prevent duplicates

### 2. Download Phase
- Calls `https://cs-demo-parsing.simonfalke.com/getDemo/{shareCode}`
- Sends webhook URL for notifications
- Waits for `/webhooks/demoReady` callback

### 3. Parsing Phase
- Receives demoReady webhook
- Calls `https://cs-demo-parsing.simonfalke.com/parseDemo/{shareCode}`
- Sends webhook URL for parsing completion
- Waits for `/webhooks/demoParsed` callback

### 4. Notification Phase
- Receives demoParsed webhook
- Finds all guilds with registered players
- Sends rich embed summaries to each guild's notification channel

## Discord Notifications

### Match Summary Embed

**Title:** 🎯 CS Match Summary
**Color:** Green (#00ff00)

**Fields:**
- **Share Code:** Formatted as code block
- **Demo File:** File path/name
- **Players:** Count of participants
- **Registered Players:** List of registered players in the guild
- **Stats:** Placeholder for future implementation

**Example:**
```
🎯 CS Match Summary

Share Code: CSGO-ABCDE-FGHIJ-KLMNO-PQRST
Demo File: /demos/match_001.dem
Players: 10 players

Registered Players:
76561198000000001
76561198000000002

Stats: 📊 Match statistics available (schema TBD)

Match analysis completed
```

## API Endpoints

The bot provides REST API endpoints for querying data.

### `GET /api/v1/match/{shareCode}`

Get match information by share code.

**Response:**
```json
{
    "uuid": "550e8400-e29b-41d4-a716-446655440000",
    "share_code": "CSGO-ABCDE-FGHIJ-KLMNO-PQRST",
    "demo_name": "/demos/match_001.dem",
    "steam_ids": ["76561198000000001", "76561198000000002"],
    "created_at": "2024-01-15T10:00:00Z",
    "updated_at": "2024-01-15T10:05:00Z"
}
```

### `GET /api/v1/user/{steamID}`

Get user information by Steam ID.

**Response:**
```json
{
    "uuid": "660e8400-e29b-41d4-a716-446655440001",
    "steam_id": "76561198000000001",
    "game_count": 5,
    "created_at": "2024-01-15T09:00:00Z",
    "updated_at": "2024-01-15T10:00:00Z"
}
```

### `GET /api/v1/guild/{guildID}`

Get guild information by Discord guild ID.

**Response:**
```json
{
    "uuid": "770e8400-e29b-41d4-a716-446655440002",
    "guild_id": "123456789012345678",
    "channel_id": "987654321098765432",
    "user_count": 3,
    "game_count": 8,
    "created_at": "2024-01-15T08:00:00Z",
    "updated_at": "2024-01-15T10:00:00Z"
}
```

## Configuration

### Environment Variables

**Required:**
- `DISCORD_BOT_TOKEN` - Discord bot token
- `STEAM_API_KEY` - Steam API key for polling

**Optional:**
- `WEBHOOK_HOST` - Host for webhook server (default: localhost)
- `WEBHOOK_PORT` - Port for webhook server (default: 8080)
- `WEBHOOK_BASE_URL` - Base URL for webhook callbacks (default: https://cs-bot.simonfalke.com)
- `DEMO_PARSE_BASE_URL` - Base URL for demo parsing service (default: https://cs-demo-parsing.simonfalke.com)
- `DB_HOST` - Database host (default: localhost)
- `DB_PORT` - Database port (default: 5432)
- `DB_USER` - Database username (default: postgres)
- `DB_PASSWORD` - Database password (default: postgres)
- `DB_NAME` - Database name (default: cs)

### Discord Permissions

**Required Bot Permissions:**
- Send Messages
- Use Slash Commands
- Embed Links
- Read Message History

**Required Intents:**
- Guilds (for guild events)
- Guild Messages (for legacy commands)
- Message Content (for legacy command parsing)

## Error Handling

### Slash Commands
- Input validation with clear error messages
- Permission checks for admin commands
- Database error handling with user-friendly responses
- Ephemeral error messages (only visible to command user)

### Steam API Polling
- Continues polling other users if one fails
- Logs errors for monitoring and debugging
- Handles API rate limits and timeouts
- Automatic retry for failed requests

### Webhook Processing
- Validates payload structure before processing
- Returns appropriate HTTP status codes
- Logs all webhook events for debugging
- Graceful handling of missing data

### Database Operations
- Connection pooling and timeout handling
- Transaction management for complex operations
- Proper error logging and user feedback
- Automatic reconnection on connection loss

## Performance Considerations

### Polling Efficiency
- 10-second intervals balance responsiveness with API limits
- Duplicate prevention reduces unnecessary API calls
- Batch processing for multiple users with same match
- Memory-efficient processed codes cache with cleanup

### Database Optimization
- Indexed columns for fast lookups
- JSONB arrays for flexible relationships
- Connection pooling for concurrent requests
- Efficient queries to minimize database load

### Discord Rate Limits
- Built-in rate limit handling in Discord.js
- Batched notifications to avoid spam
- Efficient embed generation and reuse
- Proper error handling for rate limit responses

## Security Considerations

### Data Protection
- Auth codes encrypted in database (recommended)
- Steam API keys stored as environment variables
- No sensitive data in logs or error messages
- Proper input validation and sanitization

### Access Control
- Slash commands respect Discord permissions
- Admin commands require proper guild permissions
- API endpoints are read-only for security
- Webhook endpoints validate payload structure

### Rate Limiting
- Steam API respects official rate limits
- Discord API uses built-in rate limiting
- Webhook endpoints can be rate-limited if needed
- Database queries optimized to prevent abuse