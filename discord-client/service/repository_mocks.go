package service

import (
	"context"
	"time"

	"gambler/discord-client/events"
	"gambler/discord-client/models"

	"github.com/stretchr/testify/mock"
)

// MockUserRepository is a mock implementation of UserRepository
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) GetByDiscordID(ctx context.Context, discordID int64) (*models.User, error) {
	args := m.Called(ctx, discordID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) Create(ctx context.Context, discordID int64, username string, initialBalance int64) (*models.User, error) {
	args := m.Called(ctx, discordID, username, initialBalance)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) UpdateBalance(ctx context.Context, discordID int64, newBalance int64) error {
	args := m.Called(ctx, discordID, newBalance)
	return args.Error(0)
}

func (m *MockUserRepository) GetUsersWithPositiveBalance(ctx context.Context) ([]*models.User, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.User), args.Error(1)
}

func (m *MockUserRepository) GetAll(ctx context.Context) ([]*models.User, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.User), args.Error(1)
}

// MockBalanceHistoryRepository is a mock implementation of BalanceHistoryRepository
type MockBalanceHistoryRepository struct {
	mock.Mock
}

func (m *MockBalanceHistoryRepository) Record(ctx context.Context, history *models.BalanceHistory) error {
	args := m.Called(ctx, history)
	return args.Error(0)
}

func (m *MockBalanceHistoryRepository) GetByUser(ctx context.Context, discordID int64, limit int) ([]*models.BalanceHistory, error) {
	args := m.Called(ctx, discordID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.BalanceHistory), args.Error(1)
}

func (m *MockBalanceHistoryRepository) GetByDateRange(ctx context.Context, discordID int64, from, to time.Time) ([]*models.BalanceHistory, error) {
	args := m.Called(ctx, discordID, from, to)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.BalanceHistory), args.Error(1)
}

// MockBetRepository is a mock implementation of BetRepository
type MockBetRepository struct {
	mock.Mock
}

func (m *MockBetRepository) Create(ctx context.Context, bet *models.Bet) error {
	args := m.Called(ctx, bet)
	return args.Error(0)
}

func (m *MockBetRepository) GetByID(ctx context.Context, id int64) (*models.Bet, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Bet), args.Error(1)
}

func (m *MockBetRepository) GetByUser(ctx context.Context, discordID int64, limit int) ([]*models.Bet, error) {
	args := m.Called(ctx, discordID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Bet), args.Error(1)
}

func (m *MockBetRepository) GetStats(ctx context.Context, discordID int64) (*models.BetStats, error) {
	args := m.Called(ctx, discordID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BetStats), args.Error(1)
}

func (m *MockBetRepository) GetByUserSince(ctx context.Context, discordID int64, since time.Time) ([]*models.Bet, error) {
	args := m.Called(ctx, discordID, since)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Bet), args.Error(1)
}

// MockGroupWagerRepository is a mock implementation of GroupWagerRepository
type MockGroupWagerRepository struct {
	mock.Mock
}

func (m *MockGroupWagerRepository) Create(ctx context.Context, wager *models.GroupWager) error {
	args := m.Called(ctx, wager)
	return args.Error(0)
}

func (m *MockGroupWagerRepository) GetByID(ctx context.Context, id int64) (*models.GroupWager, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.GroupWager), args.Error(1)
}

func (m *MockGroupWagerRepository) GetByMessageID(ctx context.Context, messageID int64) (*models.GroupWager, error) {
	args := m.Called(ctx, messageID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.GroupWager), args.Error(1)
}

func (m *MockGroupWagerRepository) GetByExternalReference(ctx context.Context, ref models.ExternalReference) (*models.GroupWager, error) {
	args := m.Called(ctx, ref)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.GroupWager), args.Error(1)
}

func (m *MockGroupWagerRepository) Update(ctx context.Context, wager *models.GroupWager) error {
	args := m.Called(ctx, wager)
	return args.Error(0)
}

