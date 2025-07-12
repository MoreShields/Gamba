package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"gambler/events"
	"gambler/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Test utilities

func createTestGroupWagerService() (GroupWagerService, *MockUserRepository, *MockGroupWagerRepository, *MockBalanceHistoryRepository, *MockEventPublisher) {
	mockUserRepo := new(MockUserRepository)
	mockGroupWagerRepo := new(MockGroupWagerRepository)
	mockBalanceHistoryRepo := new(MockBalanceHistoryRepository)
	mockEventPublisher := new(MockEventPublisher)

	service := NewGroupWagerService(mockGroupWagerRepo, mockUserRepo, mockBalanceHistoryRepo, mockEventPublisher)
	return service, mockUserRepo, mockGroupWagerRepo, mockBalanceHistoryRepo, mockEventPublisher
}

func createTestUser(userID int64, balance int64) *models.User {
	return &models.User{
		DiscordID:        userID,
		Username:         "testuser",
		Balance:          balance,
		AvailableBalance: balance,
	}
}

func createTestGroupWager(groupWagerID int64, state models.GroupWagerState, totalPot int64) *models.GroupWager {
	futureTime := time.Now().Add(24 * time.Hour)
	return &models.GroupWager{
		ID:               groupWagerID,
		CreatorDiscordID: 999999,
		Condition:        "Test wager",
		State:            state,
		TotalPot:         totalPot,
		MinParticipants:  3,
		VotingEndsAt:     &futureTime,
		MessageID:        123,
		ChannelID:        456,
	}
}

func createTestGroupWagerOption(optionID, groupWagerID int64, text string, order int16, totalAmount int64) *models.GroupWagerOption {
	return &models.GroupWagerOption{
		ID:           optionID,
		GroupWagerID: groupWagerID,
		OptionText:   text,
		OptionOrder:  order,
		TotalAmount:  totalAmount,
	}
}

func createTestParticipant(id, groupWagerID, userID, optionID, amount int64) *models.GroupWagerParticipant {
	return &models.GroupWagerParticipant{
		ID:           id,
		GroupWagerID: groupWagerID,
		DiscordID:    userID,
		OptionID:     optionID,
		Amount:       amount,
	}
}

func createTestGroupWagerDetail(groupWager *models.GroupWager, options []*models.GroupWagerOption, participants []*models.GroupWagerParticipant) *models.GroupWagerDetail {
	return &models.GroupWagerDetail{
		Wager:        groupWager,
		Options:      options,
		Participants: participants,
	}
}

// Mock helper functions

func setupUserMocks(mockUserRepo *MockUserRepository, user *models.User) {
	mockUserRepo.On("GetByDiscordID", mock.Anything, user.DiscordID).Return(user, nil)
}

func setupGroupWagerMocks(mockGroupWagerRepo *MockGroupWagerRepository, groupWager *models.GroupWager, detail *models.GroupWagerDetail) {
	mockGroupWagerRepo.On("GetByID", mock.Anything, groupWager.ID).Return(groupWager, nil)
	mockGroupWagerRepo.On("GetDetailByID", mock.Anything, groupWager.ID).Return(detail, nil)
}

func setupParticipantMocks(mockGroupWagerRepo *MockGroupWagerRepository, groupWagerID, userID int64, participant *models.GroupWagerParticipant) {
	mockGroupWagerRepo.On("GetParticipant", mock.Anything, groupWagerID, userID).Return(participant, nil)
}

func assertAllMockExpectations(t *testing.T, mocks ...interface{}) {
	for _, m := range mocks {
		if mockObj, ok := m.(interface{ AssertExpectations(mock.TestingT) bool }); ok {
			mockObj.AssertExpectations(t)
		}
	}
}

// Tests

