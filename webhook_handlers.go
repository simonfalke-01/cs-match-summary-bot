package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/gin-gonic/gin"
)

// DemoReadyPayload represents the webhook payload when a demo is ready
type DemoReadyPayload struct {
	Success bool `json:"success"`
	Message string `json:"message"`
	Data struct {
		ShareCode string `json:"share_code"`
		DemoPath  string `json:"demo_path"`
	} `json:"data"`
}

// DemoParsedPayload represents the webhook payload when a demo is parsed
type DemoParsedPayload struct {
	Success bool `json:"success"`
	Message string `json:"message"`
	Data struct {
		ShareCode string      `json:"share_code"`
		DemoPath  string      `json:"demo_path"`
		Stats     interface{} `json:"stats"` // Placeholder for future stats implementation
	} `json:"data"`
}

// WebhookContext holds the context needed for webhook handlers
type WebhookContext struct {
	DiscordSession *discordgo.Session
}

var webhookCtx *WebhookContext

// SetWebhookContext sets the global webhook context
func SetWebhookContext(session *discordgo.Session) {
	webhookCtx = &WebhookContext{
		DiscordSession: session,
	}
}

// HandleDemoReady processes the demo ready webhook
func HandleDemoReady(c *gin.Context) {
	var payload DemoReadyPayload
	
	if err := c.ShouldBindJSON(&payload); err != nil {
		log.Printf("Invalid JSON payload: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON payload"})
		return
	}
	
	// Validate payload structure
	if !payload.Success {
		log.Printf("Demo ready webhook reported failure: %s", payload.Message)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Demo processing failed"})
		return
	}
	
	if payload.Data.ShareCode == "" || payload.Data.DemoPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing required fields: share_code, demo_path"})
		return
	}
	
	log.Printf("Demo ready received: %s at %s", payload.Data.ShareCode, payload.Data.DemoPath)
	
	// Create or update game record
	_, err := createOrUpdateGame(payload.Data.ShareCode, payload.Data.DemoPath)
	if err != nil {
		log.Printf("Error creating/updating game: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process game"})
		return
	}
	
	// Request demo parsing
	if steamPoller != nil {
		err = steamPoller.GetDemoParsingRequest()(payload.Data.ShareCode)
		if err != nil {
			log.Printf("Error requesting demo parsing for %s: %v", payload.Data.ShareCode, err)
			// Don't fail the webhook, just log the error
		} else {
			log.Printf("Successfully requested demo parsing for %s", payload.Data.ShareCode)
		}
	}
	
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Demo ready processed successfully",
	})
}

// HandleDemoParsed processes the demo parsed webhook
func HandleDemoParsed(c *gin.Context) {
	var payload DemoParsedPayload
	
	if err := c.ShouldBindJSON(&payload); err != nil {
		log.Printf("Invalid JSON payload: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON payload"})
		return
	}
	
	// Validate payload structure
	if !payload.Success {
		log.Printf("Demo parsing webhook reported failure: %s", payload.Message)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Demo parsing failed"})
		return
	}
	
	if payload.Data.ShareCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing required field: share_code"})
		return
	}
	
	log.Printf("Demo parsing completed for: %s", payload.Data.ShareCode)
	
	// Get the game from database
	game, err := getGameByShareCode(payload.Data.ShareCode)
	if err != nil {
		log.Printf("Error getting game %s: %v", payload.Data.ShareCode, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get game"})
		return
	}
	
	// Send match summary to all guilds that have this game
	err = sendMatchSummaryToGuilds(game, payload.Data.Stats)
	if err != nil {
		log.Printf("Error sending match summaries: %v", err)
		// Don't fail the webhook, just log the error
	}
	
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Demo parsing completed successfully",
	})
}

