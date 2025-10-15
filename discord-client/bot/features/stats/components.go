package stats

import (
	"github.com/bwmarrin/discordgo"
)

// BuildScoreboardNavButtons creates navigation buttons for scoreboard pages
func BuildScoreboardNavButtons(currentPage string) []discordgo.MessageComponent {
	buttons := []discordgo.MessageComponent{}

	for _, page := range ScoreboardPages {
		var style discordgo.ButtonStyle
		disabled := false

		if page == currentPage {
			// Current page - use primary style and disable
			style = discordgo.PrimaryButton
			disabled = true
		} else {
			// Other pages - use secondary (grey) style
			style = discordgo.SecondaryButton
		}

		buttons = append(buttons, discordgo.Button{
			Label:    page,
			Style:    style,
			CustomID: "stats_page_" + page,
			Disabled: disabled,
		})
	}

	return []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: buttons,
		},
	}
}
