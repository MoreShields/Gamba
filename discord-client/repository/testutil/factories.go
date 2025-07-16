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
		WagerType:           models.GroupWagerTypePool, // Default to pool
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

// CreateTestGroupWagerWithType creates a test group wager with specific type
func CreateTestGroupWagerWithType(creatorID int64, condition string, wagerType models.GroupWagerType) *models.GroupWager {
	wager := CreateTestGroupWager(creatorID, condition)
	wager.WagerType = wagerType
	return wager
}

// CreateTestHouseWager creates a test house group wager
func CreateTestHouseWager(creatorID int64, condition string) *models.GroupWager {
	return CreateTestGroupWagerWithType(creatorID, condition, models.GroupWagerTypeHouse)
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
		GroupWagerID:   groupWagerID,
		OptionText:     text,
		OptionOrder:    order,
		TotalAmount:    0,
		OddsMultiplier: 0, // Default to 0 for pool wagers
		CreatedAt:      time.Now(),
	}
}

// CreateTestGroupWagerOptionWithOdds creates a test group wager option with specific odds
func CreateTestGroupWagerOptionWithOdds(groupWagerID int64, text string, order int16, oddsMultiplier float64) *models.GroupWagerOption {
	option := CreateTestGroupWagerOption(groupWagerID, text, order)
	option.OddsMultiplier = oddsMultiplier
	return option
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

// CreateTestSummoner creates a test summoner with default values
func CreateTestSummoner(summonerName, region string) *models.Summoner {
	now := time.Now()
	return &models.Summoner{
		SummonerName: summonerName,
		Region:       region,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// CreateTestGuildSummonerWatch creates a test guild summoner watch
func CreateTestGuildSummonerWatch(guildID, summonerID int64) *models.GuildSummonerWatch {
	return &models.GuildSummonerWatch{
		GuildID:    guildID,
		SummonerID: summonerID,
		CreatedAt:  time.Now(),
	}
}

// CreateTestSummonerWatchDetail creates a test summoner watch detail
func CreateTestSummonerWatchDetail(guildID int64, summonerName, region string) *models.SummonerWatchDetail {
	now := time.Now()
	return &models.SummonerWatchDetail{
		WatchID:      1,
		GuildID:      guildID,
		WatchedAt:    now,
		SummonerID:   1,
		SummonerName: summonerName,
		Region:       region,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

