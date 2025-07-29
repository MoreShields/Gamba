package service

import (
	"testing"

	"gambler/discord-client/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// AssertionHelper provides domain-specific assertions
type AssertionHelper struct {
	t *testing.T
}

// NewAssertionHelper creates a new assertion helper
func NewAssertionHelper(t *testing.T) *AssertionHelper {
	return &AssertionHelper{t: t}
}

// AssertWagerResolved verifies a wager was properly resolved
func (a *AssertionHelper) AssertWagerResolved(result *models.GroupWagerResult, expectedWinnerCount, expectedLoserCount int) {
	require.NotNil(a.t, result)
	assert.Equal(a.t, models.GroupWagerStateResolved, result.GroupWager.State)
	assert.NotNil(a.t, result.GroupWager.ResolverDiscordID)
	assert.NotNil(a.t, result.GroupWager.WinningOptionID)
	assert.NotNil(a.t, result.GroupWager.ResolvedAt)
	assert.Len(a.t, result.Winners, expectedWinnerCount)
	assert.Len(a.t, result.Losers, expectedLoserCount)
}

// AssertPayouts verifies payout calculations
func (a *AssertionHelper) AssertPayouts(result *models.GroupWagerResult, expectedPayouts map[int64]int64) {
	require.NotNil(a.t, result.PayoutDetails)

	for userID, expectedPayout := range expectedPayouts {
		actualPayout, exists := result.PayoutDetails[userID]
		require.True(a.t, exists, "Expected payout for user %d", userID)
		assert.Equal(a.t, expectedPayout, actualPayout, "Incorrect payout for user %d", userID)
	}

	// Verify no unexpected payouts
	assert.Len(a.t, result.PayoutDetails, len(expectedPayouts))
}

// AssertPoolWagerPayouts verifies pool wager specific payout logic
func (a *AssertionHelper) AssertPoolWagerPayouts(result *models.GroupWagerResult) {
	// For pool wagers, verify proportional distribution
	var totalWinnerBets int64
	var totalPayouts int64

	for _, winner := range result.Winners {
		totalWinnerBets += winner.Amount
		if winner.PayoutAmount != nil {
			totalPayouts += *winner.PayoutAmount
		}
	}

	// Total payouts should equal total pot (allowing for rounding)
	assert.InDelta(a.t, float64(result.TotalPot), float64(totalPayouts), 1.0,
		"Total payouts should approximately equal total pot")
}

// AssertHouseWagerPayouts verifies house wager specific payout logic
func (a *AssertionHelper) AssertHouseWagerPayouts(result *models.GroupWagerResult, winningOption *models.GroupWagerOption) {
	// For house wagers, verify fixed odds payouts
	for _, winner := range result.Winners {
		require.NotNil(a.t, winner.PayoutAmount)
		expectedPayout := int64(float64(winner.Amount) * winningOption.OddsMultiplier)
		assert.Equal(a.t, expectedPayout, *winner.PayoutAmount,
			"House wager payout should be bet amount * odds multiplier")
	}

	// Losers should have 0 payout
	for _, loser := range result.Losers {
		require.NotNil(a.t, loser.PayoutAmount)
		assert.Equal(a.t, int64(0), *loser.PayoutAmount)
	}
}

// AssertBalanceChanges verifies expected balance changes
func (a *AssertionHelper) AssertBalanceChanges(wagerType models.GroupWagerType, winners, losers []*models.GroupWagerParticipant) {
	if wagerType == models.GroupWagerTypePool {
		// Pool wager: winners get net profit, losers lose bet amount
		for _, winner := range winners {
			require.NotNil(a.t, winner.PayoutAmount)
			netProfit := *winner.PayoutAmount - winner.Amount
			assert.True(a.t, netProfit >= 0, "Winner should have non-negative profit")
		}

		for _, loser := range losers {
			require.NotNil(a.t, loser.PayoutAmount)
			assert.Equal(a.t, int64(0), *loser.PayoutAmount, "Loser should have 0 payout")
		}
	} else {
		// House wager: winners get full payout (bet already deducted), losers get 0
		for _, winner := range winners {
			require.NotNil(a.t, winner.PayoutAmount)
			assert.True(a.t, *winner.PayoutAmount > 0, "Winner should have positive payout")
		}

		for _, loser := range losers {
			require.NotNil(a.t, loser.PayoutAmount)
			assert.Equal(a.t, int64(0), *loser.PayoutAmount, "Loser should have 0 payout")
		}
	}
}

// AssertParticipantPayout verifies a specific participant's payout
func (a *AssertionHelper) AssertParticipantPayout(participant *models.GroupWagerParticipant, expectedPayout int64, hasBalanceHistory bool) {
	require.NotNil(a.t, participant.PayoutAmount)
	assert.Equal(a.t, expectedPayout, *participant.PayoutAmount)

	if hasBalanceHistory {
		assert.NotNil(a.t, participant.BalanceHistoryID)
	} else {
		assert.Nil(a.t, participant.BalanceHistoryID)
	}
}

// AssertWagerState verifies wager state
func (a *AssertionHelper) AssertWagerState(wager *models.GroupWager, expectedState models.GroupWagerState) {
	assert.Equal(a.t, expectedState, wager.State)
}

// AssertValidationError verifies an error contains expected validation message
func (a *AssertionHelper) AssertValidationError(err error, expectedMessage string) {
	require.Error(a.t, err)
	assert.Contains(a.t, err.Error(), expectedMessage)
}

// AssertNoError verifies no error occurred
func (a *AssertionHelper) AssertNoError(err error) {
	require.NoError(a.t, err)
}

// AssertGroupWagerCreated verifies a group wager was created successfully
func (a *AssertionHelper) AssertGroupWagerCreated(result *models.GroupWagerDetail, wagerType models.GroupWagerType, optionCount int) {
	require.NotNil(a.t, result)
	require.NotNil(a.t, result.Wager)
	assert.Equal(a.t, wagerType, result.Wager.WagerType)
	assert.Equal(a.t, models.GroupWagerStateActive, result.Wager.State)
	assert.Len(a.t, result.Options, optionCount)
	assert.Empty(a.t, result.Participants)
}

// AssertOptionOdds verifies option odds
func (a *AssertionHelper) AssertOptionOdds(options []*models.GroupWagerOption, expectedOdds map[int]float64) {
	for i, option := range options {
		if expectedOdd, exists := expectedOdds[i]; exists {
			assert.Equal(a.t, expectedOdd, option.OddsMultiplier,
				"Option %d (%s) should have odds %.2f", i, option.OptionText, expectedOdd)
		}
	}
}
