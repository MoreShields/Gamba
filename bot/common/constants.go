package common

// Discord color constants
const (
	ColorPrimary = 0x5865F2 // Discord blurple
	ColorSuccess = 0x57F287 // Green
	ColorDanger  = 0xED4245 // Red
	ColorWarning = 0xFEE75C // Yellow
)

// Betting constants
const (
	MinBetAmount = 1
	MinOdds      = 0.01
	MaxOdds      = 0.99
)

// Wager constants
const (
	WagerVotingDuration = 24 * 60 * 60 // 24 hours in seconds
	MinWagerAmount      = 100
)

// UI constants
const (
	MaxButtonsPerRow = 5
	MaxActionRows    = 5
)