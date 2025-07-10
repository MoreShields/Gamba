package balance

import (
	"context"
	"fmt"
	"strconv"

	"github.com/bwmarrin/discordgo"
	"gambler/bot/common"
	log "github.com/sirupsen/logrus"
)

func (f *Feature) handleBalance(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()

	// Convert Discord string ID to int64
	discordID, err := strconv.ParseInt(i.Member.User.ID, 10, 64)
	if err != nil {
		log.Errorf("Error parsing Discord ID %s: %v", i.Member.User.ID, err)
		common.RespondWithError(s, i, "Unable to process request. Please try again.")
		return
	}

	// Get or create user
	user, err := f.userService.GetOrCreateUser(ctx, discordID, i.Member.User.Username)
	if err != nil {
		log.Errorf("Error getting user %d: %v", discordID, err)
		common.RespondWithError(s, i, "Unable to retrieve balance. Please try again.")
		return
	}

	// Get display name
	displayName := common.GetDisplayName(s, i.GuildID, i.Member.User.ID)

	// Format and send response
	message := fmt.Sprintf("%s, your current balance: **%s bits**", displayName, common.FormatBalance(user.AvailableBalance))
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: message,
		},
	})
	if err != nil {
		log.Errorf("Error responding to balance command: %v", err)
	}
}