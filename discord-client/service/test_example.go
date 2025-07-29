//go:build example
// +build example

// This file contains example test patterns and is not meant to be compiled.
// It serves as a reference for writing new tests using the streamlined patterns.
// Copy and adapt these examples for your specific test needs.

package service

import (
	"context"
	"testing"

	"gambler/discord-client/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ExampleTest demonstrates the streamlined testing patterns
// This is a template for writing new service tests

func TestServiceMethod_Example(t *testing.T) {
	ctx := context.Background()

	// Example 1: Table-driven test for multiple scenarios
	t.Run("table driven example", func(t *testing.T) {
		testCases := []struct {
			name          string
			setupScenario func() *GroupWagerScenario
			expectedError string
			validate      func(*testing.T, interface{})
		}{
			{
				name: "successful case",
				setupScenario: func() *GroupWagerScenario {
					return NewGroupWagerScenario().
						WithPoolWager(TestResolverID, "Test condition").
						WithOptions("Option A", "Option B").
						WithUser(TestUser1ID, "user1", TestInitialBalance).
						Build()
				},
				validate: func(t *testing.T, result interface{}) {
					require.NotNil(t, result)
					// Add specific assertions
				},
			},
			{
				name: "error case",
				setupScenario: func() *GroupWagerScenario {
					return NewGroupWagerScenario().
						WithPoolWager(TestResolverID, "Test").
						WithOptions("A", "B").
						Build()
				},
				expectedError: "expected error message",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Setup
				mocks := NewTestMocks()
				helper := NewMockHelper(mocks)
				service := NewGroupWagerService(
					mocks.GroupWagerRepo,
					mocks.UserRepo,
					mocks.BalanceHistoryRepo,
					mocks.EventPublisher,
				)

				// Build scenario
				scenario := tc.setupScenario()

				// Setup mocks using helper
				// helper.ExpectUserLookup(...)
				// helper.ExpectWagerLookup(...)

				// Execute
				// result, err := service.Method(ctx, ...)

				// Validate
				if tc.expectedError != "" {
					require.Error(t, err)
					require.Contains(t, err.Error(), tc.expectedError)
				} else {
					require.NoError(t, err)
					tc.validate(t, result)
				}

				mocks.AssertAllExpectations(t)
			})
		}
	})

	// Example 2: Focused test for specific functionality
	t.Run("specific functionality example", func(t *testing.T) {
		// Setup
		mocks := NewTestMocks()
		helper := NewMockHelper(mocks)
		assertions := NewAssertionHelper(t)

		service := NewGroupWagerService(
			mocks.GroupWagerRepo,
			mocks.UserRepo,
			mocks.BalanceHistoryRepo,
			mocks.EventPublisher,
		)

		// Build a specific scenario
		scenario := NewGroupWagerScenario().
			WithHouseWager(TestResolverID, "Specific test").
			WithOptions("Yes", "No").
			WithOdds(2.0, 3.0).
			WithUser(TestUser1ID, "user1", 50000).
			WithParticipant(TestUser1ID, 0, 1000).
			Build()

		// Setup mocks for this specific case
		helper.ExpectWagerLookup(TestWagerID, scenario.Wager)
		helper.ExpectUserLookup(TestUser1ID, scenario.Users[TestUser1ID])

		// Add custom mock expectation if needed
		mocks.GroupWagerRepo.On("CustomMethod", ctx, mock.MatchedBy(func(param interface{}) bool {
			// Custom matching logic
			return true
		})).Return(nil)

		// Execute
		// result, err := service.Method(ctx, ...)

		// Use assertion helpers
		assertions.AssertNoError(err)
		// assertions.AssertWagerState(wager, expectedState)
		// assertions.AssertPayouts(result, expectedPayouts)

		mocks.AssertAllExpectations(t)
	})

	// Example 3: Testing both wager types
	t.Run("both wager types example", func(t *testing.T) {
		wagerTypes := []struct {
			name      string
			wagerType models.GroupWagerType
			odds      []float64
		}{
			{
				name:      "pool wager",
				wagerType: models.GroupWagerTypePool,
				odds:      nil, // Pool wagers calculate their own odds
			},
			{
				name:      "house wager",
				wagerType: models.GroupWagerTypeHouse,
				odds:      []float64{2.5, 1.8},
			},
		}

		for _, wt := range wagerTypes {
			t.Run(wt.name, func(t *testing.T) {
				// Setup for specific wager type
				mocks := NewTestMocks()
				service := NewGroupWagerService(
					mocks.GroupWagerRepo,
					mocks.UserRepo,
					mocks.BalanceHistoryRepo,
					mocks.EventPublisher,
				)

				// Build scenario based on wager type
				builder := NewGroupWagerScenario()

				if wt.wagerType == models.GroupWagerTypePool {
					builder.WithPoolWager(TestResolverID, "Pool test")
				} else {
					builder.WithHouseWager(TestResolverID, "House test")
				}

				scenario := builder.
					WithOptions("A", "B").
					WithOdds(wt.odds...).
					Build()

				// Test logic specific to wager type
				// ...

				mocks.AssertAllExpectations(t)
			})
		}
	})
}

// Example of a helper function specific to your test needs
func setupComplexScenario(t *testing.T, helper *MockHelper, numParticipants int) *GroupWagerScenario {
	builder := NewGroupWagerScenario().
		WithPoolWager(TestResolverID, "Complex scenario").
		WithOptions("Option 1", "Option 2", "Option 3")

	// Add multiple participants
	for i := 0; i < numParticipants; i++ {
		userID := int64(100000 + i)
		builder.
			WithUser(userID, "user", TestInitialBalance).
			WithParticipant(userID, i%3, int64(1000*(i+1)))
	}

	return builder.Build()
}

// Example of testing error scenarios
func TestServiceMethod_ErrorScenarios(t *testing.T) {
	ctx := context.Background()

	errorTests := []struct {
		name          string
		setupMocks    func(*MockHelper)
		expectedError string
	}{
		{
			name: "database error",
			setupMocks: func(helper *MockHelper) {
				helper.mocks.GroupWagerRepo.On("GetByID", ctx, int64(TestWagerID)).
					Return(nil, assert.AnError)
			},
			expectedError: "failed to get group wager",
		},
		{
			name: "not found",
			setupMocks: func(helper *MockHelper) {
				helper.ExpectWagerLookup(TestWagerID, nil)
			},
			expectedError: "group wager not found",
		},
	}

	for _, tt := range errorTests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mocks := NewTestMocks()
			helper := NewMockHelper(mocks)
			service := NewGroupWagerService(
				mocks.GroupWagerRepo,
				mocks.UserRepo,
				mocks.BalanceHistoryRepo,
				mocks.EventPublisher,
			)

			// Setup error mocks
			tt.setupMocks(helper)

			// Execute
			// _, err := service.Method(ctx, ...)

			// Verify error
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.expectedError)

			mocks.AssertAllExpectations(t)
		})
	}
}
