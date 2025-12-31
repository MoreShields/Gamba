package lottery

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"gambler/discord-client/application"
	"gambler/discord-client/bot/common"
	"gambler/discord-client/domain/entities"
	"gambler/discord-client/domain/interfaces"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

// Feature represents the lottery feature
type Feature struct {
	session    *discordgo.Session
	uowFactory application.UnitOfWorkFactory
}

// NewFeature creates a new lottery feature instance
func NewFeature(session *discordgo.Session, uowFactory application.UnitOfWorkFactory) *Feature {
	return &Feature{
		session:    session,
		uowFactory: uowFactory,
	}
}

// HandleInteraction handles lottery button interactions and modals
func (f *Feature) HandleInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionMessageComponent:
		f.handleComponentInteraction(s, i)
	case discordgo.InteractionModalSubmit:
		f.handleModalSubmit(s, i)
	default:
		log.Warnf("Unknown interaction type in lottery: %v", i.Type)
	}
}

// handleComponentInteraction routes button clicks based on custom ID
func (f *Feature) handleComponentInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID

	// Lottery button interactions use format: lotto_buy_<draw_id>
	if strings.HasPrefix(customID, "lotto_buy_") {
		f.handleBuyButton(s, i)
		return
	}

	common.RespondWithError(s, i, "Unknown lottery interaction")
}

// handleModalSubmit handles lottery modal submissions
func (f *Feature) handleModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.ModalSubmitData().CustomID

	if strings.HasPrefix(customID, "lotto_buy_modal_") {
		f.handleBuyModalSubmit(s, i)
		return
	}

	log.Warnf("Unknown lottery modal customID: %s", customID)
	common.RespondWithError(s, i, "Unknown lottery modal")
}

// PostLotteryResult updates the existing lottery message with draw results (implements LotteryPoster)
func (f *Feature) PostLotteryResult(ctx context.Context, draw *entities.LotteryDraw, result *interfaces.LotteryDrawResult, participants []*entities.LotteryParticipantInfo) error {
	if !draw.HasMessage() {
		log.Warnf("Draw %d has no message to update with results", draw.ID)
		return nil
	}

	channelIDStr := fmt.Sprintf("%d", *draw.ChannelID)
	messageIDStr := fmt.Sprintf("%d", *draw.MessageID)

	// Create result embed
	embed := CreateDrawResultEmbed(result, draw, participants)
	components := CreateCompletedLotteryComponents(draw)

	// Update the existing message
	_, err := f.session.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel:    channelIDStr,
		ID:         messageIDStr,
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
	})
	if err != nil {
		return fmt.Errorf("failed to update lottery message with results: %w", err)
	}

	log.WithFields(log.Fields{
		"draw_id":    draw.ID,
		"channel_id": *draw.ChannelID,
		"message_id": *draw.MessageID,
		"won":        !result.RolledOver,
	}).Info("Posted lottery result to Discord")

	return nil
}

// PostNewLotteryDraw posts a new lottery draw message (implements LotteryPoster)
func (f *Feature) PostNewLotteryDraw(ctx context.Context, drawInfo *interfaces.LotteryDrawInfo, channelID int64) (messageID int64, err error) {
	channelIDStr := fmt.Sprintf("%d", channelID)

	embed := CreateLotteryEmbed(drawInfo)
	components := CreateLotteryComponents(drawInfo.Draw)

	msg, err := f.session.ChannelMessageSendComplex(channelIDStr, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: components,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to post new lottery draw: %w", err)
	}

	messageID, err = strconv.ParseInt(msg.ID, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse message ID: %w", err)
	}

	log.WithFields(log.Fields{
		"draw_id":    drawInfo.Draw.ID,
		"channel_id": channelID,
		"message_id": messageID,
	}).Info("Posted new lottery draw message to Discord")

	return messageID, nil
}

// UpdateLotteryEmbed updates an existing lottery embed (implements LotteryPoster)
func (f *Feature) UpdateLotteryEmbed(ctx context.Context, draw *entities.LotteryDraw, drawInfo *interfaces.LotteryDrawInfo) error {
	if !draw.HasMessage() {
		return nil
	}

	channelIDStr := fmt.Sprintf("%d", *draw.ChannelID)
	messageIDStr := fmt.Sprintf("%d", *draw.MessageID)

	embed := CreateLotteryEmbed(drawInfo)
	components := CreateLotteryComponents(draw)

	_, err := f.session.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel:    channelIDStr,
		ID:         messageIDStr,
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
	})
	if err != nil {
		return fmt.Errorf("failed to update lottery embed: %w", err)
	}

	return nil
}
