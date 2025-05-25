package main

import (
	"fmt"
	"log"

	"github.com/google/uuid"
)

// InitializeTables creates all necessary tables in the database
func initializeTables() error {
	_, err := db.Exec(CreateTablesSQL)
	if err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}
	log.Println("Database tables initialized successfully")
	return nil
}

// Guild database operations

// CreateGuild inserts a new guild into the database
func createGuild(guildID, channelID string) (*Guild, error) {
	guild := &Guild{
		UUID:      uuid.New(),
		GuildID:   guildID,
		ChannelID: channelID,
		UserIDs:   StringSlice{},
		GameIDs:   StringSlice{},
	}

	query := `
		INSERT INTO guilds (uuid, guild_id, channel_id, user_ids, game_ids)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at, updated_at`

	err := db.QueryRow(query, guild.UUID, guild.GuildID, guild.ChannelID, guild.UserIDs, guild.GameIDs).
		Scan(&guild.CreatedAt, &guild.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create guild: %w", err)
	}

	return guild, nil
}

// GetGuildByGuildID retrieves a guild by its Discord guild ID
func getGuildByGuildID(guildID string) (*Guild, error) {
	guild := &Guild{}
	query := `
		SELECT uuid, guild_id, channel_id, user_ids, game_ids, created_at, updated_at
		FROM guilds WHERE guild_id = $1`

	err := db.QueryRow(query, guildID).Scan(
		&guild.UUID, &guild.GuildID, &guild.ChannelID, &guild.UserIDs, &guild.GameIDs,
		&guild.CreatedAt, &guild.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get guild: %w", err)
	}

	return guild, nil
}

// UpdateGuild updates an existing guild
func updateGuild(guild *Guild) error {
	query := `
		UPDATE guilds 
		SET channel_id = $2, user_ids = $3, game_ids = $4
		WHERE uuid = $1`

	_, err := db.Exec(query, guild.UUID, guild.ChannelID, guild.UserIDs, guild.GameIDs)
	if err != nil {
		return fmt.Errorf("failed to update guild: %w", err)
	}

	return nil
}

// AddUserToGuild adds a user UUID to a guild's user list
func addUserToGuild(guildID string, userUUID uuid.UUID) error {
	query := `
		UPDATE guilds 
		SET user_ids = COALESCE(user_ids, '[]'::jsonb) || $2::jsonb
		WHERE guild_id = $1 AND NOT (user_ids @> $2::jsonb)`

	userJSON := fmt.Sprintf(`["%s"]`, userUUID.String())
	_, err := db.Exec(query, guildID, userJSON)
	if err != nil {
		return fmt.Errorf("failed to add user to guild: %w", err)
	}

	return nil
}

// AddGameToGuild adds a game UUID to a guild's game list
func addGameToGuild(guildID string, gameUUID uuid.UUID) error {
	query := `
		UPDATE guilds 
		SET game_ids = COALESCE(game_ids, '[]'::jsonb) || $2::jsonb
		WHERE guild_id = $1 AND NOT (game_ids @> $2::jsonb)`

	gameJSON := fmt.Sprintf(`["%s"]`, gameUUID.String())
	_, err := db.Exec(query, guildID, gameJSON)
	if err != nil {
		return fmt.Errorf("failed to add game to guild: %w", err)
	}

	return nil
}

// User database operations

