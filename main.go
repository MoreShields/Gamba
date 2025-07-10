package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
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
