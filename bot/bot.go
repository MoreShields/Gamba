package bot

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"gambler/events"
	"gambler/service"

	"github.com/bwmarrin/discordgo"
)

// Config holds bot configuration
type Config struct {
	Token             string
	GuildID           string
	HighRollerRoleID  string
	HighRollerEnabled bool
}

type Bot struct {
	config            Config
	session           *discordgo.Session
	userService       service.UserService
	gamblingService   service.GamblingService
	transferService   service.TransferService
	wagerService      service.WagerService
	statsService      service.StatsService
	groupWagerService service.GroupWagerService
	eventBus          *events.Bus
}

func New(config Config, userService service.UserService, gamblingService service.GamblingService, transferService service.TransferService, wagerService service.WagerService, statsService service.StatsService, groupWagerService service.GroupWagerService, eventBus *events.Bus) (*Bot, error) {
	dg, err := discordgo.New("Bot " + config.Token)
	if err != nil {
		return nil, fmt.Errorf("error creating discord session: %w", err)
	}
	dg.Identify.Intents = discordgo.IntentsAll

	bot := &Bot{
		config:            config,
		session:           dg,
		userService:       userService,
		gamblingService:   gamblingService,
		transferService:   transferService,
		wagerService:      wagerService,
		statsService:      statsService,
		groupWagerService: groupWagerService,
		eventBus:          eventBus,
	}

	// Register slash command handlers
	dg.AddHandler(bot.handleCommands)

	// Register component interaction handlers
	dg.AddHandler(bot.handleBetInteraction)
	dg.AddHandler(bot.handleWagerInteractions)
	dg.AddHandler(bot.handleGroupWagerInteractions)

	// Open websocket connection
	if err := dg.Open(); err != nil {
		return nil, fmt.Errorf("error opening connection: %w", err)
	}

	// Register slash commands with Discord
	if err := bot.registerCommands(); err != nil {
		dg.Close()
		return nil, fmt.Errorf("error registering commands: %w", err)
	}

	// Start periodic cleanup of old bet sessions
	go bot.startSessionCleanup()

	// Subscribe to balance change events for high roller role updates
	if bot.config.HighRollerEnabled {
		eventBus.Subscribe(events.EventTypeBalanceChange, func(ctx context.Context, event events.Event) {
			if _, ok := event.(events.BalanceChangeEvent); ok {
				if err := bot.updateHighRollerRole(ctx); err != nil {
					log.Errorf("Failed to update high roller role: %v", err)
				}
			}
		})
		log.Info("High roller role management enabled")

		// Perform initial sync of high roller role
		go func() {
			// Wait a moment for Discord connection to be fully established
			time.Sleep(2 * time.Second)
			ctx := context.Background()
			if err := bot.updateHighRollerRole(ctx); err != nil {
				log.Errorf("Failed to sync high roller role on startup: %v", err)
			} else {
				log.Info("High roller role synced on startup")
			}
		}()
	}

	return bot, nil
}

func (b *Bot) Close() error {
	return b.session.Close()
}

// updateHighRollerRole checks and updates the high roller role assignment
func (b *Bot) updateHighRollerRole(ctx context.Context) error {
	if !b.config.HighRollerEnabled || b.config.HighRollerRoleID == "" {
		return nil // Feature disabled
	}

	// Get current high roller from database
	highRoller, err := b.userService.GetCurrentHighRoller(ctx)
	log.Infof("High Roller: %s", highRoller.Username)
	if err != nil {
		return fmt.Errorf("failed to get current high roller: %w", err)
	}

	if highRoller == nil {
		// No users in database yet
		return nil
	}

	// Get all guild members with the high roller role
	members, err := b.session.GuildMembers(b.config.GuildID, "", 1000)
	log.Infof("guildmembers: %+v", members)
	if err != nil {
		return fmt.Errorf("failed to get guild members: %w", err)
	}

	// Find who currently has the role
	var currentHolders []string
	highRollerDiscordID := strconv.FormatInt(highRoller.DiscordID, 10)

	for _, member := range members {
		for _, roleID := range member.Roles {
			if roleID == b.config.HighRollerRoleID {
				currentHolders = append(currentHolders, member.User.ID)
				break
			}
		}
	}

	// Remove role from anyone who shouldn't have it
	for _, holderID := range currentHolders {
		if holderID != highRollerDiscordID {
			if err := b.session.GuildMemberRoleRemove(b.config.GuildID, holderID, b.config.HighRollerRoleID); err != nil {
				log.Errorf("Failed to remove high roller role from user %s: %v", holderID, err)
			} else {
				log.Infof("Removed high roller role from user %s", holderID)
			}
		}
	}

	// Add role to the high roller if they don't have it
	hasRole := false
	for _, holderID := range currentHolders {
		if holderID == highRollerDiscordID {
			hasRole = true
			break
		}
	}

	if !hasRole {
		if err := b.session.GuildMemberRoleAdd(b.config.GuildID, highRollerDiscordID, b.config.HighRollerRoleID); err != nil {
			log.Errorf("Failed to add high roller role to user %s: %v", highRollerDiscordID, err)
		} else {
			log.Infof("Added high roller role to user %s (balance: %d)", highRollerDiscordID, highRoller.Balance)
		}
	}

	return nil
}

