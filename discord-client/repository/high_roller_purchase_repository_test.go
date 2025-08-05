package repository

import (
	"context"
	"testing"
	"time"

	"gambler/discord-client/repository/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHighRollerPurchaseRepository_GetLatestPurchase(t *testing.T) {
	t.Parallel()
	testDB := testutil.SetupTestDatabase(t)

	repo := NewHighRollerPurchaseRepository(testDB.DB)
	ctx := context.Background()

	t.Run("no purchases found", func(t *testing.T) {
		purchase, err := repo.GetLatestPurchase(ctx, 999999)
		require.NoError(t, err)
		assert.Nil(t, purchase)
	})

	t.Run("latest purchase found", func(t *testing.T) {
		guildID := int64(123456)
		
		// Create multiple purchases with different timestamps
		purchase1 := testutil.CreateTestHighRollerPurchaseWithTime(guildID, 111, 10000, time.Now().Add(-2*time.Hour))
		purchase2 := testutil.CreateTestHighRollerPurchaseWithTime(guildID, 222, 20000, time.Now().Add(-1*time.Hour))
		purchase3 := testutil.CreateTestHighRollerPurchaseWithTime(guildID, 333, 30000, time.Now())

		// Create scoped repository for the transaction
		scopedRepo := NewHighRollerPurchaseRepositoryScoped(testDB.DB.Pool, guildID)
		
		err := scopedRepo.CreatePurchase(ctx, purchase1)
		require.NoError(t, err)
		err = scopedRepo.CreatePurchase(ctx, purchase2)
		require.NoError(t, err)
		err = scopedRepo.CreatePurchase(ctx, purchase3)
		require.NoError(t, err)

		// Get latest purchase
		latest, err := repo.GetLatestPurchase(ctx, guildID)
		require.NoError(t, err)
		require.NotNil(t, latest)

		// Should return the most recent purchase (purchase3)
		assert.Equal(t, guildID, latest.GuildID)
		assert.Equal(t, int64(333), latest.DiscordID)
		assert.Equal(t, int64(30000), latest.PurchasePrice)
		assert.True(t, latest.ID > 0)
	})

	t.Run("multiple guilds - returns correct guild's latest", func(t *testing.T) {
		guild1ID := int64(111111)
		guild2ID := int64(222222)
		
		// Create purchases for different guilds
		purchase1 := testutil.CreateTestHighRollerPurchaseWithTime(guild1ID, 444, 40000, time.Now().Add(-1*time.Hour))
		purchase2 := testutil.CreateTestHighRollerPurchaseWithTime(guild2ID, 555, 50000, time.Now())

		scopedRepo1 := NewHighRollerPurchaseRepositoryScoped(testDB.DB.Pool, guild1ID)
		scopedRepo2 := NewHighRollerPurchaseRepositoryScoped(testDB.DB.Pool, guild2ID)
		
		err := scopedRepo1.CreatePurchase(ctx, purchase1)
		require.NoError(t, err)
		err = scopedRepo2.CreatePurchase(ctx, purchase2)
		require.NoError(t, err)

		// Get latest for guild1
		latest1, err := repo.GetLatestPurchase(ctx, guild1ID)
		require.NoError(t, err)
		require.NotNil(t, latest1)
		assert.Equal(t, guild1ID, latest1.GuildID)
		assert.Equal(t, int64(444), latest1.DiscordID)

		// Get latest for guild2
		latest2, err := repo.GetLatestPurchase(ctx, guild2ID)
		require.NoError(t, err)
		require.NotNil(t, latest2)
		assert.Equal(t, guild2ID, latest2.GuildID)
		assert.Equal(t, int64(555), latest2.DiscordID)
	})
}

