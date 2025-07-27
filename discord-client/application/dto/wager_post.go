package dto

import "time"

// HouseWagerPostDTO contains all the information needed to post a house wager to Discord
type HouseWagerPostDTO struct {
	// Guild and channel information
	GuildID   int64
	ChannelID int64 // If 0, use guild's primary channel

	// Wager information
	WagerID      int64
	Title        string
	Description  string
	State        string    // Wager state (active, pending_resolution, resolved, cancelled)
	Options      []WagerOptionDTO
	VotingEndsAt *time.Time // When the voting period ends

	// Participant information for real-time display
	Participants []ParticipantDTO
	TotalPot     int64
}

// WagerOptionDTO represents a single option in a house wager
type WagerOptionDTO struct {
	ID          int64
	Text        string
	Order       int16
	Multiplier  float64
	TotalAmount int64 // Total amount bet on this option
}

// ParticipantDTO represents a participant in a house wager
type ParticipantDTO struct {
	DiscordID int64
	OptionID  int64
	Amount    int64
}

// PostResult contains the result of posting a wager to Discord
type PostResult struct {
	MessageID int64
	ChannelID int64
}
