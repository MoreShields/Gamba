package service

import (
	"errors"
	"testing"

	"gambler/discord-client/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestGroupWagerService_EdgeCases tests common error scenarios and edge cases
func TestGroupWagerService_EdgeCases(t *testing.T) {
	fixture := NewGroupWagerTestFixture(t)

	t.Run("wager not found errors", func(t *testing.T) {
		fixture.Reset()

		// Test different operations when wager doesn't exist
		testCases := []struct {
			name      string
			operation func() error
		}{
			{
				name: "place bet on non-existent wager",
				operation: func() error {
					fixture.Helper.ExpectWagerNotFound(TestWagerID)
					_, err := fixture.Service.PlaceBet(fixture.Ctx, TestWagerID, TestUser1ID, TestOption1ID, 1000)
					return err
				},
			},
			{
				name: "resolve non-existent wager",
				operation: func() error {
					fixture.Helper.ExpectWagerNotFound(TestWagerID)
					resolverID := int64(TestResolverID)
					_, err := fixture.Service.ResolveGroupWager(fixture.Ctx, TestWagerID, &resolverID, TestOption1ID)
					return err
				},
			},
			{
				name: "cancel non-existent wager",
				operation: func() error {
					fixture.Helper.ExpectWagerNotFound(TestWagerID)
					creatorID := int64(TestUser1ID)
					return fixture.Service.CancelGroupWager(fixture.Ctx, TestWagerID, &creatorID)
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				fixture.Reset()
				err := tc.operation()
				fixture.Assertions.AssertValidationError(err, "group wager not found")
				fixture.AssertAllMocks()
			})
		}
	})

	t.Run("database connection failures", func(t *testing.T) {
		fixture.Reset()

		// Test handling of database connection errors
		fixture.Mocks.GroupWagerRepo.On("GetByID", fixture.Ctx, int64(TestWagerID)).Return(nil, errors.New("connection failed"))

		_, err := fixture.Service.PlaceBet(fixture.Ctx, TestWagerID, TestUser1ID, TestOption1ID, 1000)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get group wager")
		fixture.AssertAllMocks()
	})

	t.Run("user not found during operations", func(t *testing.T) {
		fixture.Reset()

		// Setup valid wager but user doesn't exist
		scenario := NewGroupWagerScenario().
			WithPoolWager(TestResolverID, "Test wager").
			WithOptions("Yes", "No").
			Build()

		fixture.Helper.ExpectWagerLookup(TestWagerID, scenario.Wager)
		fixture.Helper.ExpectWagerDetailLookup(TestWagerID, &models.GroupWagerDetail{
			Wager:        scenario.Wager,
			Options:      scenario.Options,
			Participants: scenario.Participants,
		})
		fixture.Helper.ExpectUserNotFound(TestUser1ID)

		_, err := fixture.Service.PlaceBet(fixture.Ctx, TestWagerID, TestUser1ID, TestOption1ID, 1000)

		fixture.Assertions.AssertValidationError(err, "user 111111 not found")
		fixture.AssertAllMocks()
	})

	t.Run("invalid option ID validation", func(t *testing.T) {
		fixture.Reset()

		scenario := NewGroupWagerScenario().
			WithPoolWager(TestResolverID, "Test wager").
			WithOptions("Option A", "Option B").
			WithUser(TestUser1ID, "user1", TestInitialBalance).
			Build()

		fixture.Helper.ExpectWagerLookup(TestWagerID, scenario.Wager)
		fixture.Helper.ExpectWagerDetailLookup(TestWagerID, &models.GroupWagerDetail{
			Wager:        scenario.Wager,
			Options:      scenario.Options,
			Participants: scenario.Participants,
		})

		// Try to bet on non-existent option
		_, err := fixture.Service.PlaceBet(fixture.Ctx, TestWagerID, TestUser1ID, 99999, 1000)

		fixture.Assertions.AssertValidationError(err, "invalid option")
		fixture.AssertAllMocks()
	})

	t.Run("bet amount validation", func(t *testing.T) {
		fixture.Reset()

		testCases := []struct {
			name          string
			amount        int64
			expectedError string
		}{
			{"negative amount", -1000, "bet amount must be positive"},
			{"zero amount", 0, "bet amount must be positive"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				fixture.Reset()

				_, err := fixture.Service.PlaceBet(fixture.Ctx, TestWagerID, TestUser1ID, TestOption1ID, tc.amount)

				fixture.Assertions.AssertValidationError(err, tc.expectedError)
				fixture.AssertAllMocks()
			})
		}
	})

	t.Run("insufficient balance scenarios", func(t *testing.T) {
		fixture.Reset()

		// User with very low balance
		scenario := NewGroupWagerScenario().
			WithPoolWager(TestResolverID, "Test wager").
			WithOptions("Yes", "No").
			WithUser(TestUser1ID, "poor_user", 500). // Only 500 balance
			Build()

		fixture.Helper.ExpectWagerLookup(TestWagerID, scenario.Wager)
		fixture.Helper.ExpectWagerDetailLookup(TestWagerID, &models.GroupWagerDetail{
			Wager:        scenario.Wager,
			Options:      scenario.Options,
			Participants: scenario.Participants,
		})
		fixture.Helper.ExpectUserLookup(TestUser1ID, scenario.Users[TestUser1ID])
		fixture.Helper.ExpectParticipantLookup(TestWagerID, TestUser1ID, nil)

		// Try to bet more than balance
		_, err := fixture.Service.PlaceBet(fixture.Ctx, TestWagerID, TestUser1ID, TestOption1ID, 1000)

		fixture.Assertions.AssertValidationError(err, "insufficient balance")
		fixture.AssertAllMocks()
	})

	t.Run("wager state validation", func(t *testing.T) {
		invalidStates := []struct {
			state         models.GroupWagerState
			expectedError string
		}{
			{models.GroupWagerStateResolved, "cannot place bets on resolved wager"},
			{models.GroupWagerStateCancelled, "cannot place bets on cancelled wager"},
		}

		for _, tc := range invalidStates {
			t.Run(string(tc.state), func(t *testing.T) {
				fixture.Reset()

				wager := &models.GroupWager{
					ID:    TestWagerID,
					State: tc.state,
				}
				fixture.Helper.ExpectWagerLookup(TestWagerID, wager)

				_, err := fixture.Service.PlaceBet(fixture.Ctx, TestWagerID, TestUser1ID, TestOption1ID, 1000)

				fixture.Assertions.AssertValidationError(err, tc.expectedError)
				fixture.AssertAllMocks()
			})
		}
	})

	t.Run("repository operation failures", func(t *testing.T) {
		fixture.Reset()

		// Setup valid scenario but simulate repository failure
		scenario := NewGroupWagerScenario().
			WithPoolWager(TestResolverID, "Test wager").
			WithOptions("Yes", "No").
			WithUser(TestUser1ID, "user1", TestInitialBalance).
			Build()

		fixture.Helper.ExpectWagerLookup(TestWagerID, scenario.Wager)
		fixture.Helper.ExpectWagerDetailLookup(TestWagerID, &models.GroupWagerDetail{
			Wager:        scenario.Wager,
			Options:      scenario.Options,
			Participants: scenario.Participants,
		})
		fixture.Helper.ExpectUserLookup(TestUser1ID, scenario.Users[TestUser1ID])
		fixture.Helper.ExpectParticipantLookup(TestWagerID, TestUser1ID, nil)

		// Simulate participant creation failure
		fixture.Mocks.GroupWagerRepo.On("SaveParticipant", fixture.Ctx, mock.Anything).Return(errors.New("database write failed"))

		_, err := fixture.Service.PlaceBet(fixture.Ctx, TestWagerID, TestUser1ID, TestOption1ID, 1000)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create participant")
		fixture.AssertAllMocks()
	})

	t.Run("concurrent modification scenarios", func(t *testing.T) {
		fixture.Reset()

		// Simulate scenario where wager state changes between lookup and update
		scenario := NewGroupWagerScenario().
			WithPoolWager(TestResolverID, "Test wager").
			WithOptions("Yes", "No").
			WithUser(TestUser1ID, "user1", TestInitialBalance).
			Build()

		fixture.Helper.ExpectWagerLookup(TestWagerID, scenario.Wager)
		fixture.Helper.ExpectWagerDetailLookup(TestWagerID, &models.GroupWagerDetail{
			Wager:        scenario.Wager,
			Options:      scenario.Options,
			Participants: scenario.Participants,
		})
		fixture.Helper.ExpectUserLookup(TestUser1ID, scenario.Users[TestUser1ID])
		fixture.Helper.ExpectParticipantLookup(TestWagerID, TestUser1ID, nil)
		fixture.Helper.ExpectNewParticipant(TestWagerID, TestUser1ID, TestOption1ID, 1000)
		fixture.Helper.ExpectOptionTotalUpdate(TestOption1ID, 1000)

		// Simulate optimistic locking failure on wager update
		fixture.Mocks.GroupWagerRepo.On("Update", fixture.Ctx, mock.Anything).Return(errors.New("row was modified by another transaction"))

		_, err := fixture.Service.PlaceBet(fixture.Ctx, TestWagerID, TestUser1ID, TestOption1ID, 1000)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update group wager")
		fixture.AssertAllMocks()
	})
}

