package service

import (
	"testing"
	"time"

	"gambler/discord-client/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
		WagerType:        models.GroupWagerTypePool, // Default to pool for existing tests
		TotalPot:         totalPot,
		MinParticipants:  3,
		VotingEndsAt:     &futureTime,
		MessageID:        123,
		ChannelID:        456,
	}
}

func createTestGroupWagerWithType(groupWagerID int64, state models.GroupWagerState, wagerType models.GroupWagerType, totalPot int64) *models.GroupWager {
	futureTime := time.Now().Add(24 * time.Hour)
	return &models.GroupWager{
		ID:               groupWagerID,
		CreatorDiscordID: 999999,
		Condition:        "Test wager",
		State:            state,
		WagerType:        wagerType,
		TotalPot:         totalPot,
		MinParticipants:  3,
		VotingEndsAt:     &futureTime,
		MessageID:        123,
		ChannelID:        456,
	}
}

func createTestGroupWagerOption(optionID, groupWagerID int64, text string, order int16, totalAmount int64) *models.GroupWagerOption {
	return &models.GroupWagerOption{
		ID:             optionID,
		GroupWagerID:   groupWagerID,
		OptionText:     text,
		OptionOrder:    order,
		TotalAmount:    totalAmount,
		OddsMultiplier: 0, // Default to 0 for existing tests
	}
}

func createTestGroupWagerOptionWithOdds(optionID, groupWagerID int64, text string, order int16, totalAmount int64, oddsMultiplier float64) *models.GroupWagerOption {
	return &models.GroupWagerOption{
		ID:             optionID,
		GroupWagerID:   groupWagerID,
		OptionText:     text,
		OptionOrder:    order,
		TotalAmount:    totalAmount,
		OddsMultiplier: oddsMultiplier,
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
