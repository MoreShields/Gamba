package services

import (
	"context"
	"testing"
	"time"

	"gambler/discord-client/config"
	"gambler/discord-client/domain/entities"
	"gambler/discord-client/domain/events"
	"gambler/discord-client/domain/testhelpers"
	
	"github.com/stretchr/testify/mock"
)

// Test constants for consistent test data
const (
	TestResolverID     = int64(999999)
	TestMessageID      = int64(123456789)
	TestChannelID      = int64(987654321)
	TestGuildID        = int64(555555555)
	TestInitialBalance = int64(100000)
	TestWagerID        = int64(1)
	TestUser1ID        = int64(100)
	TestUser2ID        = int64(200)
	TestUser3ID        = int64(300)
	TestUser4ID        = int64(400)
	TestOption1ID      = int64(1)
	TestOption2ID      = int64(2)
)

// TestMocks aggregates all repository mocks for testing
type TestMocks struct {
	GroupWagerRepo     *testhelpers.MockGroupWagerRepository
	UserRepo           *testhelpers.MockUserRepository
	BalanceHistoryRepo *testhelpers.MockBalanceHistoryRepository
	EventPublisher     *testhelpers.MockEventPublisher
	BetRepo            *testhelpers.MockBetRepository
	WagerRepo          *testhelpers.MockWagerRepository
	WagerVoteRepo      *testhelpers.MockWagerVoteRepository
	GuildSettingsRepo  *testhelpers.MockGuildSettingsRepository
	SummonerWatchRepo  *testhelpers.MockSummonerWatchRepository
}

// NewTestMocks creates a new set of mocks
func NewTestMocks() *TestMocks {
	return &TestMocks{
		GroupWagerRepo:     &testhelpers.MockGroupWagerRepository{},
		UserRepo:           &testhelpers.MockUserRepository{},
		BalanceHistoryRepo: &testhelpers.MockBalanceHistoryRepository{},
		EventPublisher:     &testhelpers.MockEventPublisher{},
		BetRepo:            &testhelpers.MockBetRepository{},
		WagerRepo:          &testhelpers.MockWagerRepository{},
		WagerVoteRepo:      &testhelpers.MockWagerVoteRepository{},
		GuildSettingsRepo:  &testhelpers.MockGuildSettingsRepository{},
		SummonerWatchRepo:  &testhelpers.MockSummonerWatchRepository{},
	}
}

