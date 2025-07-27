package service

import (
	"context"
	"testing"
	"time"

	"gambler/discord-client/config"
	"gambler/discord-client/models"

	"github.com/stretchr/testify/mock"
)

// Test IDs - Using meaningful constants instead of magic numbers
const (
	TestUser1ID        = 111111
	TestUser2ID        = 222222
	TestUser3ID        = 333333
	TestUser4ID        = 444444
	TestResolverID     = 999999
	TestWagerID        = 1
	TestOptionYesID    = 10
	TestOptionNoID     = 20
	TestOption1ID      = 10
	TestOption2ID      = 20
	TestOption3ID      = 30
	TestInitialBalance = 100000
	TestMessageID      = 123456
	TestChannelID      = 789012
)

// TestMocks holds all mock repositories for easy access
type TestMocks struct {
	UserRepo           *MockUserRepository
	GroupWagerRepo     *MockGroupWagerRepository
	BalanceHistoryRepo *MockBalanceHistoryRepository
	EventPublisher     *MockEventPublisher
}

// NewTestMocks creates a new set of mocks
func NewTestMocks() *TestMocks {
	return &TestMocks{
		UserRepo:           new(MockUserRepository),
		GroupWagerRepo:     new(MockGroupWagerRepository),
		BalanceHistoryRepo: new(MockBalanceHistoryRepository),
		EventPublisher:     new(MockEventPublisher),
	}
}

// AssertAllExpectations asserts all mock expectations
func (m *TestMocks) AssertAllExpectations(t *testing.T) {
	m.UserRepo.AssertExpectations(t)
	m.GroupWagerRepo.AssertExpectations(t)
	m.BalanceHistoryRepo.AssertExpectations(t)
	m.EventPublisher.AssertExpectations(t)
}

// GroupWagerScenario represents a complete test scenario
type GroupWagerScenario struct {
	Wager        *models.GroupWager
	Options      []*models.GroupWagerOption
	Participants []*models.GroupWagerParticipant
	Users        map[int64]*models.User
}

// GroupWagerScenarioBuilder builds test scenarios fluently
type GroupWagerScenarioBuilder struct {
	scenario *GroupWagerScenario
}

// NewGroupWagerScenario creates a new scenario builder
func NewGroupWagerScenario() *GroupWagerScenarioBuilder {
	return &GroupWagerScenarioBuilder{
		scenario: &GroupWagerScenario{
			Users: make(map[int64]*models.User),
		},
	}
}

// WithPoolWager sets up a pool wager
func (b *GroupWagerScenarioBuilder) WithPoolWager(creatorID int64, condition string) *GroupWagerScenarioBuilder {
	now := time.Now()
	votingEndsAt := now.Add(time.Hour) // 1 hour from now
	
	b.scenario.Wager = &models.GroupWager{
		ID:                  TestWagerID,
		CreatorDiscordID:    &creatorID,
		Condition:           condition,
		State:               models.GroupWagerStateActive,
		WagerType:           models.GroupWagerTypePool,
		TotalPot:            0,
		MinParticipants:     3,
		VotingPeriodMinutes: 60,
		VotingStartsAt:      &now,
		VotingEndsAt:        &votingEndsAt,
		MessageID:           TestMessageID,
		ChannelID:           TestChannelID,
	}
	return b
}

// WithHouseWager sets up a house wager
func (b *GroupWagerScenarioBuilder) WithHouseWager(creatorID int64, condition string) *GroupWagerScenarioBuilder {
	now := time.Now()
	votingEndsAt := now.Add(time.Hour) // 1 hour from now
	
	b.scenario.Wager = &models.GroupWager{
		ID:                  TestWagerID,
		CreatorDiscordID:    &creatorID,
		Condition:           condition,
		State:               models.GroupWagerStateActive,
		WagerType:           models.GroupWagerTypeHouse,
		TotalPot:            0,
		MinParticipants:     3,
		VotingPeriodMinutes: 60,
		VotingStartsAt:      &now,
		VotingEndsAt:        &votingEndsAt,
		MessageID:           TestMessageID,
		ChannelID:           TestChannelID,
	}
	return b
}

// WithOptions adds options to the wager
func (b *GroupWagerScenarioBuilder) WithOptions(optionTexts ...string) *GroupWagerScenarioBuilder {
	b.scenario.Options = make([]*models.GroupWagerOption, len(optionTexts))
	for i, text := range optionTexts {
		optionID := int64(10 + i*10) // 10, 20, 30, etc.
		b.scenario.Options[i] = &models.GroupWagerOption{
			ID:             optionID,
			GroupWagerID:   TestWagerID,
			OptionText:     text,
			OptionOrder:    int16(i),
			TotalAmount:    0,
			OddsMultiplier: 0,
		}
	}
	return b
}

