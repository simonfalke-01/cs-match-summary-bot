package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"github.com/bwmarrin/discordgo"
	"github.com/gin-gonic/gin"
)

// DemoReadyPayload represents the webhook payload when a demo is ready
type DemoReadyPayload struct {
	ShareCode string   `json:"share_code"`
	DemoName  string   `json:"demo_name"`
	GuildID   string   `json:"guild_id"`
	SteamIDs  []string `json:"steam_ids"`
	ChannelID string   `json:"channel_id,omitempty"` // Optional, will use guild default if not provided
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
	
	// Validate required fields
	if payload.ShareCode == "" || payload.DemoName == "" || payload.GuildID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing required fields: share_code, demo_name, guild_id"})
		return
	}
	
	log.Printf("Processing demo ready for guild %s: %s", payload.GuildID, payload.ShareCode)
	
	// Ensure guild exists in database
	guild, err := ensureGuildExists(payload.GuildID)
	if err != nil {
		log.Printf("Error ensuring guild exists: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process guild"})
		return
	}
	
	// Process the match
	game, err := processMatchShare(payload.GuildID, payload.ShareCode, payload.DemoName, payload.SteamIDs)
	if err != nil {
		log.Printf("Error processing match: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process match"})
		return
	}
	
	// Send notification to Discord if session is available
	if webhookCtx != nil && webhookCtx.DiscordSession != nil {
		err = sendMatchNotification(guild, game, payload.ChannelID)
		if err != nil {
			log.Printf("Error sending Discord notification: %v", err)
			// Don't fail the webhook, just log the error
		}
	}
	
	log.Printf("Successfully processed match %s for guild %s", game.ShareCode, guild.GuildID)
	
	c.JSON(http.StatusOK, gin.H{
		"status":     "success",
		"game_uuid":  game.UUID.String(),
		"guild_uuid": guild.UUID.String(),
		"message":    "Match processed successfully",
	})
}

// sendMatchNotification sends a Discord notification about the new match
func sendMatchNotification(guild *Guild, game *Game, overrideChannelID string) error {
	if webhookCtx == nil || webhookCtx.DiscordSession == nil {
		return fmt.Errorf("Discord session not available")
	}
	
	// Determine which channel to use
	channelID := guild.ChannelID
	if overrideChannelID != "" {
		channelID = overrideChannelID
	}
	
	// Create embed for the match notification
	embed := &discordgo.MessageEmbed{
		Title: "🎮 New CS Match Demo Ready!",
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
			Text: "Match automatically processed via webhook",
		},
	}
	
	// Add player list if not too many
	if len(game.SteamIDs) > 0 && len(game.SteamIDs) <= 10 {
		playerList := ""
		for i, steamID := range game.SteamIDs {
			if i > 0 {
				playerList += "\n"
			}
			
			// Try to get user info from database
			_, err := getUserBySteamID(steamID)
			if err == nil {
				playerList += fmt.Sprintf("• %s", steamID)
			} else {
				playerList += fmt.Sprintf("• %s (unregistered)", steamID)
			}
		}
		
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Player Steam IDs",
			Value:  playerList,
			Inline: false,
		})
	}
	
	_, err := webhookCtx.DiscordSession.ChannelMessageSendEmbed(channelID, embed)
	return err
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