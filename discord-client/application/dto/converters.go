package dto

import "gambler/discord-client/models"

// GroupWagerDetailToHouseWagerPostDTO converts a GroupWagerDetail to a HouseWagerPostDTO
// This is used when a house wager needs to be displayed or updated in Discord
func GroupWagerDetailToHouseWagerPostDTO(detail *models.GroupWagerDetail) HouseWagerPostDTO {
	dto := HouseWagerPostDTO{
		GuildID:         detail.Wager.GuildID,
		ChannelID:       detail.Wager.ChannelID,
		WagerID:         detail.Wager.ID,
		Title:           detail.Wager.Condition, // Use condition as title (like group wagers)
		Description:     "",                     // Description will be set separately if needed
		State:           string(detail.Wager.State),
		Options:         make([]WagerOptionDTO, len(detail.Options)),
		VotingEndsAt:    detail.Wager.VotingEndsAt,
		WinningOptionID: detail.Wager.WinningOptionID,
		Participants:    make([]ParticipantDTO, len(detail.Participants)),
		TotalPot:        detail.Wager.TotalPot,
	}

	// Convert options
	for i, opt := range detail.Options {
		dto.Options[i] = WagerOptionDTO{
			ID:          opt.ID,
			Text:        opt.OptionText,
			Order:       opt.OptionOrder,
			Multiplier:  opt.OddsMultiplier,
			TotalAmount: opt.TotalAmount,
		}
	}

	// Convert participants
	for i, participant := range detail.Participants {
		dto.Participants[i] = ParticipantDTO{
			DiscordID: participant.DiscordID,
			OptionID:  participant.OptionID,
			Amount:    participant.Amount,
		}
	}

	return dto
}

// GroupWagerDetailToGroupWagerDTO converts a GroupWagerDetail to a GroupWagerDTO
// This would be used for regular group wagers if a similar DTO pattern is needed
func GroupWagerDetailToGroupWagerDTO(detail *models.GroupWagerDetail) *models.GroupWagerDetail {
	// For now, we can return the detail as-is since groupwagers.CreateGroupWagerEmbed
	// already accepts *models.GroupWagerDetail directly
	// If a specific DTO is needed in the future, it can be implemented here
	return detail
}