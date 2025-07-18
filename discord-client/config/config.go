package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	
	"gambler/discord-client/database"
)

// Config holds all application configuration
type Config struct {
	// Discord configuration
	DiscordToken string
	GuildID      string // Primary Discord guild ID

	// Database configuration
	DatabaseURL  string
	DatabaseName string

	// Bot configuration
	StartingBalance     int64
	DailyGambleLimit    int64 // Daily gambling limit per user
	DailyLimitResetHour int   // Hour in UTC when daily limit resets (0-23)

	// High Roller Role configuration
	GambaChannelID string // Channel ID for high roller change notifications

	// Group Wager configuration
	ResolverDiscordIDs []int64 // Discord IDs that can resolve group wagers

	// Environment
	Environment string // "development" or "production"
}

var (
	instance *Config
	once     sync.Once
)

// Get returns the global configuration instance
func Get() *Config {
	once.Do(func() {
		var err error
		instance, err = load()
		if err != nil {
			panic(fmt.Sprintf("failed to load config: %v", err))
		}
	})
	return instance
}

// GetDatabaseURL constructs the full database URL by combining base URL and database name
func (c *Config) GetDatabaseURL() string {
	return database.ConstructDatabaseURL(c.DatabaseURL, c.DatabaseName)
}

// load loads configuration from environment variables
func load() (*Config, error) {
	config := &Config{
		// Discord
		DiscordToken: os.Getenv("DISCORD_TOKEN"),
		GuildID:      os.Getenv("GUILD_ID"),

		// Database
		DatabaseURL:  os.Getenv("DATABASE_URL"),
		DatabaseName: os.Getenv("DATABASE_NAME"),

		// Bot settings with defaults
		StartingBalance:     100000,
		DailyGambleLimit:    10000, // 10k daily limit default
		DailyLimitResetHour: 12,    // 12:00 PM UTC default

		// High Roller Role
		GambaChannelID: os.Getenv("GAMBA_CHANNEL_ID"),

		// Environment
		Environment: os.Getenv("ENVIRONMENT"),
	}

	// Override defaults if environment variables are set
	if balance := os.Getenv("STARTING_BALANCE"); balance != "" {
		if parsedBalance, err := strconv.ParseInt(balance, 10, 64); err == nil {
			config.StartingBalance = parsedBalance
		}
	}
	if limit := os.Getenv("DAILY_GAMBLE_LIMIT"); limit != "" {
		if parsedLimit, err := strconv.ParseInt(limit, 10, 64); err == nil {
			config.DailyGambleLimit = parsedLimit
		}
	}

	// Parse resolver Discord IDs
	if resolverIDs := os.Getenv("RESOLVER_DISCORD_IDS"); resolverIDs != "" {
		idStrings := strings.Split(resolverIDs, ",")
		for _, idStr := range idStrings {
			idStr = strings.TrimSpace(idStr)
			if idStr != "" {
				if id, err := strconv.ParseInt(idStr, 10, 64); err == nil {
					config.ResolverDiscordIDs = append(config.ResolverDiscordIDs, id)
				}
			}
		}
	}

	// Set default environment if not specified
	if config.Environment == "" {
		config.Environment = "development"
	}

	if config.Environment != "test" {
		// Validate required configuration
		if config.DiscordToken == "" {
			return nil, fmt.Errorf("DISCORD_TOKEN is required")
		}
		if config.DatabaseURL == "" {
			return nil, fmt.Errorf("DATABASE_URL is required")
		}
		// If DatabaseName is provided, ensure it's not empty
		if config.DatabaseName != "" && strings.TrimSpace(config.DatabaseName) == "" {
			return nil, fmt.Errorf("DATABASE_NAME cannot be empty when provided")
		}
	}

	return config, nil
}
