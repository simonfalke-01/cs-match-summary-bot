package main

import (
	"fmt"
	"log"

	"github.com/google/uuid"
)

// ExampleUsage demonstrates how to use the Guild, User, and Game models
func ExampleUsage() {
	// Example 1: Creating a new guild
	fmt.Println("=== Guild Operations ===")
	
	// Create a new guild
	guild, err := createGuild("123456789012345678", "987654321098765432")
	if err != nil {
		log.Printf("Error creating guild: %v", err)
		return
	}
	fmt.Printf("Created guild: %+v\n", guild)
	
	// Retrieve guild by Discord guild ID
	retrievedGuild, err := getGuildByGuildID("123456789012345678")
	if err != nil {
		log.Printf("Error retrieving guild: %v", err)
	} else {
		fmt.Printf("Retrieved guild: %+v\n", retrievedGuild)
	}

	// Example 2: Creating users
	fmt.Println("\n=== User Operations ===")
	
	// Create first user
	user1, err := createUser("76561198000000001", "auth_code_123", "CSGO-ABCDE-FGHIJ-KLMNO-PQRST")
	if err != nil {
		log.Printf("Error creating user1: %v", err)
		return
	}
	fmt.Printf("Created user1: %+v\n", user1)
	
	// Create second user
	user2, err := createUser("76561198000000002", "auth_code_456", "CSGO-ZYXWV-UTSRQ-PONML-KJIHG")
	if err != nil {
		log.Printf("Error creating user2: %v", err)
		return
	}
	fmt.Printf("Created user2: %+v\n", user2)
	
	// Retrieve user by Steam ID
	retrievedUser, err := getUserBySteamID("76561198000000001")
	if err != nil {
		log.Printf("Error retrieving user: %v", err)
	} else {
		fmt.Printf("Retrieved user: %+v\n", retrievedUser)
	}

	// Example 3: Creating a game
	fmt.Println("\n=== Game Operations ===")
	
	// Create a new game with multiple Steam IDs
	steamIDs := []string{"76561198000000001", "76561198000000002", "76561198000000003"}
	game, err := createGame("CSGO-XXXXX-XXXXX-XXXXX-XXXXX", "match_2024_01_15_001.dem", steamIDs)
	if err != nil {
		log.Printf("Error creating game: %v", err)
		return
	}
	fmt.Printf("Created game: %+v\n", game)
	
	// Retrieve game by share code
	retrievedGame, err := getGameByShareCode("CSGO-XXXXX-XXXXX-XXXXX-XXXXX")
	if err != nil {
		log.Printf("Error retrieving game: %v", err)
	} else {
		fmt.Printf("Retrieved game: %+v\n", retrievedGame)
	}

	// Example 4: Linking entities together
	fmt.Println("\n=== Linking Operations ===")
	
	// Add users to guild
	err = addUserToGuild(guild.GuildID, user1.UUID)
	if err != nil {
		log.Printf("Error adding user1 to guild: %v", err)
	} else {
		fmt.Printf("Added user1 to guild\n")
	}
	
	err = addUserToGuild(guild.GuildID, user2.UUID)
	if err != nil {
		log.Printf("Error adding user2 to guild: %v", err)
	} else {
		fmt.Printf("Added user2 to guild\n")
	}
	
	// Add game to guild
	err = addGameToGuild(guild.GuildID, game.UUID)
	if err != nil {
		log.Printf("Error adding game to guild: %v", err)
	} else {
		fmt.Printf("Added game to guild\n")
	}
	
	// Add game to users
	err = addGameToUser(user1.SteamID, game.UUID)
	if err != nil {
		log.Printf("Error adding game to user1: %v", err)
	} else {
		fmt.Printf("Added game to user1\n")
	}
	
	err = addGameToUser(user2.SteamID, game.UUID)
	if err != nil {
		log.Printf("Error adding game to user2: %v", err)
	} else {
		fmt.Printf("Added game to user2\n")
	}

	// Example 5: Querying related data
	fmt.Println("\n=== Query Operations ===")
	
	// Get all games for a specific Steam ID
	userGames, err := getGamesBySteamID("76561198000000001")
	if err != nil {
		log.Printf("Error getting games for Steam ID: %v", err)
	} else {
		fmt.Printf("Games for Steam ID 76561198000000001: %d games\n", len(userGames))
		for _, g := range userGames {
			fmt.Printf("  - Game: %s (Demo: %s)\n", g.ShareCode, g.DemoName)
		}
	}
	
	// Get all games for a guild
	guildGames, err := getGamesForGuild(guild.GuildID)
	if err != nil {
		log.Printf("Error getting games for guild: %v", err)
	} else {
		fmt.Printf("Games for guild %s: %d games\n", guild.GuildID, len(guildGames))
		for _, g := range guildGames {
			fmt.Printf("  - Game: %s (Demo: %s)\n", g.ShareCode, g.DemoName)
		}
	}
	
	// Update operations example
	fmt.Println("\n=== Update Operations ===")
	
	// Update user's auth code
	user1.AuthCode = "new_auth_code_789"
	err = updateUser(user1)
	if err != nil {
		log.Printf("Error updating user: %v", err)
	} else {
		fmt.Printf("Updated user1 auth code\n")
	}
	
	// Update game's demo name
	game.DemoName = "match_2024_01_15_001_processed.dem"
	err = updateGame(game)
	if err != nil {
		log.Printf("Error updating game: %v", err)
	} else {
		fmt.Printf("Updated game demo name\n")
	}
	
	// Update guild's channel ID
	guild.ChannelID = "111111111111111111"
	err = updateGuild(guild)
	if err != nil {
		log.Printf("Error updating guild: %v", err)
	} else {
		fmt.Printf("Updated guild channel ID\n")
	}
}

