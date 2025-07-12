package service

import (
	"context"
	"time"

	"gambler/events"
	"gambler/models"
)

// UserRepository defines the interface for user data access
type UserRepository interface {
	// GetByDiscordID retrieves a user by their Discord ID
	GetByDiscordID(ctx context.Context, discordID int64) (*models.User, error)

	// Create creates a new user with the initial balance
	Create(ctx context.Context, discordID int64, username string, initialBalance int64) (*models.User, error)

	// UpdateBalance updates a user's balance atomically
	UpdateBalance(ctx context.Context, discordID int64, newBalance int64) error

	// AddBalance adds to a user's balance atomically
	AddBalance(ctx context.Context, discordID int64, amount int64) error

	// DeductBalance deducts from a user's balance atomically, failing if insufficient funds
	DeductBalance(ctx context.Context, discordID int64, amount int64) error

	// GetUsersWithPositiveBalance returns all users with balance > 0
	GetUsersWithPositiveBalance(ctx context.Context) ([]*models.User, error)

	// GetAll returns all users
	GetAll(ctx context.Context) ([]*models.User, error)
}

// BalanceHistoryRepository defines the interface for balance history tracking
type BalanceHistoryRepository interface {
	// Record creates a new balance history entry
	Record(ctx context.Context, history *models.BalanceHistory) error

	// GetByUser returns balance history for a specific user
	GetByUser(ctx context.Context, discordID int64, limit int) ([]*models.BalanceHistory, error)

	// GetByDateRange returns balance history within a date range
	GetByDateRange(ctx context.Context, discordID int64, from, to time.Time) ([]*models.BalanceHistory, error)
}


// BetRepository defines the interface for bet data access
type BetRepository interface {
	// Create creates a new bet record
	Create(ctx context.Context, bet *models.Bet) error

	// GetByID retrieves a bet by its ID
	GetByID(ctx context.Context, id int64) (*models.Bet, error)

	// GetByUser returns bets for a specific user
	GetByUser(ctx context.Context, discordID int64, limit int) ([]*models.Bet, error)

	// GetStats returns betting statistics for a user
	GetStats(ctx context.Context, discordID int64) (*models.BetStats, error)

	// GetByUserSince returns all bets for a user since a specific time
	GetByUserSince(ctx context.Context, discordID int64, since time.Time) ([]*models.Bet, error)
}

// UserService defines the interface for user operations
type UserService interface {
	// GetOrCreateUser retrieves an existing user or creates a new one with initial balance
	GetOrCreateUser(ctx context.Context, discordID int64, username string) (*models.User, error)

	// GetCurrentHighRoller returns the user with the highest balance
	GetCurrentHighRoller(ctx context.Context) (*models.User, error)

	// TransferBetweenUsers transfers amount from sender to recipient
	TransferBetweenUsers(ctx context.Context, fromDiscordID, toDiscordID int64, amount int64, fromUsername, toUsername string) error
}

// GamblingService defines the interface for gambling operations
type GamblingService interface {
	// PlaceBet places a bet for a user with the given win probability and amount
	PlaceBet(ctx context.Context, discordID int64, winProbability float64, betAmount int64) (*models.BetResult, error)

	// GetDailyRiskAmount returns the total amount risked by a user since a given time
	GetDailyRiskAmount(ctx context.Context, discordID int64, since time.Time) (int64, error)

	// CheckDailyLimit checks if a bet amount would exceed the user's daily limit
	// Returns remaining amount and any error
	CheckDailyLimit(ctx context.Context, discordID int64, betAmount int64) (remaining int64, err error)
}


// WagerRepository defines the interface for wager data access
type WagerRepository interface {
	// Create creates a new wager
	Create(ctx context.Context, wager *models.Wager) error

	// GetByID retrieves a wager by its ID
	GetByID(ctx context.Context, id int64) (*models.Wager, error)

	// GetByMessageID retrieves a wager by its Discord message ID
	GetByMessageID(ctx context.Context, messageID int64) (*models.Wager, error)

	// Update updates a wager's state and related fields
	Update(ctx context.Context, wager *models.Wager) error

	// GetActiveByUser returns all active wagers for a user
	GetActiveByUser(ctx context.Context, discordID int64) ([]*models.Wager, error)

	// GetAllByUser returns all wagers for a user with limit
	GetAllByUser(ctx context.Context, discordID int64, limit int) ([]*models.Wager, error)

	// GetStats returns wager statistics for a user
	GetStats(ctx context.Context, discordID int64) (*models.WagerStats, error)
}

