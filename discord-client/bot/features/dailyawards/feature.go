package dailyawards

import (
	"context"
	"fmt"
	"strings"

	"gambler/discord-client/application"
	"gambler/discord-client/application/dto"
	"gambler/discord-client/bot/common"
	"gambler/discord-client/service"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

// Feature handles daily awards posting functionality
type Feature struct {
	session    *discordgo.Session
	uowFactory application.UnitOfWorkFactory
}

// NewFeature creates a new daily awards feature instance
func NewFeature(session *discordgo.Session, uowFactory application.UnitOfWorkFactory) *Feature {
	return &Feature{
		session:    session,
		uowFactory: uowFactory,
	}
}

// PostDailyAwardsSummary posts a daily awards summary to a Discord channel
func (f *Feature) PostDailyAwardsSummary(ctx context.Context, guildID int64, channelID string, summary *service.DailyAwardsSummary) error {
	// Create embed fields for different award types
	var fields []*discordgo.MessageEmbedField

	// Group awards by type
	awardsByType := make(map[service.DailyAwardType][]service.DailyAward)
	for _, award := range summary.Awards {
		awardsByType[award.GetType()] = append(awardsByType[award.GetType()], award)
	}

	// Format Wordle awards if present
	if wordleAwards, ok := awardsByType[service.DailyAwardTypeWordle]; ok && len(wordleAwards) > 0 {
		// Build the table content
		var tableContent strings.Builder
		tableContent.WriteString("```\n")
		tableContent.WriteString("User                Score  Reward\n")
		tableContent.WriteString("──────────────────  ─────  ──────\n")

		// Convert guild ID to string for Discord API
		guildIDStr := fmt.Sprintf("%d", guildID)

		for _, award := range wordleAwards {
			// Get username from guild member
			username := f.getUsername(guildIDStr, fmt.Sprintf("%d", award.GetDiscordID()))
			if len(username) > 18 {
				username = username[:15] + "..."
			}

			// Format the row
			tableContent.WriteString(fmt.Sprintf("%-18s  %-5s  %s\n",
				username,
				award.GetDetails(),
				common.FormatBalance(award.GetReward()),
			))
		}
		tableContent.WriteString("```")

		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "🧩 Wordle Completions",
			Value:  tableContent.String(),
			Inline: false,
		})
	}

	// Add total payout field
	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   "💰 Total Payout",
		Value:  common.FormatBalance(summary.TotalPayout),
		Inline: true,
	})

	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   "👥 Recipients",
		Value:  fmt.Sprintf("%d", len(summary.Awards)),
		Inline: true,
	})

	// Add total server bits field
	if summary.TotalServerBits > 0 {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "🏦 Total Server Bits",
			Value:  common.FormatBalance(summary.TotalServerBits),
			Inline: false,
		})
	}

	// Create the embed
	embed := &discordgo.MessageEmbed{
		Title:       "📊 Daily Summary",
		Description: "",
		Color:       common.ColorInfo,
		Fields:      fields,
		Footer:      &discordgo.MessageEmbedFooter{},
	}

	// Send the message
	_, err := f.session.ChannelMessageSendEmbed(channelID, embed)
	if err != nil {
		return fmt.Errorf("failed to send daily awards summary: %w", err)
	}

	return nil
}

