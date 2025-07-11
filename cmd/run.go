package cmd

import (
	"context"
	"fmt"
	"log"
	"time"

	"gambler/bot"
	"gambler/bot/features/betting"
	"gambler/config"
	"gambler/database"
	"gambler/events"
	"gambler/repository"
	"gambler/service"
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

	// Initialize services
	log.Println("Initializing services...")
	userService := service.NewUserService(uowFactory)
	gamblingService := service.NewGamblingService(uowFactory)
	transferService := service.NewTransferService(uowFactory)
	wagerService := service.NewWagerService(uowFactory)
	statsService := service.NewStatsService(uowFactory)
	groupWagerService := service.NewGroupWagerService(uowFactory)
	log.Println("Services initialized successfully")

	// Initialize Discord bot
	log.Println("Initializing Discord bot...")
	botConfig := bot.Config{
		Token:             cfg.DiscordToken,
		GuildID:           cfg.DiscordGuildID,
		HighRollerRoleID:  cfg.HighRollerRoleID,
		HighRollerEnabled: cfg.HighRollerEnabled,
		GambaChannelID:    cfg.GambaChannelID,
	}
	gamblingConfig := &betting.GamblingConfig{
		DailyGambleLimit:    cfg.DailyGambleLimit,
		DailyLimitResetHour: cfg.DailyLimitResetHour,
	}
	discordBot, err := bot.New(botConfig, gamblingConfig, userService, gamblingService, transferService, wagerService, statsService, groupWagerService, eventBus)
	if err != nil {
		return fmt.Errorf("failed to initialize Discord bot: %w", err)
	}
	log.Println("Discord bot initialized successfully")

	// Start background worker for group wager expiration
	log.Println("Starting group wager expiration worker...")
	go runGroupWagerExpirationWorker(ctx, groupWagerService)

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

// runGroupWagerExpirationWorker runs a background task to check for expired group wagers
func runGroupWagerExpirationWorker(ctx context.Context, groupWagerService service.GroupWagerService) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	// Run immediately on startup
	if err := groupWagerService.TransitionExpiredWagers(context.Background()); err != nil {
		log.Printf("Error transitioning expired group wagers: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			log.Println("Group wager expiration worker shutting down...")
			return
		case <-ticker.C:
			// Use a separate context for the operation so it's not cancelled by the parent
			if err := groupWagerService.TransitionExpiredWagers(context.Background()); err != nil {
				log.Printf("Error transitioning expired group wagers: %v", err)
			}
		}
	}
}
