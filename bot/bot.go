package bot

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gambler/bot/features/balance"
	"gambler/bot/features/betting"
	"gambler/bot/features/groupwagers"
	"gambler/bot/features/stats"
	"gambler/bot/features/transfer"
	"gambler/bot/features/wagers"
	"gambler/events"
	"gambler/service"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

// Config holds bot configuration
type Config struct {
	Token             string
	GuildID           string
	HighRollerRoleID  string
	HighRollerEnabled bool
}

// Bot manages the Discord bot and all feature modules
type Bot struct {
	// Core components
	config   Config
	session  *discordgo.Session
	eventBus *events.Bus

	// Services
	userService       service.UserService
	gamblingService   service.GamblingService
	transferService   service.TransferService
	wagerService      service.WagerService
	statsService      service.StatsService
	groupWagerService service.GroupWagerService

	// Feature modules
	betting     *betting.Feature
	wagers      *wagers.Feature
	groupWagers *groupwagers.Feature
	stats       *stats.Feature
	balance     *balance.Feature
	transfer    *transfer.Feature
}

// New creates a new bot instance with all features
func New(config Config, gamblingConfig *betting.GamblingConfig, userService service.UserService, gamblingService service.GamblingService, transferService service.TransferService, wagerService service.WagerService, statsService service.StatsService, groupWagerService service.GroupWagerService, eventBus *events.Bus) (*Bot, error) {
	// Create Discord session
	dg, err := discordgo.New("Bot " + config.Token)
	if err != nil {
		return nil, fmt.Errorf("error creating discord session: %w", err)
	}
	dg.Identify.Intents = discordgo.IntentsAll

	// Create bot instance
	bot := &Bot{
		config:            config,
		session:           dg,
		eventBus:          eventBus,
		userService:       userService,
		gamblingService:   gamblingService,
		transferService:   transferService,
		wagerService:      wagerService,
		statsService:      statsService,
		groupWagerService: groupWagerService,
	}

	// Create feature modules
	bot.betting = betting.NewFeature(dg, gamblingConfig, userService, gamblingService, config.GuildID)
	bot.wagers = wagers.NewFeature(dg, wagerService, userService, config.GuildID)
	bot.groupWagers = groupwagers.NewFeature(dg, groupWagerService, userService, config.GuildID)
	bot.stats = stats.NewFeature(dg, statsService, userService, config.GuildID)
	bot.balance = balance.New(userService)
	bot.transfer = transfer.New(transferService, userService)

	// Register handlers
	dg.AddHandler(bot.handleCommands)
	dg.AddHandler(bot.handleInteractions)

	// Open websocket connection
	if err := dg.Open(); err != nil {
		return nil, fmt.Errorf("error opening connection: %w", err)
	}

	// Register slash commands with Discord
	if err := bot.registerCommands(); err != nil {
		dg.Close()
		return nil, fmt.Errorf("error registering commands: %w", err)
	}

	// Subscribe to group wager state change events to update Discord embeds
	eventBus.Subscribe(events.EventTypeGroupWagerStateChange, bot.handleGroupWagerStateChange)
	log.Info("Group wager state change listener enabled")

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

// Close gracefully shuts down the bot
func (b *Bot) Close() error {
	return b.session.Close()
}

// GetSession returns the Discord session
func (b *Bot) GetSession() *discordgo.Session {
	return b.session
}

// GetConfig returns the bot configuration
func (b *Bot) GetConfig() Config {
	return b.config
}

// updateHighRollerRole updates the high roller role based on current balances
func (b *Bot) updateHighRollerRole(ctx context.Context) error {
	if !b.config.HighRollerEnabled || b.config.HighRollerRoleID == "" {
		return nil
	}

	// Get the current high roller
	highRoller, err := b.userService.GetCurrentHighRoller(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current high roller: %w", err)
	}

	if highRoller == nil {
		// No users in database yet
		return nil
	}

	// Get all guild members with the high roller role
	members, err := b.session.GuildMembers(b.config.GuildID, "", 1000)
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

// handleCommands routes slash commands to appropriate handlers
func (b *Bot) handleCommands(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	switch i.ApplicationCommandData().Name {
	case "balance":
		b.balance.HandleCommand(s, i)
	case "gamble":
		b.betting.HandleCommand(s, i)
	case "donate":
		b.transfer.HandleCommand(s, i)
	case "wager":
		b.wagers.HandleCommand(s, i)
	case "groupwager":
		b.groupWagers.HandleCommand(s, i)
	case "stats":
		b.stats.HandleCommand(s, i)
	}
}

// handleInteractions routes component interactions to appropriate features
func (b *Bot) handleInteractions(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionMessageComponent:
		customID := i.MessageComponentData().CustomID
		b.routeComponentInteraction(s, i, customID)

	case discordgo.InteractionModalSubmit:
		customID := i.ModalSubmitData().CustomID
		b.routeModalInteraction(s, i, customID)
	}
}

// routeComponentInteraction routes button and select menu interactions
func (b *Bot) routeComponentInteraction(s *discordgo.Session, i *discordgo.InteractionCreate, customID string) {
	switch {
	case strings.HasPrefix(customID, "bet_"):
		b.betting.HandleInteraction(s, i)

	case strings.HasPrefix(customID, "wager_"):
		b.wagers.HandleInteraction(s, i)

	case strings.HasPrefix(customID, "groupwager_"), strings.HasPrefix(customID, "group_wager_"):
		b.groupWagers.HandleInteraction(s, i)
	}
}

// routeModalInteraction routes modal submit interactions
func (b *Bot) routeModalInteraction(s *discordgo.Session, i *discordgo.InteractionCreate, customID string) {
	switch {
	case strings.HasPrefix(customID, "wager_condition_modal_"):
		b.wagers.HandleInteraction(s, i)

	case strings.HasPrefix(customID, "groupwager_"), strings.HasPrefix(customID, "group_wager_"):
		b.groupWagers.HandleInteraction(s, i)

	case customID == "bet_amount_modal":
		b.betting.HandleInteraction(s, i)
	}
}

// handleGroupWagerStateChange handles group wager state change events and updates Discord embeds
func (b *Bot) handleGroupWagerStateChange(ctx context.Context, event events.Event) {
	e, ok := event.(events.GroupWagerStateChangeEvent)
	if !ok {
		return
	}

	// Skip if no message to update
	if e.MessageID == 0 || e.ChannelID == 0 {
		return
	}

	// Get updated wager details
	detail, err := b.groupWagerService.GetGroupWagerDetail(ctx, e.GroupWagerID)
	if err != nil {
		log.Errorf("Failed to get group wager detail for event update: %v", err)
		return
	}

	// Create updated embed and components
	embed := groupwagers.CreateGroupWagerEmbed(detail)
	components := groupwagers.CreateGroupWagerComponents(detail)

	// Update the Discord message
	channelID := strconv.FormatInt(e.ChannelID, 10)
	messageID := strconv.FormatInt(e.MessageID, 10)

	_, err = b.session.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel:    channelID,
		ID:         messageID,
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
	})

	if err != nil {
		log.Errorf("Failed to update group wager message (channel: %s, message: %s): %v",
			channelID, messageID, err)
	} else {
		log.Debugf("Successfully updated group wager message for wager %d (state: %s -> %s)",
			e.GroupWagerID, e.OldState, e.NewState)
	}
}