// WagerVoteRepository defines the interface for wager vote data access
type WagerVoteRepository interface {
	// CreateOrUpdate creates a new vote or updates an existing one
	CreateOrUpdate(ctx context.Context, vote *models.WagerVote) error

	// GetByWager returns all votes for a specific wager
	GetByWager(ctx context.Context, wagerID int64) ([]*models.WagerVote, error)

	// GetVoteCounts returns the vote counts for a wager
	GetVoteCounts(ctx context.Context, wagerID int64) (*models.VoteCount, error)

	// GetByVoter returns a vote by a specific voter for a wager
	GetByVoter(ctx context.Context, wagerID int64, voterDiscordID int64) (*models.WagerVote, error)

	// DeleteByWager deletes all votes for a wager
	DeleteByWager(ctx context.Context, wagerID int64) error
}

// WagerService defines the interface for wager operations
type WagerService interface {
	// ProposeWager creates a new wager proposal
	ProposeWager(ctx context.Context, proposerID, targetID int64, amount int64, condition string, messageID, channelID int64) (*models.Wager, error)

	// RespondToWager handles accepting or declining a wager
	RespondToWager(ctx context.Context, wagerID int64, responderID int64, accept bool) (*models.Wager, error)

	// CastVote records or updates a vote on a wager
	CastVote(ctx context.Context, wagerID int64, voterID int64, voteForID int64) (*models.WagerVote, *models.VoteCount, error)

	// GetWagerByID retrieves a wager by ID
	GetWagerByID(ctx context.Context, wagerID int64) (*models.Wager, error)

	// GetWagerByMessageID retrieves a wager by message ID
	GetWagerByMessageID(ctx context.Context, messageID int64) (*models.Wager, error)

	// GetActiveWagersByUser returns active wagers for a user
	GetActiveWagersByUser(ctx context.Context, discordID int64) ([]*models.Wager, error)

	// CancelWager cancels a proposed wager
	CancelWager(ctx context.Context, wagerID int64, cancellerID int64) error
}

// EventPublisher defines the interface for publishing events
type EventPublisher interface {
	Publish(event events.Event)
}

// StatsService defines the interface for statistics operations
type StatsService interface {
	// GetScoreboard returns the top users with their statistics
	GetScoreboard(ctx context.Context, limit int) ([]*models.ScoreboardEntry, error)

	// GetUserStats returns detailed statistics for a specific user
	GetUserStats(ctx context.Context, discordID int64) (*models.UserStats, error)
}

// GroupWagerRepository defines the interface for all group wager related data access
type GroupWagerRepository interface {
	// Core wager operations
	CreateWithOptions(ctx context.Context, wager *models.GroupWager, options []*models.GroupWagerOption) error
	GetByID(ctx context.Context, id int64) (*models.GroupWager, error)
	GetByMessageID(ctx context.Context, messageID int64) (*models.GroupWager, error)
	Update(ctx context.Context, wager *models.GroupWager) error
	GetActiveByUser(ctx context.Context, discordID int64) ([]*models.GroupWager, error)
	GetAll(ctx context.Context, state *models.GroupWagerState) ([]*models.GroupWager, error)

	// Detail operations (returns full wager with options and participants)
	GetDetailByID(ctx context.Context, id int64) (*models.GroupWagerDetail, error)
	GetDetailByMessageID(ctx context.Context, messageID int64) (*models.GroupWagerDetail, error)

	// Participant operations
	SaveParticipant(ctx context.Context, participant *models.GroupWagerParticipant) error
	GetParticipant(ctx context.Context, groupWagerID int64, discordID int64) (*models.GroupWagerParticipant, error)
	GetActiveParticipationsByUser(ctx context.Context, discordID int64) ([]*models.GroupWagerParticipant, error)
	UpdateParticipantPayouts(ctx context.Context, participants []*models.GroupWagerParticipant) error

	// Option operations
	UpdateOptionTotal(ctx context.Context, optionID int64, totalAmount int64) error
	
	// Stats operations
	GetStats(ctx context.Context, discordID int64) (*models.GroupWagerStats, error)
	
	// Expiration operations
	GetExpiredActiveWagers(ctx context.Context) ([]*models.GroupWager, error)
	GetWagersPendingResolution(ctx context.Context) ([]*models.GroupWager, error)
	GetGuildsWithActiveWagers(ctx context.Context) ([]int64, error)
}

