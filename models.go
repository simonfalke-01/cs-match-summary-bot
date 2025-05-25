package main

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// StringSlice is a custom type for handling string slices in PostgreSQL
type StringSlice []string

func (s StringSlice) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	return json.Marshal(s)
}

func (s *StringSlice) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("cannot scan non-[]byte value into StringSlice")
	}

	return json.Unmarshal(bytes, s)
}

// Guild represents a Discord guild with associated users and games
type Guild struct {
	UUID      uuid.UUID   `json:"uuid" db:"uuid"`
	GuildID   string      `json:"guild_id" db:"guild_id"`
	ChannelID string      `json:"channel_id" db:"channel_id"`
	UserIDs   StringSlice `json:"user_ids" db:"user_ids"`
	GameIDs   StringSlice `json:"game_ids" db:"game_ids"`
	CreatedAt time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt time.Time   `json:"updated_at" db:"updated_at"`
}

// User represents a user with Steam integration
type User struct {
	UUID          uuid.UUID   `json:"uuid" db:"uuid"`
	SteamID       string      `json:"steam_id" db:"steam_id"`
	AuthCode      string      `json:"auth_code" db:"auth_code"`
	LastShareCode string      `json:"last_share_code" db:"last_share_code"`
	GameIDs       StringSlice `json:"game_ids" db:"game_ids"`
	CreatedAt     time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at" db:"updated_at"`
}

// Game represents a CS match with demo information
type Game struct {
	UUID      uuid.UUID   `json:"uuid" db:"uuid"`
	ShareCode string      `json:"share_code" db:"share_code"`
	DemoName  string      `json:"demo_name" db:"demo_name"`
	SteamIDs  StringSlice `json:"steam_ids" db:"steam_ids"`
	CreatedAt time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt time.Time   `json:"updated_at" db:"updated_at"`
}

// CreateTablesSQL contains the SQL statements to create all tables
const CreateTablesSQL = `
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS guilds (
    uuid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    guild_id VARCHAR(255) UNIQUE NOT NULL,
    channel_id VARCHAR(255) NOT NULL,
    user_ids JSONB DEFAULT '[]',
    game_ids JSONB DEFAULT '[]',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS users (
    uuid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    steam_id VARCHAR(255) UNIQUE NOT NULL,
    auth_code VARCHAR(255) NOT NULL,
    last_share_code VARCHAR(255) DEFAULT '',
    game_ids JSONB DEFAULT '[]',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS games (
    uuid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    share_code VARCHAR(255) UNIQUE NOT NULL,
    demo_name VARCHAR(255) NOT NULL,
    steam_ids JSONB DEFAULT '[]',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_guilds_guild_id ON guilds(guild_id);
CREATE INDEX IF NOT EXISTS idx_users_steam_id ON users(steam_id);
CREATE INDEX IF NOT EXISTS idx_games_share_code ON games(share_code);

-- Create triggers to automatically update updated_at timestamps
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE OR REPLACE TRIGGER update_guilds_updated_at BEFORE UPDATE ON guilds FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE OR REPLACE TRIGGER update_users_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE OR REPLACE TRIGGER update_games_updated_at BEFORE UPDATE ON games FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
`