// handleWagerInteractions handles wager-related component interactions and modals
func (b *Bot) handleWagerInteractions(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionMessageComponent:
		customID := i.MessageComponentData().CustomID
		// Check if this is a wager-related interaction
		if strings.HasPrefix(customID, "wager_") {
			b.handleWagerInteraction(s, i)
		}

	case discordgo.InteractionModalSubmit:
		customID := i.ModalSubmitData().CustomID
		// Check if this is a wager condition modal
		if strings.HasPrefix(customID, "wager_condition_modal_") {
			b.handleWagerConditionModal(s, i)
		}
	}
}

// startSessionCleanup runs periodic cleanup of old bet sessions
func (b *Bot) startSessionCleanup() {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		cleanupSessions()
	}
}

func (b *Bot) registerCommands() error {
	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "balance",
			Description: "Check your current balance",
		},
		{
			Name:        "gamble",
			Description: "Open the interactive betting interface",
		},
		{
			Name:        "donate",
			Description: "Transfer bits to another player",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "amount",
					Description: "Amount to donate in bits",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "user",
					Description: "User to donate to",
					Required:    true,
				},
			},
		},
		{
			Name:        "wager",
			Description: "Create and manage wagers",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "propose",
					Description: "Propose a wager against another player",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionUser,
							Name:        "user",
							Description: "User to wager against",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "amount",
							Description: "Amount to wager in bits",
							Required:    true,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "list",
					Description: "List your active wagers",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "cancel",
					Description: "Cancel a proposed wager",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "id",
							Description: "Wager ID to cancel",
							Required:    true,
						},
					},
				},
			},
		},
		{
			Name:        "groupwager",
			Description: "Create and manage group wagers",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "create",
					Description: "Create a new group wager (opens modal for details)",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "resolve",
					Description: "Resolve a group wager (resolvers only)",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "id",
							Description: "Group wager ID to resolve",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "winning_option",
							Description: "ID of the winning option",
							Required:    true,
						},
					},
				},
			},
		},
		{
			Name:        "stats",
			Description: "View player statistics",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "scoreboard",
					Description: "Display the top players scoreboard",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "balance",
					Description: "Display detailed statistics for a player",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionUser,
							Name:        "user",
							Description: "User to check stats for (defaults to you)",
							Required:    false,
						},
					},
				},
			},
		},
	}

	for _, cmd := range commands {
		_, err := b.session.ApplicationCommandCreate(b.session.State.User.ID, "", cmd)
		if err != nil {
			return fmt.Errorf("cannot create '%s' command: %w", cmd.Name, err)
		}
	}

	// Delete a command by ID
	// err := b.session.ApplicationCommandDelete(b.session.State.User.ID, "", "1391958820778938440")
	// if err != nil {
	// 	log.Errorf("%s", err)
	// }

	return nil
}

func (b *Bot) handleCommands(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	switch i.ApplicationCommandData().Name {
	case "balance":
		b.handleBalance(s, i)
	case "gamble":
		b.handleBetCommand(s, i)
	case "donate":
		b.handleDonate(s, i)
	case "wager":
		b.handleWagerCommand(s, i)
	case "groupwager":
		b.handleGroupWagerCommand(s, i)
	case "stats":
		b.handleStatsCommand(s, i)
	}
}

