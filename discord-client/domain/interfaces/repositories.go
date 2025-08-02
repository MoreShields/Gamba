package interfaces

import (
	"context"
	"time"

	"gambler/discord-client/domain/entities"
	"gambler/discord-client/domain/events"
)

// UserRepository defines the interface for user data access
type UserRepository interface {
	// GetByDiscordID retrieves a user by their Discord ID
	GetByDiscordID(ctx context.Context, discordID int64) (*entities.User, error)

	// Create creates a new user with the initial balance
	Create(ctx context.Context, discordID int64, username string, initialBalance int64) (*entities.User, error)

	// UpdateBalance updates a user's balance atomically
	UpdateBalance(ctx context.Context, discordID int64, newBalance int64) error

	// GetUsersWithPositiveBalance returns all users with balance > 0
	GetUsersWithPositiveBalance(ctx context.Context) ([]*entities.User, error)

	// GetAll returns all users
	GetAll(ctx context.Context) ([]*entities.User, error)
}

// BalanceHistoryRepository defines the interface for balance history tracking
type BalanceHistoryRepository interface {
	// Record creates a new balance history entry
	Record(ctx context.Context, history *entities.BalanceHistory) error

	// GetByUser returns balance history for a specific user
	GetByUser(ctx context.Context, discordID int64, limit int) ([]*entities.BalanceHistory, error)

	// GetByDateRange returns balance history within a date range
	GetByDateRange(ctx context.Context, discordID int64, from, to time.Time) ([]*entities.BalanceHistory, error)

	// GetTotalVolumeByUser returns the total volume (sum of absolute balance changes) for a user
	GetTotalVolumeByUser(ctx context.Context, discordID int64) (int64, error)
}

// BetRepository defines the interface for bet data access
type BetRepository interface {
	// Create creates a new bet record
	Create(ctx context.Context, bet *entities.Bet) error

	// GetByID retrieves a bet by its ID
	GetByID(ctx context.Context, id int64) (*entities.Bet, error)

	// GetByUser returns bets for a specific user
	GetByUser(ctx context.Context, discordID int64, limit int) ([]*entities.Bet, error)

	// GetStats returns betting statistics for a user
	GetStats(ctx context.Context, discordID int64) (*entities.BetStats, error)

	// GetByUserSince returns all bets for a user since a specific time
	GetByUserSince(ctx context.Context, discordID int64, since time.Time) ([]*entities.Bet, error)
}

// WagerRepository defines the interface for wager data access
type WagerRepository interface {
	// Create creates a new wager
	Create(ctx context.Context, wager *entities.Wager) error

	// GetByID retrieves a wager by its ID
	GetByID(ctx context.Context, id int64) (*entities.Wager, error)

	// GetByMessageID retrieves a wager by its Discord message ID
	GetByMessageID(ctx context.Context, messageID int64) (*entities.Wager, error)

	// Update updates a wager's state and related fields
	Update(ctx context.Context, wager *entities.Wager) error

	// GetActiveByUser returns all active wagers for a user
	GetActiveByUser(ctx context.Context, discordID int64) ([]*entities.Wager, error)

	// GetAllByUser returns all wagers for a user with limit
	GetAllByUser(ctx context.Context, discordID int64, limit int) ([]*entities.Wager, error)

	// GetStats returns wager statistics for a user
	GetStats(ctx context.Context, discordID int64) (*entities.WagerStats, error)
}

// WagerVoteRepository defines the interface for wager vote data access
type WagerVoteRepository interface {
	// CreateOrUpdate creates a new vote or updates an existing one
	CreateOrUpdate(ctx context.Context, vote *entities.WagerVote) error

	// GetByWager returns all votes for a specific wager
	GetByWager(ctx context.Context, wagerID int64) ([]*entities.WagerVote, error)

	// GetVoteCounts returns the vote counts for a wager
	GetVoteCounts(ctx context.Context, wagerID int64) (*entities.VoteCount, error)

	// GetByVoter returns a vote by a specific voter for a wager
	GetByVoter(ctx context.Context, wagerID int64, voterDiscordID int64) (*entities.WagerVote, error)

	// DeleteByWager deletes all votes for a wager
	DeleteByWager(ctx context.Context, wagerID int64) error
}

