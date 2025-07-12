package bot

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gambler/bot/common"
	"gambler/bot/features/balance"
	"gambler/bot/features/betting"
	"gambler/bot/features/groupwagers"
	"gambler/bot/features/settings"
	"gambler/bot/features/stats"
	"gambler/bot/features/transfer"
	"gambler/bot/features/wagers"
	"gambler/events"
	"gambler/models"
	"gambler/service"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

// Config holds bot configuration
type Config struct {
	Token          string
	GuildID        string
	GambaChannelID string
}

// Bot manages the Discord bot and all feature modules
type Bot struct {
	// Core components
	config   Config
	session  *discordgo.Session
	eventBus *events.Bus

	// High roller tracking
	lastHighRollerID int64

	// Services
	userService          service.UserService
	gamblingService      service.GamblingService
	transferService      service.TransferService
	wagerService         service.WagerService
	statsService         service.StatsService
	groupWagerService    service.GroupWagerService
	guildSettingsService service.GuildSettingsService

	// Feature modules
	betting     *betting.Feature
	wagers      *wagers.Feature
	groupWagers *groupwagers.Feature
	stats       *stats.Feature
	balance     *balance.Feature
	transfer    *transfer.Feature
	settings    *settings.Feature
}

// New creates a new bot instance with all features
func New(config Config, gamblingConfig *betting.GamblingConfig, userService service.UserService, gamblingService service.GamblingService, transferService service.TransferService, wagerService service.WagerService, statsService service.StatsService, groupWagerService service.GroupWagerService, guildSettingsService service.GuildSettingsService, eventBus *events.Bus) (*Bot, error) {
	// Create Discord session
	dg, err := discordgo.New("Bot " + config.Token)
	if err != nil {
		return nil, fmt.Errorf("error creating discord session: %w", err)
	}
	dg.Identify.Intents = discordgo.IntentsAll

	// Create bot instance
	bot := &Bot{
		config:               config,
		session:              dg,
		eventBus:             eventBus,
		userService:          userService,
		gamblingService:      gamblingService,
		transferService:      transferService,
		wagerService:         wagerService,
		statsService:         statsService,
		groupWagerService:    groupWagerService,
		guildSettingsService: guildSettingsService,
	}

	// Create feature modules
	bot.betting = betting.NewFeature(dg, gamblingConfig, userService, gamblingService, config.GuildID)
	bot.wagers = wagers.NewFeature(dg, wagerService, userService, config.GuildID)
	bot.groupWagers = groupwagers.NewFeature(dg, groupWagerService, userService, config.GuildID)
	bot.stats = stats.NewFeature(dg, statsService, userService, config.GuildID)
	bot.balance = balance.New(userService)
	bot.transfer = transfer.New(transferService, userService)
	bot.settings = settings.NewFeature(dg, guildSettingsService)

	// Register handlers
	dg.AddHandler(bot.handleCommands)
	dg.AddHandler(bot.handleInteractions)
	dg.AddHandler(bot.handleGuildCreate)

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
	eventBus.Subscribe(events.EventTypeBalanceChange, func(ctx context.Context, event events.Event) {
		if _, ok := event.(events.BalanceChangeEvent); ok {
			// TODO: Extract guild ID from the balance change event when user table includes guild_id
			// For now, use the primary guild from config as a fallback
			if config.GuildID != "" {
				guildID, err := strconv.ParseInt(config.GuildID, 10, 64)
				if err != nil {
					log.Errorf("Failed to parse primary guild ID: %v", err)
					return
				}
				if err := bot.updateHighRollerRole(ctx, guildID); err != nil {
					log.Errorf("Failed to update high roller role: %v", err)
				}
			} else {
				log.Warn("No primary guild configured, skipping high roller role update")
			}
		}
	})
	log.Info("High roller role management enabled")

	// Perform initial sync of high roller role for all guilds
	go func() {
		// Wait a moment for Discord connection to be fully established
		time.Sleep(2 * time.Second)
		ctx := context.Background()
		
		// Get all guilds the bot is in
		guilds := bot.session.State.Guilds
		log.Infof("Syncing high roller roles for %d guilds", len(guilds))
		
		for _, guild := range guilds {
			guildID, err := strconv.ParseInt(guild.ID, 10, 64)
			if err != nil {
				log.Errorf("Failed to parse guild ID %s: %v", guild.ID, err)
				continue
			}
			
			if err := bot.updateHighRollerRole(ctx, guildID); err != nil {
				log.Errorf("Failed to sync high roller role for guild %s: %v", guild.Name, err)
			} else {
				log.Infof("High roller role synced for guild %s", guild.Name)
			}
		}
		log.Info("High roller role sync completed for all guilds")
	}()

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
func (b *Bot) updateHighRollerRole(ctx context.Context, guildID int64) error {
	// Get guild-specific settings
	settings, err := b.guildSettingsService.GetOrCreateSettings(ctx, guildID)
	if err != nil {
		return fmt.Errorf("failed to get guild settings: %w", err)
	}

	// Check if high roller feature is enabled for this guild
	if settings.HighRollerRoleID == nil {
		return nil // Feature disabled for this guild
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

	// Check if the high roller has changed
	hasChanged := b.lastHighRollerID != highRoller.DiscordID
	if hasChanged && b.lastHighRollerID != 0 {
		// Post notification message in the gamba channel
		b.postHighRollerChangeMessage(ctx, guildID, highRoller)
	}

	// Update the tracked high roller
	b.lastHighRollerID = highRoller.DiscordID

	// Get all guild members with the high roller role
	guildIDStr := strconv.FormatInt(guildID, 10)
	members, err := b.session.GuildMembers(guildIDStr, "", 1000)
	if err != nil {
		return fmt.Errorf("failed to get guild members: %w", err)
	}

	// Find who currently has the role
	var currentHolders []string
	highRollerDiscordID := strconv.FormatInt(highRoller.DiscordID, 10)
	roleIDStr := strconv.FormatInt(*settings.HighRollerRoleID, 10)

	for _, member := range members {
		for _, roleID := range member.Roles {
			if roleID == roleIDStr {
				currentHolders = append(currentHolders, member.User.ID)
				break
			}
		}
	}

	// Remove role from anyone who shouldn't have it
	for _, holderID := range currentHolders {
		if holderID != highRollerDiscordID {
			if err := b.session.GuildMemberRoleRemove(guildIDStr, holderID, roleIDStr); err != nil {
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
		if err := b.session.GuildMemberRoleAdd(guildIDStr, highRollerDiscordID, roleIDStr); err != nil {
			log.Errorf("Failed to add high roller role to user %s: %v", highRollerDiscordID, err)
		} else {
			log.Infof("Added high roller role to user %s (balance: %d)", highRollerDiscordID, highRoller.Balance)
		}
	}

	return nil
}

// postHighRollerChangeMessage posts a message to the gamba channel when the high roller changes
func (b *Bot) postHighRollerChangeMessage(ctx context.Context, guildID int64, newHighRoller *models.User) {
	if b.config.GambaChannelID == "" {
		return
	}

	// Get the scoreboard
	entries, err := b.statsService.GetScoreboard(ctx, 10)
	if err != nil {
		log.Errorf("Failed to get scoreboard for high roller notification: %v", err)
		return
	}

	// Create the scoreboard embed
	guildIDStr := strconv.FormatInt(guildID, 10)
	embed := stats.BuildScoreboardEmbed(entries, b.session, guildIDStr)

	// Update the title to indicate a new high roller
	embed.Title = "👑 NEW HIGH ROLLER! 👑"

	// Create the message content with mention
	highRollerDiscordID := strconv.FormatInt(newHighRoller.DiscordID, 10)
	content := fmt.Sprintf("🎉 Congratulations <@%s>! You are now the high roller with **%s bits**! 🎉",
		highRollerDiscordID, common.FormatBalance(newHighRoller.Balance))

	// Send the message
	_, err = b.session.ChannelMessageSendComplex(b.config.GambaChannelID, &discordgo.MessageSend{
		Content: content,
		Embeds:  []*discordgo.MessageEmbed{embed},
	})

	if err != nil {
		log.Errorf("Failed to send high roller change message to channel %s: %v", b.config.GambaChannelID, err)
	} else {
		log.Infof("Posted high roller change notification for user %d", newHighRoller.DiscordID)
	}
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
	case "settings":
		b.settings.HandleCommand(s, i)
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

// handleGuildCreate handles when the bot joins a new guild
func (b *Bot) handleGuildCreate(s *discordgo.Session, g *discordgo.GuildCreate) {
	ctx := context.Background()

	guildID, err := strconv.ParseInt(g.ID, 10, 64)
	if err != nil {
		log.Errorf("Failed to parse guild ID %s: %v", g.ID, err)
		return
	}

	// Get or create settings for this guild
	settings, err := b.guildSettingsService.GetOrCreateSettings(ctx, guildID)
	if err != nil {
		log.Errorf("Failed to track new guild %s (%s): %v", g.Name, g.ID, err)
		return
	}

	log.Infof("Bot joined new guild: %s (ID: %d, Primary Channel: %v, High Roller Role: %v)",
		g.Name, settings.GuildID, settings.PrimaryChannelID, settings.HighRollerRoleID)
}
