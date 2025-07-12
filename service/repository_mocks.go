package service

import (
	"context"
	"time"

	"gambler/events"
	"gambler/models"

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

func (m *MockUserRepository) AddBalance(ctx context.Context, discordID int64, amount int64) error {
	args := m.Called(ctx, discordID, amount)
	return args.Error(0)
}

func (m *MockUserRepository) DeductBalance(ctx context.Context, discordID int64, amount int64) error {
	args := m.Called(ctx, discordID, amount)
	return args.Error(0)
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

// MockEventPublisher is a mock implementation of EventPublisher for testing
type MockEventPublisher struct {
	mock.Mock
}

func (m *MockEventPublisher) Publish(event events.Event) {
	m.Called(event)
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

