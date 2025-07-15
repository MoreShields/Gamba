package testutil

import (
	"time"

	"gambler/discord-client/models"
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

// CreateTestGroupWager creates a test group wager with sensible defaults
func CreateTestGroupWager(creatorID int64, condition string) *models.GroupWager {
	futureTime := time.Now().Add(24 * time.Hour)
	return &models.GroupWager{
		CreatorDiscordID:    creatorID,
		Condition:           condition,
		State:               models.GroupWagerStateActive,
		TotalPot:            0,
		MinParticipants:     2,
		VotingPeriodMinutes: 60,
		VotingStartsAt:      &time.Time{},
		VotingEndsAt:        &futureTime,
		MessageID:           123456,
		ChannelID:           789012,
		CreatedAt:           time.Now(),
	}
}

// CreateTestGroupWagerResolved creates a resolved test group wager
func CreateTestGroupWagerResolved(creatorID int64, condition string, resolverID int64) *models.GroupWager {
	wager := CreateTestGroupWager(creatorID, condition)
	wager.State = models.GroupWagerStateResolved
	wager.ResolverDiscordID = &resolverID
	resolvedAt := time.Now()
	wager.ResolvedAt = &resolvedAt
	return wager
}

// CreateTestGroupWagerOption creates a test group wager option
func CreateTestGroupWagerOption(groupWagerID int64, text string, order int16) *models.GroupWagerOption {
	return &models.GroupWagerOption{
		GroupWagerID: groupWagerID,
		OptionText:   text,
		OptionOrder:  order,
		TotalAmount:  0,
		CreatedAt:    time.Now(),
	}
}

// CreateTestGroupWagerParticipant creates a test group wager participant
func CreateTestGroupWagerParticipant(groupWagerID, discordID, optionID int64, amount int64) *models.GroupWagerParticipant {
	return &models.GroupWagerParticipant{
		GroupWagerID: groupWagerID,
		DiscordID:    discordID,
		OptionID:     optionID,
		Amount:       amount,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

// CreateTestGroupWagerParticipantWithPayout creates a test participant with payout info
func CreateTestGroupWagerParticipantWithPayout(groupWagerID, discordID, optionID int64, amount, payout int64, balanceHistoryID int64) *models.GroupWagerParticipant {
	participant := CreateTestGroupWagerParticipant(groupWagerID, discordID, optionID, amount)
	participant.PayoutAmount = &payout
	participant.BalanceHistoryID = &balanceHistoryID
	return participant
}

