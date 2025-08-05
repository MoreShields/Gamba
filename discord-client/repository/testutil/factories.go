package testutil

import (
	"time"

	"gambler/discord-client/domain/entities"
)

// CreateTestUser creates a test user with default values
func CreateTestUser(discordID int64, username string) *entities.User {
	now := time.Now()
	return &entities.User{
		DiscordID: discordID,
		Username:  username,
		Balance:   100000,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// CreateTestUserWithBalance creates a test user with a specific balance
func CreateTestUserWithBalance(discordID int64, username string, balance int64) *entities.User {
	user := CreateTestUser(discordID, username)
	user.Balance = balance
	return user
}

// CreateTestBalanceHistory creates a test balance history entry
func CreateTestBalanceHistory(discordID int64, transactionType entities.TransactionType) *entities.BalanceHistory {
	return &entities.BalanceHistory{
		DiscordID:       discordID,
		BalanceBefore:   100000,
		BalanceAfter:    90000,
		ChangeAmount:    -10000,
		TransactionType: transactionType,
		TransactionMetadata: map[string]interface{}{
			"test": true,
		},
		CreatedAt: time.Now(),
	}
}

// CreateTestBalanceHistoryWithAmounts creates a test balance history with specific amounts
func CreateTestBalanceHistoryWithAmounts(discordID int64, before, after, change int64, transactionType entities.TransactionType) *entities.BalanceHistory {
	history := CreateTestBalanceHistory(discordID, transactionType)
	history.BalanceBefore = before
	history.BalanceAfter = after
	history.ChangeAmount = change
	return history
}

// CreateTestGroupWager creates a test group wager with sensible defaults
func CreateTestGroupWager(creatorID int64, condition string) *entities.GroupWager {
	futureTime := time.Now().Add(24 * time.Hour)
	return &entities.GroupWager{
		CreatorDiscordID:    &creatorID,
		Condition:           condition,
		State:               entities.GroupWagerStateActive,
		WagerType:           entities.GroupWagerTypePool, // Default to pool
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
func CreateTestGroupWagerWithType(creatorID int64, condition string, wagerType entities.GroupWagerType) *entities.GroupWager {
	wager := CreateTestGroupWager(creatorID, condition)
	wager.WagerType = wagerType
	return wager
}

// CreateTestHouseWager creates a test house group wager
func CreateTestHouseWager(creatorID int64, condition string) *entities.GroupWager {
	return CreateTestGroupWagerWithType(creatorID, condition, entities.GroupWagerTypeHouse)
}

// CreateTestGroupWagerResolved creates a resolved test group wager
func CreateTestGroupWagerResolved(creatorID int64, condition string, resolverID int64) *entities.GroupWager {
	wager := CreateTestGroupWager(creatorID, condition)
	wager.State = entities.GroupWagerStateResolved
	wager.ResolverDiscordID = &resolverID
	resolvedAt := time.Now()
	wager.ResolvedAt = &resolvedAt
	return wager
}

// CreateTestGroupWagerOption creates a test group wager option
func CreateTestGroupWagerOption(groupWagerID int64, text string, order int16) *entities.GroupWagerOption {
	return &entities.GroupWagerOption{
		GroupWagerID:   groupWagerID,
		OptionText:     text,
		OptionOrder:    order,
		TotalAmount:    0,
		OddsMultiplier: 0, // Default to 0 for pool wagers
		CreatedAt:      time.Now(),
	}
}

// CreateTestGroupWagerOptionWithOdds creates a test group wager option with specific odds
func CreateTestGroupWagerOptionWithOdds(groupWagerID int64, text string, order int16, oddsMultiplier float64) *entities.GroupWagerOption {
	option := CreateTestGroupWagerOption(groupWagerID, text, order)
	option.OddsMultiplier = oddsMultiplier
	return option
}

// CreateTestGroupWagerParticipant creates a test group wager participant
func CreateTestGroupWagerParticipant(groupWagerID, discordID, optionID int64, amount int64) *entities.GroupWagerParticipant {
	return &entities.GroupWagerParticipant{
		GroupWagerID: groupWagerID,
		DiscordID:    discordID,
		OptionID:     optionID,
		Amount:       amount,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

// CreateTestGroupWagerParticipantWithPayout creates a test participant with payout info
func CreateTestGroupWagerParticipantWithPayout(groupWagerID, discordID, optionID int64, amount, payout int64, balanceHistoryID int64) *entities.GroupWagerParticipant {
	participant := CreateTestGroupWagerParticipant(groupWagerID, discordID, optionID, amount)
	participant.PayoutAmount = &payout
	participant.BalanceHistoryID = &balanceHistoryID
	return participant
}

// CreateTestSummoner creates a test summoner with default values
func CreateTestSummoner(summonerName, tagLine string) *entities.Summoner {
	now := time.Now()
	return &entities.Summoner{
		SummonerName: summonerName,
		TagLine:      tagLine,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// CreateTestGuildSummonerWatch creates a test guild summoner watch
func CreateTestGuildSummonerWatch(guildID, summonerID int64) *entities.GuildSummonerWatch {
	return &entities.GuildSummonerWatch{
		GuildID:    guildID,
		SummonerID: summonerID,
		CreatedAt:  time.Now(),
	}
}

// CreateTestSummonerWatchDetail creates a test summoner watch detail
func CreateTestSummonerWatchDetail(guildID int64, summonerName, tagLine string) *entities.SummonerWatchDetail {
	now := time.Now()
	return &entities.SummonerWatchDetail{
		GuildID:      guildID,
		WatchedAt:    now,
		SummonerName: summonerName,
		TagLine:      tagLine,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// CreateTestHighRollerPurchase creates a test high roller purchase with default values
func CreateTestHighRollerPurchase(guildID, discordID int64, purchasePrice int64) *entities.HighRollerPurchase {
	return &entities.HighRollerPurchase{
		GuildID:       guildID,
		DiscordID:     discordID,
		PurchasePrice: purchasePrice,
		PurchasedAt:   time.Now(),
	}
}

// CreateTestHighRollerPurchaseWithTime creates a test high roller purchase with a specific time
func CreateTestHighRollerPurchaseWithTime(guildID, discordID int64, purchasePrice int64, purchasedAt time.Time) *entities.HighRollerPurchase {
	purchase := CreateTestHighRollerPurchase(guildID, discordID, purchasePrice)
	purchase.PurchasedAt = purchasedAt
	return purchase
}
