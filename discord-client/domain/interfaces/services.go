package interfaces

import (
	"context"
	"time"

	"gambler/discord-client/domain/entities"
)

// UserService defines the interface for user operations
type UserService interface {
	// GetOrCreateUser retrieves an existing user or creates a new one with initial balance
	GetOrCreateUser(ctx context.Context, discordID int64, username string) (*entities.User, error)

	// GetCurrentHighRoller returns the user with the highest balance
	GetCurrentHighRoller(ctx context.Context) (*entities.User, error)

	// TransferBetweenUsers transfers amount from sender to recipient
	TransferBetweenUsers(ctx context.Context, fromDiscordID, toDiscordID int64, amount int64, fromUsername, toUsername string) error
}

// GamblingService defines the interface for gambling operations
type GamblingService interface {
	// PlaceBet places a bet for a user with the given win probability and amount
	PlaceBet(ctx context.Context, discordID int64, winProbability float64, betAmount int64) (*entities.BetResult, error)

	// GetDailyRiskAmount returns the total amount risked by a user since a given time
	GetDailyRiskAmount(ctx context.Context, discordID int64, since time.Time) (int64, error)

	// CheckDailyLimit checks if a bet amount would exceed the user's daily limit
	// Returns remaining amount and any error
	CheckDailyLimit(ctx context.Context, discordID int64, betAmount int64) (remaining int64, err error)
}

// WagerService defines the interface for wager operations
type WagerService interface {
	// ProposeWager creates a new wager proposal
	ProposeWager(ctx context.Context, proposerID, targetID int64, amount int64, condition string, messageID, channelID int64) (*entities.Wager, error)

	// RespondToWager handles accepting or declining a wager
	RespondToWager(ctx context.Context, wagerID int64, responderID int64, accept bool) (*entities.Wager, error)

	// CastVote records or updates a participant's vote on a wager
	CastVote(ctx context.Context, wagerID int64, voterID int64, voteForID int64) (*entities.WagerVote, *entities.VoteCount, error)

	// GetWagerByID retrieves a wager by ID
	GetWagerByID(ctx context.Context, wagerID int64) (*entities.Wager, error)

	// GetWagerByMessageID retrieves a wager by message ID
	GetWagerByMessageID(ctx context.Context, messageID int64) (*entities.Wager, error)

	// GetActiveWagersByUser returns active wagers for a user
	GetActiveWagersByUser(ctx context.Context, discordID int64) ([]*entities.Wager, error)

	// CancelWager cancels a proposed wager
	CancelWager(ctx context.Context, wagerID int64, cancellerID int64) error

	// UpdateMessageIDs updates the message and channel IDs for a wager
	UpdateMessageIDs(ctx context.Context, wagerID int64, messageID int64, channelID int64) error

	// BothParticipantsAgree checks if both participants have voted for the same winner
	// Returns the winner's Discord ID if they agree, 0 otherwise
	BothParticipantsAgree(wager *entities.Wager, voteCounts *entities.VoteCount) int64
}

// GroupWagerService defines the interface for group wager operations
type GroupWagerService interface {
	// CreateGroupWager creates a new group wager with options
	CreateGroupWager(ctx context.Context, creatorID *int64, condition string, options []string, votingPeriodMinutes int, messageID, channelID int64, wagerType entities.GroupWagerType, oddsMultipliers []float64) (*entities.GroupWagerDetail, error)

	// PlaceBet allows a user to place or update their bet on a group wager option
	PlaceBet(ctx context.Context, groupWagerID int64, userID int64, optionID int64, amount int64) (*entities.GroupWagerParticipant, error)

	// ResolveGroupWager resolves a group wager with the winning option
	ResolveGroupWager(ctx context.Context, groupWagerID int64, resolverID *int64, winningOptionID int64) (*entities.GroupWagerResult, error)

	// GetGroupWagerDetail retrieves full details of a group wager
	GetGroupWagerDetail(ctx context.Context, groupWagerID int64) (*entities.GroupWagerDetail, error)

	// GetGroupWagerByMessageID retrieves a group wager by message ID
	GetGroupWagerByMessageID(ctx context.Context, messageID int64) (*entities.GroupWagerDetail, error)

	// GetActiveGroupWagersByUser returns active group wagers where user is participating
	GetActiveGroupWagersByUser(ctx context.Context, discordID int64) ([]*entities.GroupWager, error)

	// IsResolver checks if a user can resolve group wagers
	IsResolver(discordID int64) bool

	// UpdateMessageIDs updates the message and channel IDs for a group wager
	UpdateMessageIDs(ctx context.Context, groupWagerID int64, messageID int64, channelID int64) error

	// TransitionExpiredWagers finds and transitions expired active wagers to pending_resolution
	TransitionExpiredWagers(ctx context.Context) error

	// CancelGroupWager cancels an active group wager
	CancelGroupWager(ctx context.Context, groupWagerID int64, cancellerID *int64) error
}