func (m *MockGroupWagerRepository) GetActiveByUser(ctx context.Context, discordID int64) ([]*models.GroupWager, error) {
	args := m.Called(ctx, discordID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.GroupWager), args.Error(1)
}

func (m *MockGroupWagerRepository) GetAll(ctx context.Context, state *models.GroupWagerState) ([]*models.GroupWager, error) {
	args := m.Called(ctx, state)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.GroupWager), args.Error(1)
}

func (m *MockGroupWagerRepository) CreateWithOptions(ctx context.Context, wager *models.GroupWager, options []*models.GroupWagerOption) error {
	args := m.Called(ctx, wager, options)
	return args.Error(0)
}

func (m *MockGroupWagerRepository) GetDetailByID(ctx context.Context, id int64) (*models.GroupWagerDetail, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.GroupWagerDetail), args.Error(1)
}

func (m *MockGroupWagerRepository) GetDetailByMessageID(ctx context.Context, messageID int64) (*models.GroupWagerDetail, error) {
	args := m.Called(ctx, messageID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.GroupWagerDetail), args.Error(1)
}

func (m *MockGroupWagerRepository) SaveParticipant(ctx context.Context, participant *models.GroupWagerParticipant) error {
	args := m.Called(ctx, participant)
	return args.Error(0)
}

func (m *MockGroupWagerRepository) GetParticipant(ctx context.Context, groupWagerID int64, discordID int64) (*models.GroupWagerParticipant, error) {
	args := m.Called(ctx, groupWagerID, discordID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.GroupWagerParticipant), args.Error(1)
}

func (m *MockGroupWagerRepository) GetActiveParticipationsByUser(ctx context.Context, discordID int64) ([]*models.GroupWagerParticipant, error) {
	args := m.Called(ctx, discordID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.GroupWagerParticipant), args.Error(1)
}

func (m *MockGroupWagerRepository) UpdateParticipantPayouts(ctx context.Context, participants []*models.GroupWagerParticipant) error {
	args := m.Called(ctx, participants)
	return args.Error(0)
}

func (m *MockGroupWagerRepository) UpdateOptionTotal(ctx context.Context, optionID int64, totalAmount int64) error {
	args := m.Called(ctx, optionID, totalAmount)
	return args.Error(0)
}

func (m *MockGroupWagerRepository) UpdateOptionOdds(ctx context.Context, optionID int64, oddsMultiplier float64) error {
	args := m.Called(ctx, optionID, oddsMultiplier)
	return args.Error(0)
}

func (m *MockGroupWagerRepository) UpdateAllOptionOdds(ctx context.Context, groupWagerID int64, oddsMultipliers map[int64]float64) error {
	args := m.Called(ctx, groupWagerID, oddsMultipliers)
	return args.Error(0)
}

func (m *MockGroupWagerRepository) GetStats(ctx context.Context, discordID int64) (*models.GroupWagerStats, error) {
	args := m.Called(ctx, discordID)
	return args.Get(0).(*models.GroupWagerStats), args.Error(1)
}

func (m *MockGroupWagerRepository) GetExpiredActiveWagers(ctx context.Context) ([]*models.GroupWager, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*models.GroupWager), args.Error(1)
}

func (m *MockGroupWagerRepository) GetWagersPendingResolution(ctx context.Context) ([]*models.GroupWager, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*models.GroupWager), args.Error(1)
}

func (m *MockGroupWagerRepository) GetGuildsWithActiveWagers(ctx context.Context) ([]int64, error) {
	args := m.Called(ctx)
	return args.Get(0).([]int64), args.Error(1)
}

// MockWagerRepository is a mock implementation of WagerRepository for testing
type MockWagerRepository struct {
	mock.Mock
}

func (m *MockWagerRepository) GetByID(ctx context.Context, wagerID int64) (*models.Wager, error) {
	args := m.Called(ctx, wagerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Wager), args.Error(1)
}

