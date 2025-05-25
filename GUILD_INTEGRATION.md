# Guild Integration Documentation

This document explains how the CS Match Summary Bot automatically handles Discord guild integration, including auto-registration when added to new servers and webhook-driven match processing.

## Automatic Guild Registration

### When Bot Joins a New Guild

When the bot is invited to a new Discord server, it automatically:

1. **Detects Guild Join**: The `guildCreate` event handler triggers
2. **Finds Default Channel**: Searches for the first text channel where the bot has send message permissions
3. **Creates Database Entry**: Automatically creates a guild record with:
   - `uuid`: Auto-generated unique identifier
   - `guild_id`: Discord guild ID
   - `channel_id`: Selected default channel for notifications
   - `user_ids`: Empty array (populated as users register)
   - `game_ids`: Empty array (populated as matches are added)
4. **Sends Welcome Message**: Posts an introduction message in the selected channel

### Channel Selection Logic

The bot uses the following priority order for selecting the default notification channel:

1. **First available text channel** where bot has `SEND_MESSAGES` permission
2. **System channel** (if configured and accessible)
3. **First channel in the list** (as fallback)
4. **Guild ID** (emergency fallback)

### Startup Registration

When the bot starts up, it also:

1. **Scans Existing Guilds**: Checks all guilds the bot is currently in
2. **Registers Missing Guilds**: Creates database entries for any guilds not already registered
3. **Preserves Existing Data**: Skips guilds that are already in the database

This ensures the bot works correctly even if it was added to servers while offline.

## Guild Management Commands

### Admin Commands

All admin commands require **Administrator** or **Manage Server** permissions:

```
!cs setchannel [#channel]     # Set notification channel (uses current if no channel specified)
!cs stats                     # Show guild statistics (users, games)
!cs register <steam_id> <auth_code>  # Register a Steam user
!cs addmatch <share_code> <demo_name> [steam_ids...]  # Manually add a match
!cs listusers                 # List registered users (max 10 shown)
!cs listgames                 # List tracked games (max 10 shown)
```

### General Commands

```
!cs help                      # Show command help
!cs ping                      # Test bot responsiveness
```

## Webhook Integration

### Demo Ready Webhook

The bot provides a `/demoReady` POST endpoint for external systems to automatically add matches:

**Endpoint**: `POST http://your-server:8080/demoReady`

**Payload**:
```json
{
    "share_code": "CSGO-XXXXX-XXXXX-XXXXX-XXXXX",
    "demo_name": "match_2024_01_15_001.dem",
    "guild_id": "123456789012345678",
    "steam_ids": ["76561198000000001", "76561198000000002"],
    "channel_id": "987654321098765432"  // Optional override
}
```

**Response**:
```json
{
    "status": "success",
    "game_uuid": "550e8400-e29b-41d4-a716-446655440000",
    "guild_uuid": "660e8400-e29b-41d4-a716-446655440001",
    "message": "Match processed successfully"
}
```

### Automatic Processing

When a webhook is received, the bot:

1. **Validates Payload**: Ensures required fields are present
2. **Ensures Guild Exists**: Creates guild record if missing
3. **Processes Match**: Creates or updates game record
4. **Links Entities**: Associates game with guild and participating users
5. **Sends Discord Notification**: Posts match details to the configured channel
6. **Returns Response**: Confirms successful processing

### Discord Notifications

Webhook-triggered matches generate rich embed notifications containing:

- **Share Code**: The CS match share code
- **Demo File**: Name/path of the demo file
- **Player Count**: Number of participating players
- **Player List**: Steam IDs of participants (up to 10 shown)
- **Registration Status**: Shows which players are registered vs unregistered

## API Endpoints

### Query APIs

The bot also provides REST APIs for querying data:

#### Get Match Information
```
GET /api/v1/match/{shareCode}
```

#### Get User Information
```
GET /api/v1/user/{steamID}
```

#### Get Guild Information
```
GET /api/v1/guild/{guildID}
```

## Database Schema Integration

