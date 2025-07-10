package testutil

import (
	"time"

	"gambler/models"
)

// CreateTestUser creates a test user with default values
func CreateTestUser(discordID int64, username string) *models.User {
	now := time.Now()
	return &models.User{
		DiscordID: discordID,
		Username:  username,
		Balance:   100000,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// CreateTestUserWithBalance creates a test user with a specific balance
func CreateTestUserWithBalance(discordID int64, username string, balance int64) *models.User {
	user := CreateTestUser(discordID, username)
	user.Balance = balance
	return user
}

// CreateTestBalanceHistory creates a test balance history entry
func CreateTestBalanceHistory(discordID int64, transactionType models.TransactionType) *models.BalanceHistory {
	return &models.BalanceHistory{
		DiscordID:           discordID,
		BalanceBefore:       100000,
		BalanceAfter:        90000,
		ChangeAmount:        -10000,
		TransactionType:     transactionType,
		TransactionMetadata: map[string]interface{}{
			"test": true,
		},
		CreatedAt: time.Now(),
	}
}

// CreateTestBalanceHistoryWithAmounts creates a test balance history with specific amounts
func CreateTestBalanceHistoryWithAmounts(discordID int64, before, after, change int64, transactionType models.TransactionType) *models.BalanceHistory {
	history := CreateTestBalanceHistory(discordID, transactionType)
	history.BalanceBefore = before
	history.BalanceAfter = after
	history.ChangeAmount = change
	return history
}

// CreateTestInterestRun creates a test interest run
func CreateTestInterestRun(runDate time.Time) *models.InterestRun {
	return &models.InterestRun{
		RunDate:                  runDate,
		TotalInterestDistributed: 5000,
		UsersAffected:            10,
		ExecutionSummary: map[string]interface{}{
			"average_interest": 500,
			"max_interest":     1000,
			"min_interest":     100,
		},
		CreatedAt: time.Now(),
	}
}

// CreateTestInterestRunWithDetails creates a test interest run with specific details
func CreateTestInterestRunWithDetails(runDate time.Time, total int64, users int) *models.InterestRun {
	run := CreateTestInterestRun(runDate)
	run.TotalInterestDistributed = total
	run.UsersAffected = users
	return run
}