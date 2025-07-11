package service

import (
	"context"
	"testing"
	"time"

	"gambler/config"
	"gambler/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGroupWagerService_PlaceBet_ChangingOptionUpdatesTotalsCorrectly(t *testing.T) {
	ctx := context.Background()

	// Setup mocks
	mockUoW := new(MockUnitOfWork)
	mockFactory := new(MockUnitOfWorkFactory)
	mockUserRepo := new(MockUserRepository)
	mockGroupWagerRepo := new(MockGroupWagerRepository)

	// Configure unit of work
	mockUoW.SetRepositories(mockUserRepo, nil, nil)
	mockUoW.SetGroupWagerRepository(mockGroupWagerRepo)

	service := NewGroupWagerService(mockFactory, &config.Config{ResolverDiscordIDs: []int64{}})

	// Test data
	userID := int64(123456)
	groupWagerID := int64(1)
	option1ID := int64(10)
	option2ID := int64(20)
	initialBetAmount := int64(1000)
	newBetAmount := int64(2000)

	// Create test user
	testUser := &models.User{
		DiscordID:        userID,
		Username:         "testuser",
		Balance:          10000,
		AvailableBalance: 10000,
	}

	// Create test group wager
	futureTime := time.Now().Add(24 * time.Hour) // Set voting to end 24 hours from now
	groupWager := &models.GroupWager{
		ID:               groupWagerID,
		CreatorDiscordID: 999999,
		Condition:        "Test wager",
		State:            models.GroupWagerStateActive,
		TotalPot:         initialBetAmount, // Already has initial bet
		MinParticipants:  3,
		VotingEndsAt:     &futureTime,
		MessageID:        123,
		ChannelID:        456,
	}

	// Create test options
	option1 := &models.GroupWagerOption{
		ID:           option1ID,
		GroupWagerID: groupWagerID,
		OptionText:   "Option 1",
		OptionOrder:  0,
		TotalAmount:  initialBetAmount, // User's initial bet is on option 1
	}
	option2 := &models.GroupWagerOption{
		ID:           option2ID,
		GroupWagerID: groupWagerID,
		OptionText:   "Option 2",
		OptionOrder:  1,
		TotalAmount:  0, // No bets on option 2 yet
	}
	options := []*models.GroupWagerOption{option1, option2}

	// Existing participant (bet on option 1)
	existingParticipant := &models.GroupWagerParticipant{
		ID:           1,
		GroupWagerID: groupWagerID,
		DiscordID:    userID,
		OptionID:     option1ID,
		Amount:       initialBetAmount,
	}

	// Mock expectations for changing bet from option1 to option2
	mockFactory.On("Create").Return(mockUoW)
	mockUoW.On("Begin", ctx).Return(nil)
	mockUoW.On("Commit").Return(nil)
	mockUoW.On("Rollback").Return(nil)

	// Get group wager
	mockGroupWagerRepo.On("GetByID", ctx, groupWagerID).Return(groupWager, nil)

	// Get full detail (includes options and participants)
	detail := &models.GroupWagerDetail{
		Wager:        groupWager,
		Options:      options,
		Participants: []*models.GroupWagerParticipant{existingParticipant},
	}
	mockGroupWagerRepo.On("GetDetailByID", ctx, groupWagerID).Return(detail, nil)

	// Get user
	mockUserRepo.On("GetByDiscordID", ctx, userID).Return(testUser, nil)

	// Get existing participant
	mockGroupWagerRepo.On("GetParticipant", ctx, groupWagerID, userID).Return(existingParticipant, nil)

	// Update participant to new option and amount
	mockGroupWagerRepo.On("SaveParticipant", ctx, mock.MatchedBy(func(p *models.GroupWagerParticipant) bool {
		return p.ID == existingParticipant.ID &&
			p.OptionID == option2ID &&
			p.Amount == newBetAmount
	})).Return(nil)

	// Update option 1 total (removing the old bet)
	mockGroupWagerRepo.On("UpdateOptionTotal", ctx, option1ID, int64(0)).Return(nil).Once()

	// CORRECT BEHAVIOR: Option 2 should have the full new amount (2000)
	// This was fixed - when changing options, the new option gets the full amount
	mockGroupWagerRepo.On("UpdateOptionTotal", ctx, option2ID, newBetAmount).Return(nil).Once()

	// Update group wager total pot
	mockGroupWagerRepo.On("Update", ctx, mock.MatchedBy(func(gw *models.GroupWager) bool {
		return gw.ID == groupWagerID && gw.TotalPot == 2000 // 1000 + 1000 net change
	})).Return(nil)

	// Execute the bet change
	participant, err := service.PlaceBet(ctx, groupWagerID, userID, option2ID, newBetAmount)

	// Verify results
	assert.NoError(t, err)
	assert.NotNil(t, participant)
	assert.Equal(t, option2ID, participant.OptionID)
	assert.Equal(t, newBetAmount, participant.Amount)

	// Verify all expectations were met
	mockFactory.AssertExpectations(t)
	mockUoW.AssertExpectations(t)
	mockUserRepo.AssertExpectations(t)
	mockGroupWagerRepo.AssertExpectations(t)
}

