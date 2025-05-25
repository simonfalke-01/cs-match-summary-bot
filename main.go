package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"cs-match-summary-bot/webhooks"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found or could not be loaded")
	}

	// Initialize database connection
	if err := initDB(); err != nil {
		log.Fatal("Failed to initialize database: ", err)
	}
	defer closeDB()

	// Get bot token from environment variable
	token := os.Getenv("DISCORD_BOT_TOKEN")
	if token == "" {
		log.Fatal("DISCORD_BOT_TOKEN environment variable is required")
	}

	// Create a new Discord session
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatal("Error creating Discord session: ", err)
	}

	// Register event handlers
	dg.AddHandler(messageCreate)
	dg.AddHandler(guildCreate)
	dg.AddHandler(guildDelete)
	dg.AddHandler(ready)

	// Set intents
	dg.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentMessageContent | discordgo.IntentsGuilds

	// Open a websocket connection to Discord and begin listening
	err = dg.Open()
	if err != nil {
		log.Fatal("Error opening connection: ", err)
	}

	// Get webhook configuration from environment variables
	webhookHost := os.Getenv("WEBHOOK_HOST")
	if webhookHost == "" {
		webhookHost = "localhost"
	}
	webhookPort := os.Getenv("WEBHOOK_PORT")
	if webhookPort == "" {
		webhookPort = "8080"
	}

	// Set up webhook context with Discord session
	SetWebhookContext(dg)
	
	// Configure webhook handlers
	handlers := &webhooks.HandlerFunctions{
		DemoReady:  HandleDemoReady,
		MatchQuery: HandleMatchQuery,
		UserQuery:  HandleUserQuery,
		GuildQuery: HandleGuildQuery,
	}
	
	// Start webhook server
	go func() {
		if err := webhooks.StartServer(webhookHost, webhookPort, handlers); err != nil {
			log.Fatal("Failed to start webhook server: ", err)
		}
	}()

	fmt.Println("Bot is now running. Press CTRL-C to exit.")
	fmt.Printf("Webhook server listening on %s:%s\n", webhookHost, webhookPort)
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Cleanly close down the Discord session
	dg.Close()
}

// This function will be called every time a new message is created
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Handle CS bot commands
	if strings.HasPrefix(m.Content, "!cs ") {
		parts := strings.Fields(m.Content)
		if len(parts) < 2 {
			return
		}
		
		command := parts[1]
		args := parts[2:]
		
		switch command {
		case "help":
			handleHelpCommand(s, m)
		case "ping":
			s.ChannelMessageSend(m.ChannelID, "🏓 Pong!")
		default:
			// Check if it's an admin command
			adminCommands := []string{"setchannel", "stats", "register", "addmatch", "listusers", "listgames"}
			for _, adminCmd := range adminCommands {
				if command == adminCmd {
					handleAdminCommand(s, m, command, args)
					return
				}
			}
			// Unknown command
			s.ChannelMessageSend(m.ChannelID, "❌ Unknown command. Use `!cs help` for available commands.")
		}
		return
	}

	// Legacy ping command for backward compatibility
	if m.Content == "ping" {
		s.ChannelMessageSend(m.ChannelID, "Pong!")
	}
}