func TestGroupWagerService_PlaceBet(t *testing.T) {
	ctx := context.Background()

	t.Run("successful first bet", func(t *testing.T) {
		// Setup
		service, mockUserRepo, mockGroupWagerRepo, mockBalanceHistoryRepo, mockEventPublisher := createTestGroupWagerService()

		userID := int64(123456)
		groupWagerID := int64(1)
		optionID := int64(10)
		betAmount := int64(1000)

		testUser := createTestUser(userID, 10000)
		groupWager := createTestGroupWager(groupWagerID, models.GroupWagerStateActive, 0)
		option := createTestGroupWagerOption(optionID, groupWagerID, "Option 1", 0, 0)
		detail := createTestGroupWagerDetail(groupWager, []*models.GroupWagerOption{option}, []*models.GroupWagerParticipant{})

		// Setup mocks
		setupUserMocks(mockUserRepo, testUser)
		setupGroupWagerMocks(mockGroupWagerRepo, groupWager, detail)
		setupParticipantMocks(mockGroupWagerRepo, groupWagerID, userID, nil) // No existing participant

		// Expect new participant creation
		mockGroupWagerRepo.On("SaveParticipant", mock.Anything, mock.MatchedBy(func(p *models.GroupWagerParticipant) bool {
			return p.GroupWagerID == groupWagerID &&
				p.DiscordID == userID &&
				p.OptionID == optionID &&
				p.Amount == betAmount
		})).Return(nil)

		// Expect option total update
		mockGroupWagerRepo.On("UpdateOptionTotal", mock.Anything, optionID, betAmount).Return(nil)

		// Expect group wager total update
		mockGroupWagerRepo.On("Update", mock.Anything, mock.MatchedBy(func(gw *models.GroupWager) bool {
			return gw.ID == groupWagerID && gw.TotalPot == betAmount
		})).Return(nil)

		// Execute
		participant, err := service.PlaceBet(ctx, groupWagerID, userID, optionID, betAmount)

		// Verify
		require.NoError(t, err)
		require.NotNil(t, participant)
		assert.Equal(t, optionID, participant.OptionID)
		assert.Equal(t, betAmount, participant.Amount)

		assertAllMockExpectations(t, mockUserRepo, mockGroupWagerRepo, mockBalanceHistoryRepo, mockEventPublisher)
	})

	t.Run("increase existing bet same option", func(t *testing.T) {
		// Setup
		service, mockUserRepo, mockGroupWagerRepo, mockBalanceHistoryRepo, mockEventPublisher := createTestGroupWagerService()

		userID := int64(123456)
		groupWagerID := int64(1)
		optionID := int64(10)
		initialAmount := int64(1000)
		newAmount := int64(3000)

		testUser := createTestUser(userID, 10000)
		groupWager := createTestGroupWager(groupWagerID, models.GroupWagerStateActive, initialAmount)
		option := createTestGroupWagerOption(optionID, groupWagerID, "Option 1", 0, initialAmount)
		existingParticipant := createTestParticipant(1, groupWagerID, userID, optionID, initialAmount)
		detail := createTestGroupWagerDetail(groupWager, []*models.GroupWagerOption{option}, []*models.GroupWagerParticipant{existingParticipant})

		// Setup mocks
		setupUserMocks(mockUserRepo, testUser)
		setupGroupWagerMocks(mockGroupWagerRepo, groupWager, detail)
		setupParticipantMocks(mockGroupWagerRepo, groupWagerID, userID, existingParticipant)

		// Expect participant update
		mockGroupWagerRepo.On("SaveParticipant", mock.Anything, mock.MatchedBy(func(p *models.GroupWagerParticipant) bool {
			return p.ID == existingParticipant.ID &&
				p.OptionID == optionID &&
				p.Amount == newAmount
		})).Return(nil)

		// Expect option total update to new amount
		mockGroupWagerRepo.On("UpdateOptionTotal", mock.Anything, optionID, newAmount).Return(nil)

		// Expect group wager total update
		mockGroupWagerRepo.On("Update", mock.Anything, mock.MatchedBy(func(gw *models.GroupWager) bool {
			return gw.ID == groupWagerID && gw.TotalPot == newAmount
		})).Return(nil)

		// Execute
		participant, err := service.PlaceBet(ctx, groupWagerID, userID, optionID, newAmount)

		// Verify
		require.NoError(t, err)
		require.NotNil(t, participant)
		assert.Equal(t, optionID, participant.OptionID)
		assert.Equal(t, newAmount, participant.Amount)

		assertAllMockExpectations(t, mockUserRepo, mockGroupWagerRepo, mockBalanceHistoryRepo, mockEventPublisher)
	})

	t.Run("change to different option", func(t *testing.T) {
		// Setup
		service, mockUserRepo, mockGroupWagerRepo, mockBalanceHistoryRepo, mockEventPublisher := createTestGroupWagerService()

		userID := int64(123456)
		groupWagerID := int64(1)
		option1ID := int64(10)
		option2ID := int64(20)
		initialAmount := int64(1000)
		newAmount := int64(2000)

		testUser := createTestUser(userID, 10000)
		groupWager := createTestGroupWager(groupWagerID, models.GroupWagerStateActive, initialAmount)
		option1 := createTestGroupWagerOption(option1ID, groupWagerID, "Option 1", 0, initialAmount)
		option2 := createTestGroupWagerOption(option2ID, groupWagerID, "Option 2", 1, 0)
		existingParticipant := createTestParticipant(1, groupWagerID, userID, option1ID, initialAmount)
		detail := createTestGroupWagerDetail(groupWager, []*models.GroupWagerOption{option1, option2}, []*models.GroupWagerParticipant{existingParticipant})

		// Setup mocks
		setupUserMocks(mockUserRepo, testUser)
		setupGroupWagerMocks(mockGroupWagerRepo, groupWager, detail)
		setupParticipantMocks(mockGroupWagerRepo, groupWagerID, userID, existingParticipant)

		// Expect participant update to new option
		mockGroupWagerRepo.On("SaveParticipant", mock.Anything, mock.MatchedBy(func(p *models.GroupWagerParticipant) bool {
			return p.ID == existingParticipant.ID &&
				p.OptionID == option2ID &&
				p.Amount == newAmount
		})).Return(nil)

		// Expect old option total to be reduced to 0
		mockGroupWagerRepo.On("UpdateOptionTotal", mock.Anything, option1ID, int64(0)).Return(nil)

		// Expect new option total to be set to new amount
		mockGroupWagerRepo.On("UpdateOptionTotal", mock.Anything, option2ID, newAmount).Return(nil)

		// Expect group wager total update
		mockGroupWagerRepo.On("Update", mock.Anything, mock.MatchedBy(func(gw *models.GroupWager) bool {
			return gw.ID == groupWagerID && gw.TotalPot == newAmount
		})).Return(nil)

		// Execute
		participant, err := service.PlaceBet(ctx, groupWagerID, userID, option2ID, newAmount)

		// Verify
		require.NoError(t, err)
		require.NotNil(t, participant)
		assert.Equal(t, option2ID, participant.OptionID)
		assert.Equal(t, newAmount, participant.Amount)

		assertAllMockExpectations(t, mockUserRepo, mockGroupWagerRepo, mockBalanceHistoryRepo, mockEventPublisher)
	})

	t.Run("user not found", func(t *testing.T) {
		// Setup
		service, mockUserRepo, mockGroupWagerRepo, mockBalanceHistoryRepo, mockEventPublisher := createTestGroupWagerService()

		userID := int64(123456)
		groupWagerID := int64(1)
		optionID := int64(10)
		betAmount := int64(1000)

		// Create test group wager and detail for the calls that happen before user validation
		groupWager := createTestGroupWager(groupWagerID, models.GroupWagerStateActive, 0)
		option := createTestGroupWagerOption(optionID, groupWagerID, "Option 1", 0, 0)
		detail := createTestGroupWagerDetail(groupWager, []*models.GroupWagerOption{option}, []*models.GroupWagerParticipant{})

		// Setup mocks - group wager is found but user is not
		mockGroupWagerRepo.On("GetByID", mock.Anything, groupWagerID).Return(groupWager, nil)
		mockGroupWagerRepo.On("GetDetailByID", mock.Anything, groupWagerID).Return(detail, nil)
		mockUserRepo.On("GetByDiscordID", mock.Anything, userID).Return(nil, nil)

		// Execute
		participant, err := service.PlaceBet(ctx, groupWagerID, userID, optionID, betAmount)

		// Verify
		assert.Error(t, err)
		assert.Nil(t, participant)
		assert.Contains(t, err.Error(), "user not found")

		assertAllMockExpectations(t, mockUserRepo, mockGroupWagerRepo, mockBalanceHistoryRepo, mockEventPublisher)
	})

	t.Run("group wager not found", func(t *testing.T) {
		// Setup
		service, mockUserRepo, mockGroupWagerRepo, mockBalanceHistoryRepo, mockEventPublisher := createTestGroupWagerService()

		userID := int64(123456)
		groupWagerID := int64(1)
		optionID := int64(10)
		betAmount := int64(1000)

		// Setup mocks - group wager not found
		mockGroupWagerRepo.On("GetByID", mock.Anything, groupWagerID).Return(nil, nil)

		// Execute
		participant, err := service.PlaceBet(ctx, groupWagerID, userID, optionID, betAmount)

		// Verify
		assert.Error(t, err)
		assert.Nil(t, participant)
		assert.Contains(t, err.Error(), "group wager not found")

		assertAllMockExpectations(t, mockUserRepo, mockGroupWagerRepo, mockBalanceHistoryRepo, mockEventPublisher)
	})

	t.Run("insufficient balance", func(t *testing.T) {
		// Setup
		service, mockUserRepo, mockGroupWagerRepo, mockBalanceHistoryRepo, mockEventPublisher := createTestGroupWagerService()

		userID := int64(123456)
		groupWagerID := int64(1)
		optionID := int64(10)
		betAmount := int64(10000)

		// User with insufficient balance
		testUser := createTestUser(userID, 1000) // Only 1000, but betting 10000
		groupWager := createTestGroupWager(groupWagerID, models.GroupWagerStateActive, 0)
		option := createTestGroupWagerOption(optionID, groupWagerID, "Option 1", 0, 0)
		detail := createTestGroupWagerDetail(groupWager, []*models.GroupWagerOption{option}, []*models.GroupWagerParticipant{})

		// Setup mocks
		setupUserMocks(mockUserRepo, testUser)
		setupGroupWagerMocks(mockGroupWagerRepo, groupWager, detail)
		setupParticipantMocks(mockGroupWagerRepo, groupWagerID, userID, nil) // No existing participant

		// Execute
		participant, err := service.PlaceBet(ctx, groupWagerID, userID, optionID, betAmount)

		// Verify
		assert.Error(t, err)
		assert.Nil(t, participant)
		assert.Contains(t, err.Error(), "insufficient balance")

		assertAllMockExpectations(t, mockUserRepo, mockGroupWagerRepo, mockBalanceHistoryRepo, mockEventPublisher)
	})

	t.Run("wager not active", func(t *testing.T) {
		// Setup
		service, mockUserRepo, mockGroupWagerRepo, mockBalanceHistoryRepo, mockEventPublisher := createTestGroupWagerService()

		userID := int64(123456)
		groupWagerID := int64(1)
		optionID := int64(10)
		betAmount := int64(1000)

		// Group wager is resolved, not active
		groupWager := createTestGroupWager(groupWagerID, models.GroupWagerStateResolved, 0)

		// Setup mocks - only need GetByID since service fails early when wager is not active
		mockGroupWagerRepo.On("GetByID", mock.Anything, groupWagerID).Return(groupWager, nil)

		// Execute
		participant, err := service.PlaceBet(ctx, groupWagerID, userID, optionID, betAmount)

		// Verify
		assert.Error(t, err)
		assert.Nil(t, participant)
		assert.Contains(t, err.Error(), "not accepting bets")

		assertAllMockExpectations(t, mockUserRepo, mockGroupWagerRepo, mockBalanceHistoryRepo, mockEventPublisher)
	})

	t.Run("voting period expired", func(t *testing.T) {
		// Setup
		service, mockUserRepo, mockGroupWagerRepo, mockBalanceHistoryRepo, mockEventPublisher := createTestGroupWagerService()

		userID := int64(123456)
		groupWagerID := int64(1)
		optionID := int64(10)
		betAmount := int64(1000)

		// Create expired wager
		expiredTime := time.Now().Add(-1 * time.Hour) // 1 hour ago
		groupWager := &models.GroupWager{
			ID:               groupWagerID,
			CreatorDiscordID: 999999,
			Condition:        "Test wager",
			State:            models.GroupWagerStateActive,
			TotalPot:         0,
			MinParticipants:  3,
			VotingEndsAt:     &expiredTime, // Expired
			MessageID:        123,
			ChannelID:        456,
		}

		// Setup mocks - only need GetByID since service fails early when voting period expired
		mockGroupWagerRepo.On("GetByID", mock.Anything, groupWagerID).Return(groupWager, nil)

		// Execute
		participant, err := service.PlaceBet(ctx, groupWagerID, userID, optionID, betAmount)

		// Verify
		assert.Error(t, err)
		assert.Nil(t, participant)
		assert.Contains(t, err.Error(), "voting period has ended")

		assertAllMockExpectations(t, mockUserRepo, mockGroupWagerRepo, mockBalanceHistoryRepo, mockEventPublisher)
	})

	t.Run("invalid option", func(t *testing.T) {
		// Setup
		service, mockUserRepo, mockGroupWagerRepo, mockBalanceHistoryRepo, mockEventPublisher := createTestGroupWagerService()

		userID := int64(123456)
		groupWagerID := int64(1)
		validOptionID := int64(10)
		invalidOptionID := int64(99) // Non-existent option
		betAmount := int64(1000)

		groupWager := createTestGroupWager(groupWagerID, models.GroupWagerStateActive, 0)
		option := createTestGroupWagerOption(validOptionID, groupWagerID, "Option 1", 0, 0)
		detail := createTestGroupWagerDetail(groupWager, []*models.GroupWagerOption{option}, []*models.GroupWagerParticipant{})

		// Setup mocks - need GetByID and GetDetailByID to validate options
		mockGroupWagerRepo.On("GetByID", mock.Anything, groupWagerID).Return(groupWager, nil)
		mockGroupWagerRepo.On("GetDetailByID", mock.Anything, groupWagerID).Return(detail, nil)

		// Execute with invalid option ID
		participant, err := service.PlaceBet(ctx, groupWagerID, userID, invalidOptionID, betAmount)

		// Verify
		assert.Error(t, err)
		assert.Nil(t, participant)
		assert.Contains(t, err.Error(), "invalid option")

		assertAllMockExpectations(t, mockUserRepo, mockGroupWagerRepo, mockBalanceHistoryRepo, mockEventPublisher)
	})
}

