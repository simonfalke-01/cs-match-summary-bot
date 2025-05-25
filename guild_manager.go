package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

// ensureGuildExists checks if a guild exists in the database, creates it if not
func ensureGuildExists(guildID string) (*Guild, error) {
	guild, err := getGuildByGuildID(guildID)
	if err != nil {
		if err == sql.ErrNoRows {
			// Guild doesn't exist, create it with a default channel
			return createGuild(guildID, guildID) // Use guild ID as temporary channel ID
		}
		return nil, err
	}
	return guild, nil
}

// updateGuildChannel updates the channel for guild notifications
func updateGuildChannel(guildID, channelID string) error {
	guild, err := ensureGuildExists(guildID)
	if err != nil {
		return err
	}
	
	guild.ChannelID = channelID
	return updateGuild(guild)
}

// getGuildStats returns statistics about a guild
func getGuildStats(guildID string) (map[string]int, error) {
	guild, err := getGuildByGuildID(guildID)
	if err != nil {
		return nil, err
	}
	
	stats := map[string]int{
		"users": len(guild.UserIDs),
		"games": len(guild.GameIDs),
	}
	
	return stats, nil
}

// registerUserToGuild registers a Steam user to a guild
func registerUserToGuild(guildID, steamID, authCode string) (*User, error) {
	// Ensure guild exists
	_, err := ensureGuildExists(guildID)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure guild exists: %w", err)
	}
	
	// Check if user already exists
	user, err := getUserBySteamID(steamID)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}
	
	// Create user if doesn't exist
	if err == sql.ErrNoRows {
		user, err = createUser(steamID, authCode)
		if err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}
	} else {
		// Update auth code if user exists
		user.AuthCode = authCode
		err = updateUser(user)
		if err != nil {
			return nil, fmt.Errorf("failed to update user: %w", err)
		}
	}
	
	// Add user to guild
	err = addUserToGuild(guildID, user.UUID)
	if err != nil {
		return nil, fmt.Errorf("failed to add user to guild: %w", err)
	}
	
	return user, nil
}

// processMatchShare processes a match share code and adds it to the guild
func processMatchShare(guildID, shareCode, demoName string, steamIDs []string) (*Game, error) {
	// Ensure guild exists
	_, err := ensureGuildExists(guildID)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure guild exists: %w", err)
	}
	
	// Check if game already exists
	game, err := getGameByShareCode(shareCode)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to check existing game: %w", err)
	}
	
	// Create game if doesn't exist
	if err == sql.ErrNoRows {
		game, err = createGame(shareCode, demoName, steamIDs)
		if err != nil {
			return nil, fmt.Errorf("failed to create game: %w", err)
		}
	} else {
		// Update demo name if game exists
		game.DemoName = demoName
		err = updateGame(game)
		if err != nil {
			return nil, fmt.Errorf("failed to update game: %w", err)
		}
	}
	
	// Add game to guild
	err = addGameToGuild(guildID, game.UUID)
	if err != nil {
		return nil, fmt.Errorf("failed to add game to guild: %w", err)
	}
	
	// Add game to all users who participated
	for _, steamID := range steamIDs {
		err = addGameToUser(steamID, game.UUID)
		if err != nil {
			log.Printf("Warning: failed to add game to user %s: %v", steamID, err)
		}
	}
	
	return game, nil
}

// handleAdminCommand processes admin commands from Discord
func handleAdminCommand(s *discordgo.Session, m *discordgo.MessageCreate, command string, args []string) {
	// Check if user has admin permissions
	member, err := s.GuildMember(m.GuildID, m.Author.ID)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "❌ Error checking permissions.")
		return
	}
	
	hasAdminPerms := false
	for _, roleID := range member.Roles {
		role, err := s.State.Role(m.GuildID, roleID)
		if err != nil {
			continue
		}
		if role.Permissions&discordgo.PermissionAdministrator != 0 || 
		   role.Permissions&discordgo.PermissionManageGuild != 0 {
			hasAdminPerms = true
			break
		}
	}
	
	if !hasAdminPerms {
		s.ChannelMessageSend(m.ChannelID, "❌ You need Administrator or Manage Server permissions to use this command.")
		return
	}
	
	switch command {
	case "setchannel":
		handleSetChannel(s, m, args)
	case "stats":
		handleStats(s, m)
	case "register":
		handleRegister(s, m, args)
	case "addmatch":
		handleAddMatch(s, m, args)
	case "listusers":
		handleListUsers(s, m)
	case "listgames":
		handleListGames(s, m)
	default:
		s.ChannelMessageSend(m.ChannelID, "❌ Unknown admin command. Use `!cs help` for available commands.")
	}
}

func handleSetChannel(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	var channelID string
	
	if len(args) > 0 {
		// Try to parse channel mention or ID
		channelID = strings.Trim(args[0], "<>#")
	} else {
		// Use current channel
		channelID = m.ChannelID
	}
	
	err := updateGuildChannel(m.GuildID, channelID)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("❌ Error updating channel: %v", err))
		return
	}
	
	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("✅ Bot channel updated to <#%s>", channelID))
}

