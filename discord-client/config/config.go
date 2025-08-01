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

	// LoL Wager Limits
	MaxLolWagerPerGame int64 // Maximum bet amount per LoL game

	// High Roller Role configuration
	GambaChannelID string // Channel ID for high roller change notifications

	// Group Wager configuration
	ResolverDiscordIDs []int64 // Discord IDs that can resolve group wagers

	// Summoner Service configuration
	SummonerServiceAddr string // Address of the summoner tracking service

	// NATS configuration
	NATSServers string // NATS server addresses (comma-separated)

	// Message streaming configuration
	StreamChannelTypes []string // Channel types to stream to NATS (e.g., ["lol_channel", "primary_channel"])

	// Wordle configuration
	WordleBotID        string // Discord ID of the Wordle bot to monitor
	WordleRewardAmount int64  // Amount of bits to award for daily Wordle completion

	// Daily Awards configuration
	DailyAwardsHour int // Hour in UTC when daily awards summary is posted (0-23)

	// Environment
	Environment string // "development" or "production"
}

var (
	instance *Config
	once     sync.Once
)

// Get returns the global configuration instance
func Get() *Config {
	// If instance is already set (e.g., by tests), return it
	if instance != nil {
		return instance
	}

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

		// LoL Wager Limit default
		MaxLolWagerPerGame: 50000,

		// High Roller Role
		GambaChannelID: os.Getenv("GAMBA_CHANNEL_ID"),

		// Summoner Service
		SummonerServiceAddr: getEnvWithDefault("SUMMONER_SERVICE_ADDR", "lol-tracker:9000"),

		// NATS
		NATSServers: getEnvWithDefault("NATS_SERVERS", "nats://nats:4222"),

		// Wordle
		WordleBotID:        os.Getenv("WORDLE_BOT_ID"),
		WordleRewardAmount: 10000,

		// Daily Awards
		DailyAwardsHour: 14, // 2pm UTC / 9am CST

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
	if reward := os.Getenv("WORDLE_REWARD_AMOUNT"); reward != "" {
		if parsedReward, err := strconv.ParseInt(reward, 10, 64); err == nil {
			config.WordleRewardAmount = parsedReward
		}
	}
	if maxLolWager := os.Getenv("MAX_LOL_WAGER_PER_GAME"); maxLolWager != "" {
		if parsedMax, err := strconv.ParseInt(maxLolWager, 10, 64); err == nil {
			config.MaxLolWagerPerGame = parsedMax
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

	// Parse stream channel types
	if streamChannelTypes := os.Getenv("STREAM_CHANNEL_TYPES"); streamChannelTypes != "" {
		channelTypes := strings.Split(streamChannelTypes, ",")
		for _, channelType := range channelTypes {
			channelType = strings.TrimSpace(channelType)
			if channelType != "" {
				config.StreamChannelTypes = append(config.StreamChannelTypes, channelType)
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

// getEnvWithDefault returns the environment variable value or a default if not set
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Test helpers - only use in tests

// SetTestConfig overrides the global config instance for testing
// This should only be called from test files
func SetTestConfig(testConfig *Config) {
	instance = testConfig
}

// ResetConfig resets the global config instance and sync.Once for testing
// This should only be called from test files
func ResetConfig() {
	instance = nil
	once = sync.Once{}
}

// NewTestConfig creates a minimal config suitable for unit tests
func NewTestConfig() *Config {
	return &Config{
		Environment:         "test",
		ResolverDiscordIDs:  []int64{999999}, // Default test resolver ID
		StartingBalance:     100000,
		DailyGambleLimit:    10000,
		DailyLimitResetHour: 0,
		MaxLolWagerPerGame:  10000,
	}
}