// GroupWagerService defines the interface for group wager operations
type GroupWagerService interface {
	// CreateGroupWager creates a new group wager with options
	CreateGroupWager(ctx context.Context, creatorID int64, condition string, options []string, votingPeriodMinutes int, messageID, channelID int64) (*models.GroupWagerDetail, error)

	// PlaceBet allows a user to place or update their bet on a group wager option
	PlaceBet(ctx context.Context, groupWagerID int64, userID int64, optionID int64, amount int64) (*models.GroupWagerParticipant, error)

	// ResolveGroupWager resolves a group wager with the winning option
	ResolveGroupWager(ctx context.Context, groupWagerID int64, resolverID int64, winningOptionID int64) (*models.GroupWagerResult, error)

	// GetGroupWagerDetail retrieves full details of a group wager
	GetGroupWagerDetail(ctx context.Context, groupWagerID int64) (*models.GroupWagerDetail, error)

	// GetGroupWagerByMessageID retrieves a group wager by message ID
	GetGroupWagerByMessageID(ctx context.Context, messageID int64) (*models.GroupWagerDetail, error)

	// GetActiveGroupWagersByUser returns active group wagers where user is participating
	GetActiveGroupWagersByUser(ctx context.Context, discordID int64) ([]*models.GroupWager, error)

	// IsResolver checks if a user can resolve group wagers
	IsResolver(discordID int64) bool

	// UpdateMessageIDs updates the message and channel IDs for a group wager
	UpdateMessageIDs(ctx context.Context, groupWagerID int64, messageID int64, channelID int64) error
	
	// TransitionExpiredWagers finds and transitions expired active wagers to pending_resolution
	TransitionExpiredWagers(ctx context.Context) error
}

// GuildSettingsRepository defines the interface for guild settings data access
type GuildSettingsRepository interface {
	// GetOrCreateGuildSettings retrieves guild settings or creates default ones if not found
	GetOrCreateGuildSettings(ctx context.Context, guildID int64) (*models.GuildSettings, error)

	// UpdateGuildSettings updates guild settings
	UpdateGuildSettings(ctx context.Context, settings *models.GuildSettings) error
}

// GuildSettingsService defines the interface for guild settings operations
type GuildSettingsService interface {
	// GetOrCreateSettings retrieves guild settings or creates default ones if not found
	GetOrCreateSettings(ctx context.Context, guildID int64) (*models.GuildSettings, error)

	// UpdatePrimaryChannel updates the primary channel for a guild
	UpdatePrimaryChannel(ctx context.Context, guildID int64, channelID int64) error

	// UpdateHighRollerRole updates the high roller role for a guild
	UpdateHighRollerRole(ctx context.Context, guildID int64, roleID *int64) error
}

// UnitOfWork defines the interface for transactional repository operations
type UnitOfWork interface {
	// Begin starts a new transaction
	Begin(ctx context.Context) error

	// Commit commits the transaction
	Commit() error

	// Rollback rolls back the transaction
	Rollback() error

	// Repository getters
	UserRepository() UserRepository
	BalanceHistoryRepository() BalanceHistoryRepository
	BetRepository() BetRepository
	WagerRepository() WagerRepository
	WagerVoteRepository() WagerVoteRepository
	GroupWagerRepository() GroupWagerRepository
	GuildSettingsRepository() GuildSettingsRepository
	EventBus() EventPublisher
}

// UnitOfWorkFactory defines the interface for creating UnitOfWork instances
type UnitOfWorkFactory interface {
	// CreateForGuild creates a new UnitOfWork instance scoped to a specific guild
	CreateForGuild(guildID int64) UnitOfWork
}