func handleStats(s *discordgo.Session, m *discordgo.MessageCreate) {
	stats, err := getGuildStats(m.GuildID)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("❌ Error getting stats: %v", err))
		return
	}
	
	embed := &discordgo.MessageEmbed{
		Title: "📊 Guild Statistics",
		Color: 0x00ff00,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Registered Users",
				Value:  fmt.Sprintf("%d", stats["users"]),
				Inline: true,
			},
			{
				Name:   "Tracked Games",
				Value:  fmt.Sprintf("%d", stats["games"]),
				Inline: true,
			},
		},
	}
	
	s.ChannelMessageSendEmbed(m.ChannelID, embed)
}

func handleRegister(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	if len(args) < 2 {
		s.ChannelMessageSend(m.ChannelID, "❌ Usage: `!cs register <steam_id> <auth_code>`")
		return
	}
	
	steamID := args[0]
	authCode := args[1]
	
	user, err := registerUserToGuild(m.GuildID, steamID, authCode)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("❌ Error registering user: %v", err))
		return
	}
	
	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("✅ User registered successfully!\n**Steam ID:** %s\n**UUID:** %s", user.SteamID, user.UUID))
}

func handleAddMatch(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	if len(args) < 2 {
		s.ChannelMessageSend(m.ChannelID, "❌ Usage: `!cs addmatch <share_code> <demo_name> [steam_id1] [steam_id2] ...`")
		return
	}
	
	shareCode := args[0]
	demoName := args[1]
	steamIDs := args[2:]
	
	game, err := processMatchShare(m.GuildID, shareCode, demoName, steamIDs)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("❌ Error adding match: %v", err))
		return
	}
	
	embed := &discordgo.MessageEmbed{
		Title: "🎮 Match Added Successfully",
		Color: 0x00ff00,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Share Code",
				Value:  game.ShareCode,
				Inline: true,
			},
			{
				Name:   "Demo File",
				Value:  game.DemoName,
				Inline: true,
			},
			{
				Name:   "Players",
				Value:  fmt.Sprintf("%d players", len(game.SteamIDs)),
				Inline: true,
			},
		},
	}
	
	s.ChannelMessageSendEmbed(m.ChannelID, embed)
}

func handleListUsers(s *discordgo.Session, m *discordgo.MessageCreate) {
	guild, err := getGuildByGuildID(m.GuildID)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("❌ Error getting guild: %v", err))
		return
	}
	
	if len(guild.UserIDs) == 0 {
		s.ChannelMessageSend(m.ChannelID, "📝 No users registered in this guild.")
		return
	}
	
	var userInfo []string
	for i, userIDStr := range guild.UserIDs {
		if i >= 10 { // Limit to first 10 users
			userInfo = append(userInfo, fmt.Sprintf("... and %d more", len(guild.UserIDs)-10))
			break
		}
		
		userUUID, err := uuid.Parse(userIDStr)
		if err != nil {
			continue
		}
		
		user, err := getUserByUUID(userUUID)
		if err != nil {
			continue
		}
		
		userInfo = append(userInfo, fmt.Sprintf("• Steam ID: `%s`", user.SteamID))
	}
	
	embed := &discordgo.MessageEmbed{
		Title:       "👥 Registered Users",
		Description: strings.Join(userInfo, "\n"),
		Color:       0x0099ff,
	}
	
	s.ChannelMessageSendEmbed(m.ChannelID, embed)
}

func handleListGames(s *discordgo.Session, m *discordgo.MessageCreate) {
	games, err := getGamesForGuild(m.GuildID)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("❌ Error getting games: %v", err))
		return
	}
	
	if len(games) == 0 {
		s.ChannelMessageSend(m.ChannelID, "📝 No games tracked in this guild.")
		return
	}
	
	var gameInfo []string
	for i, game := range games {
		if i >= 10 { // Limit to first 10 games
			gameInfo = append(gameInfo, fmt.Sprintf("... and %d more", len(games)-10))
			break
		}
		
		gameInfo = append(gameInfo, fmt.Sprintf("• **%s** - %s (%d players)", 
			game.ShareCode, game.DemoName, len(game.SteamIDs)))
	}
	
	embed := &discordgo.MessageEmbed{
		Title:       "🎮 Tracked Games",
		Description: strings.Join(gameInfo, "\n"),
		Color:       0xff9900,
	}
	
	s.ChannelMessageSendEmbed(m.ChannelID, embed)
}

func handleHelpCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	embed := &discordgo.MessageEmbed{
		Title: "🎮 CS Match Summary Bot - Commands",
		Color: 0x0099ff,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:  "General Commands",
				Value: "`!cs help` - Show this help message\n`!cs ping` - Test bot responsiveness",
			},
			{
				Name:  "Admin Commands (Requires Admin/Manage Server)",
				Value: "`!cs setchannel [#channel]` - Set notification channel\n" +
					   "`!cs stats` - Show guild statistics\n" +
					   "`!cs register <steam_id> <auth_code>` - Register a user\n" +
					   "`!cs addmatch <share_code> <demo_name> [steam_ids...]` - Add a match\n" +
					   "`!cs listusers` - List registered users\n" +
					   "`!cs listgames` - List tracked games",
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "CS Match Summary Bot - Track your matches with ease!",
		},
	}
	
	s.ChannelMessageSendEmbed(m.ChannelID, embed)
}