package main

import (
	"context"
	"fmt"
	"gambler/config"
	"gambler/events"
	"gambler/models"
	"gambler/repository"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"gambler/cmd"
	"gambler/database"
)

func main() {
	// Check for migration subcommands
	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		if err := handleMigrationCommand(); err != nil {
			log.Fatal("Migration error:", err)
		}
		return
	}

	fmt.Println(os.Args)
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
	db, _ := database.NewConnection(context.Background(), cfg.DatabaseURL)
	uowFactory := repository.NewUnitOfWorkFactory(db, events.NewBus())
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

	history := &models.BalanceHistory{
		DiscordID:       userId,
		GuildID:         0, // Will be set by repository from UoW's guild scope
		BalanceBefore:   initialBalance,
		BalanceAfter:    balance,
		ChangeAmount:    balance - initialBalance,
		TransactionType: models.TransactionTypeTransferIn,
		TransactionMetadata: map[string]any{
			"admin": "true",
		},
	}
	uow.BalanceHistoryRepository().Record(ctx, history)
	uow.Commit()
	return nil
}