// WithOdds sets odds multipliers for house wagers
func (b *GroupWagerScenarioBuilder) WithOdds(odds ...float64) *GroupWagerScenarioBuilder {
	if len(odds) != len(b.scenario.Options) {
		panic("odds count must match options count")
	}
	for i, multiplier := range odds {
		b.scenario.Options[i].OddsMultiplier = multiplier
	}
	return b
}

// WithParticipant adds a participant
func (b *GroupWagerScenarioBuilder) WithParticipant(userID int64, optionIndex int, amount int64) *GroupWagerScenarioBuilder {
	if optionIndex >= len(b.scenario.Options) {
		panic("invalid option index")
	}
	
	participant := &models.GroupWagerParticipant{
		ID:           int64(len(b.scenario.Participants) + 1),
		GroupWagerID: TestWagerID,
		DiscordID:    userID,
		OptionID:     b.scenario.Options[optionIndex].ID,
		Amount:       amount,
	}
	
	b.scenario.Participants = append(b.scenario.Participants, participant)
	
	// Update option total
	b.scenario.Options[optionIndex].TotalAmount += amount
	
	// Update wager total pot
	b.scenario.Wager.TotalPot += amount
	
	return b
}

// WithUser adds a user to the scenario
func (b *GroupWagerScenarioBuilder) WithUser(userID int64, username string, balance int64) *GroupWagerScenarioBuilder {
	b.scenario.Users[userID] = &models.User{
		DiscordID:        userID,
		Username:         username,
		Balance:          balance,
		AvailableBalance: balance,
	}
	return b
}

// Build returns the complete scenario
func (b *GroupWagerScenarioBuilder) Build() *GroupWagerScenario {
	// Calculate odds for pool wagers if not set
	if b.scenario.Wager != nil && b.scenario.Wager.IsPoolWager() && b.scenario.Wager.TotalPot > 0 {
		for _, option := range b.scenario.Options {
			if option.TotalAmount > 0 {
				option.OddsMultiplier = float64(b.scenario.Wager.TotalPot) / float64(option.TotalAmount)
			}
		}
	}
	return b.scenario
}

// MockHelper provides common mock setup patterns
type MockHelper struct {
	mocks *TestMocks
	ctx   context.Context
}

// NewMockHelper creates a new mock helper
func NewMockHelper(mocks *TestMocks) *MockHelper {
	return &MockHelper{
		mocks: mocks,
		ctx:   context.Background(),
	}
}

// ExpectUserLookup sets up user lookup expectations
func (h *MockHelper) ExpectUserLookup(userID int64, user *models.User) {
	h.mocks.UserRepo.On("GetByDiscordID", h.ctx, userID).Return(user, nil)
}

// ExpectUserNotFound sets up user not found expectations
func (h *MockHelper) ExpectUserNotFound(userID int64) {
	h.mocks.UserRepo.On("GetByDiscordID", h.ctx, userID).Return(nil, nil)
}

// ExpectWagerLookup sets up wager lookup expectations
// This now mocks GetDetailByID since all service methods use GetDetailByID
func (h *MockHelper) ExpectWagerLookup(wagerID int64, wager *models.GroupWager) {
	detail := &models.GroupWagerDetail{
		Wager:        wager,
		Options:      []*models.GroupWagerOption{},
		Participants: []*models.GroupWagerParticipant{},
	}
	h.mocks.GroupWagerRepo.On("GetDetailByID", h.ctx, wagerID).Return(detail, nil)
}

// ExpectWagerDetailLookup sets up wager detail lookup expectations
func (h *MockHelper) ExpectWagerDetailLookup(wagerID int64, detail *models.GroupWagerDetail) {
	h.mocks.GroupWagerRepo.On("GetDetailByID", h.ctx, wagerID).Return(detail, nil)
}

// ExpectParticipantLookup sets up participant lookup expectations
func (h *MockHelper) ExpectParticipantLookup(wagerID, userID int64, participant *models.GroupWagerParticipant) {
	h.mocks.GroupWagerRepo.On("GetParticipant", h.ctx, wagerID, userID).Return(participant, nil)
}

// ExpectNewParticipant sets up expectations for creating a new participant
func (h *MockHelper) ExpectNewParticipant(wagerID, userID, optionID, amount int64) {
	h.mocks.GroupWagerRepo.On("SaveParticipant", h.ctx, mock.MatchedBy(func(p *models.GroupWagerParticipant) bool {
		return p.GroupWagerID == wagerID &&
			p.DiscordID == userID &&
			p.OptionID == optionID &&
			p.Amount == amount
	})).Return(nil)
}

