-- Database setup script for CS Match Summary Bot
-- Run this script as a PostgreSQL superuser to create the database and user

-- Create database
CREATE DATABASE cs;

-- Create user (optional - you can use existing postgres user)
-- CREATE USER cs_bot WITH PASSWORD 'your_password_here';

-- Grant privileges
-- GRANT ALL PRIVILEGES ON DATABASE cs TO cs_bot;

-- Connect to the cs database
\c cs;

-- Create UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create tables
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