// GuildSettingsService defines the interface for guild settings operations
type GuildSettingsService interface {
	// GetOrCreateSettings retrieves guild settings or creates default ones if not found
	GetOrCreateSettings(ctx context.Context, guildID int64) (*entities.GuildSettings, error)

	// UpdatePrimaryChannel updates the primary channel for a guild
	UpdatePrimaryChannel(ctx context.Context, guildID int64, channelID *int64) error

	// UpdateLolChannel updates the LOL channel for a guild
	UpdateLolChannel(ctx context.Context, guildID int64, channelID *int64) error

	// UpdateTftChannel updates the TFT channel for a guild
	UpdateTftChannel(ctx context.Context, guildID int64, channelID *int64) error

	// UpdateWordleChannel updates the Wordle channel for a guild
	UpdateWordleChannel(ctx context.Context, guildID int64, channelID *int64) error

	// UpdateHighRollerRole updates the high roller role for a guild
	UpdateHighRollerRole(ctx context.Context, guildID int64, roleID *int64) error

	// GetHighRollerRoleID returns the high roller role ID for a guild
	GetHighRollerRoleID(ctx context.Context, guildID int64) (*int64, error)
}

// HighRollerService defines the interface for high roller operations
type HighRollerService interface {
	// GetCurrentHighRoller returns information about the current high roller
	GetCurrentHighRoller(ctx context.Context, guildID int64) (*HighRollerInfo, error)

	// PurchaseHighRollerRole processes a high roller role purchase
	PurchaseHighRollerRole(ctx context.Context, discordID, guildID, offerAmount int64) error
}

// HighRollerInfo contains information about the current high roller
type HighRollerInfo struct {
	CurrentHolder   *entities.User
	CurrentPrice    int64
	LastPurchasedAt *time.Time
}

// SummonerWatchService defines the interface for summoner watch operations
type SummonerWatchService interface {
	// AddWatch creates a new summoner watch for a guild
	AddWatch(ctx context.Context, guildID int64, summonerName, tagLine string) (*entities.SummonerWatchDetail, error)

	// RemoveWatch removes a summoner watch for a guild
	RemoveWatch(ctx context.Context, guildID int64, summonerName, tagLine string) error

	// ListWatches returns all summoner watches for a specific guild
	ListWatches(ctx context.Context, guildID int64) ([]*entities.SummonerWatchDetail, error)
}

// UserMetricsService consolidates user statistics and analytics operations
type UserMetricsService interface {
	// General statistics methods

	// GetScoreboard returns the top users with their statistics
	GetScoreboard(ctx context.Context, limit int) ([]*entities.ScoreboardEntry, int64, error)

	// GetUserStats returns detailed statistics for a specific user
	GetUserStats(ctx context.Context, discordID int64) (*entities.UserStats, error)

	// Prediction analytics methods

	// GetLOLPredictionStats calculates LOL-specific prediction stats for all users in a guild
	// Returns a map of Discord IDs to their LOL prediction performance
	GetLOLPredictionStats(ctx context.Context) (map[int64]*entities.LOLPredictionStats, error)

	// GetWagerPredictionStats calculates generic prediction stats for all users in a guild
	// Can optionally filter by external system (pass nil for all wagers)
	GetWagerPredictionStats(ctx context.Context, externalSystem *entities.ExternalSystem) (map[int64]*entities.WagerPredictionStats, error)

	// GetLOLLeaderboard returns LoL prediction leaderboard entries
	// Filters users with minimum wager count and calculates profit/loss
	GetLOLLeaderboard(ctx context.Context, minWagers int) ([]*entities.LOLLeaderboardEntry, int64, error)
}