// CreateUser inserts a new user into the database
func createUser(steamID, authCode, lastShareCode string) (*User, error) {
	user := &User{
		UUID:          uuid.New(),
		SteamID:       steamID,
		AuthCode:      authCode,
		LastShareCode: lastShareCode,
		GameIDs:       StringSlice{},
	}

	query := `
		INSERT INTO users (uuid, steam_id, auth_code, last_share_code, game_ids)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at, updated_at`

	err := db.QueryRow(query, user.UUID, user.SteamID, user.AuthCode, user.LastShareCode, user.GameIDs).
		Scan(&user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// GetUserBySteamID retrieves a user by their Steam ID
func getUserBySteamID(steamID string) (*User, error) {
	user := &User{}
	query := `
		SELECT uuid, steam_id, auth_code, last_share_code, game_ids, created_at, updated_at
		FROM users WHERE steam_id = $1`

	err := db.QueryRow(query, steamID).Scan(
		&user.UUID, &user.SteamID, &user.AuthCode, &user.LastShareCode, &user.GameIDs,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// GetUserByUUID retrieves a user by their UUID
func getUserByUUID(userUUID uuid.UUID) (*User, error) {
	user := &User{}
	query := `
		SELECT uuid, steam_id, auth_code, last_share_code, game_ids, created_at, updated_at
		FROM users WHERE uuid = $1`

	err := db.QueryRow(query, userUUID).Scan(
		&user.UUID, &user.SteamID, &user.AuthCode, &user.LastShareCode, &user.GameIDs,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// UpdateUser updates an existing user
func updateUser(user *User) error {
	query := `
		UPDATE users 
		SET auth_code = $2, last_share_code = $3, game_ids = $4
		WHERE uuid = $1`

	_, err := db.Exec(query, user.UUID, user.AuthCode, user.LastShareCode, user.GameIDs)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// AddGameToUser adds a game UUID to a user's game list
func addGameToUser(steamID string, gameUUID uuid.UUID) error {
	query := `
		UPDATE users 
		SET game_ids = COALESCE(game_ids, '[]'::jsonb) || $2::jsonb
		WHERE steam_id = $1 AND NOT (game_ids @> $2::jsonb)`

	gameJSON := fmt.Sprintf(`["%s"]`, gameUUID.String())
	_, err := db.Exec(query, steamID, gameJSON)
	if err != nil {
		return fmt.Errorf("failed to add game to user: %w", err)
	}

	return nil
}

// Game database operations

// CreateGame inserts a new game into the database
func createGame(shareCode, demoName string, steamIDs []string) (*Game, error) {
	game := &Game{
		UUID:      uuid.New(),
		ShareCode: shareCode,
		DemoName:  demoName,
		SteamIDs:  StringSlice(steamIDs),
	}

	query := `
		INSERT INTO games (uuid, share_code, demo_name, steam_ids)
		VALUES ($1, $2, $3, $4)
		RETURNING created_at, updated_at`

	err := db.QueryRow(query, game.UUID, game.ShareCode, game.DemoName, game.SteamIDs).
		Scan(&game.CreatedAt, &game.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create game: %w", err)
	}

	return game, nil
}

// GetGameByShareCode retrieves a game by its share code
func getGameByShareCode(shareCode string) (*Game, error) {
	game := &Game{}
	query := `
		SELECT uuid, share_code, demo_name, steam_ids, created_at, updated_at
		FROM games WHERE share_code = $1`

	err := db.QueryRow(query, shareCode).Scan(
		&game.UUID, &game.ShareCode, &game.DemoName, &game.SteamIDs,
		&game.CreatedAt, &game.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get game: %w", err)
	}

	return game, nil
}

// GetGameByUUID retrieves a game by its UUID
func getGameByUUID(gameUUID uuid.UUID) (*Game, error) {
	game := &Game{}
	query := `
		SELECT uuid, share_code, demo_name, steam_ids, created_at, updated_at
		FROM games WHERE uuid = $1`

	err := db.QueryRow(query, gameUUID).Scan(
		&game.UUID, &game.ShareCode, &game.DemoName, &game.SteamIDs,
		&game.CreatedAt, &game.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get game: %w", err)
	}

	return game, nil
}

// UpdateGame updates an existing game
func updateGame(game *Game) error {
	query := `
		UPDATE games 
		SET demo_name = $2, steam_ids = $3
		WHERE uuid = $1`

	_, err := db.Exec(query, game.UUID, game.DemoName, game.SteamIDs)
	if err != nil {
		return fmt.Errorf("failed to update game: %w", err)
	}

	return nil
}

// GetGamesBySteamID retrieves all games that include a specific Steam ID
func getGamesBySteamID(steamID string) ([]*Game, error) {
	query := `
		SELECT uuid, share_code, demo_name, steam_ids, created_at, updated_at
		FROM games WHERE steam_ids @> $1::jsonb`

	steamIDJSON := fmt.Sprintf(`["%s"]`, steamID)
	rows, err := db.Query(query, steamIDJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to get games by steam ID: %w", err)
	}
	defer rows.Close()

	var games []*Game
	for rows.Next() {
		game := &Game{}
		err := rows.Scan(
			&game.UUID, &game.ShareCode, &game.DemoName, &game.SteamIDs,
			&game.CreatedAt, &game.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan game: %w", err)
		}
		games = append(games, game)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over games: %w", err)
	}

	return games, nil
}

// GetAllUsers retrieves all users for polling
func getAllUsers() ([]*User, error) {
	query := `
		SELECT uuid, steam_id, auth_code, last_share_code, game_ids, created_at, updated_at
		FROM users ORDER BY created_at`

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all users: %w", err)
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		user := &User{}
		err := rows.Scan(
			&user.UUID, &user.SteamID, &user.AuthCode, &user.LastShareCode, &user.GameIDs,
			&user.CreatedAt, &user.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over users: %w", err)
	}

	return users, nil
}

// DeleteUser removes a user from the database
func deleteUser(steamID string) error {
	// First get the user to remove from guilds
	user, err := getUserBySteamID(steamID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Remove user from all guilds
	_, err = db.Exec(`
		UPDATE guilds 
		SET user_ids = user_ids - $1::text
		WHERE user_ids @> jsonb_build_array($1::text)`,
		user.UUID.String())
	if err != nil {
		return fmt.Errorf("failed to remove user from guilds: %w", err)
	}

	// Delete the user
	_, err = db.Exec(`DELETE FROM users WHERE steam_id = $1`, steamID)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}

// UpdateUserLastShareCode updates only the last share code for a user
func updateUserLastShareCode(steamID, shareCode string) error {
	query := `
		UPDATE users 
		SET last_share_code = $2
		WHERE steam_id = $1`

	_, err := db.Exec(query, steamID, shareCode)
	if err != nil {
		return fmt.Errorf("failed to update user last share code: %w", err)
	}

	return nil
}

// GetGamesForGuild retrieves all games associated with a guild
func getGamesForGuild(guildID string) ([]*Game, error) {
	query := `
		SELECT g.uuid, g.share_code, g.demo_name, g.steam_ids, g.created_at, g.updated_at
		FROM games g
		JOIN guilds guild ON guild.game_ids @> jsonb_build_array(g.uuid::text)
		WHERE guild.guild_id = $1`

	rows, err := db.Query(query, guildID)
	if err != nil {
		return nil, fmt.Errorf("failed to get games for guild: %w", err)
	}
	defer rows.Close()

	var games []*Game
	for rows.Next() {
		game := &Game{}
		err := rows.Scan(
			&game.UUID, &game.ShareCode, &game.DemoName, &game.SteamIDs,
			&game.CreatedAt, &game.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan game: %w", err)
		}
		games = append(games, game)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over games: %w", err)
	}

	return games, nil
}