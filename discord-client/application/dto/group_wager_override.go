package dto

type OverrideGroupWagerResultRequest struct {
	GuildID           int64
	AdminDiscordID    int64
	GroupWagerID      int64
	WinningOptionText string
}

type OverrideGroupWagerWinnerDTO struct {
	DiscordID    int64
	PayoutAmount int64
}

type OverrideGroupWagerResultResponse struct {
	GroupWagerID          int64
	PreviousState         string
	PreviousWinningOption string
	CorrectedState        string
	WinningOption         string
	WinningOptionID       int64
	Winners               []OverrideGroupWagerWinnerDTO
	OriginalMessageID     int64
	OriginalChannelID     int64
}
