package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

// RegisterSlashCommands registers all slash commands with Discord
func registerSlashCommands(s *discordgo.Session) error {
	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "register",
			Description: "Register a new user with Steam ID, auth code, and last share code",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "steam_id",
					Description: "Your Steam ID",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "auth_code",
					Description: "Your Steam authentication code",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "last_share_code",
					Description: "Your last known CS match share code",
					Required:    true,
				},
			},
		},
		{
			Name:        "remove",
			Description: "Remove a user from the system",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "steam_id",
					Description: "Steam ID of the user to remove",
					Required:    true,
				},
			},
		},
		{
			Name:        "users",
			Description: "Show list of registered users",
		},
		{
			Name:                     "set_channel",
			Description:              "Set the channel for match summaries (Admin only)",
			DefaultMemberPermissions: &[]int64{discordgo.PermissionManageGuild}[0],
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionChannel,
					Name:        "channel",
					Description: "Channel to send match summaries to",
					Required:    false,
					ChannelTypes: []discordgo.ChannelType{
						discordgo.ChannelTypeGuildText,
					},
				},
			},
		},
	}

	for _, command := range commands {
		_, err := s.ApplicationCommandCreate(s.State.User.ID, "", command)
		if err != nil {
			return fmt.Errorf("failed to create command %s: %w", command.Name, err)
		}
		log.Printf("Registered slash command: %s", command.Name)
	}

	return nil
}

// HandleSlashCommand handles incoming slash command interactions
func handleSlashCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.ApplicationCommandData().Name == "" {
		return
	}

	switch i.ApplicationCommandData().Name {
	case "register":
		handleRegisterSlashCommand(s, i)
	case "remove":
		handleRemoveSlashCommand(s, i)
	case "users":
		handleUsersSlashCommand(s, i)
	case "set_channel":
		handleSetChannelSlashCommand(s, i)
	}
}

func handleRegisterSlashCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	
	var steamID, authCode, lastShareCode string
	for _, option := range options {
		switch option.Name {
		case "steam_id":
			steamID = option.StringValue()
		case "auth_code":
			authCode = option.StringValue()
		case "last_share_code":
			lastShareCode = option.StringValue()
		}
	}

	// Basic validation
	if steamID == "" || authCode == "" || lastShareCode == "" {
		respondWithError(s, i, "All fields are required")
		return
	}

	// Validate share code format
	if !strings.HasPrefix(lastShareCode, "CSGO-") {
		respondWithError(s, i, "Invalid share code format. Must start with 'CSGO-'")
		return
	}

	// Ensure guild exists
	_, err := ensureGuildExists(i.GuildID)
	if err != nil {
		log.Printf("Error ensuring guild exists: %v", err)
		respondWithError(s, i, "Failed to process guild")
		return
	}

	// Check if user already exists
	existingUser, err := getUserBySteamID(steamID)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("Error checking existing user: %v", err)
		respondWithError(s, i, "Failed to check user registration")
		return
	}

	if err != sql.ErrNoRows {
		// User exists, update their info
		existingUser.AuthCode = authCode
		existingUser.LastShareCode = lastShareCode
		err = updateUser(existingUser)
		if err != nil {
			log.Printf("Error updating user: %v", err)
			respondWithError(s, i, "Failed to update user")
			return
		}

		// Add user to guild if not already added
		err = addUserToGuild(i.GuildID, existingUser.UUID)
		if err != nil {
			log.Printf("Error adding user to guild: %v", err)
		}

		respondWithSuccess(s, i, fmt.Sprintf("✅ User updated successfully!\n**Steam ID:** %s\n**Last Share Code:** %s", steamID, lastShareCode))
		return
	}

	// Create new user
	user, err := createUser(steamID, authCode, lastShareCode)
	if err != nil {
		log.Printf("Error creating user: %v", err)
		respondWithError(s, i, "Failed to create user")
		return
	}

	// Add user to guild
	err = addUserToGuild(i.GuildID, user.UUID)
	if err != nil {
		log.Printf("Error adding user to guild: %v", err)
		respondWithError(s, i, "User created but failed to add to guild")
		return
	}

	respondWithSuccess(s, i, fmt.Sprintf("✅ User registered successfully!\n**Steam ID:** %s\n**UUID:** %s\n**Last Share Code:** %s", user.SteamID, user.UUID, user.LastShareCode))
}

func handleRemoveSlashCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	if len(options) == 0 {
		respondWithError(s, i, "Steam ID is required")
		return
	}

	steamID := options[0].StringValue()

	// Check if user exists
	_, err := getUserBySteamID(steamID)
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(s, i, "User not found")
		} else {
			log.Printf("Error checking user: %v", err)
			respondWithError(s, i, "Failed to check user")
		}
		return
	}

	// Delete user
	err = deleteUser(steamID)
	if err != nil {
		log.Printf("Error deleting user: %v", err)
		respondWithError(s, i, "Failed to remove user")
		return
	}

	respondWithSuccess(s, i, fmt.Sprintf("✅ User with Steam ID %s has been removed", steamID))
}

func handleUsersSlashCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	guild, err := getGuildByGuildID(i.GuildID)
	if err != nil {
		log.Printf("Error getting guild: %v", err)
		respondWithError(s, i, "Failed to get guild information")
		return
	}

	if len(guild.UserIDs) == 0 {
		respondWithSuccess(s, i, "📝 No users registered in this guild.")
		return
	}

	var userInfo []string
	userCount := 0
	maxUsers := 25 // Discord embed field limit

	for _, userIDStr := range guild.UserIDs {
		if userCount >= maxUsers {
			userInfo = append(userInfo, fmt.Sprintf("... and %d more users", len(guild.UserIDs)-maxUsers))
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

		userInfo = append(userInfo, fmt.Sprintf("• **%s** - Last: `%s`", user.SteamID, user.LastShareCode))
		userCount++
	}

	embed := &discordgo.MessageEmbed{
		Title:       "👥 Registered Users",
		Description: strings.Join(userInfo, "\n"),
		Color:       0x0099ff,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Total: %d users", len(guild.UserIDs)),
		},
	}

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
	if err != nil {
		log.Printf("Error responding to users command: %v", err)
	}
}

func handleSetChannelSlashCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Get channel from options or use current channel
	var channelID string
	options := i.ApplicationCommandData().Options
	
	if len(options) > 0 && options[0].ChannelValue(s) != nil {
		channelID = options[0].ChannelValue(s).ID
	} else {
		channelID = i.ChannelID
	}

	// Update guild channel
	err := updateGuildChannel(i.GuildID, channelID)
	if err != nil {
		log.Printf("Error updating guild channel: %v", err)
		respondWithError(s, i, "Failed to update channel")
		return
	}

	respondWithSuccess(s, i, fmt.Sprintf("✅ Bot notification channel updated to <#%s>", channelID))
}

func respondWithError(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "❌ " + message,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		log.Printf("Error responding with error: %v", err)
	}
}

func respondWithSuccess(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: message,
		},
	})
	if err != nil {
		log.Printf("Error responding with success: %v", err)
	}
}