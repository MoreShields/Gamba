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

		// Create users first (required for FK constraint)
		userRepo := NewUserRepository(testDB.DB)
		_, err := userRepo.Create(ctx, 111, "user111", 100000)
		require.NoError(t, err)
		_, err = userRepo.Create(ctx, 222, "user222", 100000)
		require.NoError(t, err)
		_, err = userRepo.Create(ctx, 333, "user333", 100000)
		require.NoError(t, err)

		// Create multiple purchases with different timestamps
		purchase1 := testutil.CreateTestHighRollerPurchaseWithTime(guildID, 111, 10000, time.Now().Add(-2*time.Hour))
		purchase2 := testutil.CreateTestHighRollerPurchaseWithTime(guildID, 222, 20000, time.Now().Add(-1*time.Hour))
		purchase3 := testutil.CreateTestHighRollerPurchaseWithTime(guildID, 333, 30000, time.Now())

		// Create scoped repository for the transaction
		scopedRepo := NewHighRollerPurchaseRepositoryScoped(testDB.DB.Pool, guildID)

		err = scopedRepo.CreatePurchase(ctx, purchase1)
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

		// Create users first (required for FK constraint)
		userRepo := NewUserRepository(testDB.DB)
		_, err := userRepo.Create(ctx, 444, "user444", 100000)
		require.NoError(t, err)
		_, err = userRepo.Create(ctx, 555, "user555", 100000)
		require.NoError(t, err)

		// Create purchases for different guilds
		purchase1 := testutil.CreateTestHighRollerPurchaseWithTime(guild1ID, 444, 40000, time.Now().Add(-1*time.Hour))
		purchase2 := testutil.CreateTestHighRollerPurchaseWithTime(guild2ID, 555, 50000, time.Now())

		scopedRepo1 := NewHighRollerPurchaseRepositoryScoped(testDB.DB.Pool, guild1ID)
		scopedRepo2 := NewHighRollerPurchaseRepositoryScoped(testDB.DB.Pool, guild2ID)

		err = scopedRepo1.CreatePurchase(ctx, purchase1)
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
		// Create user first (required for FK constraint)
		userRepo := NewUserRepository(testDB.DB)
		_, err := userRepo.Create(ctx, 789, "user789", 100000)
		require.NoError(t, err)

		purchase := testutil.CreateTestHighRollerPurchase(guildID, 789, 25000)

		scopedRepo := NewHighRollerPurchaseRepositoryScoped(testDB.DB.Pool, guildID)
		err = scopedRepo.CreatePurchase(ctx, purchase)
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
		// Create users first (required for FK constraint)
		userRepo := NewUserRepository(testDB.DB)
		_, err := userRepo.Create(ctx, 1111, "user1111", 100000)
		require.NoError(t, err)
		_, err = userRepo.Create(ctx, 2222, "user2222", 100000)
		require.NoError(t, err)

		purchase1 := testutil.CreateTestHighRollerPurchaseWithTime(guildID, 1111, 10000, time.Now().Add(-1*time.Hour))
		purchase2 := testutil.CreateTestHighRollerPurchaseWithTime(guildID, 2222, 20000, time.Now())

		scopedRepo := NewHighRollerPurchaseRepositoryScoped(testDB.DB.Pool, guildID)

		err = scopedRepo.CreatePurchase(ctx, purchase1)
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

		// Create users first (required for FK constraint)
		userRepo := NewUserRepository(testDB.DB)
		for _, id := range []int64{1110, 2220, 3330, 4440, 5550} {
			_, err := userRepo.Create(ctx, id, "user", 100000)
			require.NoError(t, err)
		}

		// Create 5 purchases with different timestamps
		baseTime := time.Now()
		type purchaseData struct {
			GuildID   int64
			DiscordID int64
			Price     int64
			Time      time.Time
		}
		purchases := []purchaseData{
			{GuildID: guildID, DiscordID: 1110, Price: 10000, Time: baseTime.Add(-4 * time.Hour)},
			{GuildID: guildID, DiscordID: 2220, Price: 20000, Time: baseTime.Add(-3 * time.Hour)},
			{GuildID: guildID, DiscordID: 3330, Price: 30000, Time: baseTime.Add(-2 * time.Hour)},
			{GuildID: guildID, DiscordID: 4440, Price: 40000, Time: baseTime.Add(-1 * time.Hour)},
			{GuildID: guildID, DiscordID: 5550, Price: 50000, Time: baseTime},
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
		assert.Equal(t, int64(5550), history[0].DiscordID) // Most recent
		assert.Equal(t, int64(4440), history[1].DiscordID)
		assert.Equal(t, int64(3330), history[2].DiscordID)

		// Verify all have correct guild ID
		for _, h := range history {
			assert.Equal(t, guildID, h.GuildID)
		}
	})

	t.Run("purchase history full list", func(t *testing.T) {
		guildID := int64(789012)

		// Create users first (required for FK constraint)
		userRepo := NewUserRepository(testDB.DB)
		_, err := userRepo.Create(ctx, 6660, "user6660", 100000)
		require.NoError(t, err)
		_, err = userRepo.Create(ctx, 7770, "user7770", 100000)
		require.NoError(t, err)

		// Create 2 purchases
		purchase1 := testutil.CreateTestHighRollerPurchaseWithTime(guildID, 6660, 60000, time.Now().Add(-1*time.Hour))
		purchase2 := testutil.CreateTestHighRollerPurchaseWithTime(guildID, 7770, 70000, time.Now())

		scopedRepo := NewHighRollerPurchaseRepositoryScoped(testDB.DB.Pool, guildID)

		err = scopedRepo.CreatePurchase(ctx, purchase1)
		require.NoError(t, err)
		err = scopedRepo.CreatePurchase(ctx, purchase2)
		require.NoError(t, err)

		// Get all history (limit higher than actual count)
		history, err := repo.GetPurchaseHistory(ctx, guildID, 10)
		require.NoError(t, err)
		require.Len(t, history, 2)

		// Should be ordered by purchased_at DESC
		assert.Equal(t, int64(7770), history[0].DiscordID) // Most recent
		assert.Equal(t, int64(6660), history[1].DiscordID)
	})

	t.Run("multiple guilds - returns correct guild's history", func(t *testing.T) {
		guild1ID := int64(111111)
		guild2ID := int64(222222)

		// Create users first (required for FK constraint)
		userRepo := NewUserRepository(testDB.DB)
		_, err := userRepo.Create(ctx, 8880, "user8880", 100000)
		require.NoError(t, err)
		_, err = userRepo.Create(ctx, 9990, "user9990", 100000)
		require.NoError(t, err)

		// Create purchases for different guilds
		purchase1 := testutil.CreateTestHighRollerPurchase(guild1ID, 8880, 80000)
		purchase2 := testutil.CreateTestHighRollerPurchase(guild2ID, 9990, 90000)

		scopedRepo1 := NewHighRollerPurchaseRepositoryScoped(testDB.DB.Pool, guild1ID)
		scopedRepo2 := NewHighRollerPurchaseRepositoryScoped(testDB.DB.Pool, guild2ID)

		err = scopedRepo1.CreatePurchase(ctx, purchase1)
		require.NoError(t, err)
		err = scopedRepo2.CreatePurchase(ctx, purchase2)
		require.NoError(t, err)

		// Get history for guild1
		history1, err := repo.GetPurchaseHistory(ctx, guild1ID, 10)
		require.NoError(t, err)
		require.Len(t, history1, 1)
		assert.Equal(t, guild1ID, history1[0].GuildID)
		assert.Equal(t, int64(8880), history1[0].DiscordID)

		// Get history for guild2
		history2, err := repo.GetPurchaseHistory(ctx, guild2ID, 10)
		require.NoError(t, err)
		require.Len(t, history2, 1)
		assert.Equal(t, guild2ID, history2[0].GuildID)
		assert.Equal(t, int64(9990), history2[0].DiscordID)
	})
}