func (b *Bot) handleBalance(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()

	// Convert Discord string ID to int64
	discordID, err := strconv.ParseInt(i.Member.User.ID, 10, 64)
	if err != nil {
		log.Printf("Error parsing Discord ID %s: %v", i.Member.User.ID, err)
		b.respondWithError(s, i, "Unable to process request. Please try again.")
		return
	}

	// Get or create user
	user, err := b.userService.GetOrCreateUser(ctx, discordID, i.Member.User.Username)
	if err != nil {
		log.Printf("Error getting user %d: %v", discordID, err)
		b.respondWithError(s, i, "Unable to retrieve balance. Please try again.")
		return
	}

	// Get display name
	displayName := GetDisplayName(s, i.GuildID, i.Member.User.ID)

	// Format and send response
	message := fmt.Sprintf("%s, your current balance: **%s bits**", displayName, FormatBalance(user.AvailableBalance))
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: message,
		},
	})
	if err != nil {
		log.Printf("Error responding to balance command: %v", err)
	}
}

func (b *Bot) handleDonate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()

	// Extract command options
	options := i.ApplicationCommandData().Options
	if len(options) != 2 {
		b.respondWithError(s, i, "Invalid command options. Please provide both amount and user.")
		return
	}

	// Extract amount
	var amount int64
	var recipientUser *discordgo.User
	for _, opt := range options {
		switch opt.Name {
		case "amount":
			amount = opt.IntValue()
		case "user":
			recipientUser = opt.UserValue(s)
		}
	}

	if amount <= 0 {
		b.respondWithError(s, i, "Amount must be positive.")
		return
	}

	if recipientUser == nil {
		b.respondWithError(s, i, "Invalid recipient user.")
		return
	}

	// Convert Discord string IDs to int64
	fromDiscordID, err := strconv.ParseInt(i.Member.User.ID, 10, 64)
	if err != nil {
		log.Printf("Error parsing sender Discord ID %s: %v", i.Member.User.ID, err)
		b.respondWithError(s, i, "Unable to process request. Please try again.")
		return
	}

	toDiscordID, err := strconv.ParseInt(recipientUser.ID, 10, 64)
	if err != nil {
		log.Printf("Error parsing recipient Discord ID %s: %v", recipientUser.ID, err)
		b.respondWithError(s, i, "Unable to process request. Please try again.")
		return
	}

	// Check for self-transfer
	if fromDiscordID == toDiscordID {
		b.respondWithError(s, i, "You cannot donate to yourself.")
		return
	}

	// Ensure both users exist in the database
	_, err = b.userService.GetOrCreateUser(ctx, fromDiscordID, i.Member.User.Username)
	if err != nil {
		log.Printf("Error getting/creating sender user %d: %v", fromDiscordID, err)
		b.respondWithError(s, i, "Unable to process request. Please try again.")
		return
	}

	_, err = b.userService.GetOrCreateUser(ctx, toDiscordID, recipientUser.Username)
	if err != nil {
		log.Printf("Error getting/creating recipient user %d: %v", toDiscordID, err)
		b.respondWithError(s, i, "Unable to process request. Please try again.")
		return
	}

	// Perform the transfer
	result, err := b.transferService.Transfer(ctx, fromDiscordID, toDiscordID, amount)
	if err != nil {
		log.Printf("Error transferring %d bits from %d to %d: %v", amount, fromDiscordID, toDiscordID, err)

		// Check for specific error types to provide better user feedback
		if err.Error() == fmt.Sprintf("insufficient balance: have %d, need %d", 0, amount) ||
			err.Error()[:20] == "insufficient balance" {
			b.respondWithError(s, i, "Insufficient balance for this donation.")
		} else {
			b.respondWithError(s, i, "Unable to process donation. Please try again.")
		}
		return
	}

	// Get display names
	senderName := GetDisplayName(s, i.GuildID, i.Member.User.ID)
	recipientName := GetDisplayName(s, i.GuildID, recipientUser.ID)

	// Format and send success response
	message := fmt.Sprintf("✅ **%s** transferred **%s bits** to **%s**",
		senderName, FormatBalance(result.Amount), recipientName)
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: message,
		},
	})
	if err != nil {
		log.Printf("Error responding to donate command: %v", err)
	}
}

func (b *Bot) respondWithError(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: message,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		log.Printf("Error sending error response: %v", err)
	}
}
