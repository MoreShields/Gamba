package application

import (
	"context"
	"testing"

	"gambler/discord-client/application/dto"
	"gambler/discord-client/config"
	"gambler/discord-client/domain/entities"
	"gambler/discord-client/domain/events"
	"gambler/discord-client/domain/interfaces"
	"gambler/discord-client/domain/testhelpers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGroupWagerAdminHandler_OverrideResult(t *testing.T) {
	testConfig := config.NewTestConfig()
	testConfig.DiscordToken = "test-token"
	config.SetTestConfig(testConfig)

	ctx := context.Background()
	guildID := int64(555)
	adminID := int64(999)
	wagerID := int64(42)
	winOptionID := int64(1001)

	mocks := newApplicationTestMocks()
	factory := &mockUnitOfWorkFactory{uow: mocks.uow}
	handler := NewGroupWagerAdminHandler(factory)

	wager := &entities.GroupWager{
		ID:                  wagerID,
		GuildID:             guildID,
		Condition:           "Correct short game",
		State:               entities.GroupWagerStateCancelled,
		WagerType:           entities.GroupWagerTypeHouse,
		TotalPot:            1500,
		MessageID:           123,
		ChannelID:           456,
		VotingPeriodMinutes: 5,
	}
	options := []*entities.GroupWagerOption{
		{ID: winOptionID, GroupWagerID: wagerID, OptionText: "Win", TotalAmount: 1500, OddsMultiplier: 2.0},
		{ID: 1002, GroupWagerID: wagerID, OptionText: "Loss", TotalAmount: 0, OddsMultiplier: 2.0},
	}
	participants := []*entities.GroupWagerParticipant{
		{ID: 10, GroupWagerID: wagerID, DiscordID: 111, OptionID: winOptionID, Amount: 1500},
	}

	mocks.uow.On("Begin", ctx).Return(nil)
	mocks.groupWagerRepo.On("GetDetailByID", ctx, wagerID).Return(&entities.GroupWagerDetail{Wager: wager, Options: options, Participants: participants}, nil).Twice()
	mocks.balanceHistoryRepo.On("GetByRelated", ctx, entities.RelatedTypeGroupWager, wagerID).Return([]*entities.BalanceHistory{}, nil)
	mocks.userRepo.On("GetByDiscordID", ctx, int64(111)).Return(&entities.User{DiscordID: 111, Balance: 10_000}, nil)
	mocks.userRepo.On("UpdateBalance", ctx, int64(111), int64(11_500)).Return(nil)
	mocks.balanceHistoryRepo.On("Record", ctx, mock.MatchedBy(func(history *entities.BalanceHistory) bool {
		return history.DiscordID == 111 && history.ChangeAmount == 1500 && history.TransactionType == entities.TransactionTypeGroupWagerWin
	})).Run(func(args mock.Arguments) {
		history := args.Get(1).(*entities.BalanceHistory)
		history.ID = 77
	}).Return(nil)
	mocks.eventPublisher.On("Publish", mock.MatchedBy(func(event events.Event) bool {
		balanceChange, ok := event.(events.BalanceChangeEvent)
		return ok && balanceChange.UserID == 111 && balanceChange.ChangeAmount == 1500
	})).Return(nil)
	mocks.groupWagerRepo.On("UpdateParticipantPayouts", ctx, mock.MatchedBy(func(updated []*entities.GroupWagerParticipant) bool {
		return len(updated) == 1 && updated[0].PayoutAmount != nil && *updated[0].PayoutAmount == 3000 && updated[0].BalanceHistoryID != nil && *updated[0].BalanceHistoryID == 77
	})).Return(nil)
	mocks.groupWagerRepo.On("Update", ctx, mock.MatchedBy(func(updated *entities.GroupWager) bool {
		return updated.ID == wagerID && updated.State == entities.GroupWagerStateResolved && updated.WinningOptionID != nil && *updated.WinningOptionID == winOptionID && updated.ResolverDiscordID != nil && *updated.ResolverDiscordID == adminID
	})).Return(nil)
	mocks.eventPublisher.On("Publish", mock.MatchedBy(func(event events.Event) bool {
		stateChange, ok := event.(events.GroupWagerStateChangeEvent)
		return ok && stateChange.GroupWagerID == wagerID && stateChange.OldState == string(entities.GroupWagerStateCancelled) && stateChange.NewState == string(entities.GroupWagerStateResolved)
	})).Return(nil)
	mocks.uow.On("Commit").Return(nil)
	mocks.uow.On("Rollback").Return(nil)

	resp, err := handler.OverrideResult(ctx, dto.OverrideGroupWagerResultRequest{
		GuildID:           guildID,
		AdminDiscordID:    adminID,
		GroupWagerID:      wagerID,
		WinningOptionText: "Win",
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, wagerID, resp.GroupWagerID)
	assert.Equal(t, "cancelled", resp.PreviousState)
	assert.Equal(t, "resolved", resp.CorrectedState)
	assert.Equal(t, "Win", resp.WinningOption)
	assert.Equal(t, winOptionID, resp.WinningOptionID)
	require.Len(t, resp.Winners, 1)
	assert.Equal(t, int64(111), resp.Winners[0].DiscordID)
	assert.Equal(t, int64(3000), resp.Winners[0].PayoutAmount)

	mocks.assertExpectations(t)
}

type applicationTestMocks struct {
	uow                *mockUnitOfWork
	userRepo           *testhelpers.MockUserRepository
	balanceHistoryRepo *testhelpers.MockBalanceHistoryRepository
	groupWagerRepo     *testhelpers.MockGroupWagerRepository
	eventPublisher     *testhelpers.MockEventPublisher
}

func newApplicationTestMocks() *applicationTestMocks {
	m := &applicationTestMocks{
		uow:                &mockUnitOfWork{},
		userRepo:           &testhelpers.MockUserRepository{},
		balanceHistoryRepo: &testhelpers.MockBalanceHistoryRepository{},
		groupWagerRepo:     &testhelpers.MockGroupWagerRepository{},
		eventPublisher:     &testhelpers.MockEventPublisher{},
	}
	m.uow.userRepo = m.userRepo
	m.uow.balanceHistoryRepo = m.balanceHistoryRepo
	m.uow.groupWagerRepo = m.groupWagerRepo
	m.uow.eventPublisher = m.eventPublisher
	return m
}

func (m *applicationTestMocks) assertExpectations(t *testing.T) {
	m.uow.AssertExpectations(t)
	m.userRepo.AssertExpectations(t)
	m.balanceHistoryRepo.AssertExpectations(t)
	m.groupWagerRepo.AssertExpectations(t)
	m.eventPublisher.AssertExpectations(t)
}

type mockUnitOfWorkFactory struct {
	uow UnitOfWork
}

func (f *mockUnitOfWorkFactory) CreateForGuild(guildID int64) UnitOfWork {
	return f.uow
}

type mockUnitOfWork struct {
	mock.Mock
	userRepo           interfaces.UserRepository
	balanceHistoryRepo interfaces.BalanceHistoryRepository
	groupWagerRepo     interfaces.GroupWagerRepository
	eventPublisher     interfaces.EventPublisher
}

func (m *mockUnitOfWork) Begin(ctx context.Context) error { return m.Called(ctx).Error(0) }
func (m *mockUnitOfWork) Commit() error                   { return m.Called().Error(0) }
func (m *mockUnitOfWork) Rollback() error                 { return m.Called().Error(0) }
func (m *mockUnitOfWork) UserRepository() interfaces.UserRepository {
	return m.userRepo
}
func (m *mockUnitOfWork) BalanceHistoryRepository() interfaces.BalanceHistoryRepository {
	return m.balanceHistoryRepo
}
func (m *mockUnitOfWork) GroupWagerRepository() interfaces.GroupWagerRepository {
	return m.groupWagerRepo
}
func (m *mockUnitOfWork) EventBus() interfaces.EventPublisher                         { return m.eventPublisher }
func (m *mockUnitOfWork) BetRepository() interfaces.BetRepository                     { return nil }
func (m *mockUnitOfWork) WagerRepository() interfaces.WagerRepository                 { return nil }
func (m *mockUnitOfWork) WagerVoteRepository() interfaces.WagerVoteRepository         { return nil }
func (m *mockUnitOfWork) GuildSettingsRepository() interfaces.GuildSettingsRepository { return nil }
func (m *mockUnitOfWork) SummonerWatchRepository() interfaces.SummonerWatchRepository { return nil }
func (m *mockUnitOfWork) WordleCompletionRepo() interfaces.WordleCompletionRepository { return nil }
func (m *mockUnitOfWork) HighRollerPurchaseRepository() interfaces.HighRollerPurchaseRepository {
	return nil
}
func (m *mockUnitOfWork) LotteryDrawRepository() interfaces.LotteryDrawRepository     { return nil }
func (m *mockUnitOfWork) LotteryTicketRepository() interfaces.LotteryTicketRepository { return nil }
func (m *mockUnitOfWork) LotteryWinnerRepository() interfaces.LotteryWinnerRepository { return nil }
func (m *mockUnitOfWork) StagedBetRepository() interfaces.StagedBetRepository         { return nil }

var _ UnitOfWork = (*mockUnitOfWork)(nil)
var _ UnitOfWorkFactory = (*mockUnitOfWorkFactory)(nil)
