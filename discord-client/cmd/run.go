package cmd

import (
	"context"
	"fmt"
	"log"
	"time"

	"gambler/discord-client/bot"
	"gambler/discord-client/bot/features/betting"
	"gambler/discord-client/config"
	"gambler/discord-client/database"
	"gambler/discord-client/events"
	"gambler/discord-client/repository"
)

// Run initializes and starts the application
func Run(ctx context.Context) error {
	log.Println("Starting gambler bot...")

	// Load configuration
	cfg := config.Get()

	// Initialize database connection
	log.Println("Connecting to database...")
	db, err := database.NewConnection(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	log.Println("Database connection established successfully")

	// Initialize event bus
	log.Println("Initializing event bus...")
	eventBus := events.NewBus()
	log.Println("Event bus initialized successfully")

	// Initialize unit of work factory
	log.Println("Initializing unit of work factory...")
	uowFactory := repository.NewUnitOfWorkFactory(db, eventBus)
	log.Println("Unit of work factory initialized successfully")

	// Initialize Discord bot
	log.Println("Initializing Discord bot...")
	botConfig := bot.Config{
		Token:          cfg.DiscordToken,
		GuildID:        cfg.GuildID,
		GambaChannelID: cfg.GambaChannelID,
	}
	gamblingConfig := &betting.GamblingConfig{
		DailyGambleLimit:    cfg.DailyGambleLimit,
		DailyLimitResetHour: cfg.DailyLimitResetHour,
	}
	discordBot, err := bot.New(botConfig, gamblingConfig, uowFactory, eventBus)
	if err != nil {
		return fmt.Errorf("failed to initialize Discord bot: %w", err)
	}
	log.Println("Discord bot initialized successfully")

	// Wait for context cancellation
	log.Printf("Bot is running in %s mode...", cfg.Environment)
	<-ctx.Done()

	// Cleanup resources
	log.Println("Shutting down bot...")

	// Close Discord bot connection
	if err := discordBot.Close(); err != nil {
		log.Printf("Error closing Discord bot: %v", err)
	}

	// Give cleanup operations time to complete
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Close database connection
	log.Println("Closing database connection...")
	db.Close()

	select {
	case <-shutdownCtx.Done():
		log.Println("Shutdown timeout exceeded")
	case <-time.After(1 * time.Second):
		log.Println("Shutdown completed")
	}

	return nil
}
