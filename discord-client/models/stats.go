package models

// BetStats represents aggregated betting statistics
type BetStats struct {
	TotalBets    int
	TotalWins    int
	TotalLosses  int
	TotalWagered int64
	TotalWon     int64
	TotalLost    int64
	BiggestWin   int64
	BiggestLoss  int64
}

// WagerStats represents aggregated wager statistics
type WagerStats struct {
	TotalWagers    int
	TotalProposed  int
	TotalAccepted  int
	TotalDeclined  int
	TotalResolved  int
	TotalWon       int
	TotalLost      int
	TotalAmount    int64
	TotalWonAmount int64
	BiggestWin     int64
	BiggestLoss    int64
}

// UserStats represents combined statistics for a user
type UserStats struct {
	User             *User
	BetStats         *BetStatsDetail
	WagerStats       *WagerStatsDetail
	GroupWagerStats  *GroupWagerStats
	ReservedInWagers int64 // Amount currently locked in active wagers
}

// BetStatsDetail contains detailed betting statistics
type BetStatsDetail struct {
	TotalBets     int
	TotalWins     int
	TotalLosses   int
	WinPercentage float64
	TotalWagered  int64
	TotalWon      int64
	TotalLost     int64
	NetProfit     int64
	BiggestWin    int64
	BiggestLoss   int64
}

// WagerStatsDetail contains detailed wager statistics
type WagerStatsDetail struct {
	TotalWagers    int
	TotalProposed  int
	TotalAccepted  int
	TotalDeclined  int
	TotalResolved  int
	TotalWon       int
	TotalLost      int
	WinPercentage  float64
	TotalAmount    int64
	TotalWonAmount int64
	BiggestWin     int64
	BiggestLoss    int64
}

// GroupWagerStats represents aggregated group wager statistics
type GroupWagerStats struct {
	TotalGroupWagers int
	TotalProposed    int
	TotalWon         int
	TotalWonAmount   int64
}

// ScoreboardEntry represents a user's entry in the scoreboard
type ScoreboardEntry struct {
	Rank             int
	DiscordID        int64
	Username         string
	TotalBalance     int64
	AvailableBalance int64
	ActiveWagerCount int
	WagerWinRate     float64 // Percentage as 0-100
	BetWinRate       float64 // Percentage as 0-100
}

// GroupWagerPrediction represents a user's prediction for any group wager
type GroupWagerPrediction struct {
	DiscordID       int64           `db:"discord_id"`
	GroupWagerID    int64           `db:"group_wager_id"`
	OptionID        int64           `db:"option_id"`
	OptionText      string          `db:"option_text"`
	WinningOptionID int64           `db:"winning_option_id"`
	Amount          int64           `db:"amount"`
	WasCorrect      bool            `db:"was_correct"`
	ExternalSystem  *ExternalSystem `db:"external_system"` // nil for regular wagers
	ExternalID      *string         `db:"external_id"`
}

// WagerPredictionStats represents prediction statistics for a user
type WagerPredictionStats struct {
	DiscordID          int64   `json:"discord_id"`
	CorrectPredictions int     `json:"correct_predictions"`
	TotalPredictions   int     `json:"total_predictions"`
	AccuracyPercentage float64 `json:"accuracy_percentage"`
	TotalAmountWagered int64   `json:"total_amount_wagered"`
}

// CalculateAccuracy computes the accuracy percentage
func (s *WagerPredictionStats) CalculateAccuracy() {
	if s.TotalPredictions > 0 {
		s.AccuracyPercentage = (float64(s.CorrectPredictions) / float64(s.TotalPredictions)) * 100
	} else {
		s.AccuracyPercentage = 0
	}
}

// HasData checks if the stats contain any prediction data
func (s *WagerPredictionStats) HasData() bool {
	return s.TotalPredictions > 0
}

// LOLPredictionStats represents LOL-specific prediction statistics
type LOLPredictionStats struct {
	DiscordID          int64   `json:"discord_id"`
	CorrectPredictions int     `json:"correct_predictions"`
	TotalPredictions   int     `json:"total_predictions"`
	AccuracyPercentage float64 `json:"accuracy_percentage"`
	TotalAmountWagered int64   `json:"total_amount_wagered"`
}

// CalculateAccuracy computes the accuracy percentage
func (s *LOLPredictionStats) CalculateAccuracy() {
	if s.TotalPredictions > 0 {
		s.AccuracyPercentage = (float64(s.CorrectPredictions) / float64(s.TotalPredictions)) * 100
	} else {
		s.AccuracyPercentage = 0
	}
}

// HasData checks if the stats contain any prediction data
func (s *LOLPredictionStats) HasData() bool {
	return s.TotalPredictions > 0
}

// LOLLeaderboardEntry represents a user's position in the LoL leaderboard
type LOLLeaderboardEntry struct {
	Rank               int     `json:"rank"`
	DiscordID          int64   `json:"discord_id"`
	CorrectPredictions int     `json:"correct_predictions"`
	TotalPredictions   int     `json:"total_predictions"`
	AccuracyPercentage float64 `json:"accuracy_percentage"`
	TotalAmountWagered int64   `json:"total_amount_wagered"`
	ProfitLoss         int64   `json:"profit_loss"`
}

// CalculateAccuracy computes the accuracy percentage
func (e *LOLLeaderboardEntry) CalculateAccuracy() {
	if e.TotalPredictions > 0 {
		e.AccuracyPercentage = (float64(e.CorrectPredictions) / float64(e.TotalPredictions)) * 100
	} else {
		e.AccuracyPercentage = 0
	}
}

// QualifiesForLeaderboard checks if user meets minimum wager requirements
func (e *LOLLeaderboardEntry) QualifiesForLeaderboard(minWagers int) bool {
	return e.TotalPredictions >= minWagers
}