// This function will be called when the bot joins a new guild
func guildCreate(s *discordgo.Session, g *discordgo.GuildCreate) {
	log.Printf("Bot joined guild: %s (%s)", g.Name, g.ID)
	
	// Find the first available text channel as default channel
	var defaultChannelID string
	for _, channel := range g.Channels {
		if channel.Type == discordgo.ChannelTypeGuildText {
			// Check if bot has permission to send messages in this channel
			permissions, err := s.UserChannelPermissions(s.State.User.ID, channel.ID)
			if err == nil && permissions&discordgo.PermissionSendMessages != 0 {
				defaultChannelID = channel.ID
				break
			}
		}
	}
	
	// If no suitable channel found, use the guild's system channel or the first channel
	if defaultChannelID == "" {
		if g.SystemChannelID != "" {
			defaultChannelID = g.SystemChannelID
		} else if len(g.Channels) > 0 {
			defaultChannelID = g.Channels[0].ID
		} else {
			log.Printf("Warning: No suitable channel found for guild %s", g.ID)
			defaultChannelID = g.ID // Fallback to guild ID
		}
	}
	
	// Create guild entry in database
	guild, err := createGuild(g.ID, defaultChannelID)
	if err != nil {
		log.Printf("Error creating guild in database: %v", err)
		return
	}
	
	log.Printf("Successfully added guild to database: %s (UUID: %s)", guild.GuildID, guild.UUID)
	
	// Send welcome message if we have a valid channel
	if defaultChannelID != g.ID {
		welcomeMessage := "🎮 **CS Match Summary Bot** has joined your server!\n\n" +
			"I can help you track CS match summaries and demo files. " +
			"Use this channel for match notifications, or update the channel with your preferred settings later."
		
		_, err = s.ChannelMessageSend(defaultChannelID, welcomeMessage)
		if err != nil {
			log.Printf("Error sending welcome message: %v", err)
		}
	}
}

// This function will be called when the bot leaves a guild
func guildDelete(s *discordgo.Session, g *discordgo.GuildDelete) {
	log.Printf("Bot left guild: %s", g.ID)
	
	// Note: We might want to keep the data for potential re-joins
	// Instead of deleting, we could add a "left_at" timestamp
	// For now, we'll just log it and keep the data
	
	guild, err := getGuildByGuildID(g.ID)
	if err != nil {
		log.Printf("Error retrieving guild from database: %v", err)
		return
	}
	
	log.Printf("Guild data preserved in database: %s (UUID: %s)", guild.GuildID, guild.UUID)
}

// This function will be called when the bot is ready
func ready(s *discordgo.Session, r *discordgo.Ready) {
	log.Printf("Bot is ready! Logged in as: %s#%s", s.State.User.Username, s.State.User.Discriminator)
	log.Printf("Bot is in %d guilds", len(r.Guilds))
	
	// Register all existing guilds in the database
	for _, guild := range r.Guilds {
		log.Printf("Checking guild: %s (%s)", guild.Name, guild.ID)
		
		// Check if guild already exists in database
		existingGuild, err := getGuildByGuildID(guild.ID)
		if err != nil && err != sql.ErrNoRows {
			log.Printf("Error checking guild %s: %v", guild.ID, err)
			continue
		}
		
		// If guild doesn't exist, create it
		if err == sql.ErrNoRows {
			log.Printf("Guild %s not found in database, creating...", guild.ID)
			
			// Find a suitable default channel
			var defaultChannelID string
			
			// Get full guild information to access channels
			fullGuild, err := s.Guild(guild.ID)
			if err != nil {
				log.Printf("Error getting full guild info for %s: %v", guild.ID, err)
				defaultChannelID = guild.ID // Use guild ID as fallback
			} else {
				// Find first text channel where bot can send messages
				for _, channel := range fullGuild.Channels {
					if channel.Type == discordgo.ChannelTypeGuildText {
						permissions, err := s.UserChannelPermissions(s.State.User.ID, channel.ID)
						if err == nil && permissions&discordgo.PermissionSendMessages != 0 {
							defaultChannelID = channel.ID
							break
						}
					}
				}
				
				// Fallback options
				if defaultChannelID == "" {
					if fullGuild.SystemChannelID != "" {
						defaultChannelID = fullGuild.SystemChannelID
					} else {
						defaultChannelID = guild.ID
					}
				}
			}
			
			// Create guild in database
			newGuild, err := createGuild(guild.ID, defaultChannelID)
			if err != nil {
				log.Printf("Error creating guild %s in database: %v", guild.ID, err)
				continue
			}
			
			log.Printf("Successfully registered existing guild: %s (UUID: %s)", newGuild.GuildID, newGuild.UUID)
		} else {
			log.Printf("Guild %s already exists in database (UUID: %s)", existingGuild.GuildID, existingGuild.UUID)
		}
	}
	
	log.Printf("Finished registering existing guilds")
}
