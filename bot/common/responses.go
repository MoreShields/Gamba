package common

import (
	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

// DeferResponse sends a deferred response to give more time for processing
func DeferResponse(s *discordgo.Session, i *discordgo.InteractionCreate, ephemeral bool) error {
	var flags discordgo.MessageFlags
	if ephemeral {
		flags = discordgo.MessageFlagsEphemeral
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: flags,
		},
	})
}

// RespondWithEmbed sends an embed as an interaction response
func RespondWithEmbed(s *discordgo.Session, i *discordgo.InteractionCreate, embed *discordgo.MessageEmbed, components []discordgo.MessageComponent, ephemeral bool) error {
	data := &discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{embed},
	}

	if ephemeral {
		data.Flags = discordgo.MessageFlagsEphemeral
	}

	if len(components) > 0 {
		data.Components = components
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: data,
	})
}

// FollowUpWithEmbed sends an embed as a follow-up message
func FollowUpWithEmbed(s *discordgo.Session, i *discordgo.InteractionCreate, embed *discordgo.MessageEmbed, components []discordgo.MessageComponent, ephemeral bool) (*discordgo.Message, error) {
	params := &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{embed},
	}

	if ephemeral {
		params.Flags = discordgo.MessageFlagsEphemeral
	}

	if len(components) > 0 {
		params.Components = components
	}

	return s.FollowupMessageCreate(i.Interaction, false, params)
}

// UpdateMessage updates an existing interaction response
func UpdateMessage(s *discordgo.Session, i *discordgo.InteractionCreate, embed *discordgo.MessageEmbed, components []discordgo.MessageComponent) error {
	edit := &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	}

	if components != nil {
		edit.Components = &components
	}

	_, err := s.InteractionResponseEdit(i.Interaction, edit)
	return err
}

// DisableComponents disables all components in a message
func DisableComponents(components []discordgo.MessageComponent) []discordgo.MessageComponent {
	disabled := make([]discordgo.MessageComponent, len(components))
	
	for i, component := range components {
		if actionRow, ok := component.(*discordgo.ActionsRow); ok {
			newRow := &discordgo.ActionsRow{
				Components: make([]discordgo.MessageComponent, len(actionRow.Components)),
			}
			
			for j, comp := range actionRow.Components {
				switch c := comp.(type) {
				case *discordgo.Button:
					newButton := *c
					newButton.Disabled = true
					newRow.Components[j] = &newButton
				case *discordgo.SelectMenu:
					newMenu := *c
					newMenu.Disabled = true
					newRow.Components[j] = &newMenu
				default:
					newRow.Components[j] = comp
				}
			}
			
			disabled[i] = newRow
		} else {
			disabled[i] = component
		}
	}
	
	return disabled
}

// RespondWithSuccess sends a success message
func RespondWithSuccess(s *discordgo.Session, i *discordgo.InteractionCreate, message string, ephemeral bool) error {
	data := &discordgo.InteractionResponseData{
		Content: "✅ " + message,
	}

	if ephemeral {
		data.Flags = discordgo.MessageFlagsEphemeral
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: data,
	})
}

// FollowUpWithSuccess sends a success message as a follow-up
func FollowUpWithSuccess(s *discordgo.Session, i *discordgo.InteractionCreate, message string, ephemeral bool) {
	params := &discordgo.WebhookParams{
		Content: "✅ " + message,
	}

	if ephemeral {
		params.Flags = discordgo.MessageFlagsEphemeral
	}

	_, err := s.FollowupMessageCreate(i.Interaction, false, params)
	if err != nil {
		log.Errorf("Error sending follow-up success message: %v", err)
	}
}