// AssertAllExpectations verifies all mock expectations were met
func (m *TestMocks) AssertAllExpectations(t *testing.T) {
	m.GroupWagerRepo.AssertExpectations(t)
	m.UserRepo.AssertExpectations(t)
	m.BalanceHistoryRepo.AssertExpectations(t)
	m.EventPublisher.AssertExpectations(t)
	m.BetRepo.AssertExpectations(t)
	m.WagerRepo.AssertExpectations(t)
	m.WagerVoteRepo.AssertExpectations(t)
	m.GuildSettingsRepo.AssertExpectations(t)
	m.SummonerWatchRepo.AssertExpectations(t)
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

// ExpectUserLookup sets up user repository mock expectations
func (h *MockHelper) ExpectUserLookup(discordID int64, user *entities.User) {
	h.mocks.UserRepo.On("GetByDiscordID", mock.Anything, discordID).Return(user, nil)
}

// ExpectWagerLookup sets up wager repository mock expectations
func (h *MockHelper) ExpectWagerLookup(wagerID int64, wager *entities.GroupWager) {
	h.mocks.GroupWagerRepo.On("GetByID", mock.Anything, wagerID).Return(wager, nil)
}

// ExpectWagerDetailLookup sets up wager detail repository mock expectations
func (h *MockHelper) ExpectWagerDetailLookup(wagerID int64, detail *entities.GroupWagerDetail) {
	h.mocks.GroupWagerRepo.On("GetDetailByID", mock.Anything, wagerID).Return(detail, nil)
}

// ExpectWagerNotFound sets up wager repository mock to return not found
func (h *MockHelper) ExpectWagerNotFound(wagerID int64) {
	h.mocks.GroupWagerRepo.On("GetByID", mock.Anything, wagerID).Return(nil, nil)
}

// ExpectWagerDetailNotFound sets up wager detail repository mock to return not found
func (h *MockHelper) ExpectWagerDetailNotFound(wagerID int64) {
	h.mocks.GroupWagerRepo.On("GetDetailByID", mock.Anything, wagerID).Return(nil, nil)
}

// ExpectEventPublish sets up event publisher mock expectations
func (h *MockHelper) ExpectEventPublish(eventType events.EventType) {
	h.mocks.EventPublisher.On("Publish", mock.MatchedBy(func(e events.Event) bool {
		return e.Type() == eventType
	})).Return(nil)
}

// ExpectUserNotFound sets up user repository mock to return not found
func (h *MockHelper) ExpectUserNotFound(discordID int64) {
	h.mocks.UserRepo.On("GetByDiscordID", mock.Anything, discordID).Return(nil, nil)
}

// ExpectParticipantLookup sets up group wager repository mock to return a participant
func (h *MockHelper) ExpectParticipantLookup(wagerID, userID int64, participant *entities.GroupWagerParticipant) {
	h.mocks.GroupWagerRepo.On("GetParticipant", mock.Anything, wagerID, userID).Return(participant, nil)
}

// ExpectNewParticipant sets up group wager repository mock to create a new participant
func (h *MockHelper) ExpectNewParticipant(args ...interface{}) {
	if len(args) == 1 {
		// Called with a participant object
		participant := args[0].(*entities.GroupWagerParticipant)
		h.mocks.GroupWagerRepo.On("SaveParticipant", mock.Anything, participant).Return(nil)
	} else if len(args) == 4 {
		// Called with individual parameters (wagerID, userID, optionID, amount)
		h.mocks.GroupWagerRepo.On("SaveParticipant", mock.Anything, mock.MatchedBy(func(p *entities.GroupWagerParticipant) bool {
			return p.GroupWagerID == args[0].(int64) &&
				p.DiscordID == args[1].(int64) &&
				p.OptionID == args[2].(int64) &&
				p.Amount == args[3].(int64)
		})).Return(nil)
	}
}

// ExpectOptionTotalUpdate sets up group wager repository mock to update option totals
func (h *MockHelper) ExpectOptionTotalUpdate(optionID int64, totalAmount int64) {
	h.mocks.GroupWagerRepo.On("UpdateOptionTotal", mock.Anything, optionID, totalAmount).Return(nil)
}

// ExpectBalanceUpdate sets up user repository mock to update balance
func (h *MockHelper) ExpectBalanceUpdate(discordID int64, newBalance int64) {
	h.mocks.UserRepo.On("UpdateBalance", mock.Anything, discordID, newBalance).Return(nil)
}

// ExpectBalanceHistoryRecord sets up balance history repository mock to record history
func (h *MockHelper) ExpectBalanceHistoryRecord(history *entities.BalanceHistory) {
	h.mocks.BalanceHistoryRepo.On("Record", mock.Anything, mock.MatchedBy(func(h *entities.BalanceHistory) bool {
		return h.DiscordID == history.DiscordID &&
			h.BalanceAfter == history.BalanceAfter &&
			h.TransactionType == history.TransactionType
	})).Return(nil)
}

// ExpectBalanceHistoryRecordSimple sets up balance history repository mock with simple parameters
func (h *MockHelper) ExpectBalanceHistoryRecordSimple(discordID int64, balanceAfter int64, transactionType entities.TransactionType) {
	h.mocks.BalanceHistoryRepo.On("Record", mock.Anything, mock.MatchedBy(func(h *entities.BalanceHistory) bool {
		return h.DiscordID == discordID &&
			h.BalanceAfter == balanceAfter &&
			h.TransactionType == transactionType
	})).Return(nil)
}

// GroupWagerScenario defines test scenario data for group wagers
type GroupWagerScenario struct {
	Users        []*entities.User
	Wager        *entities.GroupWager
	Options      []*entities.GroupWagerOption
	Participants []*entities.GroupWagerParticipant
	userMap      map[int64]*entities.User // internal map for quick user lookup
}

// GetUser retrieves a user by discord ID from the scenario
func (s *GroupWagerScenario) GetUser(discordID int64) (*entities.User, bool) {
	if s.userMap == nil {
		s.buildUserMap()
	}
	user, exists := s.userMap[discordID]
	return user, exists
}

// buildUserMap creates the internal user map for quick lookups
func (s *GroupWagerScenario) buildUserMap() {
	s.userMap = make(map[int64]*entities.User)
	for _, user := range s.Users {
		s.userMap[user.DiscordID] = user
	}
}

// getWinners returns participants who bet on the winning option
func getWinners(participants []*entities.GroupWagerParticipant, winningOptionID int64) []*entities.GroupWagerParticipant {
	var winners []*entities.GroupWagerParticipant
	for _, p := range participants {
		if p.OptionID == winningOptionID {
			winners = append(winners, p)
		}
	}
	return winners
}

// getLosers returns participants who did not bet on the winning option
func getLosers(participants []*entities.GroupWagerParticipant, winningOptionID int64) []*entities.GroupWagerParticipant {
	var losers []*entities.GroupWagerParticipant
	for _, p := range participants {
		if p.OptionID != winningOptionID {
			losers = append(losers, p)
		}
	}
	return losers
}

// WithPoolWager modifies the scenario to use a pool wager type
func (s *GroupWagerScenario) WithPoolWager(resolverID int64, condition string) *GroupWagerScenario {
	s.Wager.WagerType = entities.GroupWagerTypePool
	s.Wager.ResolverDiscordID = &resolverID
	s.Wager.Condition = condition
	return s
}

// WithOptions modifies the scenario options
func (s *GroupWagerScenario) WithOptions(options ...interface{}) *GroupWagerScenario {
	s.Options = make([]*entities.GroupWagerOption, 0, len(options))
	for i, opt := range options {
		switch v := opt.(type) {
		case string:
			// Create option from string
			s.Options = append(s.Options, &entities.GroupWagerOption{
				ID:             int64(i + 1),
				GroupWagerID:   s.Wager.ID,
				OptionText:     v,
				OptionOrder:    int16(i + 1),
				TotalAmount:    0,
				OddsMultiplier: 2.0,
			})
		case *entities.GroupWagerOption:
			s.Options = append(s.Options, v)
		}
	}
	return s
}

// WithUser adds or modifies a user in the scenario
func (s *GroupWagerScenario) WithUser(discordID int64, username string, balance int64) *GroupWagerScenario {
	// Check if user already exists
	for i, user := range s.Users {
		if user.DiscordID == discordID {
			s.Users[i].Username = username
			s.Users[i].Balance = balance
			s.Users[i].AvailableBalance = balance // Set available balance too
			return s
		}
	}
	// Add new user
	s.Users = append(s.Users, &entities.User{
		DiscordID:        discordID,
		Username:         username,
		Balance:          balance,
		AvailableBalance: balance, // Set available balance too
	})
	return s
}

// Build returns the scenario (for builder pattern compatibility)
func (s *GroupWagerScenario) Build() *GroupWagerScenario {
	return s
}

// GroupWagerScenarioBuilder is an alias for GroupWagerScenario for builder pattern
type GroupWagerScenarioBuilder = GroupWagerScenario

// WithHouseWager modifies the scenario to use a house wager type with resolver
func (s *GroupWagerScenario) WithHouseWager(resolverID int64, condition string) *GroupWagerScenario {
	s.Wager.WagerType = entities.GroupWagerTypeHouse
	s.Wager.ResolverDiscordID = &resolverID
	s.Wager.Condition = condition
	return s
}

// WithOdds sets odds multipliers for the options
func (s *GroupWagerScenario) WithOdds(odds ...float64) *GroupWagerScenario {
	for i, odd := range odds {
		if i < len(s.Options) {
			s.Options[i].OddsMultiplier = odd
		}
	}
	return s
}

// WithParticipant adds a participant to the scenario
func (s *GroupWagerScenario) WithParticipant(userID int64, optionIndex int, amount int64) *GroupWagerScenario {
	if optionIndex >= len(s.Options) {
		return s
	}

	// Check if participant already exists
	for i, p := range s.Participants {
		if p.DiscordID == userID {
			// Update existing participant - adjust option totals
			oldOptionID := s.Participants[i].OptionID
			oldAmount := s.Participants[i].Amount

			// Find and update old option total
			for _, opt := range s.Options {
				if opt.ID == oldOptionID {
					opt.TotalAmount -= oldAmount
					break
				}
			}

			// Update participant
			s.Participants[i].OptionID = s.Options[optionIndex].ID
			s.Participants[i].Amount = amount

			// Update new option total
			s.Options[optionIndex].TotalAmount += amount
			return s
		}
	}

	// Add new participant
	s.Participants = append(s.Participants, &entities.GroupWagerParticipant{
		GroupWagerID: s.Wager.ID,
		DiscordID:    userID,
		OptionID:     s.Options[optionIndex].ID,
		Amount:       amount,
	})

	// Update option total for new participant
	s.Options[optionIndex].TotalAmount += amount

	// Update wager total pot
	s.Wager.TotalPot += amount

	return s
}

// SetupTestConfig configures the test environment
func SetupTestConfig(t *testing.T) {
	config.SetTestConfig(config.NewTestConfig())
}

// NewGroupWagerScenario creates a new test scenario with default data
func NewGroupWagerScenario() *GroupWagerScenario {
	creatorID := TestUser1ID
	votingEndsAt := time.Now().Add(1 * time.Hour) // Default to 1 hour from now
	return &GroupWagerScenario{
		Users: []*entities.User{
			{
				DiscordID:        TestUser1ID,
				Username:         "user1",
				Balance:          TestInitialBalance,
				AvailableBalance: TestInitialBalance,
			},
			{
				DiscordID:        TestUser2ID,
				Username:         "user2",
				Balance:          TestInitialBalance,
				AvailableBalance: TestInitialBalance,
			},
		},
		Wager: &entities.GroupWager{
			ID:               TestWagerID,
			GuildID:          123,
			Condition:        "Test Wager",
			State:            entities.GroupWagerStateActive,
			WagerType:        entities.GroupWagerTypeHouse,
			MinParticipants:  2,
			CreatorDiscordID: &creatorID,
			TotalPot:         0,
			VotingEndsAt:     &votingEndsAt,
		},
		Options: []*entities.GroupWagerOption{
			{
				ID:             TestOption1ID,
				GroupWagerID:   TestWagerID,
				OptionText:     "Option 1",
				OptionOrder:    1,
				TotalAmount:    0,
				OddsMultiplier: 2.0,
			},
			{
				ID:             TestOption2ID,
				GroupWagerID:   TestWagerID,
				OptionText:     "Option 2",
				OptionOrder:    2,
				TotalAmount:    0,
				OddsMultiplier: 2.0,
			},
		},
		Participants: []*entities.GroupWagerParticipant{},
	}
}
