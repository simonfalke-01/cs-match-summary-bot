# CS Match Summary Bot

A Discord bot with webhook functionality for CS match summaries, written in Go.

## Setup

1. Install dependencies:
```bash
go mod tidy
```

2. Copy the example environment file and configure your settings:
```bash
cp .env.example .env
```

3. Edit `.env` with your configuration:
```
DISCORD_BOT_TOKEN=your_bot_token_here
WEBHOOK_HOST=localhost
WEBHOOK_PORT=8080
```

4. Run the bot:
```bash
go run main.go
```

## Features

### Discord Bot
- Responds to "ping" with "Pong!"

### Webhooks
- HTTP server with webhook endpoints
- `/demoReady` POST endpoint that returns `{"status": "received"}`
- Configurable host and port via environment variables

## Configuration

The bot uses environment variables for configuration:

- `DISCORD_BOT_TOKEN` - Your Discord bot token (required)
- `WEBHOOK_HOST` - Host for webhook server (default: localhost)
- `WEBHOOK_PORT` - Port for webhook server (default: 8080)

## Project Structure

```
cs-match-summary-bot/
├── webhooks/           # Webhook server package
│   └── server.go      # HTTP server and handlers
├── main.go            # Main application entry point
├── .env.example       # Example environment configuration
└── README.md          # This file
```