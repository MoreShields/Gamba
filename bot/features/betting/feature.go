package betting

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"gambler/bot/common"
	"gambler/service"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

// GamblingConfig holds gambling-specific configuration
type GamblingConfig struct {
	DailyGambleLimit    int64
	DailyLimitResetHour int
}

// Feature represents the betting feature
type Feature struct {
	session         *discordgo.Session
	config          *GamblingConfig
	userService     service.UserService
	gamblingService service.GamblingService
	guildID         string
}

// NewFeature creates a new betting feature instance
func NewFeature(session *discordgo.Session, config *GamblingConfig, userService service.UserService, gamblingService service.GamblingService, guildID string) *Feature {
	f := &Feature{
		session:         session,
		config:          config,
		userService:     userService,
		gamblingService: gamblingService,
		guildID:         guildID,
	}

	// Start session cleanup
	go f.startSessionCleanup()

	return f
}

// HandleCommand handles the /gamble command
func (f *Feature) HandleCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()

	// Parse user ID
	discordID, err := strconv.ParseInt(i.Member.User.ID, 10, 64)
	if err != nil {
		log.Errorf("Error parsing Discord ID %s: %v", i.Member.User.ID, err)
		common.RespondWithError(s, i, "Unable to process request. Please try again.")
		return
	}

	// Get or create user
	user, err := f.userService.GetOrCreateUser(ctx, discordID, i.Member.User.Username)
	if err != nil {
		log.Errorf("Error getting/creating user %d: %v", discordID, err)
		common.RespondWithError(s, i, "Unable to process request. Please try again.")
		return
	}

	// Check daily limit
	remaining, _ := f.gamblingService.CheckDailyLimit(ctx, discordID, 0)
	if remaining == 0 {
		// Format error message with Discord timestamp for reset time
		cfg := f.config
		nextReset := service.GetNextResetTime(cfg.DailyLimitResetHour)
		common.RespondWithError(s, i, fmt.Sprintf("Daily gambling limit of %s bits reached. Try again %s",
			common.FormatBalance(cfg.DailyGambleLimit),
			common.FormatDiscordTimestamp(nextReset, "R")))

		return
	}

	// Create initial embed
	embed := buildInitialBetEmbed(user.AvailableBalance, remaining)
	components := CreateInitialComponents()

	// Send initial response
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
			Flags:      discordgo.MessageFlagsEphemeral,
		},
	})

	if err != nil {
		log.Errorf("Error responding to bet command: %v", err)
		return
	}

	// Create betting session
	msg, err := s.InteractionResponse(i.Interaction)
	if err != nil {
		log.Errorf("Error getting interaction response: %v", err)
		return
	}

	createBetSession(discordID, msg.ID, msg.ChannelID, user.AvailableBalance)
}

// HandleInteraction handles betting-related component interactions
func (f *Feature) HandleInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionMessageComponent:
		f.handleComponentInteraction(s, i)
	case discordgo.InteractionModalSubmit:
		f.handleModalSubmit(s, i)
	}
}

// startSessionCleanup runs periodic cleanup of old bet sessions
func (f *Feature) startSessionCleanup() {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		cleanupSessions()
	}
}
