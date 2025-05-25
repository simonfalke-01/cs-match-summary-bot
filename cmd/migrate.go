package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

var db *sql.DB

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

func initDB() error {
	host := os.Getenv("DB_HOST")
	if host == "" {
		host = "localhost"
	}
	
	port := os.Getenv("DB_PORT")
	if port == "" {
		port = "5432"
	}
	
	user := os.Getenv("DB_USER")
	if user == "" {
		user = "postgres"
	}
	
	password := os.Getenv("DB_PASSWORD")
	if password == "" {
		password = "postgres"
	}
	
	dbname := os.Getenv("DB_NAME")
	if dbname == "" {
		dbname = "cs"
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		return err
	}

	if err = db.Ping(); err != nil {
		return err
	}

	log.Println("Database connected successfully")
	return nil
}

func closeDB() {
	if db != nil {
		db.Close()
	}
}

func initializeTables() error {
	_, err := db.Exec(CreateTablesSQL)
	if err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}
	log.Println("Database tables initialized successfully")
	return nil
}

func dropTables() error {
	dropSQL := `
		DROP TABLE IF EXISTS games CASCADE;
		DROP TABLE IF EXISTS users CASCADE;
		DROP TABLE IF EXISTS guilds CASCADE;
		DROP FUNCTION IF EXISTS update_updated_at_column() CASCADE;
	`

	_, err := db.Exec(dropSQL)
	if err != nil {
		return fmt.Errorf("failed to drop tables: %w", err)
	}

	return nil
}

func main() {
	var drop = flag.Bool("drop", false, "Drop all tables before creating them")
	var reset = flag.Bool("reset", false, "Drop and recreate all tables (same as -drop)")
	flag.Parse()

	if err := godotenv.Load("../.env"); err != nil {
		log.Println("Warning: .env file not found or could not be loaded")
	}

	if err := initDB(); err != nil {
		log.Fatal("Failed to initialize database: ", err)
	}
	defer closeDB()

	if *drop || *reset {
		fmt.Println("Dropping existing tables...")
		if err := dropTables(); err != nil {
			log.Fatal("Failed to drop tables: ", err)
		}
		fmt.Println("Tables dropped successfully")
	}

	fmt.Println("Creating tables...")
	if err := initializeTables(); err != nil {
		log.Fatal("Failed to create tables: ", err)
	}
	fmt.Println("Migration completed successfully!")
}