### Guild Table Structure

```sql
CREATE TABLE guilds (
    uuid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    guild_id VARCHAR(255) UNIQUE NOT NULL,           -- Discord guild ID
    channel_id VARCHAR(255) NOT NULL,                -- Default notification channel
    user_ids JSONB DEFAULT '[]',                     -- Array of user UUIDs
    game_ids JSONB DEFAULT '[]',                     -- Array of game UUIDs
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
```

### Relationship Management

- **Many-to-Many**: Guilds ↔ Users (via JSONB arrays)
- **Many-to-Many**: Guilds ↔ Games (via JSONB arrays)
- **Many-to-Many**: Users ↔ Games (via JSONB arrays)

## Error Handling

### Guild Creation Errors

- **Permission Issues**: Bot logs warning and uses fallback channel
- **Database Errors**: Logged but don't prevent bot operation
- **Duplicate Guilds**: Safely ignored (idempotent operations)

### Webhook Errors

- **Invalid Payload**: Returns 400 Bad Request
- **Missing Guild**: Automatically creates guild entry
- **Database Failures**: Returns 500 Internal Server Error
- **Discord Notification Failures**: Logged but don't fail the webhook

### Command Errors

- **Permission Denied**: Clear error message to user
- **Invalid Parameters**: Usage help provided
- **Database Errors**: User-friendly error messages

## Security Considerations

### Permission Checks

- **Admin Commands**: Require Administrator or Manage Server permissions
- **Guild Access**: Commands only work within the guild context
- **Channel Permissions**: Bot respects Discord channel permissions

### Data Validation

- **Webhook Payloads**: Validated for required fields and format
- **Steam IDs**: Basic format validation
- **Share Codes**: Format validation for CS match codes

### Rate Limiting

- **Discord API**: Respects Discord rate limits
- **Database Operations**: Connection pooling and timeout handling
- **Webhook Processing**: Async processing to prevent blocking

## Configuration

### Environment Variables

```bash
DISCORD_BOT_TOKEN=your_bot_token_here     # Required
WEBHOOK_HOST=localhost                     # Default: localhost
WEBHOOK_PORT=8080                         # Default: 8080
DB_HOST=localhost                         # Default: localhost
DB_PORT=5432                              # Default: 5432
DB_USER=postgres                          # Default: postgres
DB_PASSWORD=postgres                      # Default: postgres
DB_NAME=cs                                # Default: cs
```

### Discord Bot Permissions

Required bot permissions:
- **Send Messages**: For notifications and command responses
- **Use Slash Commands**: For future slash command support
- **Read Message History**: For context in command processing
- **Add Reactions**: For interactive features

Required intents:
- **Guild Messages**: To receive and respond to commands
- **Message Content**: To parse command content
- **Guilds**: To detect guild join/leave events

## Integration Examples

### External Demo Processing System

```bash
# When a demo is processed
curl -X POST http://your-bot-server:8080/demoReady \
  -H "Content-Type: application/json" \
  -d '{
    "share_code": "CSGO-ABCDE-FGHIJ-KLMNO-PQRST",
    "demo_name": "processed_match_001.dem",
    "guild_id": "123456789012345678",
    "steam_ids": ["76561198000000001", "76561198000000002"]
  }'
```

### Discord Server Setup

1. **Invite Bot**: Use Discord developer portal to generate invite link
2. **Automatic Setup**: Bot auto-registers and sends welcome message
3. **Configure Channel**: Use `!cs setchannel #your-channel` if needed
4. **Register Users**: Use `!cs register <steam_id> <auth_code>` for each player
5. **Test Integration**: Use `!cs stats` to verify setup

### Monitor Integration Health

```bash
# Check if guild is registered
curl http://your-bot-server:8080/api/v1/guild/123456789012345678

# Check user registration
curl http://your-bot-server:8080/api/v1/user/76561198000000001

# Query specific match
curl http://your-bot-server:8080/api/v1/match/CSGO-ABCDE-FGHIJ-KLMNO-PQRST
```