func TestGroupWagerService_AuthorizationEdgeCases(t *testing.T) {
	fixture := NewGroupWagerTestFixture(t)

	t.Run("resolver permission edge cases", func(t *testing.T) {
		fixture.Reset()
		fixture.SetResolvers() // No resolvers configured

		// User trying to resolve when not authorized
		// Note: No mock expectations needed since authorization fails before wager lookup
		unauthorizedUserID := int64(TestUser2ID)
		_, err := fixture.Service.ResolveGroupWager(fixture.Ctx, TestWagerID, &unauthorizedUserID, TestOption1ID)

		fixture.Assertions.AssertValidationError(err, "user is not authorized to resolve group wagers")
		fixture.AssertAllMocks()
	})

	t.Run("creator vs resolver permissions", func(t *testing.T) {
		fixture.Reset()
		fixture.SetResolvers(TestResolverID)

		testCases := []struct {
			name          string
			creatorID     *int64
			cancellerID   *int64
			shouldSucceed bool
		}{
			{"creator cancels own wager", &[]int64{TestUser1ID}[0], &[]int64{TestUser1ID}[0], true},
			{"resolver cancels any wager", &[]int64{TestUser1ID}[0], &[]int64{TestResolverID}[0], true},
			{"system cancels system wager", nil, nil, true},
			{"unauthorized user cancels", &[]int64{TestUser1ID}[0], &[]int64{TestUser2ID}[0], false},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				fixture.Reset()
				fixture.SetResolvers(TestResolverID)

				wager := &models.GroupWager{
					ID:               TestWagerID,
					CreatorDiscordID: tc.creatorID,
					State:            models.GroupWagerStateActive,
					MessageID:        TestMessageID,
					ChannelID:        TestChannelID,
				}
				fixture.Helper.ExpectWagerLookup(TestWagerID, wager)

				if tc.shouldSucceed {
					fixture.Mocks.GroupWagerRepo.On("Update", fixture.Ctx, mock.MatchedBy(func(w *models.GroupWager) bool {
						return w.State == models.GroupWagerStateCancelled
					})).Return(nil)
					fixture.Helper.ExpectEventPublish("events.GroupWagerStateChangeEvent")
				}

				err := fixture.Service.CancelGroupWager(fixture.Ctx, TestWagerID, tc.cancellerID)

				if tc.shouldSucceed {
					fixture.Assertions.AssertNoError(err)
				} else {
					fixture.Assertions.AssertValidationError(err, "only the creator or a resolver can cancel")
				}
				fixture.AssertAllMocks()
			})
		}
	})
}

func TestGroupWagerService_DataIntegrityEdgeCases(t *testing.T) {
	fixture := NewGroupWagerTestFixture(t)

	t.Run("empty participant list resolution", func(t *testing.T) {
		fixture.Reset()
		fixture.SetResolvers(TestResolverID)

		// Wager with no participants
		scenario := NewGroupWagerScenario().
			WithPoolWager(TestResolverID, "Empty wager").
			WithOptions("Yes", "No").
			Build()

		fixture.Helper.ExpectWagerLookup(TestWagerID, scenario.Wager)
		fixture.Helper.ExpectWagerDetailLookup(TestWagerID, &models.GroupWagerDetail{
			Wager:        scenario.Wager,
			Options:      scenario.Options,
			Participants: []*models.GroupWagerParticipant{}, // Empty
		})

		resolverID := int64(TestResolverID)
		_, err := fixture.Service.ResolveGroupWager(fixture.Ctx, TestWagerID, &resolverID, TestOption1ID)

		fixture.Assertions.AssertValidationError(err, "insufficient participants")
		fixture.AssertAllMocks()
	})
}