// ExampleBatchOperations demonstrates batch operations and more complex queries
func ExampleBatchOperations() {
	fmt.Println("\n=== Batch Operations Example ===")
	
	// Create multiple games for the same match but different rounds
	baseShareCode := "CSGO-BATCH-XXXXX-XXXXX-XXXXX"
	steamIDs := []string{"76561198000000001", "76561198000000002", "76561198000000003", "76561198000000004", "76561198000000005"}
	
	var gameUUIDs []uuid.UUID
	for i := 1; i <= 3; i++ {
		shareCode := fmt.Sprintf("%s-%d", baseShareCode, i)
		demoName := fmt.Sprintf("match_batch_round_%d.dem", i)
		
		game, err := createGame(shareCode, demoName, steamIDs)
		if err != nil {
			log.Printf("Error creating batch game %d: %v", i, err)
			continue
		}
		
		gameUUIDs = append(gameUUIDs, game.UUID)
		fmt.Printf("Created batch game %d: %s\n", i, game.ShareCode)
		
		// Add all games to all users
		for _, steamID := range steamIDs {
			err = addGameToUser(steamID, game.UUID)
			if err != nil {
				log.Printf("Error adding game to user %s: %v", steamID, err)
			}
		}
	}
	
	// Now query games for one of the Steam IDs to see all their games
	allUserGames, err := getGamesBySteamID("76561198000000001")
	if err != nil {
		log.Printf("Error getting all games for user: %v", err)
	} else {
		fmt.Printf("Total games for Steam ID 76561198000000001: %d\n", len(allUserGames))
	}
}

// ExampleErrorHandling demonstrates proper error handling patterns
func ExampleErrorHandling() {
	fmt.Println("\n=== Error Handling Examples ===")
	
	// Try to get a non-existent guild
	_, err := getGuildByGuildID("nonexistent_guild_id")
	if err != nil {
		fmt.Printf("Expected error for non-existent guild: %v\n", err)
	}
	
	// Try to get a non-existent user
	_, err = getUserBySteamID("nonexistent_steam_id")
	if err != nil {
		fmt.Printf("Expected error for non-existent user: %v\n", err)
	}
	
	// Try to get a non-existent game
	_, err = getGameByShareCode("nonexistent_share_code")
	if err != nil {
		fmt.Printf("Expected error for non-existent game: %v\n", err)
	}
	
	// Try to create duplicate guild (will fail on unique constraint)
	_, err = createGuild("123456789012345678", "987654321098765432")
	if err != nil {
		fmt.Printf("Expected error for duplicate guild: %v\n", err)
	}
}