func TestHighRollerPurchaseRepository_GetUserTotalDurationSince(t *testing.T) {
	t.Parallel()
	testDB := testutil.SetupTestDatabase(t)

	repo := NewHighRollerPurchaseRepository(testDB.DB)
	ctx := context.Background()

	t.Run("no purchases for user", func(t *testing.T) {
		guildID := int64(100001)
		startTime := time.Now().Add(-24 * time.Hour)

		duration, err := repo.GetUserTotalDurationSince(ctx, guildID, 999, startTime)
		require.NoError(t, err)
		assert.Equal(t, time.Duration(0), duration)
	})

	t.Run("single purchase - current holder", func(t *testing.T) {
		guildID := int64(100002)
		userID := int64(11100)
		purchaseTime := time.Now().Add(-2 * time.Hour)
		startTime := time.Now().Add(-24 * time.Hour)

		// Create user first (required for FK constraint)
		userRepo := NewUserRepository(testDB.DB)
		_, err := userRepo.Create(ctx, userID, "user11100", 100000)
		require.NoError(t, err)

		// Create single purchase (user still holds it)
		purchase := testutil.CreateTestHighRollerPurchaseWithTime(guildID, userID, 10000, purchaseTime)
		scopedRepo := NewHighRollerPurchaseRepositoryScoped(testDB.DB.Pool, guildID)
		err = scopedRepo.CreatePurchase(ctx, purchase)
		require.NoError(t, err)

		// Calculate duration
		duration, err := repo.GetUserTotalDurationSince(ctx, guildID, userID, startTime)
		require.NoError(t, err)

		// Should be approximately 2 hours (allow 2 second tolerance for test execution time)
		expectedDuration := 2 * time.Hour
		assert.InDelta(t, expectedDuration.Seconds(), duration.Seconds(), 2.0)
	})

	t.Run("multiple purchases by same user - cumulative", func(t *testing.T) {
		guildID := int64(100003)
		userA := int64(22200)
		userB := int64(33300)
		baseTime := time.Now()
		startTime := baseTime.Add(-5 * time.Hour)

		// Create users first (required for FK constraint)
		userRepo := NewUserRepository(testDB.DB)
		_, err := userRepo.Create(ctx, userA, "userA", 100000)
		require.NoError(t, err)
		_, err = userRepo.Create(ctx, userB, "userB", 100000)
		require.NoError(t, err)

		scopedRepo := NewHighRollerPurchaseRepositoryScoped(testDB.DB.Pool, guildID)

		// User A: Purchase at T-4h, held for 1h until User B took it
		purchase1 := testutil.CreateTestHighRollerPurchaseWithTime(guildID, userA, 10000, baseTime.Add(-4*time.Hour))
		err = scopedRepo.CreatePurchase(ctx, purchase1)
		require.NoError(t, err)

		// User B: Purchase at T-3h, held for 2h until User A took it back
		purchase2 := testutil.CreateTestHighRollerPurchaseWithTime(guildID, userB, 20000, baseTime.Add(-3*time.Hour))
		err = scopedRepo.CreatePurchase(ctx, purchase2)
		require.NoError(t, err)

		// User A: Purchase at T-1h, still holding
		purchase3 := testutil.CreateTestHighRollerPurchaseWithTime(guildID, userA, 30000, baseTime.Add(-1*time.Hour))
		err = scopedRepo.CreatePurchase(ctx, purchase3)
		require.NoError(t, err)

		// User A should have: 1h (first period) + 1h (current period) = 2h total
		durationA, err := repo.GetUserTotalDurationSince(ctx, guildID, userA, startTime)
		require.NoError(t, err)
		assert.InDelta(t, (2 * time.Hour).Seconds(), durationA.Seconds(), 2.0)

		// User B should have: 2h (their period)
		durationB, err := repo.GetUserTotalDurationSince(ctx, guildID, userB, startTime)
		require.NoError(t, err)
		assert.InDelta(t, (2 * time.Hour).Seconds(), durationB.Seconds(), 2.0)
	})

	t.Run("multiple users - isolation", func(t *testing.T) {
		guildID := int64(100004)
		baseTime := time.Now()
		startTime := baseTime.Add(-10 * time.Hour)

		// Create users first (required for FK constraint)
		userRepo := NewUserRepository(testDB.DB)
		for _, id := range []int64{44400, 55500, 66600} {
			_, err := userRepo.Create(ctx, id, "user", 100000)
			require.NoError(t, err)
		}

		scopedRepo := NewHighRollerPurchaseRepositoryScoped(testDB.DB.Pool, guildID)

		// User 1: Held for 3 hours
		purchase1 := testutil.CreateTestHighRollerPurchaseWithTime(guildID, 44400, 10000, baseTime.Add(-9*time.Hour))
		err := scopedRepo.CreatePurchase(ctx, purchase1)
		require.NoError(t, err)

		// User 2: Held for 4 hours
		purchase2 := testutil.CreateTestHighRollerPurchaseWithTime(guildID, 55500, 20000, baseTime.Add(-6*time.Hour))
		err = scopedRepo.CreatePurchase(ctx, purchase2)
		require.NoError(t, err)

		// User 3: Current holder, held for 2 hours
		purchase3 := testutil.CreateTestHighRollerPurchaseWithTime(guildID, 66600, 30000, baseTime.Add(-2*time.Hour))
		err = scopedRepo.CreatePurchase(ctx, purchase3)
		require.NoError(t, err)

		// Verify each user's individual duration
		duration1, err := repo.GetUserTotalDurationSince(ctx, guildID, 44400, startTime)
		require.NoError(t, err)
		assert.InDelta(t, (3 * time.Hour).Seconds(), duration1.Seconds(), 2.0)

		duration2, err := repo.GetUserTotalDurationSince(ctx, guildID, 55500, startTime)
		require.NoError(t, err)
		assert.InDelta(t, (4 * time.Hour).Seconds(), duration2.Seconds(), 2.0)

		duration3, err := repo.GetUserTotalDurationSince(ctx, guildID, 66600, startTime)
		require.NoError(t, err)
		assert.InDelta(t, (2 * time.Hour).Seconds(), duration3.Seconds(), 2.0)
	})

	t.Run("tracking start time filtering", func(t *testing.T) {
		guildID := int64(100005)
		userID := int64(77700)
		baseTime := time.Now()
		startTime := baseTime.Add(-5 * time.Hour)

		// Create users first (required for FK constraint)
		userRepo := NewUserRepository(testDB.DB)
		_, err := userRepo.Create(ctx, userID, "user77700", 100000)
		require.NoError(t, err)
		_, err = userRepo.Create(ctx, 88800, "user88800", 100000)
		require.NoError(t, err)

		scopedRepo := NewHighRollerPurchaseRepositoryScoped(testDB.DB.Pool, guildID)

		// Purchase BEFORE start time (should not count)
		purchase1 := testutil.CreateTestHighRollerPurchaseWithTime(guildID, userID, 10000, baseTime.Add(-10*time.Hour))
		err = scopedRepo.CreatePurchase(ctx, purchase1)
		require.NoError(t, err)

		// Purchase AFTER start time (should count)
		purchase2 := testutil.CreateTestHighRollerPurchaseWithTime(guildID, 88800, 20000, baseTime.Add(-4*time.Hour))
		err = scopedRepo.CreatePurchase(ctx, purchase2)
		require.NoError(t, err)

		// User 77700's purchase is before start time, should return 0
		duration777, err := repo.GetUserTotalDurationSince(ctx, guildID, userID, startTime)
		require.NoError(t, err)
		assert.Equal(t, time.Duration(0), duration777)

		// User 88800's purchase is after start time, should count
		duration888, err := repo.GetUserTotalDurationSince(ctx, guildID, 88800, startTime)
		require.NoError(t, err)
		assert.InDelta(t, (4 * time.Hour).Seconds(), duration888.Seconds(), 2.0)
	})

	t.Run("purchase at exact start time - should be included", func(t *testing.T) {
		guildID := int64(100006)
		userID := int64(99900)
		startTime := time.Now().Add(-3 * time.Hour)

		// Create user first (required for FK constraint)
		userRepo := NewUserRepository(testDB.DB)
		_, err := userRepo.Create(ctx, userID, "user99900", 100000)
		require.NoError(t, err)

		scopedRepo := NewHighRollerPurchaseRepositoryScoped(testDB.DB.Pool, guildID)

		// Purchase at EXACT start time (should be included)
		purchase := testutil.CreateTestHighRollerPurchaseWithTime(guildID, userID, 10000, startTime)
		err = scopedRepo.CreatePurchase(ctx, purchase)
		require.NoError(t, err)

		duration, err := repo.GetUserTotalDurationSince(ctx, guildID, userID, startTime)
		require.NoError(t, err)

		// Should be approximately 3 hours
		assert.InDelta(t, (3 * time.Hour).Seconds(), duration.Seconds(), 2.0)
	})

	t.Run("complex scenario - multiple role changes", func(t *testing.T) {
		guildID := int64(100007)
		baseTime := time.Now()
		startTime := baseTime.Add(-10 * time.Hour)

		// Create users first (required for FK constraint)
		userRepo := NewUserRepository(testDB.DB)
		for _, id := range []int64{11101, 22201, 33301} {
			_, err := userRepo.Create(ctx, id, "user", 100000)
			require.NoError(t, err)
		}

		scopedRepo := NewHighRollerPurchaseRepositoryScoped(testDB.DB.Pool, guildID)

		// Simulate realistic role transfers:
		// User A: T-9h to T-7h (2h)
		// User B: T-7h to T-5h (2h)
		// User C: T-5h to T-4h (1h)
		// User A: T-4h to T-2h (2h) - second period
		// User B: T-2h to now (2h) - second period

		purchases := []struct {
			userID int64
			time   time.Time
		}{
			{11101, baseTime.Add(-9 * time.Hour)}, // User A starts
			{22201, baseTime.Add(-7 * time.Hour)}, // User B takes over
			{33301, baseTime.Add(-5 * time.Hour)}, // User C takes over
			{11101, baseTime.Add(-4 * time.Hour)}, // User A takes back
			{22201, baseTime.Add(-2 * time.Hour)}, // User B takes back
		}

		for i, p := range purchases {
			purchase := testutil.CreateTestHighRollerPurchaseWithTime(guildID, p.userID, int64(10000*(i+1)), p.time)
			err := scopedRepo.CreatePurchase(ctx, purchase)
			require.NoError(t, err)
		}

		// User A: 2h (first) + 2h (second) = 4h total
		durationA, err := repo.GetUserTotalDurationSince(ctx, guildID, 11101, startTime)
		require.NoError(t, err)
		assert.InDelta(t, (4 * time.Hour).Seconds(), durationA.Seconds(), 2.0)

		// User B: 2h (first) + 2h (current) = 4h total
		durationB, err := repo.GetUserTotalDurationSince(ctx, guildID, 22201, startTime)
		require.NoError(t, err)
		assert.InDelta(t, (4 * time.Hour).Seconds(), durationB.Seconds(), 2.0)

		// User C: 1h (only period)
		durationC, err := repo.GetUserTotalDurationSince(ctx, guildID, 33301, startTime)
		require.NoError(t, err)
		assert.InDelta(t, (1 * time.Hour).Seconds(), durationC.Seconds(), 2.0)
	})

	t.Run("user never held the role", func(t *testing.T) {
		guildID := int64(100008)
		startTime := time.Now().Add(-24 * time.Hour)

		// Create user first (required for FK constraint)
		userRepo := NewUserRepository(testDB.DB)
		_, err := userRepo.Create(ctx, 11102, "user11102", 100000)
		require.NoError(t, err)

		scopedRepo := NewHighRollerPurchaseRepositoryScoped(testDB.DB.Pool, guildID)

		// Create purchases for other users
		purchase1 := testutil.CreateTestHighRollerPurchaseWithTime(guildID, 11102, 10000, time.Now().Add(-5*time.Hour))
		err = scopedRepo.CreatePurchase(ctx, purchase1)
		require.NoError(t, err)

		// Query for user who never purchased
		duration, err := repo.GetUserTotalDurationSince(ctx, guildID, 99999, startTime)
		require.NoError(t, err)
		assert.Equal(t, time.Duration(0), duration)
	})
}