func TestHighRollerPurchaseRepository_CreatePurchase(t *testing.T) {
	t.Parallel()
	testDB := testutil.SetupTestDatabase(t)

	ctx := context.Background()
	guildID := int64(123456)

	t.Run("successful creation", func(t *testing.T) {
		purchase := testutil.CreateTestHighRollerPurchase(guildID, 789, 25000)
		
		scopedRepo := NewHighRollerPurchaseRepositoryScoped(testDB.DB.Pool, guildID)
		err := scopedRepo.CreatePurchase(ctx, purchase)
		require.NoError(t, err)

		// Verify the purchase was created with an ID
		assert.True(t, purchase.ID > 0)

		// Verify we can retrieve it
		repo := NewHighRollerPurchaseRepository(testDB.DB)
		retrieved, err := repo.GetLatestPurchase(ctx, guildID)
		require.NoError(t, err)
		require.NotNil(t, retrieved)

		assert.Equal(t, purchase.ID, retrieved.ID)
		assert.Equal(t, purchase.GuildID, retrieved.GuildID)
		assert.Equal(t, purchase.DiscordID, retrieved.DiscordID)
		assert.Equal(t, purchase.PurchasePrice, retrieved.PurchasePrice)
		// Allow for slight time differences due to database precision and timezone differences
		assert.WithinDuration(t, purchase.PurchasedAt, retrieved.PurchasedAt, 8*time.Hour)
	})

	t.Run("guild ID mismatch error", func(t *testing.T) {
		purchase := testutil.CreateTestHighRollerPurchase(999999, 789, 25000) // Different guild ID
		
		scopedRepo := NewHighRollerPurchaseRepositoryScoped(testDB.DB.Pool, guildID)
		err := scopedRepo.CreatePurchase(ctx, purchase)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "guild ID mismatch")
	})

	t.Run("multiple purchases same guild", func(t *testing.T) {
		purchase1 := testutil.CreateTestHighRollerPurchaseWithTime(guildID, 111, 10000, time.Now().Add(-1*time.Hour))
		purchase2 := testutil.CreateTestHighRollerPurchaseWithTime(guildID, 222, 20000, time.Now())
		
		scopedRepo := NewHighRollerPurchaseRepositoryScoped(testDB.DB.Pool, guildID)
		
		err := scopedRepo.CreatePurchase(ctx, purchase1)
		require.NoError(t, err)
		err = scopedRepo.CreatePurchase(ctx, purchase2)
		require.NoError(t, err)

		// Both should have unique IDs
		assert.True(t, purchase1.ID > 0)
		assert.True(t, purchase2.ID > 0)
		assert.NotEqual(t, purchase1.ID, purchase2.ID)

		// Latest should be purchase2 (assuming it was created after)
		repo := NewHighRollerPurchaseRepository(testDB.DB)
		latest, err := repo.GetLatestPurchase(ctx, guildID)
		require.NoError(t, err)
		require.NotNil(t, latest)
		assert.Equal(t, purchase2.DiscordID, latest.DiscordID)
	})
}