// PostDailyAwardsForGuild fetches and posts the daily awards summary for a specific guild
func (f *Feature) PostDailyAwardsForGuild(ctx context.Context, guildID int64) error {
	// Create unit of work for this guild
	uow := f.uowFactory.CreateForGuild(guildID)
	if err := uow.Begin(ctx); err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer uow.Rollback()

	// Get guild settings to check for primary channel
	guildSettingsService := service.NewGuildSettingsService(uow.GuildSettingsRepository())
	settings, err := guildSettingsService.GetOrCreateSettings(ctx, guildID)
	if err != nil {
		return fmt.Errorf("failed to get guild settings: %w", err)
	}

	// Check if primary channel is configured
	if settings.PrimaryChannelID == nil {
		return fmt.Errorf("guild %d has no primary channel configured", guildID)
	}

	// Create daily awards service
	dailyAwardsService := service.NewDailyAwardsService(
		uow.WordleCompletionRepo(),
		uow.UserRepository(),
		uow.WagerRepository(),
		uow.BetRepository(),
		uow.GroupWagerRepository(),
	)

	// Get daily awards summary
	summary, err := dailyAwardsService.GetDailyAwardsSummary(ctx, guildID)
	if err != nil {
		return fmt.Errorf("failed to get daily awards summary: %w", err)
	}

	// Rollback the read-only transaction
	uow.Rollback()

	// Check if there are any awards to post
	if len(summary.Awards) == 0 {
		return fmt.Errorf("no daily awards found for guild %d", guildID)
	}

	// Post the summary to Discord
	channelID := fmt.Sprintf("%d", *settings.PrimaryChannelID)
	if err := f.PostDailyAwardsSummary(ctx, guildID, channelID, summary); err != nil {
		return fmt.Errorf("failed to post daily awards: %w", err)
	}

	log.WithFields(log.Fields{
		"guild_id":   guildID,
		"channel_id": channelID,
		"awards":     len(summary.Awards),
		"source":     "manual_post",
	}).Info("Daily awards summary posted")

	return nil
}

// getUsername fetches the username for a given user ID in a guild
func (f *Feature) getUsername(guildID, userID string) string {
	// Try to get guild member first for nickname
	member, err := f.session.GuildMember(guildID, userID)
	if err == nil && member != nil {
		if member.Nick != "" {
			return member.Nick
		}
		if member.User != nil {
			return member.User.Username
		}
	}

	// Fallback to user lookup
	user, err := f.session.User(userID)
	if err == nil && user != nil {
		return user.Username
	}

	return "Unknown"
}

// PostDailyAwardsSummaryFromDTO posts a daily awards summary from a DTO
func (f *Feature) PostDailyAwardsSummaryFromDTO(ctx context.Context, postDTO dto.DailyAwardsPostDTO) error {
	// Create embed fields for different award types
	var fields []*discordgo.MessageEmbedField

	// Group awards by type
	awardsByType := make(map[string][]dto.DailyAwardDTO)
	for _, award := range postDTO.Summary.Awards {
		awardsByType[award.Type] = append(awardsByType[award.Type], award)
	}

	// Format Wordle awards if present
	if wordleAwards, ok := awardsByType["wordle"]; ok && len(wordleAwards) > 0 {
		// Build the table content
		var tableContent strings.Builder
		tableContent.WriteString("```\n")
		tableContent.WriteString("User                Score  Reward\n")
		tableContent.WriteString("──────────────────  ─────  ──────\n")

		// Convert guild ID to string for Discord API
		guildIDStr := fmt.Sprintf("%d", postDTO.GuildID)

		for _, award := range wordleAwards {
			// Get username from guild member
			username := f.getUsername(guildIDStr, fmt.Sprintf("%d", award.DiscordID))
			if len(username) > 18 {
				username = username[:15] + "..."
			}

			// Use the Details field directly - no parsing needed
			tableContent.WriteString(fmt.Sprintf("%-18s  %-5s  %s\n",
				username,
				award.Details,
				common.FormatBalance(award.Reward),
			))
		}
		tableContent.WriteString("```")

		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "🧩 Wordle Completions",
			Value:  tableContent.String(),
			Inline: false,
		})
	}

	// Add total payout field
	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   "💰 Total Payout",
		Value:  common.FormatBalance(postDTO.Summary.TotalPayout),
		Inline: true,
	})

	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   "👥 Recipients",
		Value:  fmt.Sprintf("%d", len(postDTO.Summary.Awards)),
		Inline: true,
	})

	// Add total server bits field
	if postDTO.Summary.TotalServerBits > 0 {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "🏦 Total Server Bits",
			Value:  common.FormatBalance(postDTO.Summary.TotalServerBits),
			Inline: false,
		})
	}

	// Create the embed
	embed := &discordgo.MessageEmbed{
		Title:       "📊 Daily Summary",
		Description: "",
		Color:       common.ColorInfo,
		Fields:      fields,
		Footer:      &discordgo.MessageEmbedFooter{},
	}

	// Send the message
	channelID := fmt.Sprintf("%d", postDTO.ChannelID)
	_, err := f.session.ChannelMessageSendEmbed(channelID, embed)
	if err != nil {
		return fmt.Errorf("failed to send daily awards summary: %w", err)
	}

	return nil
}
