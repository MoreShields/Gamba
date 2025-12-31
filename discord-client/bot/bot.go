package bot

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"gambler/discord-client/application"
	"gambler/discord-client/application/dto"
	"gambler/discord-client/bot/features/balance"
	"gambler/discord-client/bot/features/betting"
	"gambler/discord-client/bot/features/dailyawards"
	"gambler/discord-client/bot/features/groupwagers"
	"gambler/discord-client/bot/features/highroller"
	"gambler/discord-client/bot/features/housewagers"
	"gambler/discord-client/bot/features/lottery"
	"gambler/discord-client/bot/features/settings"
	"gambler/discord-client/bot/features/stats"
	"gambler/discord-client/bot/features/summoner"
	"gambler/discord-client/bot/features/transfer"
	"gambler/discord-client/bot/features/wagers"
	"gambler/discord-client/domain/interfaces"
	"gambler/discord-client/domain/services"

	summoner_pb "gambler/discord-client/proto/services"

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
	config         Config
	session        *discordgo.Session
	uowFactory     application.UnitOfWorkFactory
	summonerClient summoner_pb.SummonerTrackingServiceClient
	userResolver   application.UserResolver

	// Event publishing
	eventPublisher interfaces.EventPublisher

	// Feature modules
	betting     *betting.Feature
	wagers      *wagers.Feature
	groupWagers *groupwagers.Feature
	houseWagers *housewagers.Feature
	stats       *stats.Feature
	balance     *balance.Feature
	transfer    *transfer.Feature
	settings    *settings.Feature
	summoner    *summoner.Feature
	dailyAwards *dailyawards.Feature
	highroller  *highroller.Feature
	lottery     *lottery.Feature

	// Worker cleanup functions
	stopGroupWagerWorker  func()
	stopDailyAwardsWorker func()
}

// New creates a new bot instance with all features
func New(config Config, uowFactory application.UnitOfWorkFactory, summonerClient summoner_pb.SummonerTrackingServiceClient, eventPublisher interfaces.EventPublisher) (*Bot, error) {
	// Create Discord session
	dg, err := discordgo.New("Bot " + config.Token)
	if err != nil {
		return nil, fmt.Errorf("error creating discord session: %w", err)
	}
	dg.Identify.Intents = discordgo.IntentsAll

	// Create shared components
	userResolver := NewUserResolver(dg)

	// Create bot instance
	bot := &Bot{
		config:         config,
		session:        dg,
		uowFactory:     uowFactory,
		summonerClient: summonerClient,
		eventPublisher: eventPublisher,
		userResolver:   userResolver,
	}

	// Create feature modules
	bot.betting = betting.New(uowFactory)
	bot.wagers = wagers.NewFeature(dg, uowFactory, config.GuildID)
	bot.groupWagers = groupwagers.NewFeature(dg, uowFactory)
	bot.houseWagers = housewagers.NewFeature(dg, uowFactory)
	bot.stats = stats.NewFeature(dg, uowFactory, config.GuildID, userResolver)
	bot.balance = balance.New(uowFactory)
	bot.transfer = transfer.New(uowFactory)
	bot.summoner = summoner.NewFeature(dg, uowFactory, summonerClient, config.GuildID)
	bot.dailyAwards = dailyawards.NewFeature(dg, uowFactory)
	bot.highroller = highroller.NewFeature(dg, uowFactory)
	bot.lottery = lottery.NewFeature(dg, uowFactory)
	// Settings depends on lottery for posting lottery messages when channel is configured
	bot.settings = settings.NewFeature(dg, uowFactory, bot.lottery)

	// Register handlers
	dg.AddHandler(bot.handleCommands)
	dg.AddHandler(bot.handleInteractions)
	dg.AddHandler(bot.handleGuildCreate)
	dg.AddHandler(bot.handleMessageCreate)

	// Open websocket connection
	if err := dg.Open(); err != nil {
		return nil, fmt.Errorf("error opening connection: %w", err)
	}

	// Register slash commands with Discord
	if err := bot.registerCommands(); err != nil {
		dg.Close()
		return nil, fmt.Errorf("error registering commands: %w", err)
	}


	// Start background workers
	ctx := context.Background()
	bot.stopGroupWagerWorker = bot.StartGroupWagerExpirationWorker(ctx)
	log.Info("Background workers started")

	// Always start debug API
	debugPort := 8899
	if err := bot.StartDebugAPI(debugPort); err != nil {
		log.Warnf("Failed to start debug API on port %d: %v", debugPort, err)
	}

	return bot, nil
}

