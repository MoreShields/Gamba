package repository

import (
	"context"
	"testing"

	"gambler/discord-client/repository/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSummonerWatchRepository_CreateWatch(t *testing.T) {
	t.Parallel()
	testDB := testutil.SetupTestDatabase(t)

	repo := NewSummonerWatchRepository(testDB.DB)
	ctx := context.Background()

	t.Run("successful creation with new summoner", func(t *testing.T) {
		guildID := int64(12345)
		summonerName := "TestSummoner"
		tagLine := "gamba"

		watch, err := repo.CreateWatch(ctx, guildID, summonerName, tagLine)
		require.NoError(t, err)
		require.NotNil(t, watch)

		assert.Equal(t, guildID, watch.GuildID)
		assert.Equal(t, "testsummoner", watch.SummonerName) // summoner names normalized to lowercase
		assert.Equal(t, tagLine, watch.TagLine)
		assert.False(t, watch.WatchedAt.IsZero())
		assert.False(t, watch.CreatedAt.IsZero())
		assert.False(t, watch.UpdatedAt.IsZero())
	})

	t.Run("successful creation with existing summoner", func(t *testing.T) {
		guildID1 := int64(11111)
		guildID2 := int64(22222)
		summonerName := "ExistingSummoner"
		tagLine := "test"

		// Create first watch
		watch1, err := repo.CreateWatch(ctx, guildID1, summonerName, tagLine)
		require.NoError(t, err)

		// Create second watch with same summoner, different guild
		watch2, err := repo.CreateWatch(ctx, guildID2, summonerName, tagLine)
		require.NoError(t, err)

		// Same summoner, different watches
		assert.Equal(t, watch1.SummonerName, watch2.SummonerName)
		assert.Equal(t, watch1.TagLine, watch2.TagLine)
		assert.NotEqual(t, watch1.GuildID, watch2.GuildID)
	})

	t.Run("duplicate watch accepted", func(t *testing.T) {
		guildID := int64(33333)
		summonerName := "DuplicateTest"
		tagLine := "NA1"

		// Create first watch
		_, err := repo.CreateWatch(ctx, guildID, summonerName, tagLine)
		require.NoError(t, err)

		// Try to create duplicate watch
		_, err = repo.CreateWatch(ctx, guildID, summonerName, tagLine)
		require.NoError(t, err)
	})

	t.Run("different tagLines create separate summoners", func(t *testing.T) {
		guildID := int64(44444)
		summonerName := "MultiRegionSummoner"
		tagLine1 := "NA1"
		tagLine2 := "EUW1"

		watch1, err := repo.CreateWatch(ctx, guildID, summonerName, tagLine1)
		require.NoError(t, err)

		watch2, err := repo.CreateWatch(ctx, guildID, summonerName, tagLine2)
		require.NoError(t, err)

		// Same summoner name, different tagLines
		assert.Equal(t, watch1.SummonerName, watch2.SummonerName)
		assert.NotEqual(t, watch1.TagLine, watch2.TagLine)
	})
}

func TestSummonerWatchRepository_GetWatchesByGuild(t *testing.T) {
	t.Parallel()
	testDB := testutil.SetupTestDatabase(t)

	repo := NewSummonerWatchRepository(testDB.DB)
	ctx := context.Background()

	t.Run("empty guild returns no watches", func(t *testing.T) {
		guildID := int64(99999)

		watches, err := repo.GetWatchesByGuild(ctx, guildID)
		require.NoError(t, err)
		assert.Empty(t, watches)
	})

	t.Run("guild with multiple watches", func(t *testing.T) {
		guildID := int64(55555)

		// Create multiple watches for the guild
		watch1, err := repo.CreateWatch(ctx, guildID, "Summoner1", "NA1")
		require.NoError(t, err)

		watch2, err := repo.CreateWatch(ctx, guildID, "Summoner2", "NA1")
		require.NoError(t, err)

		watch3, err := repo.CreateWatch(ctx, guildID, "Summoner3", "EUW1")
		require.NoError(t, err)

		// Get all watches for the guild
		watches, err := repo.GetWatchesByGuild(ctx, guildID)
		require.NoError(t, err)
		assert.Len(t, watches, 3)

		// Verify watches are returned in descending order by creation time
		assert.True(t, watches[0].WatchedAt.After(watches[1].WatchedAt) || watches[0].WatchedAt.Equal(watches[1].WatchedAt))
		assert.True(t, watches[1].WatchedAt.After(watches[2].WatchedAt) || watches[1].WatchedAt.Equal(watches[2].WatchedAt))

		// Find our created watches
		var foundWatch1, foundWatch2, foundWatch3 bool
		for _, w := range watches {
			switch w.SummonerName {
			case "summoner1":
				assert.Equal(t, watch1.TagLine, w.TagLine)
				foundWatch1 = true
			case "summoner2":
				assert.Equal(t, watch2.TagLine, w.TagLine)
				foundWatch2 = true
			case "summoner3":
				assert.Equal(t, watch3.TagLine, w.TagLine)
				foundWatch3 = true
			}
		}
		assert.True(t, foundWatch1 && foundWatch2 && foundWatch3)
	})

	t.Run("guild isolation - watches from other guilds not returned", func(t *testing.T) {
		guildID1 := int64(66666)
		guildID2 := int64(77777)

		// Create watches for guild1
		_, err := repo.CreateWatch(ctx, guildID1, "Guild1Summoner1", "NA1")
		require.NoError(t, err)
		_, err = repo.CreateWatch(ctx, guildID1, "Guild1Summoner2", "NA1")
		require.NoError(t, err)

		// Create watches for guild2
		_, err = repo.CreateWatch(ctx, guildID2, "Guild2Summoner1", "NA1")
		require.NoError(t, err)

		// Get watches for guild1 only
		watches1, err := repo.GetWatchesByGuild(ctx, guildID1)
		require.NoError(t, err)
		assert.Len(t, watches1, 2)

		// Get watches for guild2 only
		watches2, err := repo.GetWatchesByGuild(ctx, guildID2)
		require.NoError(t, err)
		assert.Len(t, watches2, 1)

		// Verify no cross-contamination
		for _, w := range watches1 {
			assert.Equal(t, guildID1, w.GuildID)
		}
		for _, w := range watches2 {
			assert.Equal(t, guildID2, w.GuildID)
		}
	})
}

