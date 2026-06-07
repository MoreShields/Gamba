package dto

// OverrideGroupWagerResultRequest contains the application-level data needed to correct a closed group wager.
type OverrideGroupWagerResultRequest struct {
	GuildID           int64
	AdminDiscordID    int64
	GroupWagerID      int64
	WinningOptionText string
}

// OverrideGroupWagerWinnerDTO describes a participant who backed the corrected winning option.
type OverrideGroupWagerWinnerDTO struct {
	DiscordID    int64
	PayoutAmount int64
}

// OverrideGroupWagerResultResponse contains the result of a group wager override.
type OverrideGroupWagerResultResponse struct {
	GroupWagerID      int64
	PreviousState     string
	CorrectedState    string
	WinningOption     string
	WinningOptionID   int64
	Winners           []OverrideGroupWagerWinnerDTO
	OriginalMessageID int64
	OriginalChannelID int64
}