// WordleCompletionRepository defines the interface for wordle completion data access
type WordleCompletionRepository interface {
	// Create creates a new wordle completion record
	Create(ctx context.Context, completion *entities.WordleCompletion) error

	// GetByUserToday retrieves today's completion for a specific user
	GetByUserToday(ctx context.Context, discordID, guildID int64) (*entities.WordleCompletion, error)

	// GetRecentCompletions returns recent completions for streak calculation
	GetRecentCompletions(ctx context.Context, discordID, guildID int64, limit int) ([]*entities.WordleCompletion, error)

	// GetTodaysCompletions retrieves all completions for today in the repository's guild
	GetTodaysCompletions(ctx context.Context) ([]*entities.WordleCompletion, error)
}

// GroupWagerRepository defines the interface for all group wager related data access
type GroupWagerRepository interface {
	// Core wager operations
	CreateWithOptions(ctx context.Context, wager *entities.GroupWager, options []*entities.GroupWagerOption) error
	GetByID(ctx context.Context, id int64) (*entities.GroupWager, error)
	GetByMessageID(ctx context.Context, messageID int64) (*entities.GroupWager, error)
	GetByExternalReference(ctx context.Context, ref entities.ExternalReference) (*entities.GroupWager, error)
	Update(ctx context.Context, wager *entities.GroupWager) error
	GetActiveByUser(ctx context.Context, discordID int64) ([]*entities.GroupWager, error)
	GetAll(ctx context.Context, state *entities.GroupWagerState) ([]*entities.GroupWager, error)

	// Detail operations (returns full wager with options and participants)
	GetDetailByID(ctx context.Context, id int64) (*entities.GroupWagerDetail, error)
	GetDetailByMessageID(ctx context.Context, messageID int64) (*entities.GroupWagerDetail, error)

	// Participant operations
	SaveParticipant(ctx context.Context, participant *entities.GroupWagerParticipant) error
	GetParticipant(ctx context.Context, groupWagerID int64, discordID int64) (*entities.GroupWagerParticipant, error)
	GetActiveParticipationsByUser(ctx context.Context, discordID int64) ([]*entities.GroupWagerParticipant, error)
	UpdateParticipantPayouts(ctx context.Context, participants []*entities.GroupWagerParticipant) error

	// Option operations
	UpdateOptionTotal(ctx context.Context, optionID int64, totalAmount int64) error
	UpdateOptionOdds(ctx context.Context, optionID int64, oddsMultiplier float64) error
	UpdateAllOptionOdds(ctx context.Context, groupWagerID int64, oddsMultipliers map[int64]float64) error

	// Stats operations
	GetStats(ctx context.Context, discordID int64) (*entities.GroupWagerStats, error)

	// Analytics operations
	GetGroupWagerPredictions(ctx context.Context, externalSystem *entities.ExternalSystem) ([]*entities.GroupWagerPrediction, error)

	// Expiration operations
	GetExpiredActiveWagers(ctx context.Context) ([]*entities.GroupWager, error)
	GetWagersPendingResolution(ctx context.Context) ([]*entities.GroupWager, error)
	GetGuildsWithActiveWagers(ctx context.Context) ([]int64, error)
}

// GuildSettingsRepository defines the interface for guild settings data access
type GuildSettingsRepository interface {
	// GetOrCreateGuildSettings retrieves guild settings or creates default ones if not found
	GetOrCreateGuildSettings(ctx context.Context, guildID int64) (*entities.GuildSettings, error)

	// UpdateGuildSettings updates guild settings
	UpdateGuildSettings(ctx context.Context, settings *entities.GuildSettings) error
}

// SummonerWatchRepository defines the interface for summoner watch data access
type SummonerWatchRepository interface {
	// CreateWatch creates a new summoner watch for a guild
	// Handles upsert of summoner and creation of watch relationship
	CreateWatch(ctx context.Context, guildID int64, summonerName, tagLine string) (*entities.SummonerWatchDetail, error)

	// GetWatchesByGuild returns all summoner watches for a specific guild
	GetWatchesByGuild(ctx context.Context, guildID int64) ([]*entities.SummonerWatchDetail, error)

	// GetGuildsWatchingSummoner returns all guild-summoner watch relationships for a specific summoner
	GetGuildsWatchingSummoner(ctx context.Context, summonerName, tagLine string) ([]*entities.GuildSummonerWatch, error)

	// DeleteWatch removes a summoner watch for a guild
	DeleteWatch(ctx context.Context, guildID int64, summonerName, tagLine string) error

	// GetWatch retrieves a specific summoner watch for a guild
	GetWatch(ctx context.Context, guildID int64, summonerName, tagLine string) (*entities.SummonerWatchDetail, error)
}

// EventPublisher defines the interface for publishing events
type EventPublisher interface {
	Publish(event events.Event) error
}