func TestSummonerWatchRepository_GetGuildsWatchingSummoner(t *testing.T) {
	t.Parallel()
	testDB := testutil.SetupTestDatabase(t)

	repo := NewSummonerWatchRepository(testDB.DB)
	ctx := context.Background()

	t.Run("summoner not watched by any guild", func(t *testing.T) {
		summonerName := "UnwatchedSummoner"
		tagLine := "NA1"

		watches, err := repo.GetGuildsWatchingSummoner(ctx, summonerName, tagLine)
		require.NoError(t, err)
		assert.Empty(t, watches)
	})

	t.Run("summoner watched by multiple guilds", func(t *testing.T) {
		summonerName := "PopularSummoner"
		tagLine := "NA1"
		guild1 := int64(11111)
		guild2 := int64(22222)
		guild3 := int64(33333)

		// Create watches from multiple guilds for the same summoner
		_, err := repo.CreateWatch(ctx, guild1, summonerName, tagLine)
		require.NoError(t, err)

		_, err = repo.CreateWatch(ctx, guild2, summonerName, tagLine)
		require.NoError(t, err)

		_, err = repo.CreateWatch(ctx, guild3, summonerName, tagLine)
		require.NoError(t, err)

		// Get all guilds watching this summoner
		watches, err := repo.GetGuildsWatchingSummoner(ctx, summonerName, tagLine)
		require.NoError(t, err)
		assert.Len(t, watches, 3)

		// Verify all guilds are represented
		guildIDs := make(map[int64]bool)
		for _, w := range watches {
			guildIDs[w.GuildID] = true
		}
		assert.True(t, guildIDs[guild1])
		assert.True(t, guildIDs[guild2])
		assert.True(t, guildIDs[guild3])
	})

	t.Run("tagLine isolation", func(t *testing.T) {
		summonerName := "RegionTestSummoner"
		tagLine1 := "NA1"
		tagLine2 := "EUW1"
		guildID := int64(88888)

		// Create watches for same summoner name in different tagLines
		_, err := repo.CreateWatch(ctx, guildID, summonerName, tagLine1)
		require.NoError(t, err)

		_, err = repo.CreateWatch(ctx, guildID, summonerName, tagLine2)
		require.NoError(t, err)

		// Get watches for tagLine1 only
		watches1, err := repo.GetGuildsWatchingSummoner(ctx, summonerName, tagLine1)
		require.NoError(t, err)
		assert.Len(t, watches1, 1)

		// Get watches for tagLine2 only
		watches2, err := repo.GetGuildsWatchingSummoner(ctx, summonerName, tagLine2)
		require.NoError(t, err)
		assert.Len(t, watches2, 1)

		// Verify different summoner IDs (different tagLines result in different summoners)
		assert.NotEqual(t, watches1[0].SummonerID, watches2[0].SummonerID)
	})
}

