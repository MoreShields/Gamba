package dto

import (
	"fmt"
	"strings"

	"gambler/discord-client/domain/entities"
)

// GroupWagerDetailToHouseWagerPostDTO converts a GroupWagerDetail to a HouseWagerPostDTO
// This is used when a house wager needs to be displayed or updated in Discord
func GroupWagerDetailToHouseWagerPostDTO(detail *entities.GroupWagerDetail) HouseWagerPostDTO {
	// Parse the condition to extract title and description
	// Split on first newline - everything before is title, everything after is description
	parts := strings.SplitN(detail.Wager.Condition, "\n", 2)
	title := parts[0]
	description := ""
	if len(parts) > 1 {
		description = parts[1]
	}

	// Update URL for resolved wagers from League of Legends
	if detail.Wager.State == entities.GroupWagerStateResolved &&
		detail.Wager.ExternalRef != nil &&
		detail.Wager.ExternalRef.System == entities.SystemLeagueOfLegends {
		// Replace porofessor URL with leagueofgraphs URL for resolved wagers
		leagueOfGraphsURL := fmt.Sprintf("https://www.leagueofgraphs.com/match/NA/%s", detail.Wager.ExternalRef.ID)
		// Update the description to use the new URL
		description = fmt.Sprintf("[View Match Results](%s)", leagueOfGraphsURL)
	}

	dto := HouseWagerPostDTO{
		GuildID:         detail.Wager.GuildID,
		ChannelID:       detail.Wager.ChannelID,
		WagerID:         detail.Wager.ID,
		Title:           title,       // Title from first line
		Description:     description, // Description from remaining lines
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
func GroupWagerDetailToGroupWagerDTO(detail *entities.GroupWagerDetail) *entities.GroupWagerDetail {
	// For now, we can return the detail as-is since groupwagers.CreateGroupWagerEmbed
	// already accepts *entities.GroupWagerDetail directly
	// If a specific DTO is needed in the future, it can be implemented here
	return detail
}
