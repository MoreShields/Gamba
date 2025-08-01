package debug

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"gambler/discord-client/application"
	"gambler/discord-client/database"

	log "github.com/sirupsen/logrus"
)

// Shell represents the debug shell interface
type Shell struct {
	debugClient *DebugClient // Client for connecting to running bot's debug API
	db          *database.DB
	uowFactory  application.UnitOfWorkFactory
	commands    map[string]Command
	history     []string
	currentGuild int64  // Current guild context for commands
	currentGuildName string // Current guild name for display
	dryRun      bool
	running     bool
}

// Command represents a debug command
type Command struct {
	Handler     CommandHandler
	Description string
	Usage       string
	Category    string // "read", "admin", "utility"
}

// CommandHandler is a function that handles a debug command
type CommandHandler func(s *Shell, args []string) error

// NewShell creates a new debug shell instance
func NewShell(db *database.DB, uowFactory application.UnitOfWorkFactory) *Shell {
	s := &Shell{
		db:         db,
		uowFactory: uowFactory,
		history:    []string{},
		dryRun:     false,
		running:    true,
	}
	
	// Perform startup checks and connect to debug API
	s.performStartupChecks()
	
	// Initialize commands
	s.initializeCommands()

	return s
}

// performStartupChecks validates the environment and connects to the debug API
func (s *Shell) performStartupChecks() {
	fmt.Println("🎰 Gambler Debug Shell 🎰")
	fmt.Println("=========================")
	fmt.Println()
	
	// Check if we're in a container
	if !isInContainer() {
		fmt.Println("⚠️  Warning: Not running in container")
		fmt.Println("")
		fmt.Println("For production use, run:")
		fmt.Println("  docker exec -it discord-bot debug-shell")
		fmt.Println("")
		fmt.Println("Attempting to connect to local debug API...")
		fmt.Println()
	}
	
	// Connect to debug API
	debugPort := 8899
	client := NewDebugClient(debugPort)
	
	fmt.Printf("Connecting to debug API on port %d...\n", debugPort)
	if err := client.CheckConnection(); err == nil {
		s.debugClient = client
		fmt.Printf("✅ Connected to debug API\n")
		fmt.Println()
		fmt.Println("Available commands:")
		fmt.Println("  • guild - Select guild from menu (auto-selects if only one)")
		fmt.Println("  • replay <channel_id> <message_id> - Replay a Discord message")
		fmt.Println("  • adjust-balance [guild_id] <user_id> <amount> - Adjust user balance")
		fmt.Println("  • admin-transfer [guild_id] <from> <to> <amount> - Transfer between users")
		fmt.Println("  • help - Show all available commands")
		fmt.Println()
		fmt.Println("Tip: Run 'guild' to select a default guild and omit guild_id from commands")
		fmt.Println()
	} else {
		log.Fatalf("❌ Failed to connect to debug API: %v\n\nPlease check that the bot is running and the debug API is enabled.", err)
	}
}

// isInContainer checks if we're running inside a Docker container
func isInContainer() bool {
	// Check for .dockerenv file
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}
	// Check for docker in cgroup
	if data, err := os.ReadFile("/proc/1/cgroup"); err == nil {
		if strings.Contains(string(data), "docker") || strings.Contains(string(data), "containerd") {
			return true
		}
	}
	return false
}

// Run starts the interactive debug shell
func (s *Shell) Run(ctx context.Context) error {
	// Create scanner for stdin
	scanner := bufio.NewScanner(os.Stdin)

	// Main shell loop
	for s.running {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		// Print prompt with guild context
		if s.currentGuild != 0 {
			if s.currentGuildName != "" {
				fmt.Printf("\n🎲 debug [%s]> ", s.currentGuildName)
			} else {
				fmt.Printf("\n🎲 debug [guild:%d]> ", s.currentGuild)
			}
		} else {
			fmt.Print("\n🎲 debug> ")
		}

		// Read input
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		// Add to history
		s.history = append(s.history, input)

		// Parse command and arguments
		parts := strings.Fields(input)
		if len(parts) == 0 {
			continue
		}

		cmdName := parts[0]
		args := parts[1:]

		// Handle built-in commands
		switch cmdName {
		case "exit", "quit":
			s.running = false
			fmt.Println("👋 Exiting debug shell. Bot will continue running.")
			continue
		case "clear":
			fmt.Print("\033[H\033[2J")
			continue
		case "dry-run":
			if err := s.handleDryRun(args); err != nil {
				s.printError(err)
			}
			continue
		}

		// Look up command
		cmd, exists := s.commands[cmdName]
		if !exists {
			s.printError(fmt.Errorf("unknown command: %s. Type 'help' for available commands", cmdName))
			continue
		}

		// All commands now require debug API connection
		if s.debugClient == nil {
			s.printError(fmt.Errorf("not connected to bot debug API"))
			continue
		}

		// Execute command with rate limiting
		time.Sleep(100 * time.Millisecond) // Basic rate limiting

		if err := cmd.Handler(s, args); err != nil {
			s.printError(err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanner error: %w", err)
	}

	return nil
}

// printError displays an error message in red
func (s *Shell) printError(err error) {
	fmt.Printf("\033[31m❌ Error: %s\033[0m\n", err.Error())
}

// printSuccess displays a success message in green
func (s *Shell) printSuccess(msg string) {
	fmt.Printf("\033[32m✅ %s\033[0m\n", msg)
}

// printWarning displays a warning message in yellow
func (s *Shell) printWarning(msg string) {
	fmt.Printf("\033[33m⚠️  %s\033[0m\n", msg)
}

// printInfo displays an info message in blue
func (s *Shell) printInfo(msg string) {
	fmt.Printf("\033[34mℹ️  %s\033[0m\n", msg)
}

// confirmAction prompts the user for confirmation
func (s *Shell) confirmAction(prompt string) bool {
	if s.dryRun {
		s.printInfo("Dry-run mode: Would execute action")
		return false
	}

	fmt.Printf("\n\033[33m⚠️  %s [y/N]: \033[0m", prompt)
	
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return false
	}

	response := strings.ToLower(strings.TrimSpace(scanner.Text()))
	return response == "y" || response == "yes"
}

// handleDryRun toggles dry-run mode
func (s *Shell) handleDryRun(args []string) error {
	if len(args) == 0 {
		status := "off"
		if s.dryRun {
			status = "on"
		}
		s.printInfo(fmt.Sprintf("Dry-run mode is currently: %s", status))
		return nil
	}

	switch strings.ToLower(args[0]) {
	case "on", "true", "1":
		s.dryRun = true
		s.printWarning("Dry-run mode enabled - no changes will be made")
	case "off", "false", "0":
		s.dryRun = false
		s.printSuccess("Dry-run mode disabled")
	default:
		return fmt.Errorf("invalid dry-run value. Use 'on' or 'off'")
	}

	return nil
}

// logAdminAction logs admin actions for audit purposes
func (s *Shell) logAdminAction(action string, details map[string]interface{}) {
	fields := log.Fields{
		"action":    action,
		"timestamp": time.Now().Unix(),
		"source":    "debug_shell",
	}

	for k, v := range details {
		fields[k] = v
	}

	log.WithFields(fields).Info("Admin action executed via debug shell")
}