// GetDiscordPoster returns a DiscordPoster implementation that supports both operations
func (b *Bot) GetDiscordPoster() application.DiscordPoster {
	return &discordPoster{
		houseWagers: b.houseWagers,
		groupWagers: b.groupWagers,
		dailyAwards: b.dailyAwards,
	}
}

// Close gracefully shuts down the bot
func (b *Bot) Close() error {
	// Stop background workers
	if b.stopGroupWagerWorker != nil {
		b.stopGroupWagerWorker()
	}
	if b.stopDailyAwardsWorker != nil {
		b.stopDailyAwardsWorker()
	}
	log.Info("Background workers stopped")

	return b.session.Close()
}

// GetSession returns the Discord session
func (b *Bot) GetSession() *discordgo.Session {
	return b.session
}

// SetDailyAwardsWorkerCleanup sets the cleanup function for the daily awards worker
func (b *Bot) SetDailyAwardsWorkerCleanup(cleanup func()) {
	b.stopDailyAwardsWorker = cleanup
}

// GetConfig returns the bot configuration
func (b *Bot) GetConfig() Config {
	return b.config
}

// GetLotteryPoster returns the lottery feature as a LotteryPoster
func (b *Bot) GetLotteryPoster() application.LotteryPoster {
	return b.lottery
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
	case "summoner":
		b.summoner.HandleCommand(s, i)
	case "highroller":
		b.highroller.HandleCommand(s, i)
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

	case strings.HasPrefix(customID, "house_wager_"):
		b.houseWagers.HandleInteraction(s, i)

	case strings.HasPrefix(customID, "stats_"):
		b.stats.HandleInteraction(s, i)

	case strings.HasPrefix(customID, "lotto_"):
		b.lottery.HandleInteraction(s, i)
	}
}

// routeModalInteraction routes modal submit interactions
func (b *Bot) routeModalInteraction(s *discordgo.Session, i *discordgo.InteractionCreate, customID string) {
	switch {
	case strings.HasPrefix(customID, "wager_condition_modal_"):
		b.wagers.HandleInteraction(s, i)

	case strings.HasPrefix(customID, "groupwager_"), strings.HasPrefix(customID, "group_wager_"):
		b.groupWagers.HandleInteraction(s, i)

	case strings.HasPrefix(customID, "house_wager_"):
		b.houseWagers.HandleInteraction(s, i)

	case customID == "bet_amount_modal":
		b.betting.HandleInteraction(s, i)

	case strings.HasPrefix(customID, "lotto_"):
		b.lottery.HandleInteraction(s, i)
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

	// Create guild-scoped unit of work
	uow := b.uowFactory.CreateForGuild(guildID)
	if err := uow.Begin(ctx); err != nil {
		log.Errorf("Failed to begin transaction: %v", err)
		return
	}
	defer uow.Rollback()

	// Instantiate service with repositories from UnitOfWork
	guildSettingsService := services.NewGuildSettingsService(
		uow.GuildSettingsRepository(),
	)

	// Get or create settings for this guild
	settings, err := guildSettingsService.GetOrCreateSettings(ctx, guildID)
	if err != nil {
		log.Errorf("Failed to track new guild %s (%s): %v", g.Name, g.ID, err)
		return
	}

	// Commit the transaction
	if err := uow.Commit(); err != nil {
		log.Errorf("Failed to commit transaction: %v", err)
		return
	}

	log.Infof("Bot joined new guild: %s (ID: %d, Primary Channel: %v, High Roller Role: %v, lol-channel: %v)",
		g.Name, settings.GuildID, settings.PrimaryChannelID, settings.HighRollerRoleID, settings.LolChannelID)
}

// discordPoster implements the application.DiscordPoster interface
// by delegating to the appropriate feature based on the operation
type discordPoster struct {
	houseWagers *housewagers.Feature
	groupWagers *groupwagers.Feature
	dailyAwards *dailyawards.Feature
}

// PostHouseWager delegates to the houseWagers feature
func (p *discordPoster) PostHouseWager(ctx context.Context, dto dto.HouseWagerPostDTO) (*application.PostResult, error) {
	return p.houseWagers.PostHouseWager(ctx, dto)
}

// UpdateHouseWager delegates to the houseWagers feature
func (p *discordPoster) UpdateHouseWager(ctx context.Context, messageID, channelID int64, dto dto.HouseWagerPostDTO) error {
	return p.houseWagers.UpdateHouseWager(ctx, messageID, channelID, dto)
}

// UpdateGroupWager delegates to the groupWagers feature
func (p *discordPoster) UpdateGroupWager(ctx context.Context, messageID, channelID int64, detail interface{}) error {
	return p.groupWagers.UpdateGroupWager(ctx, messageID, channelID, detail)
}

// PostDailyAwards delegates to the dailyAwards feature
func (p *discordPoster) PostDailyAwards(ctx context.Context, dto dto.DailyAwardsPostDTO) error {
	// Pass the DTO directly to the feature which will handle formatting
	return p.dailyAwards.PostDailyAwardsSummaryFromDTO(ctx, dto)
}

// handleMessageCreate handles incoming Discord messages and publishes them to NATS if configured
func (b *Bot) handleMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Skip messages from our own bot to avoid loops
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Skip if message is not from a guild
	if m.GuildID == "" {
		log.Debugf("Skipping message %s - not from a guild (possibly a DM)", m.ID)
		return
	}

	ctx := context.Background()

	// Publish Discord message as a domain event
	// The infrastructure layer will handle local handlers and NATS publishing
	if err := b.publishDiscordMessage(ctx, m); err != nil {
		log.WithError(err).WithFields(log.Fields{
			"guild_id":   m.GuildID,
			"channel_id": m.ChannelID,
			"message_id": m.ID,
		}).Error("Failed to publish Discord message event")
	}
}