func TestGroupWagerService_PlaceBet_SameOptionUpdatesTotalsCorrectly(t *testing.T) {
	ctx := context.Background()

	// Setup mocks
	mockUoW := new(MockUnitOfWork)
	mockFactory := new(MockUnitOfWorkFactory)
	mockUserRepo := new(MockUserRepository)
	mockGroupWagerRepo := new(MockGroupWagerRepository)

	// Configure unit of work
	mockUoW.SetRepositories(mockUserRepo, nil, nil)
	mockUoW.SetGroupWagerRepository(mockGroupWagerRepo)

	service := NewGroupWagerService(mockFactory, &config.Config{ResolverDiscordIDs: []int64{}})

	// Test data
	userID := int64(123456)
	groupWagerID := int64(1)
	optionID := int64(10)
	initialBetAmount := int64(1000)
	newBetAmount := int64(3000) // Increasing bet on same option

	// Create test user
	testUser := &models.User{
		DiscordID:        userID,
		Username:         "testuser",
		Balance:          10000,
		AvailableBalance: 10000,
	}

	// Create test group wager
	futureTime2 := time.Now().Add(24 * time.Hour) // Set voting to end 24 hours from now
	groupWager := &models.GroupWager{
		ID:               groupWagerID,
		CreatorDiscordID: 999999,
		Condition:        "Test wager",
		State:            models.GroupWagerStateActive,
		TotalPot:         initialBetAmount,
		MinParticipants:  3,
		VotingEndsAt:     &futureTime2,
		MessageID:        123,
		ChannelID:        456,
	}

	// Create test option
	option := &models.GroupWagerOption{
		ID:           optionID,
		GroupWagerID: groupWagerID,
		OptionText:   "Option 1",
		OptionOrder:  0,
		TotalAmount:  initialBetAmount,
	}
	options := []*models.GroupWagerOption{option}

	// Existing participant (bet on same option)
	existingParticipant := &models.GroupWagerParticipant{
		ID:           1,
		GroupWagerID: groupWagerID,
		DiscordID:    userID,
		OptionID:     optionID,
		Amount:       initialBetAmount,
	}

	// Mock expectations for increasing bet on same option
	mockFactory.On("Create").Return(mockUoW)
	mockUoW.On("Begin", ctx).Return(nil)
	mockUoW.On("Commit").Return(nil)
	mockUoW.On("Rollback").Return(nil)

	// Get group wager
	mockGroupWagerRepo.On("GetByID", ctx, groupWagerID).Return(groupWager, nil)

	// Get full detail (includes options and participants)
	detail := &models.GroupWagerDetail{
		Wager:        groupWager,
		Options:      options,
		Participants: []*models.GroupWagerParticipant{existingParticipant},
	}
	mockGroupWagerRepo.On("GetDetailByID", ctx, groupWagerID).Return(detail, nil)

	// Get user
	mockUserRepo.On("GetByDiscordID", ctx, userID).Return(testUser, nil)

	// Get existing participant
	mockGroupWagerRepo.On("GetParticipant", ctx, groupWagerID, userID).Return(existingParticipant, nil)

	// Update participant amount
	mockGroupWagerRepo.On("SaveParticipant", ctx, mock.MatchedBy(func(p *models.GroupWagerParticipant) bool {
		return p.ID == existingParticipant.ID &&
			p.OptionID == optionID &&
			p.Amount == newBetAmount
	})).Return(nil)

	// Update option total by the net change
	// Total should increase by 2000 (3000 - 1000)
	mockGroupWagerRepo.On("UpdateOptionTotal", ctx, optionID, int64(3000)).Return(nil).Once()

	// Update group wager total pot
	mockGroupWagerRepo.On("Update", ctx, mock.MatchedBy(func(gw *models.GroupWager) bool {
		return gw.ID == groupWagerID && gw.TotalPot == 3000 // 1000 + 2000 net change
	})).Return(nil)

	// Execute the bet increase
	participant, err := service.PlaceBet(ctx, groupWagerID, userID, optionID, newBetAmount)

	// Verify results
	assert.NoError(t, err)
	assert.NotNil(t, participant)
	assert.Equal(t, optionID, participant.OptionID)
	assert.Equal(t, newBetAmount, participant.Amount)

	// Verify all expectations were met
	mockFactory.AssertExpectations(t)
	mockUoW.AssertExpectations(t)
	mockUserRepo.AssertExpectations(t)
	mockGroupWagerRepo.AssertExpectations(t)
}