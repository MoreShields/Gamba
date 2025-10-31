package services

import (
	"context"
	"math"
	"testing"

	"gambler/discord-client/config"
	"gambler/discord-client/domain/entities"
	"gambler/discord-client/domain/testhelpers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestGamblingService_ProbabilityAccuracy_10Percent tests if a 10% win probability
// actually wins approximately 10% of the time over many trials
func TestGamblingService_ProbabilityAccuracy_10Percent(t *testing.T) {
	testProbabilityAccuracy(t, 0.10, 10000, 0.02) // 10% with 2% tolerance
}

// TestGamblingService_ProbabilityAccuracy_25Percent tests 25% win probability
func TestGamblingService_ProbabilityAccuracy_25Percent(t *testing.T) {
	testProbabilityAccuracy(t, 0.25, 10000, 0.02) // 25% with 2% tolerance
}

// TestGamblingService_ProbabilityAccuracy_50Percent tests 50% win probability
func TestGamblingService_ProbabilityAccuracy_50Percent(t *testing.T) {
	testProbabilityAccuracy(t, 0.50, 10000, 0.02) // 50% with 2% tolerance
}

// TestGamblingService_ProbabilityAccuracy_75Percent tests 75% win probability
func TestGamblingService_ProbabilityAccuracy_75Percent(t *testing.T) {
	testProbabilityAccuracy(t, 0.75, 10000, 0.02) // 75% with 2% tolerance
}

// TestGamblingService_ProbabilityAccuracy_90Percent tests 90% win probability
func TestGamblingService_ProbabilityAccuracy_90Percent(t *testing.T) {
	testProbabilityAccuracy(t, 0.90, 10000, 0.02) // 90% with 2% tolerance
}

// TestGamblingService_ProbabilityAccuracy_EdgeCases tests edge case probabilities
func TestGamblingService_ProbabilityAccuracy_EdgeCases(t *testing.T) {
	// Very low probability (1%)
	testProbabilityAccuracy(t, 0.01, 100000, 0.005) // 1% with 0.5% tolerance

	// Very high probability (99%)
	testProbabilityAccuracy(t, 0.99, 100000, 0.005) // 99% with 0.5% tolerance
}

// testProbabilityAccuracy is a helper function that tests if a given win probability
// produces the expected win rate over many trials
func testProbabilityAccuracy(t *testing.T, winProbability float64, numTrials int, tolerance float64) {
	// Set up test config
	config.SetTestConfig(config.NewTestConfig())

	ctx := context.Background()

	// Setup mocks
	mockUserRepo := new(testhelpers.MockUserRepository)
	mockBalanceHistoryRepo := new(testhelpers.MockBalanceHistoryRepository)
	mockBetRepo := new(testhelpers.MockBetRepository)
	mockEventPublisher := new(testhelpers.MockEventPublisher)

	service := NewGamblingService(mockUserRepo, mockBetRepo, mockBalanceHistoryRepo, mockEventPublisher)

	// Track wins
	wins := 0
	losses := 0

	// Set up user with large balance
	largeBalance := int64(1000000000) // 1 billion bits
	existingUser := &entities.User{
		DiscordID:        123456,
		Username:         "testuser",
		Balance:          largeBalance,
		AvailableBalance: largeBalance,
	}

	// Calculate expected win amount for this probability
	betAmount := int64(1000)
	expectedWinAmount := int64(float64(betAmount) * ((1 - winProbability) / winProbability))

	// Mock for daily limit check - return empty bets (no limit hit)
	mockBetRepo.On("GetByUserSince", ctx, int64(123456), mock.AnythingOfType("time.Time")).Return([]*entities.Bet{}, nil)

	// Mock GetByDiscordID to return user with sufficient balance
	mockUserRepo.On("GetByDiscordID", ctx, int64(123456)).Return(existingUser, nil)

	// Mock UpdateBalance for both wins and losses
	mockUserRepo.On("UpdateBalance", ctx, int64(123456), mock.AnythingOfType("int64")).Return(nil)

	// Mock balance history recording
	mockBalanceHistoryRepo.On("Record", ctx, mock.AnythingOfType("*entities.BalanceHistory")).Return(nil).Run(func(args mock.Arguments) {
		history := args.Get(1).(*entities.BalanceHistory)
		history.ID = 1 // Set a dummy ID
	})

	// Mock bet creation
	mockBetRepo.On("Create", ctx, mock.AnythingOfType("*entities.Bet")).Return(nil)

	// Mock event publishing
	mockEventPublisher.On("Publish", mock.Anything).Return(nil)

	// Run many trials
	for i := 0; i < numTrials; i++ {
		result, err := service.PlaceBet(ctx, 123456, winProbability, betAmount)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		// Verify win amount is calculated correctly
		assert.Equal(t, expectedWinAmount, result.WinAmount, "Win amount should be consistent")

		if result.Won {
			wins++
			// Verify the balance increased by win amount
			expectedBalance := existingUser.Balance + expectedWinAmount
			assert.Equal(t, expectedBalance, result.NewBalance, "Balance should increase by win amount on win")
		} else {
			losses++
			// Verify the balance decreased by bet amount
			expectedBalance := existingUser.Balance - betAmount
			assert.Equal(t, expectedBalance, result.NewBalance, "Balance should decrease by bet amount on loss")
		}

		// Reset user balance for next trial (keep balance constant for consistency)
		existingUser.Balance = largeBalance
	}

	// Calculate actual win rate
	actualWinRate := float64(wins) / float64(numTrials)

	t.Logf("Probability: %.2f%%, Trials: %d, Wins: %d, Losses: %d, Actual Win Rate: %.4f (%.2f%%)",
		winProbability*100, numTrials, wins, losses, actualWinRate, actualWinRate*100)

	// Assert win rate is within tolerance
	lowerBound := winProbability - tolerance
	upperBound := winProbability + tolerance

	assert.GreaterOrEqual(t, actualWinRate, lowerBound,
		"Win rate %.4f should be at least %.4f (%.2f%% - %.2f%%)",
		actualWinRate, lowerBound, winProbability*100, tolerance*100)

	assert.LessOrEqual(t, actualWinRate, upperBound,
		"Win rate %.4f should be at most %.4f (%.2f%% + %.2f%%)",
		actualWinRate, upperBound, winProbability*100, tolerance*100)

	// Additional statistical check: chi-squared test
	// For a binomial distribution, expected variance = n * p * (1-p)
	expectedWins := float64(numTrials) * winProbability
	expectedLosses := float64(numTrials) * (1 - winProbability)

	// Chi-squared statistic
	chiSquared := math.Pow(float64(wins)-expectedWins, 2)/expectedWins +
		math.Pow(float64(losses)-expectedLosses, 2)/expectedLosses

	// For 1 degree of freedom, critical value at 95% confidence is 3.841
	// At 99% confidence it's 6.635
	assert.LessOrEqual(t, chiSquared, 6.635,
		"Chi-squared statistic %.2f exceeds 99%% confidence threshold (6.635), suggesting bias",
		chiSquared)

	t.Logf("Chi-squared statistic: %.2f (should be < 6.635 for 99%% confidence)", chiSquared)
}

// TestGamblingService_WinAmount_Fairness tests that the win amounts are fair (no house edge)
func TestGamblingService_WinAmount_Fairness(t *testing.T) {
	// Set up test config
	config.SetTestConfig(config.NewTestConfig())

	testCases := []struct {
		name           string
		winProbability float64
		betAmount      int64
		expectedWin    int64
	}{
		{
			name:           "10% odds",
			winProbability: 0.10,
			betAmount:      1000,
			expectedWin:    9000, // (1-0.1)/0.1 * 1000 = 9000
		},
		{
			name:           "25% odds",
			winProbability: 0.25,
			betAmount:      1000,
			expectedWin:    3000, // (1-0.25)/0.25 * 1000 = 3000
		},
		{
			name:           "50% odds",
			winProbability: 0.50,
			betAmount:      1000,
			expectedWin:    1000, // (1-0.5)/0.5 * 1000 = 1000
		},
		{
			name:           "75% odds",
			winProbability: 0.75,
			betAmount:      1000,
			expectedWin:    333, // (1-0.75)/0.75 * 1000 = 333.33... truncated to 333
		},
		{
			name:           "90% odds",
			winProbability: 0.90,
			betAmount:      1000,
			expectedWin:    111, // (1-0.9)/0.9 * 1000 = 111.11... truncated to 111
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			mockUserRepo := new(testhelpers.MockUserRepository)
			mockBalanceHistoryRepo := new(testhelpers.MockBalanceHistoryRepository)
			mockBetRepo := new(testhelpers.MockBetRepository)
			mockEventPublisher := new(testhelpers.MockEventPublisher)

			service := NewGamblingService(mockUserRepo, mockBetRepo, mockBalanceHistoryRepo, mockEventPublisher)

			existingUser := &entities.User{
				DiscordID:        123456,
				Username:         "testuser",
				Balance:          100000,
				AvailableBalance: 100000,
			}

			mockBetRepo.On("GetByUserSince", ctx, int64(123456), mock.AnythingOfType("time.Time")).Return([]*entities.Bet{}, nil)
			mockUserRepo.On("GetByDiscordID", ctx, int64(123456)).Return(existingUser, nil)
			mockUserRepo.On("UpdateBalance", ctx, int64(123456), mock.AnythingOfType("int64")).Return(nil)
			mockBalanceHistoryRepo.On("Record", ctx, mock.AnythingOfType("*entities.BalanceHistory")).Return(nil).Run(func(args mock.Arguments) {
				history := args.Get(1).(*entities.BalanceHistory)
				history.ID = 1
			})
			mockBetRepo.On("Create", ctx, mock.AnythingOfType("*entities.Bet")).Return(nil)
			mockEventPublisher.On("Publish", mock.Anything).Return(nil)

			result, err := service.PlaceBet(ctx, 123456, tc.winProbability, tc.betAmount)
			assert.NoError(t, err)
			assert.NotNil(t, result)

			// Verify win amount matches expected (fair odds, no house edge)
			assert.Equal(t, tc.expectedWin, result.WinAmount,
				"Win amount should be %d for %.0f%% odds on %d bet (calculated as (1-p)/p * bet)",
				tc.expectedWin, tc.winProbability*100, tc.betAmount)

			// Verify expected value is fair (EV = 0 with no house edge)
			// EV = (probability of win * win amount) - (probability of loss * bet amount)
			expectedValue := (tc.winProbability * float64(tc.expectedWin)) - ((1 - tc.winProbability) * float64(tc.betAmount))

			// Due to integer truncation, EV might not be exactly 0, but should be very close
			// For example, with 75% odds: EV = 0.75*333 - 0.25*1000 = 249.75 - 250 = -0.25
			// This is acceptable as it's due to integer rounding
			assert.InDelta(t, 0.0, expectedValue, 1.0,
				"Expected value should be close to 0 (fair game). Got %.2f", expectedValue)

			t.Logf("Probability: %.0f%%, Bet: %d, Win Amount: %d, Expected Value: %.2f",
				tc.winProbability*100, tc.betAmount, tc.expectedWin, expectedValue)
		})
	}
}