func (m *MockWagerRepository) Create(ctx context.Context, wager *models.Wager) error {
	args := m.Called(ctx, wager)
	return args.Error(0)
}

func (m *MockWagerRepository) UpdateState(ctx context.Context, wagerID int64, newState models.WagerState) error {
	args := m.Called(ctx, wagerID, newState)
	return args.Error(0)
}

func (m *MockWagerRepository) UpdateWinner(ctx context.Context, wagerID int64, winnerID int64) error {
	args := m.Called(ctx, wagerID, winnerID)
	return args.Error(0)
}

// MockWagerVoteRepository is a mock implementation of WagerVoteRepository for testing
type MockWagerVoteRepository struct {
	mock.Mock
}

func (m *MockWagerVoteRepository) Create(ctx context.Context, vote *models.WagerVote) error {
	args := m.Called(ctx, vote)
	return args.Error(0)
}

func (m *MockWagerVoteRepository) GetByWagerAndVoter(ctx context.Context, wagerID int64, voterID int64) (*models.WagerVote, error) {
	args := m.Called(ctx, wagerID, voterID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.WagerVote), args.Error(1)
}

func (m *MockWagerVoteRepository) UpdateVote(ctx context.Context, wagerID int64, voterID int64, votedForID int64) error {
	args := m.Called(ctx, wagerID, voterID, votedForID)
	return args.Error(0)
}

func (m *MockWagerVoteRepository) CountVotes(ctx context.Context, wagerID int64) (*models.VoteCount, error) {
	args := m.Called(ctx, wagerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.VoteCount), args.Error(1)
}

// MockEventPublisher is a mock implementation of EventPublisher for testing
type MockEventPublisher struct {
	mock.Mock
}

func (m *MockEventPublisher) Publish(event events.Event) error {
	args := m.Called(event)
	return args.Error(0)
}

// MockGuildSettingsRepository is a mock implementation of GuildSettingsRepository
type MockGuildSettingsRepository struct {
	mock.Mock
}

func (m *MockGuildSettingsRepository) GetOrCreateGuildSettings(ctx context.Context, guildID int64) (*models.GuildSettings, error) {
	args := m.Called(ctx, guildID)
	return args.Get(0).(*models.GuildSettings), args.Error(1)
}

func (m *MockGuildSettingsRepository) UpdateGuildSettings(ctx context.Context, settings *models.GuildSettings) error {
	args := m.Called(ctx, settings)
	return args.Error(0)
}

// MockSummonerWatchRepository is a mock implementation of SummonerWatchRepository
type MockSummonerWatchRepository struct {
	mock.Mock
}

func (m *MockSummonerWatchRepository) CreateWatch(ctx context.Context, guildID int64, summonerName, region string) (*models.SummonerWatchDetail, error) {
	args := m.Called(ctx, guildID, summonerName, region)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SummonerWatchDetail), args.Error(1)
}

func (m *MockSummonerWatchRepository) GetWatchesByGuild(ctx context.Context, guildID int64) ([]*models.SummonerWatchDetail, error) {
	args := m.Called(ctx, guildID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.SummonerWatchDetail), args.Error(1)
}

func (m *MockSummonerWatchRepository) GetGuildsWatchingSummoner(ctx context.Context, summonerName, region string) ([]*models.GuildSummonerWatch, error) {
	args := m.Called(ctx, summonerName, region)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.GuildSummonerWatch), args.Error(1)
}

func (m *MockSummonerWatchRepository) DeleteWatch(ctx context.Context, guildID int64, summonerName, region string) error {
	args := m.Called(ctx, guildID, summonerName, region)
	return args.Error(0)
}

func (m *MockSummonerWatchRepository) GetWatch(ctx context.Context, guildID int64, summonerName, region string) (*models.SummonerWatchDetail, error) {
	args := m.Called(ctx, guildID, summonerName, region)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SummonerWatchDetail), args.Error(1)
}