// ReplayMessage fetches a message and replays it as if just received
func (b *Bot) ReplayMessage(channelID, messageID string) error {
	// Fetch the message from Discord API
	msg, err := b.session.ChannelMessage(channelID, messageID)
	if err != nil {
		return fmt.Errorf("failed to fetch message %s from channel %s: %w", messageID, channelID, err)
	}

	// Fetch channel info to get the guild ID
	channel, err := b.session.Channel(channelID)
	if err != nil {
		return fmt.Errorf("failed to fetch channel %s: %w", channelID, err)
	}

	// Set the guild ID on the message (it's not populated by ChannelMessage)
	msg.GuildID = channel.GuildID

	// Create MessageCreate event from the fetched message
	msgCreate := &discordgo.MessageCreate{
		Message: msg,
	}

	// Log the replay action
	log.WithFields(log.Fields{
		"channel_id": channelID,
		"message_id": messageID,
		"guild_id":   msg.GuildID,
		"content":    msg.Content,
		"author":     msg.Author.Username,
		"source":     "debug_replay",
	}).Info("Replaying Discord message")

	// Call the handler directly - this ensures all registered handlers work
	b.handleMessageCreate(b.session, msgCreate)

	return nil
}

// GetGuilds returns a list of guilds the bot is connected to
func (b *Bot) GetGuilds() []GuildInfo {
	guilds := make([]GuildInfo, 0)

	// Get all guilds from Discord session
	for _, guild := range b.session.State.Guilds {
		guilds = append(guilds, GuildInfo{
			ID:   guild.ID,
			Name: guild.Name,
		})
	}

	// If no guilds in state, try fetching them
	if len(guilds) == 0 {
		log.Warn("No guilds in session state, attempting to fetch user guilds")
		userGuilds, err := b.session.UserGuilds(100, "", "", false)
		if err != nil {
			log.Errorf("Failed to fetch user guilds: %v", err)
		} else {
			for _, guild := range userGuilds {
				log.Infof("User guild found: ID=%s, Name=%s", guild.ID, guild.Name)
				guilds = append(guilds, GuildInfo{
					ID:   guild.ID,
					Name: guild.Name,
				})
			}
		}
	}

	return guilds
}

// PostDailyAwardsForGuild manually posts the daily awards summary for a specific guild
func (b *Bot) PostDailyAwardsForGuild(guildIDStr string) error {
	// Parse guild ID
	guildID, err := strconv.ParseInt(guildIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid guild ID: %w", err)
	}

	ctx := context.Background()
	return b.dailyAwards.PostDailyAwardsForGuild(ctx, guildID)
}
