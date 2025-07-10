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
	ReservedInWagers int64 // Amount currently locked in active wagers
}

// BetStatsDetail contains detailed betting statistics
type BetStatsDetail struct {
	TotalBets      int
	TotalWins      int
	TotalLosses    int
	WinPercentage  float64
	TotalWagered   int64
	TotalWon       int64
	TotalLost      int64
	NetProfit      int64
	BiggestWin     int64
	BiggestLoss    int64
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

// ScoreboardEntry represents a user's entry in the scoreboard
type ScoreboardEntry struct {
	Rank              int
	DiscordID         int64
	Username          string
	TotalBalance      int64
	AvailableBalance  int64
	ActiveWagerCount  int
	WagerWinRate      float64 // Percentage as 0-100
	BetWinRate        float64 // Percentage as 0-100
}