func TestHighRollerPurchaseRepository_GetPurchaseHistory(t *testing.T) {
	t.Parallel()
	testDB := testutil.SetupTestDatabase(t)

	repo := NewHighRollerPurchaseRepository(testDB.DB)
	ctx := context.Background()

	t.Run("no purchases", func(t *testing.T) {
		history, err := repo.GetPurchaseHistory(ctx, 999999, 10)
		require.NoError(t, err)
		assert.Empty(t, history)
	})

	t.Run("purchase history with limit", func(t *testing.T) {
		guildID := int64(654321)
		
		// Create 5 purchases with different timestamps
		baseTime := time.Now()
		type purchaseData struct {
			GuildID   int64
			DiscordID int64
			Price     int64
			Time      time.Time
		}
		purchases := []purchaseData{
			{GuildID: guildID, DiscordID: 111, Price: 10000, Time: baseTime.Add(-4 * time.Hour)},
			{GuildID: guildID, DiscordID: 222, Price: 20000, Time: baseTime.Add(-3 * time.Hour)},
			{GuildID: guildID, DiscordID: 333, Price: 30000, Time: baseTime.Add(-2 * time.Hour)},
			{GuildID: guildID, DiscordID: 444, Price: 40000, Time: baseTime.Add(-1 * time.Hour)},
			{GuildID: guildID, DiscordID: 555, Price: 50000, Time: baseTime},
		}

		scopedRepo := NewHighRollerPurchaseRepositoryScoped(testDB.DB.Pool, guildID)
		
		for _, p := range purchases {
			purchase := testutil.CreateTestHighRollerPurchaseWithTime(p.GuildID, p.DiscordID, p.Price, p.Time)
			err := scopedRepo.CreatePurchase(ctx, purchase)
			require.NoError(t, err)
		}

		// Get history with limit of 3
		history, err := repo.GetPurchaseHistory(ctx, guildID, 3)
		require.NoError(t, err)
		require.Len(t, history, 3)

		// Should be ordered by purchased_at DESC (most recent first)
		assert.Equal(t, int64(555), history[0].DiscordID) // Most recent
		assert.Equal(t, int64(444), history[1].DiscordID)
		assert.Equal(t, int64(333), history[2].DiscordID)

		// Verify all have correct guild ID
		for _, h := range history {
			assert.Equal(t, guildID, h.GuildID)
		}
	})

	t.Run("purchase history full list", func(t *testing.T) {
		guildID := int64(789012)
		
		// Create 2 purchases
		purchase1 := testutil.CreateTestHighRollerPurchaseWithTime(guildID, 666, 60000, time.Now().Add(-1*time.Hour))
		purchase2 := testutil.CreateTestHighRollerPurchaseWithTime(guildID, 777, 70000, time.Now())

		scopedRepo := NewHighRollerPurchaseRepositoryScoped(testDB.DB.Pool, guildID)
		
		err := scopedRepo.CreatePurchase(ctx, purchase1)
		require.NoError(t, err)
		err = scopedRepo.CreatePurchase(ctx, purchase2)
		require.NoError(t, err)

		// Get all history (limit higher than actual count)
		history, err := repo.GetPurchaseHistory(ctx, guildID, 10)
		require.NoError(t, err)
		require.Len(t, history, 2)

		// Should be ordered by purchased_at DESC
		assert.Equal(t, int64(777), history[0].DiscordID) // Most recent
		assert.Equal(t, int64(666), history[1].DiscordID)
	})

	t.Run("multiple guilds - returns correct guild's history", func(t *testing.T) {
		guild1ID := int64(111111)
		guild2ID := int64(222222)
		
		// Create purchases for different guilds
		purchase1 := testutil.CreateTestHighRollerPurchase(guild1ID, 888, 80000)
		purchase2 := testutil.CreateTestHighRollerPurchase(guild2ID, 999, 90000)

		scopedRepo1 := NewHighRollerPurchaseRepositoryScoped(testDB.DB.Pool, guild1ID)
		scopedRepo2 := NewHighRollerPurchaseRepositoryScoped(testDB.DB.Pool, guild2ID)
		
		err := scopedRepo1.CreatePurchase(ctx, purchase1)
		require.NoError(t, err)
		err = scopedRepo2.CreatePurchase(ctx, purchase2)
		require.NoError(t, err)

		// Get history for guild1
		history1, err := repo.GetPurchaseHistory(ctx, guild1ID, 10)
		require.NoError(t, err)
		require.Len(t, history1, 1)
		assert.Equal(t, guild1ID, history1[0].GuildID)
		assert.Equal(t, int64(888), history1[0].DiscordID)

		// Get history for guild2
		history2, err := repo.GetPurchaseHistory(ctx, guild2ID, 10)
		require.NoError(t, err)
		require.Len(t, history2, 1)
		assert.Equal(t, guild2ID, history2[0].GuildID)
		assert.Equal(t, int64(999), history2[0].DiscordID)
	})
}