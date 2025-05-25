#!/bin/bash

# Quick database setup script for CS Match Summary Bot
# This script creates the PostgreSQL database and tables

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Setting up CS Match Summary Bot database...${NC}"

# Load environment variables if .env exists
if [ -f .env ]; then
    export $(cat .env | xargs)
fi

# Set defaults if not provided
DB_HOST=${DB_HOST:-localhost}
DB_PORT=${DB_PORT:-5432}
DB_USER=${DB_USER:-postgres}
DB_PASSWORD=${DB_PASSWORD:-postgres}
DB_NAME=${DB_NAME:-cs}

echo "Database configuration:"
echo "  Host: $DB_HOST"
echo "  Port: $DB_PORT"
echo "  User: $DB_USER"
echo "  Database: $DB_NAME"
echo ""

# Function to run psql command
run_psql() {
    PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d postgres -c "$1"
}

# Function to run psql command on target database
run_psql_db() {
    PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "$1"
}

# Check if PostgreSQL is running
echo -e "${YELLOW}Checking PostgreSQL connection...${NC}"
if ! PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d postgres -c '\q' 2>/dev/null; then
    echo -e "${RED}Error: Cannot connect to PostgreSQL server${NC}"
    echo "Please ensure PostgreSQL is running and credentials are correct"
    exit 1
fi
echo -e "${GREEN}✓ PostgreSQL connection successful${NC}"

# Check if database exists
echo -e "${YELLOW}Checking if database '$DB_NAME' exists...${NC}"
DB_EXISTS=$(PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname='$DB_NAME'")

if [ "$DB_EXISTS" = "1" ]; then
    echo -e "${GREEN}✓ Database '$DB_NAME' already exists${NC}"
else
    echo -e "${YELLOW}Creating database '$DB_NAME'...${NC}"
    if run_psql "CREATE DATABASE $DB_NAME;"; then
        echo -e "${GREEN}✓ Database '$DB_NAME' created successfully${NC}"
    else
        echo -e "${RED}Error: Failed to create database '$DB_NAME'${NC}"
        exit 1
    fi
fi

# Create UUID extension
echo -e "${YELLOW}Creating UUID extension...${NC}"
if run_psql_db "CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";"; then
    echo -e "${GREEN}✓ UUID extension created${NC}"
else
    echo -e "${RED}Error: Failed to create UUID extension${NC}"
    exit 1
fi

# Create tables
echo -e "${YELLOW}Creating tables...${NC}"

# Create guilds table
if run_psql_db "CREATE TABLE IF NOT EXISTS guilds (
    uuid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    guild_id VARCHAR(255) UNIQUE NOT NULL,
    channel_id VARCHAR(255) NOT NULL,
    user_ids JSONB DEFAULT '[]',
    game_ids JSONB DEFAULT '[]',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);"; then
    echo -e "${GREEN}✓ Guilds table created${NC}"
else
    echo -e "${RED}Error: Failed to create guilds table${NC}"
    exit 1
fi

# Create users table
if run_psql_db "CREATE TABLE IF NOT EXISTS users (
    uuid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    steam_id VARCHAR(255) UNIQUE NOT NULL,
    auth_code VARCHAR(255) NOT NULL,
    last_share_code VARCHAR(255) DEFAULT '',
    game_ids JSONB DEFAULT '[]',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);"; then
    echo -e "${GREEN}✓ Users table created${NC}"
else
    echo -e "${RED}Error: Failed to create users table${NC}"
    exit 1
fi

# Create games table
if run_psql_db "CREATE TABLE IF NOT EXISTS games (
    uuid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    share_code VARCHAR(255) UNIQUE NOT NULL,
    demo_name VARCHAR(255) NOT NULL,
    steam_ids JSONB DEFAULT '[]',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);"; then
    echo -e "${GREEN}✓ Games table created${NC}"
else
    echo -e "${RED}Error: Failed to create games table${NC}"
    exit 1
fi

# Create indexes
echo -e "${YELLOW}Creating indexes...${NC}"
run_psql_db "CREATE INDEX IF NOT EXISTS idx_guilds_guild_id ON guilds(guild_id);"
run_psql_db "CREATE INDEX IF NOT EXISTS idx_users_steam_id ON users(steam_id);"
run_psql_db "CREATE INDEX IF NOT EXISTS idx_games_share_code ON games(share_code);"
echo -e "${GREEN}✓ Indexes created${NC}"

# Create update function and triggers
echo -e "${YELLOW}Creating update triggers...${NC}"
run_psql_db "CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS \$\$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
\$\$ language 'plpgsql';"

run_psql_db "CREATE OR REPLACE TRIGGER update_guilds_updated_at BEFORE UPDATE ON guilds FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();"
run_psql_db "CREATE OR REPLACE TRIGGER update_users_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();"
run_psql_db "CREATE OR REPLACE TRIGGER update_games_updated_at BEFORE UPDATE ON games FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();"
echo -e "${GREEN}✓ Update triggers created${NC}"

echo ""
echo -e "${GREEN}🎉 Database setup completed successfully!${NC}"
echo -e "${GREEN}You can now run the bot with: go run .${NC}"