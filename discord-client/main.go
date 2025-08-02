package main

import (
	"context"
	"fmt"
	"gambler/discord-client/config"
	"gambler/discord-client/database"
	"gambler/discord-client/domain/events"
	"gambler/discord-client/infrastructure"
	"gambler/discord-client/domain/entities"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"

	"gambler/discord-client/cmd"
	"gambler/discord-client/cmd/debug"
)

func main() {
	// Check if invoked as debug-shell (via symlink)
	if filepath.Base(os.Args[0]) == "debug-shell" {
		if err := runDebugMode(); err != nil {
			log.Fatal("Debug mode error:", err)
		}
		return
	}

	// Check for migration subcommands
	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		if err := handleMigrationCommand(); err != nil {
			log.Fatal("Migration error:", err)
		}
		return
	}

	// Check for debug mode
	if len(os.Args) > 1 && os.Args[1] == "debug" {
		if err := runDebugMode(); err != nil {
			log.Fatal("Debug mode error:", err)
		}
		return
	}

	// Check for balance adjustment subcommands
	if len(os.Args) > 1 && os.Args[1] == "update-balance" {
		if err := handleBalanceAdjustment(); err != nil {
			log.Fatal("Balance adjustment error:", err)
		}
		return
	}

	// Normal bot operation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Received shutdown signal, shutting down gracefully...")
		cancel()
	}()

	// Run the application
	if err := cmd.Run(ctx); err != nil {
		log.Fatal("Application error:", err)
	}
}

func handleMigrationCommand() error {
	if len(os.Args) < 3 {
		return fmt.Errorf("usage: gambler migrate [up|down|status] [args...]")
	}

	command := os.Args[2]
	switch command {
	case "up":
		return database.MigrateUp()
	case "down":
		steps := "1"
		if len(os.Args) > 3 {
			steps = os.Args[3]
		}
		return database.MigrateDown(steps)
	case "status":
		return database.MigrateStatus()
	default:
		return fmt.Errorf("unknown migration command: %s", command)
	}
}

func handleBalanceAdjustment() error {
	if len(os.Args) < 5 {
		return fmt.Errorf("usage: gambler update-balance guild-id user amount")
	}
	guildId, _ := strconv.ParseInt(os.Args[2], 10, 64)
	userId, _ := strconv.ParseInt(os.Args[3], 10, 64)
	balance, _ := strconv.ParseInt(os.Args[4], 10, 64)

	ctx := context.Background()
	// Load configuration
	cfg := config.Get()
	// load infra
	db, _ := database.NewConnection(context.Background(), cfg.GetDatabaseURL())
	// Create a dummy event publisher for admin commands (events won't be processed)
	dummyEventPublisher := &dummyEventPublisher{}
	uowFactory := infrastructure.NewUnitOfWorkFactory(db, dummyEventPublisher)
	uow := uowFactory.CreateForGuild(int64(guildId))
	uow.Begin(ctx)
	defer uow.Commit()

	// Get user
	user, err := uow.UserRepository().GetByDiscordID(context.Background(), userId)
	if err != nil {
		return fmt.Errorf("failed to get recipient user: %w", err)
	}
	initialBalance := user.Balance
	uow.UserRepository().UpdateBalance(ctx, userId, int64(balance))

	history := &entities.BalanceHistory{
		DiscordID:       userId,
		GuildID:         0, // Will be set by repository from UoW's guild scope
		BalanceBefore:   initialBalance,
		BalanceAfter:    balance,
		ChangeAmount:    balance - initialBalance,
		TransactionType: entities.TransactionTypeTransferIn,
		TransactionMetadata: map[string]any{
			"admin": "true",
		},
	}
	uow.BalanceHistoryRepository().Record(ctx, history)
	uow.Commit()
	return nil
}

// dummyEventPublisher is a no-op event publisher for admin commands
type dummyEventPublisher struct{}

func (d *dummyEventPublisher) Publish(event events.Event) error {
	// No-op for admin commands
	return nil
}

// runDebugMode starts the debug shell connecting to the running bot via debug API
func runDebugMode() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Println("Starting debug shell...")
	// Run shell that connects to bot via debug API
	runDebugShell(ctx)
	return nil
}

// runDebugShell runs a simple debug shell
func runDebugShell(ctx context.Context) {
	log.Println("Debug shell ready. Type 'help' for commands.")
	
	// Load configuration
	cfg := config.Get()

	// Create database connection for shell operations
	db, err := database.NewConnection(ctx, cfg.GetDatabaseURL())
	if err != nil {
		log.Printf("Failed to connect to database: %v", err)
		return
	}
	defer db.Close()

	// Create dummy event publisher for admin operations
	eventPublisher := &dummyEventPublisher{}
	uowFactory := infrastructure.NewUnitOfWorkFactory(db, eventPublisher)

	// Create shell that will connect to bot via debug API
	shell := debug.NewShell(db, uowFactory)
	
	// Run the shell
	if err := shell.Run(ctx); err != nil {
		log.Printf("Debug shell error: %v", err)
	}
}
