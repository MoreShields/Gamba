package services

import (
	"testing"
	"time"

	"gambler/discord-client/domain/entities"
	"gambler/discord-client/domain/events"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGroupWagerService_OverrideGroupWagerResult(t *testing.T) {
	fixture := NewGroupWagerTestFixture(t)

	t.Run("overrides cancelled house wager to win", func(t *testing.T) {
		fixture.Reset()

		wager := &entities.GroupWager{
			ID:                  TestWagerID,
			GuildID:             TestGuildID,
			Condition:           "Short game correction",
			State:               entities.GroupWagerStateCancelled,
			WagerType:           entities.GroupWagerTypeHouse,
			TotalPot:            1000,
			MessageID:           TestMessageID,
			ChannelID:           TestChannelID,
			VotingPeriodMinutes: 5,
		}
		options := []*entities.GroupWagerOption{
			{ID: TestOption1ID, GroupWagerID: TestWagerID, OptionText: "Win", OptionOrder: 0, TotalAmount: 1000, OddsMultiplier: 2.0},
			{ID: TestOption2ID, GroupWagerID: TestWagerID, OptionText: "Loss", OptionOrder: 1, TotalAmount: 0, OddsMultiplier: 2.0},
		}
		participants := []*entities.GroupWagerParticipant{
			{ID: 10, GroupWagerID: TestWagerID, DiscordID: TestUser1ID, OptionID: TestOption1ID, Amount: 1000},
		}

		fixture.Helper.ExpectWagerDetailLookup(TestWagerID, &entities.GroupWagerDetail{Wager: wager, Options: options, Participants: participants})
		fixture.Mocks.BalanceHistoryRepo.On("GetByRelated", mock.Anything, entities.RelatedTypeGroupWager, TestWagerID).Return([]*entities.BalanceHistory{}, nil)
		fixture.Helper.ExpectUserLookup(TestUser1ID, &entities.User{DiscordID: TestUser1ID, Balance: TestInitialBalance})
		fixture.Helper.ExpectBalanceUpdate(TestUser1ID, TestInitialBalance+1000)
		expectHistoryRecord(fixture, TestUser1ID, int64(1000), entities.TransactionTypeGroupWagerWin, int64(101))
		fixture.Mocks.GroupWagerRepo.On("UpdateParticipantPayouts", mock.Anything, mock.MatchedBy(func(ps []*entities.GroupWagerParticipant) bool {
			return len(ps) == 1 && ps[0].DiscordID == TestUser1ID && ps[0].PayoutAmount != nil && *ps[0].PayoutAmount == 2000 && ps[0].BalanceHistoryID != nil && *ps[0].BalanceHistoryID == 101
		})).Return(nil)
		fixture.Mocks.GroupWagerRepo.On("Update", mock.Anything, mock.MatchedBy(func(gw *entities.GroupWager) bool {
			return gw.ID == TestWagerID && gw.State == entities.GroupWagerStateResolved && gw.WinningOptionID != nil && *gw.WinningOptionID == TestOption1ID && gw.ResolverDiscordID != nil && *gw.ResolverDiscordID == TestResolverID && gw.ResolvedAt != nil
		})).Return(nil)
		fixture.Mocks.EventPublisher.On("Publish", mock.MatchedBy(func(e events.Event) bool {
			sc, ok := e.(events.GroupWagerStateChangeEvent)
			return ok && sc.GroupWagerID == TestWagerID && sc.OldState == string(entities.GroupWagerStateCancelled) && sc.NewState == string(entities.GroupWagerStateResolved)
		})).Return(nil)

		resolverID := int64(TestResolverID)
		result, err := fixture.Service.OverrideGroupWagerResult(fixture.Ctx, TestWagerID, &resolverID, TestOption1ID)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, entities.GroupWagerStateResolved, result.GroupWager.State)
		assert.Equal(t, TestOption1ID, *result.GroupWager.WinningOptionID)
		assert.Len(t, result.Winners, 1)
		assert.Empty(t, result.Losers)
		fixture.AssertAllMocks()
	})

	t.Run("applies net adjustments when changing a resolved result", func(t *testing.T) {
		fixture.Reset()
		resolvedAt := time.Now().Add(-time.Hour)
		previousWinningOptionID := int64(TestOption2ID)

		wager := &entities.GroupWager{
			ID:                  TestWagerID,
			GuildID:             TestGuildID,
			Condition:           "Correct winner",
			State:               entities.GroupWagerStateResolved,
			WagerType:           entities.GroupWagerTypeHouse,
			WinningOptionID:     &previousWinningOptionID,
			TotalPot:            2000,
			MessageID:           TestMessageID,
			ChannelID:           TestChannelID,
			VotingPeriodMinutes: 5,
			ResolvedAt:          &resolvedAt,
		}
		options := []*entities.GroupWagerOption{
			{ID: TestOption1ID, GroupWagerID: TestWagerID, OptionText: "Win", OptionOrder: 0, TotalAmount: 1000, OddsMultiplier: 2.0},
			{ID: TestOption2ID, GroupWagerID: TestWagerID, OptionText: "Loss", OptionOrder: 1, TotalAmount: 1000, OddsMultiplier: 2.0},
		}
		participants := []*entities.GroupWagerParticipant{
			{ID: 10, GroupWagerID: TestWagerID, DiscordID: TestUser1ID, OptionID: TestOption1ID, Amount: 1000, PayoutAmount: int64Ptr(0), BalanceHistoryID: int64Ptr(1)},
			{ID: 11, GroupWagerID: TestWagerID, DiscordID: TestUser2ID, OptionID: TestOption2ID, Amount: 1000, PayoutAmount: int64Ptr(2000), BalanceHistoryID: int64Ptr(2)},
		}
		priorHistory := []*entities.BalanceHistory{
			{ID: 1, DiscordID: TestUser1ID, ChangeAmount: -1000, TransactionType: entities.TransactionTypeGroupWagerLoss},
			{ID: 2, DiscordID: TestUser2ID, ChangeAmount: 1000, TransactionType: entities.TransactionTypeGroupWagerWin},
		}

		fixture.Helper.ExpectWagerDetailLookup(TestWagerID, &entities.GroupWagerDetail{Wager: wager, Options: options, Participants: participants})
		fixture.Mocks.BalanceHistoryRepo.On("GetByRelated", mock.Anything, entities.RelatedTypeGroupWager, TestWagerID).Return(priorHistory, nil)
		fixture.Helper.ExpectUserLookup(TestUser1ID, &entities.User{DiscordID: TestUser1ID, Balance: TestInitialBalance - 1000})
		fixture.Helper.ExpectBalanceUpdate(TestUser1ID, TestInitialBalance+1000)
		expectHistoryRecord(fixture, TestUser1ID, int64(2000), entities.TransactionTypeGroupWagerWin, int64(201))
		fixture.Helper.ExpectUserLookup(TestUser2ID, &entities.User{DiscordID: TestUser2ID, Balance: TestInitialBalance + 1000})
		fixture.Helper.ExpectBalanceUpdate(TestUser2ID, TestInitialBalance-1000)
		expectHistoryRecord(fixture, TestUser2ID, int64(-2000), entities.TransactionTypeGroupWagerLoss, int64(202))
		fixture.Mocks.GroupWagerRepo.On("UpdateParticipantPayouts", mock.Anything, mock.MatchedBy(func(ps []*entities.GroupWagerParticipant) bool {
			if len(ps) != 2 {
				return false
			}
			return ps[0].DiscordID == TestUser1ID && *ps[0].PayoutAmount == 2000 && *ps[0].BalanceHistoryID == 201 && ps[1].DiscordID == TestUser2ID && *ps[1].PayoutAmount == 0 && *ps[1].BalanceHistoryID == 202
		})).Return(nil)
		fixture.Mocks.GroupWagerRepo.On("Update", mock.Anything, mock.MatchedBy(func(gw *entities.GroupWager) bool {
			return gw.State == entities.GroupWagerStateResolved && gw.WinningOptionID != nil && *gw.WinningOptionID == TestOption1ID
		})).Return(nil)
		fixture.Mocks.EventPublisher.On("Publish", mock.AnythingOfType("events.GroupWagerStateChangeEvent")).Return(nil)

		resolverID := int64(TestResolverID)
		result, err := fixture.Service.OverrideGroupWagerResult(fixture.Ctx, TestWagerID, &resolverID, TestOption1ID)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.Winners, 1)
		assert.Len(t, result.Losers, 1)
		assert.Equal(t, int64(2000), result.PayoutDetails[TestUser1ID])
		assert.Equal(t, int64(0), result.PayoutDetails[TestUser2ID])
		fixture.AssertAllMocks()
	})
}

func expectHistoryRecord(fixture *GroupWagerTestFixture, discordID int64, changeAmount int64, transactionType entities.TransactionType, assignedID int64) {
	fixture.Mocks.BalanceHistoryRepo.On("Record", mock.Anything, mock.MatchedBy(func(history *entities.BalanceHistory) bool {
		return history.DiscordID == discordID && history.ChangeAmount == changeAmount && history.TransactionType == transactionType && history.RelatedID != nil && *history.RelatedID == TestWagerID
	})).Run(func(args mock.Arguments) {
		history := args.Get(1).(*entities.BalanceHistory)
		history.ID = assignedID
	}).Return(nil)
	fixture.Mocks.EventPublisher.On("Publish", mock.MatchedBy(func(event events.Event) bool {
		balanceChange, ok := event.(events.BalanceChangeEvent)
		return ok && balanceChange.UserID == discordID && balanceChange.ChangeAmount == changeAmount && balanceChange.TransactionType == transactionType
	})).Return(nil)
}

func int64Ptr(v int64) *int64 {
	return &v
}