// createOrUpdateGame creates a new game or updates existing game with demo path
func createOrUpdateGame(shareCode, demoPath string) (*Game, error) {
	// Try to get existing game
	game, err := getGameByShareCode(shareCode)
	if err == sql.ErrNoRows {
		// Create new game - we'll get steam IDs when we have the stats
		game, err = createGame(shareCode, demoPath, []string{})
		if err != nil {
			return nil, fmt.Errorf("failed to create game: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("failed to check existing game: %w", err)
	} else {
		// Update existing game
		game.DemoName = demoPath
		err = updateGame(game)
		if err != nil {
			return nil, fmt.Errorf("failed to update game: %w", err)
		}
	}
	
	return game, nil
}

// sendMatchSummaryToGuilds sends match summary to all guilds that have registered users for this match
func sendMatchSummaryToGuilds(game *Game, stats interface{}) error {
	if webhookCtx == nil || webhookCtx.DiscordSession == nil {
		return fmt.Errorf("Discord session not available")
	}
	
	// Find all guilds that have users who played in this match
	guildsToNotify := make(map[string]*Guild)
	
	for _, steamID := range game.SteamIDs {
		user, err := getUserBySteamID(steamID)
		if err != nil {
			continue // User not registered, skip
		}
		
		// Find guilds that have this user
		allGuilds, err := getAllGuilds()
		if err != nil {
			continue
		}
		
		for _, guild := range allGuilds {
			for _, userIDStr := range guild.UserIDs {
				if userIDStr == user.UUID.String() {
					guildsToNotify[guild.GuildID] = guild
				}
			}
		}
	}
	
	// Send notification to each guild
	for _, guild := range guildsToNotify {
		err := sendMatchSummary(guild, game, stats)
		if err != nil {
			log.Printf("Error sending match summary to guild %s: %v", guild.GuildID, err)
		}
	}
	
	return nil
}

// sendMatchSummary sends a match summary embed to a specific guild
func sendMatchSummary(guild *Guild, game *Game, stats interface{}) error {
	embed := &discordgo.MessageEmbed{
		Title: "🎯 CS Match Summary",
		Color: 0x00ff00,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Share Code",
				Value:  fmt.Sprintf("`%s`", game.ShareCode),
				Inline: true,
			},
			{
				Name:   "Demo File",
				Value:  fmt.Sprintf("`%s`", game.DemoName),
				Inline: true,
			},
			{
				Name:   "Players",
				Value:  fmt.Sprintf("%d players", len(game.SteamIDs)),
				Inline: true,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Match analysis completed",
		},
	}
	
	// Add registered players for this guild
	var registeredPlayers []string
	for _, steamID := range game.SteamIDs {
		user, err := getUserBySteamID(steamID)
		if err == nil {
			// Check if this user is in the current guild
			for _, userIDStr := range guild.UserIDs {
				if userIDStr == user.UUID.String() {
					registeredPlayers = append(registeredPlayers, steamID)
					break
				}
			}
		}
	}
	
	if len(registeredPlayers) > 0 {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Registered Players",
			Value:  fmt.Sprintf("```\n%s\n```", strings.Join(registeredPlayers, "\n")),
			Inline: false,
		})
	}
	
	// TODO: Add stats fields when stats schema is implemented
	if stats != nil {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Stats",
			Value:  "📊 Match statistics available (schema TBD)",
			Inline: false,
		})
	}
	
	_, err := webhookCtx.DiscordSession.ChannelMessageSendEmbed(guild.ChannelID, embed)
	return err
}

// getAllGuilds retrieves all guilds from the database
func getAllGuilds() ([]*Guild, error) {
	query := `
		SELECT uuid, guild_id, channel_id, user_ids, game_ids, created_at, updated_at
		FROM guilds ORDER BY created_at`

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all guilds: %w", err)
	}
	defer rows.Close()

	var guilds []*Guild
	for rows.Next() {
		guild := &Guild{}
		err := rows.Scan(
			&guild.UUID, &guild.GuildID, &guild.ChannelID, &guild.UserIDs, &guild.GameIDs,
			&guild.CreatedAt, &guild.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan guild: %w", err)
		}
		guilds = append(guilds, guild)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over guilds: %w", err)
	}

	return guilds, nil
}

// HandleMatchQuery handles queries for match information
func HandleMatchQuery(c *gin.Context) {
	shareCode := c.Param("shareCode")
	if shareCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Share code is required"})
		return
	}
	
	game, err := getGameByShareCode(shareCode)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Match not found"})
		} else {
			log.Printf("Error querying match: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query match"})
		}
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"uuid":       game.UUID.String(),
		"share_code": game.ShareCode,
		"demo_name":  game.DemoName,
		"steam_ids":  game.SteamIDs,
		"created_at": game.CreatedAt,
		"updated_at": game.UpdatedAt,
	})
}

// HandleUserQuery handles queries for user information
func HandleUserQuery(c *gin.Context) {
	steamID := c.Param("steamID")
	if steamID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Steam ID is required"})
		return
	}
	
	user, err := getUserBySteamID(steamID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		} else {
			log.Printf("Error querying user: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query user"})
		}
		return
	}
	
	// Get user's games
	games, err := getGamesBySteamID(steamID)
	if err != nil {
		log.Printf("Error getting user games: %v", err)
		games = []*Game{} // Empty slice on error
	}
	
	c.JSON(http.StatusOK, gin.H{
		"uuid":        user.UUID.String(),
		"steam_id":    user.SteamID,
		"game_count":  len(games),
		"created_at":  user.CreatedAt,
		"updated_at":  user.UpdatedAt,
	})
}

// HandleGuildQuery handles queries for guild information
func HandleGuildQuery(c *gin.Context) {
	guildID := c.Param("guildID")
	if guildID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Guild ID is required"})
		return
	}
	
	guild, err := getGuildByGuildID(guildID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Guild not found"})
		} else {
			log.Printf("Error querying guild: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query guild"})
		}
		return
	}
	
	// Get guild's games
	games, err := getGamesForGuild(guildID)
	if err != nil {
		log.Printf("Error getting guild games: %v", err)
		games = []*Game{} // Empty slice on error
	}
	
	c.JSON(http.StatusOK, gin.H{
		"uuid":       guild.UUID.String(),
		"guild_id":   guild.GuildID,
		"channel_id": guild.ChannelID,
		"user_count": len(guild.UserIDs),
		"game_count": len(games),
		"created_at": guild.CreatedAt,
		"updated_at": guild.UpdatedAt,
	})
}