func TestGroupWagerService_ResolveGroupWager(t *testing.T) {
	ctx := context.Background()

	t.Run("successful resolution with winners and losers", func(t *testing.T) {
		// Setup
		service, mockUserRepo, mockGroupWagerRepo, mockBalanceHistoryRepo, mockEventPublisher := createTestGroupWagerService()
		
		// Set resolver IDs in config
		service.(*groupWagerService).config.ResolverDiscordIDs = []int64{999999}

		resolverID := int64(999999)
		groupWagerID := int64(1)
		winningOptionID := int64(10)

		// Create test data
		groupWager := createTestGroupWager(groupWagerID, models.GroupWagerStateActive, 5000)
		option1 := createTestGroupWagerOption(winningOptionID, groupWagerID, "Winning Option", 0, 3000)
		option2 := createTestGroupWagerOption(20, groupWagerID, "Losing Option", 1, 2000)
		
		// Create participants
		winner1 := createTestParticipant(1, groupWagerID, 111111, winningOptionID, 2000)
		winner2 := createTestParticipant(2, groupWagerID, 222222, winningOptionID, 1000)
		loser1 := createTestParticipant(3, groupWagerID, 333333, option2.ID, 1500)
		loser2 := createTestParticipant(4, groupWagerID, 444444, option2.ID, 500)

		participants := []*models.GroupWagerParticipant{winner1, winner2, loser1, loser2}
		detail := createTestGroupWagerDetail(groupWager, []*models.GroupWagerOption{option1, option2}, participants)

		// Create test users
		winnerUser1 := createTestUser(111111, 10000)
		winnerUser2 := createTestUser(222222, 10000)
		loserUser1 := createTestUser(333333, 10000)
		loserUser2 := createTestUser(444444, 10000)

		// Setup mocks
		mockGroupWagerRepo.On("GetByID", mock.Anything, groupWagerID).Return(groupWager, nil)
		mockGroupWagerRepo.On("GetDetailByID", mock.Anything, groupWagerID).Return(detail, nil)

		// Mock user lookups for all participants
		mockUserRepo.On("GetByDiscordID", mock.Anything, int64(111111)).Return(winnerUser1, nil)
		mockUserRepo.On("GetByDiscordID", mock.Anything, int64(222222)).Return(winnerUser2, nil)
		mockUserRepo.On("GetByDiscordID", mock.Anything, int64(333333)).Return(loserUser1, nil)
		mockUserRepo.On("GetByDiscordID", mock.Anything, int64(444444)).Return(loserUser2, nil)

		// Mock balance updates for winners
		// Winner1 gets 3333 (2000/3000 * 5000), net win = 3333 - 2000 = 1333
		mockUserRepo.On("AddBalance", mock.Anything, int64(111111), int64(1333)).Return(nil)
		// Winner2 gets 1666 (1000/3000 * 5000 with integer division), net win = 1666 - 1000 = 666
		mockUserRepo.On("AddBalance", mock.Anything, int64(222222), int64(666)).Return(nil)

		// Mock balance deductions for losers
		mockUserRepo.On("DeductBalance", mock.Anything, int64(333333), int64(1500)).Return(nil)
		mockUserRepo.On("DeductBalance", mock.Anything, int64(444444), int64(500)).Return(nil)

		// Mock balance history recording
		mockBalanceHistoryRepo.On("Record", mock.Anything, mock.MatchedBy(func(h *models.BalanceHistory) bool {
			return h.DiscordID == 111111 && h.TransactionType == models.TransactionTypeGroupWagerWin && h.ChangeAmount == 1333
		})).Return(nil)
		mockBalanceHistoryRepo.On("Record", mock.Anything, mock.MatchedBy(func(h *models.BalanceHistory) bool {
			return h.DiscordID == 222222 && h.TransactionType == models.TransactionTypeGroupWagerWin && h.ChangeAmount == 666
		})).Return(nil)
		mockBalanceHistoryRepo.On("Record", mock.Anything, mock.MatchedBy(func(h *models.BalanceHistory) bool {
			return h.DiscordID == 333333 && h.TransactionType == models.TransactionTypeGroupWagerLoss && h.ChangeAmount == -1500
		})).Return(nil)
		mockBalanceHistoryRepo.On("Record", mock.Anything, mock.MatchedBy(func(h *models.BalanceHistory) bool {
			return h.DiscordID == 444444 && h.TransactionType == models.TransactionTypeGroupWagerLoss && h.ChangeAmount == -500
		})).Return(nil)

		// Mock event publishing
		mockEventPublisher.On("Publish", mock.AnythingOfType("events.BalanceChangeEvent")).Return().Times(4)
		mockEventPublisher.On("Publish", mock.MatchedBy(func(e events.GroupWagerStateChangeEvent) bool {
			return e.GroupWagerID == groupWagerID && 
				e.OldState == string(models.GroupWagerStateActive) && 
				e.NewState == string(models.GroupWagerStateResolved)
		})).Return()

		// Mock participant payout updates
		mockGroupWagerRepo.On("UpdateParticipantPayouts", mock.Anything, mock.MatchedBy(func(participants []*models.GroupWagerParticipant) bool {
			// Verify all participants have payout amounts set
			for _, p := range participants {
				if p.PayoutAmount == nil {
					return false
				}
			}
			return len(participants) == 4
		})).Return(nil)

		// Mock group wager state update
		mockGroupWagerRepo.On("Update", mock.Anything, mock.MatchedBy(func(gw *models.GroupWager) bool {
			return gw.ID == groupWagerID && 
				gw.State == models.GroupWagerStateResolved &&
				gw.ResolverDiscordID != nil && *gw.ResolverDiscordID == resolverID &&
				gw.WinningOptionID != nil && *gw.WinningOptionID == winningOptionID &&
				gw.ResolvedAt != nil
		})).Return(nil)

		// Execute
		result, err := service.ResolveGroupWager(ctx, groupWagerID, resolverID, winningOptionID)

		// Verify
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, groupWagerID, result.GroupWager.ID)
		assert.Equal(t, winningOptionID, result.WinningOption.ID)
		assert.Len(t, result.Winners, 2)
		assert.Len(t, result.Losers, 2)
		assert.Equal(t, int64(5000), result.TotalPot)

		// Verify payout calculations
		assert.Equal(t, int64(3333), result.PayoutDetails[111111])
		assert.Equal(t, int64(1666), result.PayoutDetails[222222])
		assert.Equal(t, int64(0), result.PayoutDetails[333333])
		assert.Equal(t, int64(0), result.PayoutDetails[444444])

		assertAllMockExpectations(t, mockUserRepo, mockGroupWagerRepo, mockBalanceHistoryRepo, mockEventPublisher)
	})

	t.Run("user not authorized to resolve", func(t *testing.T) {
		// Setup
		service, mockUserRepo, mockGroupWagerRepo, mockBalanceHistoryRepo, mockEventPublisher := createTestGroupWagerService()
		
		// Set resolver IDs without our test user
		service.(*groupWagerService).config.ResolverDiscordIDs = []int64{999999}

		unauthorizedUserID := int64(123456) // Not in resolver list
		groupWagerID := int64(1)
		winningOptionID := int64(10)

		// Execute
		result, err := service.ResolveGroupWager(ctx, groupWagerID, unauthorizedUserID, winningOptionID)

		// Verify
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "not authorized to resolve")

		assertAllMockExpectations(t, mockUserRepo, mockGroupWagerRepo, mockBalanceHistoryRepo, mockEventPublisher)
	})

	t.Run("group wager not found", func(t *testing.T) {
		// Setup
		service, mockUserRepo, mockGroupWagerRepo, mockBalanceHistoryRepo, mockEventPublisher := createTestGroupWagerService()
		
		// Set resolver IDs
		service.(*groupWagerService).config.ResolverDiscordIDs = []int64{999999}

		resolverID := int64(999999)
		groupWagerID := int64(1)
		winningOptionID := int64(10)

		// Setup mocks
		mockGroupWagerRepo.On("GetByID", mock.Anything, groupWagerID).Return(nil, nil)

		// Execute
		result, err := service.ResolveGroupWager(ctx, groupWagerID, resolverID, winningOptionID)

		// Verify
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "group wager not found")

		assertAllMockExpectations(t, mockUserRepo, mockGroupWagerRepo, mockBalanceHistoryRepo, mockEventPublisher)
	})

	t.Run("group wager already resolved", func(t *testing.T) {
		// Setup
		service, mockUserRepo, mockGroupWagerRepo, mockBalanceHistoryRepo, mockEventPublisher := createTestGroupWagerService()
		
		// Set resolver IDs
		service.(*groupWagerService).config.ResolverDiscordIDs = []int64{999999}

		resolverID := int64(999999)
		groupWagerID := int64(1)
		winningOptionID := int64(10)

		// Create already resolved wager
		groupWager := createTestGroupWager(groupWagerID, models.GroupWagerStateResolved, 5000)

		// Setup mocks
		mockGroupWagerRepo.On("GetByID", mock.Anything, groupWagerID).Return(groupWager, nil)

		// Execute
		result, err := service.ResolveGroupWager(ctx, groupWagerID, resolverID, winningOptionID)

		// Verify
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "cannot be resolved")

		assertAllMockExpectations(t, mockUserRepo, mockGroupWagerRepo, mockBalanceHistoryRepo, mockEventPublisher)
	})

	t.Run("invalid winning option ID", func(t *testing.T) {
		// Setup
		service, mockUserRepo, mockGroupWagerRepo, mockBalanceHistoryRepo, mockEventPublisher := createTestGroupWagerService()
		
		// Set resolver IDs
		service.(*groupWagerService).config.ResolverDiscordIDs = []int64{999999}

		resolverID := int64(999999)
		groupWagerID := int64(1)
		invalidOptionID := int64(999) // Non-existent option

		// Create test data with enough participants
		groupWager := createTestGroupWager(groupWagerID, models.GroupWagerStateActive, 5000)
		option1 := createTestGroupWagerOption(10, groupWagerID, "Option 1", 0, 3000)
		option2 := createTestGroupWagerOption(20, groupWagerID, "Option 2", 1, 2000)
		
		// Create participants to meet minimum requirements
		participant1 := createTestParticipant(1, groupWagerID, 111111, option1.ID, 2000)
		participant2 := createTestParticipant(2, groupWagerID, 222222, option1.ID, 1000)
		participant3 := createTestParticipant(3, groupWagerID, 333333, option2.ID, 2000)
		
		detail := createTestGroupWagerDetail(groupWager, []*models.GroupWagerOption{option1, option2}, 
			[]*models.GroupWagerParticipant{participant1, participant2, participant3})

		// Setup mocks
		mockGroupWagerRepo.On("GetByID", mock.Anything, groupWagerID).Return(groupWager, nil)
		mockGroupWagerRepo.On("GetDetailByID", mock.Anything, groupWagerID).Return(detail, nil)

		// Execute
		result, err := service.ResolveGroupWager(ctx, groupWagerID, resolverID, invalidOptionID)

		// Verify
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "invalid winning option")

		assertAllMockExpectations(t, mockUserRepo, mockGroupWagerRepo, mockBalanceHistoryRepo, mockEventPublisher)
	})

	t.Run("no participants on winning option", func(t *testing.T) {
		// Setup
		service, mockUserRepo, mockGroupWagerRepo, mockBalanceHistoryRepo, mockEventPublisher := createTestGroupWagerService()
		
		// Set resolver IDs
		service.(*groupWagerService).config.ResolverDiscordIDs = []int64{999999}

		resolverID := int64(999999)
		groupWagerID := int64(1)
		winningOptionID := int64(10)

		// Create test data with all participants on losing option
		groupWager := createTestGroupWager(groupWagerID, models.GroupWagerStateActive, 5000)
		option1 := createTestGroupWagerOption(winningOptionID, groupWagerID, "Winning Option", 0, 0) // No participants
		option2 := createTestGroupWagerOption(20, groupWagerID, "Losing Option", 1, 5000)
		
		// Need at least 3 participants on option2 to meet minimum
		participant1 := createTestParticipant(1, groupWagerID, 111111, option2.ID, 2000)
		participant2 := createTestParticipant(2, groupWagerID, 222222, option2.ID, 2000)
		participant3 := createTestParticipant(3, groupWagerID, 333333, option2.ID, 1000)
		
		detail := createTestGroupWagerDetail(groupWager, []*models.GroupWagerOption{option1, option2}, 
			[]*models.GroupWagerParticipant{participant1, participant2, participant3})

		// Setup mocks
		mockGroupWagerRepo.On("GetByID", mock.Anything, groupWagerID).Return(groupWager, nil)
		mockGroupWagerRepo.On("GetDetailByID", mock.Anything, groupWagerID).Return(detail, nil)

		// Execute
		result, err := service.ResolveGroupWager(ctx, groupWagerID, resolverID, winningOptionID)

		// Verify - this will fail because it needs participants on at least 2 options
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "need participants on at least 2 different options")

		assertAllMockExpectations(t, mockUserRepo, mockGroupWagerRepo, mockBalanceHistoryRepo, mockEventPublisher)
	})

	t.Run("balance update failure rolls back", func(t *testing.T) {
		// Setup
		service, mockUserRepo, mockGroupWagerRepo, mockBalanceHistoryRepo, mockEventPublisher := createTestGroupWagerService()
		
		// Set resolver IDs
		service.(*groupWagerService).config.ResolverDiscordIDs = []int64{999999}

		resolverID := int64(999999)
		groupWagerID := int64(1)
		winningOptionID := int64(10)

		// Create test data with minimum 3 participants
		groupWager := createTestGroupWager(groupWagerID, models.GroupWagerStateActive, 4000)
		option1 := createTestGroupWagerOption(winningOptionID, groupWagerID, "Winning Option", 0, 3000)
		option2 := createTestGroupWagerOption(20, groupWagerID, "Losing Option", 1, 1000)
		
		winner1 := createTestParticipant(1, groupWagerID, 111111, winningOptionID, 2000)
		winner2 := createTestParticipant(2, groupWagerID, 222222, winningOptionID, 1000)
		loser := createTestParticipant(3, groupWagerID, 333333, option2.ID, 1000)
		
		detail := createTestGroupWagerDetail(groupWager, []*models.GroupWagerOption{option1, option2}, 
			[]*models.GroupWagerParticipant{winner1, winner2, loser})

		winnerUser := createTestUser(111111, 10000)

		// Setup mocks
		mockGroupWagerRepo.On("GetByID", mock.Anything, groupWagerID).Return(groupWager, nil)
		mockGroupWagerRepo.On("GetDetailByID", mock.Anything, groupWagerID).Return(detail, nil)
		mockUserRepo.On("GetByDiscordID", mock.Anything, int64(111111)).Return(winnerUser, nil)

		// Mock balance update failure on first winner
		// Winner1 would get 2000/3000 * 4000 = 2666, net win = 666
		mockUserRepo.On("AddBalance", mock.Anything, int64(111111), int64(666)).Return(fmt.Errorf("database error"))

		// Execute
		result, err := service.ResolveGroupWager(ctx, groupWagerID, resolverID, winningOptionID)

		// Verify
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to update winner balance")

		assertAllMockExpectations(t, mockUserRepo, mockGroupWagerRepo, mockBalanceHistoryRepo, mockEventPublisher)
	})
}

