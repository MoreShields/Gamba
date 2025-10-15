package entities

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

// CalculateWinPercentage computes the win percentage
func (s *BetStatsDetail) CalculateWinPercentage() {
	if s.TotalBets > 0 {
		s.WinPercentage = (float64(s.TotalWins) / float64(s.TotalBets)) * 100
	} else {
		s.WinPercentage = 0
	}
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

// CalculateWinPercentage computes the win percentage
func (s *WagerStatsDetail) CalculateWinPercentage() {
	if s.TotalResolved > 0 {
		s.WinPercentage = (float64(s.TotalWon) / float64(s.TotalResolved)) * 100
	} else {
		s.WinPercentage = 0
	}
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
	TotalVolume      int64   // Total bits exchanged (sum of absolute balance changes)
	TotalDonations   int64   // Total amount donated (transfer_out transactions)
}

// GroupWagerPrediction represents a user's prediction for any group wager
type GroupWagerPrediction struct {
	DiscordID       int64
	GroupWagerID    int64
	OptionID        int64
	OptionText      string
	WinningOptionID int64
	Amount          int64
	WasCorrect      bool
	PayoutAmount    *int64           // Actual payout from database (nil for unresolved, 0 for losers)
	ExternalSystem  *ExternalSystem // nil for regular wagers
	ExternalID      *string
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

// GamblingLeaderboardEntry represents a user's position in the gambling leaderboard
type GamblingLeaderboardEntry struct {
	Rank           int     `json:"rank"`
	DiscordID      int64   `json:"discord_id"`
	TotalBets      int     `json:"total_bets"`
	TotalWins      int     `json:"total_wins"`
	WinPercentage  float64 `json:"win_percentage"`
	TotalWagered   int64   `json:"total_wagered"`
	NetProfit      int64   `json:"net_profit"`
}

// CalculateWinPercentage computes the win percentage
func (e *GamblingLeaderboardEntry) CalculateWinPercentage() {
	if e.TotalBets > 0 {
		e.WinPercentage = (float64(e.TotalWins) / float64(e.TotalBets)) * 100
	} else {
		e.WinPercentage = 0
	}
}

// QualifiesForLeaderboard checks if user meets minimum bet requirements
func (e *GamblingLeaderboardEntry) QualifiesForLeaderboard(minBets int) bool {
	return e.TotalBets >= minBets
}