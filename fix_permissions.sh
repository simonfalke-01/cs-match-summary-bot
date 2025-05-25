#!/bin/bash

# Fix PostgreSQL table ownership and permissions for CS Match Summary Bot
# This script fixes permission issues when tables exist but are owned by different user

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Fixing PostgreSQL permissions for CS Match Summary Bot...${NC}"

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

# Check if we're on macOS or Linux and use appropriate superuser
if [[ "$OSTYPE" == "darwin"* ]]; then
    # macOS - usually use your username
    SUPERUSER=$(whoami)
    echo -e "${YELLOW}Detected macOS - using user: $SUPERUSER${NC}"
    RUN_PSQL="psql -h $DB_HOST -p $DB_PORT -U $SUPERUSER -d $DB_NAME"
else
    # Linux - try postgres user first
    SUPERUSER="postgres"
    echo -e "${YELLOW}Detected Linux - using user: $SUPERUSER${NC}"
    RUN_PSQL="sudo -u postgres psql -h $DB_HOST -p $DB_PORT -d $DB_NAME"
fi

# Function to run psql command as superuser
run_psql() {
    if [[ "$OSTYPE" == "darwin"* ]]; then
        psql -h $DB_HOST -p $DB_PORT -U $SUPERUSER -d $DB_NAME -c "$1"
    else
        sudo -u postgres psql -h $DB_HOST -p $DB_PORT -d $DB_NAME -c "$1"
    fi
}

# Check if PostgreSQL is running and database exists
echo -e "${YELLOW}Checking PostgreSQL connection and database...${NC}"
if ! run_psql '\q' 2>/dev/null; then
    echo -e "${RED}Error: Cannot connect to PostgreSQL database '$DB_NAME'${NC}"
    echo "Please ensure PostgreSQL is running and database exists"
    exit 1
fi
echo -e "${GREEN}✓ PostgreSQL connection successful${NC}"

# Check if tables exist
echo -e "${YELLOW}Checking existing tables...${NC}"
TABLES=$(run_psql "SELECT tablename FROM pg_tables WHERE schemaname = 'public';" -t | grep -E '(guilds|users|games)' | wc -l)

if [ "$TABLES" -eq 0 ]; then
    echo -e "${YELLOW}No tables found. Creating tables...${NC}"
    
    # Create UUID extension
    echo -e "${YELLOW}Creating UUID extension...${NC}"
    run_psql "CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";"
    
    # Create tables with proper ownership
    echo -e "${YELLOW}Creating guilds table...${NC}"
    run_psql "CREATE TABLE IF NOT EXISTS guilds (
        uuid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
        guild_id VARCHAR(255) UNIQUE NOT NULL,
        channel_id VARCHAR(255) NOT NULL,
        user_ids JSONB DEFAULT '[]',
        game_ids JSONB DEFAULT '[]',
        created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
    );"
    
    echo -e "${YELLOW}Creating users table...${NC}"
    run_psql "CREATE TABLE IF NOT EXISTS users (
        uuid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
        steam_id VARCHAR(255) UNIQUE NOT NULL,
        auth_code VARCHAR(255) NOT NULL,
        last_share_code VARCHAR(255) DEFAULT '',
        game_ids JSONB DEFAULT '[]',
        created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
    );"
    
    echo -e "${YELLOW}Creating games table...${NC}"
    run_psql "CREATE TABLE IF NOT EXISTS games (
        uuid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
        share_code VARCHAR(255) UNIQUE NOT NULL,
        demo_name VARCHAR(255) NOT NULL,
        steam_ids JSONB DEFAULT '[]',
        created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
    );"
    
    echo -e "${GREEN}✓ Tables created${NC}"
else
    echo -e "${GREEN}✓ Found $TABLES existing tables${NC}"
fi

# Fix ownership for all tables
echo -e "${YELLOW}Fixing table ownership...${NC}"
if [ "$DB_USER" != "$SUPERUSER" ]; then
    run_psql "ALTER TABLE IF EXISTS guilds OWNER TO $DB_USER;"
    run_psql "ALTER TABLE IF EXISTS users OWNER TO $DB_USER;"
    run_psql "ALTER TABLE IF EXISTS games OWNER TO $DB_USER;"
    echo -e "${GREEN}✓ Table ownership transferred to $DB_USER${NC}"
else
    echo -e "${GREEN}✓ User $DB_USER is already the superuser${NC}"
fi

# Grant all privileges
echo -e "${YELLOW}Granting privileges...${NC}"
run_psql "GRANT ALL PRIVILEGES ON DATABASE $DB_NAME TO $DB_USER;"
run_psql "GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO $DB_USER;"
run_psql "GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO $DB_USER;"
run_psql "GRANT ALL PRIVILEGES ON ALL FUNCTIONS IN SCHEMA public TO $DB_USER;"
echo -e "${GREEN}✓ Privileges granted${NC}"

# Create or recreate indexes
echo -e "${YELLOW}Creating indexes...${NC}"
run_psql "CREATE INDEX IF NOT EXISTS idx_guilds_guild_id ON guilds(guild_id);"
run_psql "CREATE INDEX IF NOT EXISTS idx_users_steam_id ON users(steam_id);"
run_psql "CREATE INDEX IF NOT EXISTS idx_games_share_code ON games(share_code);"
echo -e "${GREEN}✓ Indexes created${NC}"

# Create or recreate update function and triggers
echo -e "${YELLOW}Creating update function and triggers...${NC}"
run_psql "CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS \$\$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
\$\$ language 'plpgsql';"

# Drop and recreate triggers to ensure they work with new ownership
run_psql "DROP TRIGGER IF EXISTS update_guilds_updated_at ON guilds;"
run_psql "DROP TRIGGER IF EXISTS update_users_updated_at ON users;"
run_psql "DROP TRIGGER IF EXISTS update_games_updated_at ON games;"

run_psql "CREATE TRIGGER update_guilds_updated_at BEFORE UPDATE ON guilds FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();"
run_psql "CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();"
run_psql "CREATE TRIGGER update_games_updated_at BEFORE UPDATE ON games FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();"
echo -e "${GREEN}✓ Triggers created${NC}"

# Fix function ownership
run_psql "ALTER FUNCTION update_updated_at_column() OWNER TO $DB_USER;"
echo -e "${GREEN}✓ Function ownership fixed${NC}"

# Test connection with target user
echo -e "${YELLOW}Testing connection with target user...${NC}"
if PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "SELECT COUNT(*) FROM guilds;" >/dev/null 2>&1; then
    echo -e "${GREEN}✓ User $DB_USER can now access tables${NC}"
else
    echo -e "${RED}✗ User $DB_USER still cannot access tables${NC}"
    echo -e "${YELLOW}Trying alternative permission fix...${NC}"
    
    # Try granting public schema usage
    run_psql "GRANT USAGE ON SCHEMA public TO $DB_USER;"
    run_psql "GRANT CREATE ON SCHEMA public TO $DB_USER;"
    
    # Try setting default privileges
    run_psql "ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO $DB_USER;"
    run_psql "ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO $DB_USER;"
    run_psql "ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON FUNCTIONS TO $DB_USER;"
    
    if PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "SELECT COUNT(*) FROM guilds;" >/dev/null 2>&1; then
        echo -e "${GREEN}✓ Fixed! User $DB_USER can now access tables${NC}"
    else
        echo -e "${RED}✗ Still having issues. You may need to recreate the database${NC}"
        exit 1
    fi
fi

echo ""
echo -e "${GREEN}🎉 Database permissions fixed successfully!${NC}"
echo -e "${GREEN}You can now run the bot with: go run .${NC}"