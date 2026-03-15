package betwin

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"gambler/discord-client/application/dto"
	"gambler/discord-client/bot/common"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

// handleBetwin processes the /betwin command
func (f *Feature) handleBetwin(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()

	discordID, err := strconv.ParseInt(i.Member.User.ID, 10, 64)
	if err != nil {
		log.Errorf("Error parsing Discord ID %s: %v", i.Member.User.ID, err)
		common.RespondWithError(s, i, "Unable to process request. Please try again.")
		return
	}

	guildID, err := strconv.ParseInt(i.GuildID, 10, 64)
	if err != nil {
		log.Errorf("Error parsing guild ID %s: %v", i.GuildID, err)
		common.RespondWithError(s, i, "Unable to process request. Please try again.")
		return
	}

	options := i.ApplicationCommandData().Options
	var summonerStr string
	var amount int64

	for _, opt := range options {
		switch opt.Name {
		case "summoner":
			summonerStr = opt.StringValue()
		case "amount":
			amount = opt.IntValue()
		}
	}

	// Parse summoner string into name and tag (format: "name#tag")
	parts := strings.SplitN(summonerStr, "#", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		common.RespondWithError(s, i, "Invalid summoner format. Please select a summoner from the autocomplete list.")
		return
	}

	req := dto.PlaceStagedBetRequest{
		GuildID:      guildID,
		DiscordID:    discordID,
		Username:     i.Member.User.Username,
		SummonerName: parts[0],
		TagLine:      parts[1],
		Amount:       amount,
	}

	resp, err := f.stagedBetHandler.PlaceStagedBet(ctx, req)
	if err != nil {
		log.WithFields(log.Fields{
			"discordID": discordID,
			"guildID":   guildID,
			"summoner":  summonerStr,
			"amount":    amount,
			"error":     err,
		}).Warn("Failed to place staged bet")
		common.RespondWithError(s, i, err.Error())
		return
	}

	// Build success response
	action := "Queued"
	if resp.IsUpdate {
		action = "Updated queued"
	}

	message := fmt.Sprintf("%s bet of **%s bits** on **%s #%s**'s next game.\nRemaining balance: **%s bits**",
		action,
		common.FormatBalance(resp.Amount),
		resp.SummonerName, resp.TagLine,
		common.FormatBalance(resp.RemainingBalance),
	)

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: message,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		log.Errorf("Error responding to betwin command: %v", err)
	}
}

// handleAutocomplete returns matching summoner names for the autocomplete dropdown
func (f *Feature) handleAutocomplete(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()

	guildID, err := strconv.ParseInt(i.GuildID, 10, 64)
	if err != nil {
		log.Errorf("Error parsing guild ID for autocomplete: %v", err)
		return
	}

	// Get the current input value from the focused option
	var focused string
	for _, opt := range i.ApplicationCommandData().Options {
		if opt.Name == "summoner" && opt.Focused {
			focused = strings.ToLower(opt.StringValue())
		}
	}

	// Query watched summoners for this guild (read-only, no UoW transaction needed)
	uow := f.uowFactory.CreateForGuild(guildID)
	if err := uow.Begin(ctx); err != nil {
		log.Errorf("Error beginning transaction for autocomplete: %v", err)
		return
	}
	defer uow.Rollback()

	// Use NewUserService pattern for read-only query
	watches, err := uow.SummonerWatchRepository().GetWatchesByGuild(ctx, guildID)
	if err != nil {
		log.Errorf("Error fetching watches for autocomplete: %v", err)
		return
	}

	if err := uow.Commit(); err != nil {
		log.Errorf("Error committing autocomplete transaction: %v", err)
		return
	}

	// Filter and build choices
	var choices []*discordgo.ApplicationCommandOptionChoice
	for _, watch := range watches {
		// Internal format for parsing (name#tag)
		internalName := watch.GetFullName()
		lowerName := strings.ToLower(internalName)
		// Display format for Discord dropdown (name #tag)
		displayName := fmt.Sprintf("%s #%s", watch.SummonerName, watch.TagLine)

		if focused == "" || strings.Contains(lowerName, focused) {
			choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
				Name:  displayName,
				Value: lowerName,
			})
		}

		// Discord allows max 25 autocomplete choices
		if len(choices) >= 25 {
			break
		}
	}

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{
			Choices: choices,
		},
	})
	if err != nil {
		log.Errorf("Error responding to autocomplete: %v", err)
	}
}
