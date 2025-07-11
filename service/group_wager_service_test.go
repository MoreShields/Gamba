package service

import (
	"context"
	"testing"
	"time"

	"gambler/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Test utilities

func createTestGroupWagerService() (GroupWagerService, *MockUnitOfWorkFactory, *MockUnitOfWork, *MockUserRepository, *MockGroupWagerRepository) {
	mockUoW := new(MockUnitOfWork)
	mockFactory := new(MockUnitOfWorkFactory)
	mockUserRepo := new(MockUserRepository)
	mockGroupWagerRepo := new(MockGroupWagerRepository)

	mockUoW.SetRepositories(mockUserRepo, nil, nil)
	mockUoW.SetGroupWagerRepository(mockGroupWagerRepo)
	mockFactory.On("Create").Return(mockUoW)

	service := NewGroupWagerService(mockFactory)
	return service, mockFactory, mockUoW, mockUserRepo, mockGroupWagerRepo
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

func setupBasicTransactionMocks(mockUoW *MockUnitOfWork) {
	mockUoW.On("Begin", mock.Anything).Return(nil)
	mockUoW.On("Commit").Return(nil)
	mockUoW.On("Rollback").Return(nil)
}

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
		service, mockFactory, mockUoW, mockUserRepo, mockGroupWagerRepo := createTestGroupWagerService()
		setupBasicTransactionMocks(mockUoW)

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

		assertAllMockExpectations(t, mockFactory, mockUoW, mockUserRepo, mockGroupWagerRepo)
	})

	t.Run("increase existing bet same option", func(t *testing.T) {
		// Setup
		service, mockFactory, mockUoW, mockUserRepo, mockGroupWagerRepo := createTestGroupWagerService()
		setupBasicTransactionMocks(mockUoW)

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

		assertAllMockExpectations(t, mockFactory, mockUoW, mockUserRepo, mockGroupWagerRepo)
	})

	t.Run("change to different option", func(t *testing.T) {
		// Setup
		service, mockFactory, mockUoW, mockUserRepo, mockGroupWagerRepo := createTestGroupWagerService()
		setupBasicTransactionMocks(mockUoW)

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

		assertAllMockExpectations(t, mockFactory, mockUoW, mockUserRepo, mockGroupWagerRepo)
	})

	t.Run("user not found", func(t *testing.T) {
		// Setup
		service, mockFactory, mockUoW, mockUserRepo, mockGroupWagerRepo := createTestGroupWagerService()

		// For error cases, we expect Begin and Rollback but NOT Commit
		mockUoW.On("Begin", mock.Anything).Return(nil)
		mockUoW.On("Rollback").Return(nil)

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

		assertAllMockExpectations(t, mockFactory, mockUoW, mockUserRepo, mockGroupWagerRepo)
	})

	t.Run("group wager not found", func(t *testing.T) {
		// Setup
		service, mockFactory, mockUoW, mockUserRepo, mockGroupWagerRepo := createTestGroupWagerService()

		// For error cases, we expect Begin and Rollback but NOT Commit
		mockUoW.On("Begin", mock.Anything).Return(nil)
		mockUoW.On("Rollback").Return(nil)

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

		assertAllMockExpectations(t, mockFactory, mockUoW, mockUserRepo, mockGroupWagerRepo)
	})

	t.Run("insufficient balance", func(t *testing.T) {
		// Setup
		service, mockFactory, mockUoW, mockUserRepo, mockGroupWagerRepo := createTestGroupWagerService()

		// For error cases, we expect Begin and Rollback but NOT Commit
		mockUoW.On("Begin", mock.Anything).Return(nil)
		mockUoW.On("Rollback").Return(nil)

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

		assertAllMockExpectations(t, mockFactory, mockUoW, mockUserRepo, mockGroupWagerRepo)
	})

	t.Run("wager not active", func(t *testing.T) {
		// Setup
		service, mockFactory, mockUoW, mockUserRepo, mockGroupWagerRepo := createTestGroupWagerService()

		// For error cases, we expect Begin and Rollback but NOT Commit
		mockUoW.On("Begin", mock.Anything).Return(nil)
		mockUoW.On("Rollback").Return(nil)

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

		assertAllMockExpectations(t, mockFactory, mockUoW, mockUserRepo, mockGroupWagerRepo)
	})

	t.Run("voting period expired", func(t *testing.T) {
		// Setup
		service, mockFactory, mockUoW, mockUserRepo, mockGroupWagerRepo := createTestGroupWagerService()

		// For error cases, we expect Begin and Rollback but NOT Commit
		mockUoW.On("Begin", mock.Anything).Return(nil)
		mockUoW.On("Rollback").Return(nil)

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

		assertAllMockExpectations(t, mockFactory, mockUoW, mockUserRepo, mockGroupWagerRepo)
	})

	t.Run("invalid option", func(t *testing.T) {
		// Setup
		service, mockFactory, mockUoW, mockUserRepo, mockGroupWagerRepo := createTestGroupWagerService()

		// For error cases, we expect Begin and Rollback but NOT Commit
		mockUoW.On("Begin", mock.Anything).Return(nil)
		mockUoW.On("Rollback").Return(nil)

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

		assertAllMockExpectations(t, mockFactory, mockUoW, mockUserRepo, mockGroupWagerRepo)
	})
}