func TestSummonerWatchRepository_DeleteWatch(t *testing.T) {
	t.Parallel()
	testDB := testutil.SetupTestDatabase(t)

	repo := NewSummonerWatchRepository(testDB.DB)
	ctx := context.Background()

	t.Run("successful deletion", func(t *testing.T) {
		guildID := int64(12345)
		summonerName := "ToBeDeleted"
		tagLine := "NA1"

		// Create watch
		_, err := repo.CreateWatch(ctx, guildID, summonerName, tagLine)
		require.NoError(t, err)

		// Verify it exists
		watch, err := repo.GetWatch(ctx, guildID, summonerName, tagLine)
		require.NoError(t, err)
		require.NotNil(t, watch)

		// Delete it
		err = repo.DeleteWatch(ctx, guildID, summonerName, tagLine)
		require.NoError(t, err)

		// Verify it's gone
		watch, err = repo.GetWatch(ctx, guildID, summonerName, tagLine)
		require.NoError(t, err)
		assert.Nil(t, watch)
	})

	t.Run("delete non-existent watch", func(t *testing.T) {
		guildID := int64(99999)
		summonerName := "NonExistent"
		tagLine := "NA1"

		err := repo.DeleteWatch(ctx, guildID, summonerName, tagLine)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no watch found")
	})

	t.Run("delete only affects specific guild-summoner combination", func(t *testing.T) {
		summonerName := "SharedSummoner"
		tagLine := "NA1"
		guild1 := int64(11111)
		guild2 := int64(22222)

		// Create watches from two guilds for same summoner
		_, err := repo.CreateWatch(ctx, guild1, summonerName, tagLine)
		require.NoError(t, err)

		_, err = repo.CreateWatch(ctx, guild2, summonerName, tagLine)
		require.NoError(t, err)

		// Delete watch from guild1 only
		err = repo.DeleteWatch(ctx, guild1, summonerName, tagLine)
		require.NoError(t, err)

		// Verify guild1 watch is gone
		watch1, err := repo.GetWatch(ctx, guild1, summonerName, tagLine)
		require.NoError(t, err)
		assert.Nil(t, watch1)

		// Verify guild2 watch still exists
		watch2, err := repo.GetWatch(ctx, guild2, summonerName, tagLine)
		require.NoError(t, err)
		assert.NotNil(t, watch2)
	})
}

func TestSummonerWatchRepository_GetWatch(t *testing.T) {
	t.Parallel()
	testDB := testutil.SetupTestDatabase(t)

	repo := NewSummonerWatchRepository(testDB.DB)
	ctx := context.Background()

	t.Run("watch not found", func(t *testing.T) {
		guildID := int64(99999)
		summonerName := "NonExistent"
		tagLine := "NA1"

		watch, err := repo.GetWatch(ctx, guildID, summonerName, tagLine)
		require.NoError(t, err)
		assert.Nil(t, watch)
	})

	t.Run("watch found", func(t *testing.T) {
		guildID := int64(12345)
		summonerName := "FoundSummoner"
		tagLine := "NA1"

		// Create watch
		createdWatch, err := repo.CreateWatch(ctx, guildID, summonerName, tagLine)
		require.NoError(t, err)

		// Retrieve it
		foundWatch, err := repo.GetWatch(ctx, guildID, summonerName, tagLine)
		require.NoError(t, err)
		require.NotNil(t, foundWatch)

		assert.Equal(t, createdWatch.GuildID, foundWatch.GuildID)
		assert.Equal(t, createdWatch.SummonerName, foundWatch.SummonerName)
		assert.Equal(t, createdWatch.TagLine, foundWatch.TagLine)
	})

	t.Run("guild isolation", func(t *testing.T) {
		summonerName := "IsolationTestSummoner"
		tagLine := "NA1"
		guild1 := int64(11111)
		guild2 := int64(22222)

		// Create watch for guild1
		_, err := repo.CreateWatch(ctx, guild1, summonerName, tagLine)
		require.NoError(t, err)

		// Try to get watch for guild2 (should not exist)
		watch, err := repo.GetWatch(ctx, guild2, summonerName, tagLine)
		require.NoError(t, err)
		assert.Nil(t, watch)

		// Verify guild1 watch exists
		watch, err = repo.GetWatch(ctx, guild1, summonerName, tagLine)
		require.NoError(t, err)
		assert.NotNil(t, watch)
	})

	t.Run("tagLine isolation", func(t *testing.T) {
		guildID := int64(33333)
		summonerName := "RegionIsolationSummoner"
		tagLine1 := "NA1"
		tagLine2 := "EUW1"

		// Create watch for tagLine1
		_, err := repo.CreateWatch(ctx, guildID, summonerName, tagLine1)
		require.NoError(t, err)

		// Try to get watch for tagLine2 (should not exist)
		watch, err := repo.GetWatch(ctx, guildID, summonerName, tagLine2)
		require.NoError(t, err)
		assert.Nil(t, watch)

		// Verify tagLine1 watch exists
		watch, err = repo.GetWatch(ctx, guildID, summonerName, tagLine1)
		require.NoError(t, err)
		assert.NotNil(t, watch)
	})
}
