package common

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

// BotError represents a structured error with user-facing and internal messages
type BotError struct {
	UserMessage string      // Message shown to Discord user
	LogMessage  string      // Internal message for logging
	Ephemeral   bool        // Whether the error message should be ephemeral
	Err         error       // Underlying error
	Context     interface{} // Additional context for logging
}

// Error implements the error interface
func (e *BotError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.LogMessage, e.Err)
	}
	return e.LogMessage
}

// Unwrap returns the underlying error
func (e *BotError) Unwrap() error {
	return e.Err
}

// NewUserError creates an error for user-caused issues (validation, insufficient funds, etc)
func NewUserError(userMessage string, logMessage string) *BotError {
	return &BotError{
		UserMessage: userMessage,
		LogMessage:  logMessage,
		Ephemeral:   true,
	}
}

// NewSystemError creates an error for system issues (database, unexpected state, etc)
func NewSystemError(err error, logMessage string) *BotError {
	return &BotError{
		UserMessage: "❌ Something went wrong. Please try again later.",
		LogMessage:  logMessage,
		Ephemeral:   true,
		Err:         err,
	}
}

// RespondWithError sends an error message as an interaction response
func RespondWithError(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("❌ %s", message),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		log.Errorf("Error sending error response: %v", err)
	}
}

// FollowUpWithError sends an error message as a follow-up to a deferred interaction
func FollowUpWithError(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	_, err := s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
		Content: fmt.Sprintf("❌ %s", message),
		Flags:   discordgo.MessageFlagsEphemeral,
	})
	if err != nil {
		log.Errorf("Error sending follow-up error message: %v", err)
	}
}

// HandleError processes a BotError and responds appropriately
func HandleError(s *discordgo.Session, i *discordgo.InteractionCreate, err error, deferred bool) {
	if botErr, ok := err.(*BotError); ok {
		// Log the full error with context
		log.WithFields(log.Fields{
			"user_id":      i.Member.User.ID,
			"command":      i.ApplicationCommandData().Name,
			"error":        botErr.Error(),
			"user_message": botErr.UserMessage,
			"context":      botErr.Context,
		}).Error(botErr.LogMessage)

		// Send appropriate user message
		if deferred {
			FollowUpWithError(s, i, botErr.UserMessage)
		} else {
			RespondWithError(s, i, botErr.UserMessage)
		}
	} else {
		// Unexpected error - log full details but show generic message to user
		log.WithFields(log.Fields{
			"user_id": i.Member.User.ID,
			"command": i.ApplicationCommandData().Name,
			"error":   err.Error(),
		}).Error("Unexpected error in bot command")

		if deferred {
			FollowUpWithError(s, i, "Something went wrong. Please try again later.")
		} else {
			RespondWithError(s, i, "Something went wrong. Please try again later.")
		}
	}
}