// ExpectBalanceUpdate sets up balance update expectations
func (h *MockHelper) ExpectBalanceUpdate(userID, newBalance int64) {
	h.mocks.UserRepo.On("UpdateBalance", h.ctx, userID, newBalance).Return(nil)
}

// ExpectOptionTotalUpdate sets up option total update expectations
func (h *MockHelper) ExpectOptionTotalUpdate(optionID, totalAmount int64) {
	h.mocks.GroupWagerRepo.On("UpdateOptionTotal", h.ctx, optionID, totalAmount).Return(nil)
}

// ExpectWagerUpdate sets up wager update expectations
func (h *MockHelper) ExpectWagerUpdate(wager *models.GroupWager) {
	h.mocks.GroupWagerRepo.On("Update", h.ctx, mock.MatchedBy(func(gw *models.GroupWager) bool {
		return gw.ID == wager.ID
	})).Return(nil)
}

// ExpectOddsRecalculation sets up odds recalculation expectations for pool wagers
func (h *MockHelper) ExpectOddsRecalculation(wagerID int64, options []*models.GroupWagerOption) {
	oddsMap := make(map[int64]float64)
	for _, opt := range options {
		oddsMap[opt.ID] = opt.OddsMultiplier
	}
	
	h.mocks.GroupWagerRepo.On("UpdateAllOptionOdds", h.ctx, wagerID, mock.MatchedBy(func(odds map[int64]float64) bool {
		if len(odds) != len(oddsMap) {
			return false
		}
		for id, expectedOdds := range oddsMap {
			if odds[id] != expectedOdds {
				return false
			}
		}
		return true
	})).Return(nil)
}

// ExpectBalanceHistoryRecord sets up balance history recording expectations
func (h *MockHelper) ExpectBalanceHistoryRecord(userID int64, changeAmount int64, transactionType models.TransactionType) {
	h.mocks.BalanceHistoryRepo.On("Record", h.ctx, mock.MatchedBy(func(h *models.BalanceHistory) bool {
		return h.DiscordID == userID &&
			h.ChangeAmount == changeAmount &&
			h.TransactionType == transactionType
	})).Return(nil)
}

// ExpectEventPublish sets up event publishing expectations
func (h *MockHelper) ExpectEventPublish(eventType string) {
	h.mocks.EventPublisher.On("Publish", mock.AnythingOfType(eventType)).Return()
}

// ExpectSuccessfulResolution sets up all mocks for a successful wager resolution
func (h *MockHelper) ExpectSuccessfulResolution(scenario *GroupWagerScenario, winningOptionID int64, wagerType models.GroupWagerType) {
	// Basic lookups
	h.ExpectWagerLookup(scenario.Wager.ID, scenario.Wager)
	h.ExpectWagerDetailLookup(scenario.Wager.ID, &models.GroupWagerDetail{
		Wager:        scenario.Wager,
		Options:      scenario.Options,
		Participants: scenario.Participants,
	})
	
	// User lookups for all participants
	for _, participant := range scenario.Participants {
		if user, exists := scenario.Users[participant.DiscordID]; exists {
			h.ExpectUserLookup(participant.DiscordID, user)
		}
	}
	
	// Setup balance changes
	winners := getWinners(scenario.Participants, winningOptionID)
	losers := getLosers(scenario.Participants, winningOptionID)
	h.ExpectResolutionBalanceChanges(scenario, winners, losers, winningOptionID, wagerType)
	
	// Update participant payouts
	h.mocks.GroupWagerRepo.On("UpdateParticipantPayouts", h.ctx, mock.Anything).Return(nil)
	
	// Update wager state
	h.mocks.GroupWagerRepo.On("Update", h.ctx, mock.MatchedBy(func(gw *models.GroupWager) bool {
		return gw.ID == scenario.Wager.ID && gw.State == models.GroupWagerStateResolved
	})).Return(nil)
	
	// State change event
	h.ExpectEventPublish("events.GroupWagerStateChangeEvent")
}

