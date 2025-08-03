package repository

import (
	"context"
	"testing"
	"time"

	"gambler/discord-client/domain/entities"
	"gambler/discord-client/repository/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUserRepository_AvailableBalance_Integration tests the complex SQL query
// that calculates available balance by considering various wager types and states
func TestUserRepository_AvailableBalance_Integration(t *testing.T) {
	testDB := testutil.SetupTestDatabase(t)
	ctx := context.Background()
	guildID := int64(123456789)

	// Create repositories
	userRepo := NewUserRepositoryScoped(testDB.DB.Pool, guildID)
	wagerRepo := NewWagerRepositoryScoped(testDB.DB.Pool, guildID)
	groupWagerRepo := NewGroupWagerRepositoryScoped(testDB.DB.Pool, guildID)

	// Test user setup
	userID := int64(987654321)
	targetUserID := int64(987654322)
	username := "testuser"
	targetUsername := "targetuser"
	initialBalance := int64(100000)

	// Create test users
	_, err := userRepo.Create(ctx, userID, username, initialBalance)
	require.NoError(t, err)
	_, err = userRepo.Create(ctx, targetUserID, targetUsername, initialBalance)
	require.NoError(t, err)

	t.Run("no wagers - full balance available", func(t *testing.T) {
		user, err := userRepo.GetByDiscordID(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, initialBalance, user.Balance)
		assert.Equal(t, initialBalance, user.AvailableBalance)
	})

	t.Run("single voting wager reserves balance", func(t *testing.T) {
		// Create a wager in voting state
		wager := &entities.Wager{
			ProposerDiscordID: userID,
			TargetDiscordID:   targetUserID,
			Amount:            10000,
			Condition:         "Test wager 1",
			State:             entities.WagerStateVoting,
			GuildID:           guildID,
		}
		err := wagerRepo.Create(ctx, wager)
		require.NoError(t, err)

		// Check available balance for proposer
		user, err := userRepo.GetByDiscordID(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, initialBalance, user.Balance)
		assert.Equal(t, int64(90000), user.AvailableBalance) // 100000 - 10000

		// Check available balance for target
		target, err := userRepo.GetByDiscordID(ctx, targetUserID)
		require.NoError(t, err)
		assert.Equal(t, initialBalance, target.Balance)
		assert.Equal(t, int64(90000), target.AvailableBalance) // 100000 - 10000

		// Clean up
		wager.State = entities.WagerStateDeclined
		err = wagerRepo.Update(ctx, wager)
		require.NoError(t, err)
	})

	t.Run("multiple voting wagers accumulate reservations", func(t *testing.T) {
		// Create multiple wagers
		wager1 := &entities.Wager{
			ProposerDiscordID: userID,
			TargetDiscordID:   targetUserID,
			Amount:            5000,
			Condition:         "Test wager 2",
			State:             entities.WagerStateVoting,
			GuildID:           guildID,
		}
		err := wagerRepo.Create(ctx, wager1)
		require.NoError(t, err)

		wager2 := &entities.Wager{
			ProposerDiscordID: targetUserID,
			TargetDiscordID:   userID,
			Amount:            3000,
			Condition:         "Test wager 3",
			State:             entities.WagerStateVoting,
			GuildID:           guildID,
		}
		err = wagerRepo.Create(ctx, wager2)
		require.NoError(t, err)

		// Check available balance
		user, err := userRepo.GetByDiscordID(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, initialBalance, user.Balance)
		assert.Equal(t, int64(92000), user.AvailableBalance) // 100000 - 5000 - 3000

		// Clean up
		wager1.State = entities.WagerStateDeclined
		wager2.State = entities.WagerStateDeclined
		err = wagerRepo.Update(ctx, wager1)
		require.NoError(t, err)
		err = wagerRepo.Update(ctx, wager2)
		require.NoError(t, err)
	})

	t.Run("declined wagers do not reserve balance", func(t *testing.T) {
		// Create a declined wager
		wager := &entities.Wager{
			ProposerDiscordID: userID,
			TargetDiscordID:   targetUserID,
			Amount:            15000,
			Condition:         "Test wager 4",
			State:             entities.WagerStateDeclined,
			GuildID:           guildID,
		}
		err := wagerRepo.Create(ctx, wager)
		require.NoError(t, err)

		// Check available balance - should be full balance
		user, err := userRepo.GetByDiscordID(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, initialBalance, user.Balance)
		assert.Equal(t, initialBalance, user.AvailableBalance) // No reservation
	})

	t.Run("active group wager reserves balance", func(t *testing.T) {
		// Create an active group wager
		now := time.Now()
		votingEnds := now.Add(1 * time.Hour)
		groupWager := &entities.GroupWager{
			CreatorDiscordID:    &userID,
			Condition:          "Test group wager 1",
			State:              entities.GroupWagerStateActive,
			WagerType:          entities.GroupWagerTypePool,
			TotalPot:           0,
			MinParticipants:    2,
			VotingPeriodMinutes: 60,
			VotingStartsAt:     &now,
			VotingEndsAt:       &votingEnds,
			MessageID:          12345,
			ChannelID:          67890,
			GuildID:            guildID,
		}

		options := []*entities.GroupWagerOption{
			{
				OptionText:     "Option A",
				OptionOrder:    0,
				TotalAmount:    0,
				OddsMultiplier: 1.0,
			},
			{
				OptionText:     "Option B",
				OptionOrder:    1,
				TotalAmount:    0,
				OddsMultiplier: 1.0,
			},
		}

		err = groupWagerRepo.CreateWithOptions(ctx, groupWager, options)
		require.NoError(t, err)

		// Place a bet
		participant := &entities.GroupWagerParticipant{
			GroupWagerID: groupWager.ID,
			DiscordID:    userID,
			OptionID:     options[0].ID,
			Amount:       20000,
		}
		err = groupWagerRepo.SaveParticipant(ctx, participant)
		require.NoError(t, err)

		// Check available balance
		user, err := userRepo.GetByDiscordID(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, initialBalance, user.Balance)
		assert.Equal(t, int64(80000), user.AvailableBalance) // 100000 - 20000

		// Clean up
		groupWager.State = entities.GroupWagerStateCancelled
		err = groupWagerRepo.Update(ctx, groupWager)
		require.NoError(t, err)
	})

	t.Run("pending_resolution group wager reserves balance", func(t *testing.T) {
		// Create a pending_resolution group wager
		now := time.Now()
		votingEnds := now.Add(-1 * time.Hour) // Already expired
		groupWager := &entities.GroupWager{
			CreatorDiscordID:    &userID,
			Condition:          "Test group wager 2",
			State:              entities.GroupWagerStatePendingResolution,
			WagerType:          entities.GroupWagerTypePool,
			TotalPot:           0,
			MinParticipants:    2,
			VotingPeriodMinutes: 60,
			VotingStartsAt:     &now,
			VotingEndsAt:       &votingEnds,
			MessageID:          12346,
			ChannelID:          67890,
			GuildID:            guildID,
		}

		options := []*entities.GroupWagerOption{
			{
				OptionText:     "Option A",
				OptionOrder:    0,
				TotalAmount:    0,
				OddsMultiplier: 1.0,
			},
			{
				OptionText:     "Option B",
				OptionOrder:    1,
				TotalAmount:    0,
				OddsMultiplier: 1.0,
			},
		}

		err = groupWagerRepo.CreateWithOptions(ctx, groupWager, options)
		require.NoError(t, err)

		// Place a bet
		participant := &entities.GroupWagerParticipant{
			GroupWagerID: groupWager.ID,
			DiscordID:    userID,
			OptionID:     options[0].ID,
			Amount:       25000,
		}
		err = groupWagerRepo.SaveParticipant(ctx, participant)
		require.NoError(t, err)

		// Check available balance - should still be reserved
		user, err := userRepo.GetByDiscordID(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, initialBalance, user.Balance)
		assert.Equal(t, int64(75000), user.AvailableBalance) // 100000 - 25000

		// Clean up
		groupWager.State = entities.GroupWagerStateCancelled
		err = groupWagerRepo.Update(ctx, groupWager)
		require.NoError(t, err)
	})

	t.Run("cancelled group wager does not reserve balance", func(t *testing.T) {
		// Create a cancelled group wager
		now := time.Now()
		votingEnds := now.Add(-2 * time.Hour)
		groupWager := &entities.GroupWager{
			CreatorDiscordID:    &userID,
			Condition:          "Test group wager 3",
			State:              entities.GroupWagerStateCancelled,
			WagerType:          entities.GroupWagerTypePool,
			TotalPot:           30000,
			MinParticipants:    2,
			VotingPeriodMinutes: 60,
			VotingStartsAt:     &now,
			VotingEndsAt:       &votingEnds,
			MessageID:          12347,
			ChannelID:          67890,
			GuildID:            guildID,
		}

		options := []*entities.GroupWagerOption{
			{
				OptionText:     "Option A",
				OptionOrder:    0,
				TotalAmount:    30000,
				OddsMultiplier: 1.0,
			},
			{
				OptionText:     "Option B",
				OptionOrder:    1,
				TotalAmount:    0,
				OddsMultiplier: 1.0,
			},
		}

		err = groupWagerRepo.CreateWithOptions(ctx, groupWager, options)
		require.NoError(t, err)

		// Create a participant record
		participant := &entities.GroupWagerParticipant{
			GroupWagerID: groupWager.ID,
			DiscordID:    userID,
			OptionID:     options[0].ID,
			Amount:       30000,
		}
		err = groupWagerRepo.SaveParticipant(ctx, participant)
		require.NoError(t, err)

		// Check available balance - should be full balance
		user, err := userRepo.GetByDiscordID(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, initialBalance, user.Balance)
		assert.Equal(t, initialBalance, user.AvailableBalance) // No reservation
	})

	t.Run("mixed wagers and group wagers", func(t *testing.T) {
		// Clean up any existing wagers first
		testDB.DB.Pool.Exec(ctx, "DELETE FROM wagers WHERE guild_id = $1", guildID)
		testDB.DB.Pool.Exec(ctx, "DELETE FROM group_wager_participants WHERE discord_id = $1", userID)
		testDB.DB.Pool.Exec(ctx, "DELETE FROM group_wagers WHERE guild_id = $1", guildID)

		// Create a voting wager
		wager := &entities.Wager{
			ProposerDiscordID: userID,
			TargetDiscordID:   targetUserID,
			Amount:            10000,
			Condition:         "Mixed test wager",
			State:             entities.WagerStateVoting,
			GuildID:           guildID,
		}
		err := wagerRepo.Create(ctx, wager)
		require.NoError(t, err)

		// Create an active group wager
		now := time.Now()
		votingEnds := now.Add(1 * time.Hour)
		groupWager1 := &entities.GroupWager{
			CreatorDiscordID:    &userID,
			Condition:          "Mixed test group wager 1",
			State:              entities.GroupWagerStateActive,
			WagerType:          entities.GroupWagerTypePool,
			TotalPot:           0,
			MinParticipants:    2,
			VotingPeriodMinutes: 60,
			VotingStartsAt:     &now,
			VotingEndsAt:       &votingEnds,
			MessageID:          12348,
			ChannelID:          67890,
			GuildID:            guildID,
		}

		options1 := []*entities.GroupWagerOption{
			{
				OptionText:     "Option A",
				OptionOrder:    0,
				TotalAmount:    0,
				OddsMultiplier: 1.0,
			},
			{
				OptionText:     "Option B",
				OptionOrder:    1,
				TotalAmount:    0,
				OddsMultiplier: 1.0,
			},
		}

		err = groupWagerRepo.CreateWithOptions(ctx, groupWager1, options1)
		require.NoError(t, err)

		participant1 := &entities.GroupWagerParticipant{
			GroupWagerID: groupWager1.ID,
			DiscordID:    userID,
			OptionID:     options1[0].ID,
			Amount:       15000,
		}
		err = groupWagerRepo.SaveParticipant(ctx, participant1)
		require.NoError(t, err)

		// Create a pending_resolution group wager
		votingEnds2 := now.Add(-1 * time.Hour)
		groupWager2 := &entities.GroupWager{
			CreatorDiscordID:    &userID,
			Condition:          "Mixed test group wager 2",
			State:              entities.GroupWagerStatePendingResolution,
			WagerType:          entities.GroupWagerTypePool,
			TotalPot:           0,
			MinParticipants:    2,
			VotingPeriodMinutes: 60,
			VotingStartsAt:     &now,
			VotingEndsAt:       &votingEnds2,
			MessageID:          12349,
			ChannelID:          67890,
			GuildID:            guildID,
		}

		options2 := []*entities.GroupWagerOption{
			{
				OptionText:     "Option C",
				OptionOrder:    0,
				TotalAmount:    0,
				OddsMultiplier: 1.0,
			},
			{
				OptionText:     "Option D",
				OptionOrder:    1,
				TotalAmount:    0,
				OddsMultiplier: 1.0,
			},
		}

		err = groupWagerRepo.CreateWithOptions(ctx, groupWager2, options2)
		require.NoError(t, err)

		participant2 := &entities.GroupWagerParticipant{
			GroupWagerID: groupWager2.ID,
			DiscordID:    userID,
			OptionID:     options2[0].ID,
			Amount:       20000,
		}
		err = groupWagerRepo.SaveParticipant(ctx, participant2)
		require.NoError(t, err)

		// Check available balance
		// Expected: 100000 - 10000 (wager) - 15000 (active group) - 20000 (pending group) = 55000
		user, err := userRepo.GetByDiscordID(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, initialBalance, user.Balance)
		assert.Equal(t, int64(55000), user.AvailableBalance)
	})
}

// TestUserRepository_GetUsersWithPositiveBalance_AvailableBalance tests that
// the GetUsersWithPositiveBalance method also correctly calculates available balance
func TestUserRepository_GetUsersWithPositiveBalance_AvailableBalance(t *testing.T) {
	testDB := testutil.SetupTestDatabase(t)
	ctx := context.Background()
	guildID := int64(123456789)

	// Create repositories
	userRepo := NewUserRepositoryScoped(testDB.DB.Pool, guildID)
	groupWagerRepo := NewGroupWagerRepositoryScoped(testDB.DB.Pool, guildID)

	// Create test users
	users := []struct {
		id      int64
		name    string
		balance int64
	}{
		{111111, "user1", 50000},
		{222222, "user2", 30000},
		{333333, "user3", 20000},
	}

	for _, u := range users {
		_, err := userRepo.Create(ctx, u.id, u.name, u.balance)
		require.NoError(t, err)
	}

	// Create group wagers affecting user1 and user2
	now := time.Now()
	votingEnds := now.Add(1 * time.Hour)

	// Active group wager for user1
	groupWager1 := &entities.GroupWager{
		CreatorDiscordID:    &users[0].id,
		Condition:          "User1 group wager",
		State:              entities.GroupWagerStateActive,
		WagerType:          entities.GroupWagerTypePool,
		TotalPot:           0,
		MinParticipants:    2,
		VotingPeriodMinutes: 60,
		VotingStartsAt:     &now,
		VotingEndsAt:       &votingEnds,
		MessageID:          12350,
		ChannelID:          67890,
		GuildID:            guildID,
	}

	options := []*entities.GroupWagerOption{
		{
			OptionText:     "Option A",
			OptionOrder:    0,
			TotalAmount:    0,
			OddsMultiplier: 1.0,
		},
		{
			OptionText:     "Option B",
			OptionOrder:    1,
			TotalAmount:    0,
			OddsMultiplier: 1.0,
		},
	}

	err := groupWagerRepo.CreateWithOptions(ctx, groupWager1, options)
	require.NoError(t, err)

	// User1 bets 10000
	participant1 := &entities.GroupWagerParticipant{
		GroupWagerID: groupWager1.ID,
		DiscordID:    users[0].id,
		OptionID:     options[0].ID,
		Amount:       10000,
	}
	err = groupWagerRepo.SaveParticipant(ctx, participant1)
	require.NoError(t, err)

	// pending_resolution group wager for user2
	votingEnds2 := now.Add(-1 * time.Hour)
	groupWager2 := &entities.GroupWager{
		CreatorDiscordID:    &users[1].id,
		Condition:          "User2 group wager",
		State:              entities.GroupWagerStatePendingResolution,
		WagerType:          entities.GroupWagerTypePool,
		TotalPot:           0,
		MinParticipants:    2,
		VotingPeriodMinutes: 60,
		VotingStartsAt:     &now,
		VotingEndsAt:       &votingEnds2,
		MessageID:          12351,
		ChannelID:          67890,
		GuildID:            guildID,
	}

	err = groupWagerRepo.CreateWithOptions(ctx, groupWager2, options)
	require.NoError(t, err)

	// User2 bets 5000
	participant2 := &entities.GroupWagerParticipant{
		GroupWagerID: groupWager2.ID,
		DiscordID:    users[1].id,
		OptionID:     options[0].ID,
		Amount:       5000,
	}
	err = groupWagerRepo.SaveParticipant(ctx, participant2)
	require.NoError(t, err)

	// Get users with positive balance
	result, err := userRepo.GetUsersWithPositiveBalance(ctx)
	require.NoError(t, err)
	require.Len(t, result, 3)

	// Verify available balances are calculated correctly
	// user1: 50000 - 10000 = 40000
	assert.Equal(t, users[0].id, result[0].DiscordID)
	assert.Equal(t, int64(50000), result[0].Balance)
	assert.Equal(t, int64(40000), result[0].AvailableBalance)

	// user2: 30000 - 5000 = 25000
	assert.Equal(t, users[1].id, result[1].DiscordID)
	assert.Equal(t, int64(30000), result[1].Balance)
	assert.Equal(t, int64(25000), result[1].AvailableBalance)

	// user3: 20000 - 0 = 20000
	assert.Equal(t, users[2].id, result[2].DiscordID)
	assert.Equal(t, int64(20000), result[2].Balance)
	assert.Equal(t, int64(20000), result[2].AvailableBalance)
}

// TestUserRepository_GetAll_AvailableBalance tests that the GetAll method
// also correctly calculates available balance
func TestUserRepository_GetAll_AvailableBalance(t *testing.T) {
	testDB := testutil.SetupTestDatabase(t)
	ctx := context.Background()
	guildID := int64(123456789)

	// Create repositories
	userRepo := NewUserRepositoryScoped(testDB.DB.Pool, guildID)
	wagerRepo := NewWagerRepositoryScoped(testDB.DB.Pool, guildID)

	// Create test users
	user1ID := int64(444444)
	user2ID := int64(555555)
	_, err := userRepo.Create(ctx, user1ID, "user1", 40000)
	require.NoError(t, err)
	_, err = userRepo.Create(ctx, user2ID, "user2", 60000)
	require.NoError(t, err)

	// Create a voting wager between users
	wager := &entities.Wager{
		ProposerDiscordID: user1ID,
		TargetDiscordID:   user2ID,
		Amount:            15000,
		Condition:         "GetAll test wager",
		State:             entities.WagerStateVoting,
		GuildID:           guildID,
	}
	err = wagerRepo.Create(ctx, wager)
	require.NoError(t, err)

	// Get all users
	result, err := userRepo.GetAll(ctx)
	require.NoError(t, err)
	require.Len(t, result, 2)

	// Find users in result (they're ordered by created_at DESC)
	var user1, user2 *entities.User
	for _, u := range result {
		if u.DiscordID == user1ID {
			user1 = u
		} else if u.DiscordID == user2ID {
			user2 = u
		}
	}

	require.NotNil(t, user1)
	require.NotNil(t, user2)

	// Verify available balances
	// user1: 40000 - 15000 = 25000
	assert.Equal(t, int64(40000), user1.Balance)
	assert.Equal(t, int64(25000), user1.AvailableBalance)

	// user2: 60000 - 15000 = 45000
	assert.Equal(t, int64(60000), user2.Balance)
	assert.Equal(t, int64(45000), user2.AvailableBalance)
}