func TestGroupWagerService_IsResolver(t *testing.T) {
	t.Run("user is resolver", func(t *testing.T) {
		// Setup
		service, _, _, _, _ := createTestGroupWagerService()
		
		// Set resolver IDs
		service.(*groupWagerService).config.ResolverDiscordIDs = []int64{111111, 222222, 333333}

		// Test
		assert.True(t, service.IsResolver(111111))
		assert.True(t, service.IsResolver(222222))
		assert.True(t, service.IsResolver(333333))
	})

	t.Run("user is not resolver", func(t *testing.T) {
		// Setup
		service, _, _, _, _ := createTestGroupWagerService()
		
		// Set resolver IDs
		service.(*groupWagerService).config.ResolverDiscordIDs = []int64{111111, 222222, 333333}

		// Test
		assert.False(t, service.IsResolver(444444))
		assert.False(t, service.IsResolver(555555))
		assert.False(t, service.IsResolver(0))
	})

	t.Run("empty resolver list", func(t *testing.T) {
		// Setup
		service, _, _, _, _ := createTestGroupWagerService()
		
		// Set empty resolver list
		service.(*groupWagerService).config.ResolverDiscordIDs = []int64{}

		// Test
		assert.False(t, service.IsResolver(111111))
		assert.False(t, service.IsResolver(222222))
	})
}