// ExpectResolutionBalanceChanges sets up balance change expectations for resolution
func (h *MockHelper) ExpectResolutionBalanceChanges(scenario *GroupWagerScenario, winners, losers []*models.GroupWagerParticipant, winningOptionID int64, wagerType models.GroupWagerType) {
	// Find winning option
	var winningOption *models.GroupWagerOption
	for _, opt := range scenario.Options {
		if opt.ID == winningOptionID {
			winningOption = opt
			break
		}
	}
	
	// Setup winner balance updates
	for _, winner := range winners {
		user := scenario.Users[winner.DiscordID]
		var payout int64
		var balanceChange int64
		
		if wagerType == models.GroupWagerTypePool {
			payout = winner.CalculatePayout(winningOption.TotalAmount, scenario.Wager.TotalPot)
		} else {
			payout = int64(float64(winner.Amount) * winningOption.OddsMultiplier)
		}
		
		balanceChange = payout - winner.Amount
		newBalance := user.Balance + balanceChange
		
		h.ExpectBalanceUpdate(winner.DiscordID, newBalance)
		h.ExpectBalanceHistoryRecord(winner.DiscordID, balanceChange, models.TransactionTypeGroupWagerWin)
		h.ExpectEventPublish("events.BalanceChangeEvent")
	}
	
	// Setup loser balance updates
	for _, loser := range losers {
		user := scenario.Users[loser.DiscordID]
		balanceChange := -loser.Amount
		newBalance := user.Balance + balanceChange
		
		h.ExpectBalanceUpdate(loser.DiscordID, newBalance)
		h.ExpectBalanceHistoryRecord(loser.DiscordID, balanceChange, models.TransactionTypeGroupWagerLoss)
		h.ExpectEventPublish("events.BalanceChangeEvent")
	}
}

// ExpectSuccessfulBet sets up all mocks for a successful bet placement
func (h *MockHelper) ExpectSuccessfulBet(wagerID, userID, optionID, amount int64, scenario *GroupWagerScenario) {
	// Basic validations
	h.ExpectBetValidation(wagerID, scenario.Wager, scenario.Users[userID])
	h.ExpectWagerDetailLookup(wagerID, &models.GroupWagerDetail{
		Wager:        scenario.Wager,
		Options:      scenario.Options,
		Participants: scenario.Participants,
	})
	
	// Check for existing participant
	var existingParticipant *models.GroupWagerParticipant
	for _, p := range scenario.Participants {
		if p.DiscordID == userID {
			existingParticipant = p
			break
		}
	}
	h.ExpectParticipantLookup(wagerID, userID, existingParticipant)
	
	// Save participant
	if existingParticipant == nil {
		h.ExpectNewParticipant(wagerID, userID, optionID, amount)
	} else {
		h.mocks.GroupWagerRepo.On("SaveParticipant", h.ctx, mock.MatchedBy(func(p *models.GroupWagerParticipant) bool {
			return p.ID == existingParticipant.ID && p.OptionID == optionID && p.Amount == amount
		})).Return(nil)
	}
	
	// Update option total
	var option *models.GroupWagerOption
	for _, opt := range scenario.Options {
		if opt.ID == optionID {
			option = opt
			break
		}
	}
	newTotal := option.TotalAmount + amount
	if existingParticipant != nil && existingParticipant.OptionID != optionID {
		newTotal = amount // Switching options
	}
	h.ExpectOptionTotalUpdate(optionID, newTotal)
	
	// Update wager pot
	h.mocks.GroupWagerRepo.On("Update", h.ctx, mock.MatchedBy(func(gw *models.GroupWager) bool {
		return gw.ID == wagerID
	})).Return(nil)
	
	// Pool wager odds recalculation
	if scenario.Wager.IsPoolWager() {
		h.ExpectOddsRecalculation(wagerID, scenario.Options)
	}
}

// ExpectBetValidation sets up mocks for bet validation checks
func (h *MockHelper) ExpectBetValidation(wagerID int64, wager *models.GroupWager, user *models.User) {
	h.ExpectWagerLookup(wagerID, wager)
	h.ExpectUserLookup(user.DiscordID, user)
}

// ExpectWagerNotFound sets up mocks for wager not found scenario
func (h *MockHelper) ExpectWagerNotFound(wagerID int64) {
	h.mocks.GroupWagerRepo.On("GetDetailByID", h.ctx, wagerID).Return(nil, nil)
}

// ExpectInsufficientBalance sets up mocks for insufficient balance scenario
func (h *MockHelper) ExpectInsufficientBalance(userID int64, user *models.User) {
	h.ExpectUserLookup(userID, user)
}

// Helper functions to get winners and losers
func getWinners(participants []*models.GroupWagerParticipant, winningOptionID int64) []*models.GroupWagerParticipant {
	var winners []*models.GroupWagerParticipant
	for _, p := range participants {
		if p.OptionID == winningOptionID {
			winners = append(winners, p)
		}
	}
	return winners
}

func getLosers(participants []*models.GroupWagerParticipant, winningOptionID int64) []*models.GroupWagerParticipant {
	var losers []*models.GroupWagerParticipant
	for _, p := range participants {
		if p.OptionID != winningOptionID {
			losers = append(losers, p)
		}
	}
	return losers
}

// SetupTestConfig initializes a test configuration for the current test
// This should be called at the beginning of every test that uses services
func SetupTestConfig(t *testing.T) {
	// Set up test config
	testConfig := config.NewTestConfig()
	config.SetTestConfig(testConfig)
	
	// Clean up after test
	t.Cleanup(func() {
		config.ResetConfig()
	})
}