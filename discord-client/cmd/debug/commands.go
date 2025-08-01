package debug

import (
	"fmt"
	"strconv"
)

// initializeCommands sets up all available debug commands
func (s *Shell) initializeCommands() {
	s.commands = map[string]Command{
		// Essential commands only
		"help": {
			Handler:     s.handleHelp,
			Description: "Show available commands",
			Usage:       "help [command]",
			Category:    "utility",
		},
		"guild": {
			Handler:     s.handleGuild,
			Description: "Select or show current guild context",
			Usage:       "guild [guild_id] - shows menu if no ID provided",
			Category:    "utility",
		},
		"replay": {
			Handler:     s.handleReplay,
			Description: "Replay a Discord message",
			Usage:       "replay <channel_id> <message_id>",
			Category:    "admin",
		},
		"daily-awards": {
			Handler:     s.handleDailyAwards,
			Description: "Post daily awards summary for a guild",
			Usage:       "daily-awards [guild_id] - uses current guild if not specified",
			Category:    "admin",
		},
		// Admin commands are defined in admin.go
	}

	// Add admin commands
	s.addAdminCommands()
}

// handleHelp displays help information
func (s *Shell) handleHelp(shell *Shell, args []string) error {
	if len(args) > 0 {
		// Show help for specific command
		cmdName := args[0]
		if cmd, exists := s.commands[cmdName]; exists {
			fmt.Printf("\n📖 %s\n", cmdName)
			fmt.Printf("   %s\n", cmd.Description)
			fmt.Printf("   Usage: %s\n", cmd.Usage)
			fmt.Printf("   Category: %s\n", cmd.Category)
			return nil
		}
		return fmt.Errorf("unknown command: %s", cmdName)
	}

	// Show all commands
	fmt.Println("\n📚 Available Commands:")
	fmt.Println("====================")
	
	fmt.Println("\n\033[33mESSENTIAL COMMANDS:\033[0m")
	fmt.Printf("  %-20s %s\n", "replay", "Replay a Discord message")
	fmt.Printf("  %-20s %s\n", "adjust-balance", "Adjust user balance by amount (+/-)")
	fmt.Printf("  %-20s %s\n", "admin-transfer", "Transfer bits between users")
	
	fmt.Println("\n\033[34mOTHER:\033[0m")
	fmt.Printf("  %-20s %s\n", "guild", "Select guild from menu (auto-selects if only one)")
	fmt.Printf("  %-20s %s\n", "help", "Show this help message")
	fmt.Printf("  %-20s %s\n", "exit", "Exit debug shell")
	
	fmt.Println("\nType 'help <command>' for detailed usage")

	return nil
}

// handleReplay replays a Discord message
func (s *Shell) handleReplay(shell *Shell, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: replay <channel_id> <message_id>")
	}

	channelID := args[0]
	messageID := args[1]

	s.printInfo(fmt.Sprintf("Fetching message %s from channel %s...", messageID, channelID))

	// Use debug client to replay message
	if err := s.debugClient.ReplayMessage(channelID, messageID); err != nil {
		return fmt.Errorf("failed to replay message: %w", err)
	}

	s.printSuccess("Message replayed successfully")
	return nil
}

// handleDailyAwards posts the daily awards summary for a guild
func (s *Shell) handleDailyAwards(shell *Shell, args []string) error {
	var guildID int64
	
	// Determine guild ID
	if len(args) > 0 {
		// Guild ID provided as argument
		parsedGuildID, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid guild ID: %w", err)
		}
		guildID = parsedGuildID
	} else {
		// Use current guild context
		if s.currentGuild == 0 {
			return fmt.Errorf("no guild context set - use 'guild' command first or provide guild ID")
		}
		guildID = s.currentGuild
	}

	guildIDStr := strconv.FormatInt(guildID, 10)
	s.printInfo(fmt.Sprintf("Posting daily awards summary for guild %d...", guildID))

	// Use debug client to post daily awards
	if err := s.debugClient.PostDailyAwards(guildIDStr); err != nil {
		return fmt.Errorf("failed to post daily awards: %w", err)
	}

	s.printSuccess("Daily awards summary posted successfully")
	return nil
}

// handleGuild sets or shows the current guild context
func (s *Shell) handleGuild(shell *Shell, args []string) error {
	// If no args, show guild selection menu
	if len(args) == 0 {
		return s.showGuildSelection()
	}

	// Try to parse as guild ID
	guildID, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		// Not a number, might be a guild name - show selection
		return s.showGuildSelection()
	}

	// Set guild by ID
	s.currentGuild = guildID
	s.printSuccess(fmt.Sprintf("Guild context set to: %d", guildID))
	fmt.Println("This guild will be used as default for commands that require a guild_id.")
	return nil
}

// showGuildSelection displays a menu of available guilds
func (s *Shell) showGuildSelection() error {
	// Fetch guilds from the bot
	guilds, err := s.debugClient.GetGuilds()
	if err != nil {
		return fmt.Errorf("failed to fetch guilds: %w", err)
	}

	if len(guilds) == 0 {
		return fmt.Errorf("no guilds found - bot is not in any servers")
	}

	// Show current guild if set
	if s.currentGuild != 0 {
		fmt.Printf("\nCurrent guild: %d\n", s.currentGuild)
	}

	// Display guild list
	fmt.Println("\n📋 Available Guilds:")
	fmt.Println("==================")
	for i, guild := range guilds {
		fmt.Printf("%d. %s (ID: %s)\n", i+1, guild.Name, guild.ID)
	}

	// If only one guild, auto-select it
	if len(guilds) == 1 {
		guildID, _ := strconv.ParseInt(guilds[0].ID, 10, 64)
		s.currentGuild = guildID
		s.printSuccess(fmt.Sprintf("Auto-selected guild: %s", guilds[0].Name))
		return nil
	}

	// Prompt for selection
	fmt.Print("\nSelect guild number (1-", len(guilds), ") or guild ID: ")
	
	var input string
	fmt.Scanln(&input)
	
	// Try as number selection first
	if num, err := strconv.Atoi(input); err == nil {
		if num >= 1 && num <= len(guilds) {
			guildID, _ := strconv.ParseInt(guilds[num-1].ID, 10, 64)
			s.currentGuild = guildID
			s.printSuccess(fmt.Sprintf("Guild set to: %s", guilds[num-1].Name))
			return nil
		}
		return fmt.Errorf("invalid selection - please choose a number between 1 and %d", len(guilds))
	}

	// Try as guild ID
	if guildID, err := strconv.ParseInt(input, 10, 64); err == nil {
		// Verify it's a valid guild
		for _, guild := range guilds {
			if guild.ID == input {
				s.currentGuild = guildID
				s.printSuccess(fmt.Sprintf("Guild set to: %s", guild.Name))
				return nil
			}
		}
		return fmt.Errorf("guild ID %s not found in available guilds", input)
	}

	return fmt.Errorf("invalid input - please enter a number or guild ID")
}