package repository

import (
	"context"
	"testing"
	"time"

	"gambler/discord-client/models"
	"gambler/discord-client/repository/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWordleCompletionRepository_Create(t *testing.T) {
	t.Parallel()
	testDB := testutil.SetupTestDatabase(t)

	repo := NewWordleCompletionRepositoryScoped(testDB.DB.Pool, 123456789)
	ctx := context.Background()

	tests := []struct {
		name        string
		setup       func()
		completion  *models.WordleCompletion
		wantErr     bool
		errContains string
	}{
		{
			name: "successful creation",
			completion: func() *models.WordleCompletion {
				score, _ := models.NewWordleScore(3, 6)
				comp, _ := models.NewWordleCompletion(987654321, 123456789, score, time.Now())
				return comp
			}(),
			wantErr: false,
		},
		{
			name: "create with perfect score",
			completion: func() *models.WordleCompletion {
				score, _ := models.NewWordleScore(1, 6)
				comp, _ := models.NewWordleCompletion(987654322, 123456789, score, time.Now())
				return comp
			}(),
			wantErr: false,
		},
		{
			name: "create with max guesses",
			completion: func() *models.WordleCompletion {
				score, _ := models.NewWordleScore(6, 6)
				comp, _ := models.NewWordleCompletion(987654323, 123456789, score, time.Now())
				return comp
			}(),
			wantErr: false,
		},
		{
			name: "duplicate entry for same day",
			setup: func() {
				// Create an initial completion for today
				score, _ := models.NewWordleScore(3, 6)
				comp, _ := models.NewWordleCompletion(111111111, 123456789, score, time.Now())
				err := repo.Create(ctx, comp)
				require.NoError(t, err)
			},
			completion: func() *models.WordleCompletion {
				// Try to create another completion for the same user/guild/day
				score, _ := models.NewWordleScore(4, 6)
				comp, _ := models.NewWordleCompletion(111111111, 123456789, score, time.Now())
				return comp
			}(),
			wantErr:     true,
			errContains: "wordle_completions_discord_id_guild_id_date_unique",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			err := repo.Create(ctx, tt.completion)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
				assert.NotZero(t, tt.completion.ID)
				assert.False(t, tt.completion.CreatedAt.IsZero())
			}
		})
	}
}

func TestWordleCompletionRepository_GetByUserToday(t *testing.T) {
	t.Parallel()
	testDB := testutil.SetupTestDatabase(t)

	repo := NewWordleCompletionRepositoryScoped(testDB.DB.Pool, 123456789)
	ctx := context.Background()

	t.Run("no completion found", func(t *testing.T) {
		completion, err := repo.GetByUserToday(ctx, 999999999, 0)
		require.NoError(t, err)
		assert.Nil(t, completion)
	})

	t.Run("completion found for today", func(t *testing.T) {
		// Create a completion for today with a unique discord ID
		uniqueDiscordID := int64(222222222)
		score, _ := models.NewWordleScore(3, 6)
		// Use UTC to ensure consistent timezone handling
		now := time.Now().UTC()
		testCompletion, _ := models.NewWordleCompletion(uniqueDiscordID, 999999999, score, now) // guild ID will be overridden
		err := repo.Create(ctx, testCompletion)
		require.NoError(t, err)
		// Repository now updates the completion's guild ID to match the scoped value

		t.Logf("Created completion with ID %d, DiscordID %d, GuildID %d, CompletedAt %v",
			testCompletion.ID, testCompletion.DiscordID, testCompletion.GuildID, testCompletion.CompletedAt)

		// Check if we can query it directly first
		var count int
		err = testDB.DB.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM wordle_completions WHERE discord_id = $1 AND guild_id = $2", uniqueDiscordID, 123456789).Scan(&count)
		require.NoError(t, err)
		t.Logf("Direct query found %d completions for discord_id=%d, guild_id=%d", count, uniqueDiscordID, 123456789)

		// Retrieve it - note that GetByUserToday uses r.guildID internally
		completion, err := repo.GetByUserToday(ctx, uniqueDiscordID, 0)
		require.NoError(t, err)
		require.NotNil(t, completion, "Expected to find completion for discord ID %d in guild 123456789", uniqueDiscordID)

		assert.Equal(t, testCompletion.DiscordID, completion.DiscordID)
		assert.Equal(t, testCompletion.GuildID, completion.GuildID) // Both should be 123456789
		assert.Equal(t, testCompletion.Score.Guesses, completion.Score.Guesses)
		assert.Equal(t, testCompletion.Score.MaxGuesses, completion.Score.MaxGuesses)
	})

	t.Run("completion from yesterday not returned", func(t *testing.T) {
		// Create a completion for yesterday
		yesterday := time.Now().AddDate(0, 0, -1)
		score, _ := models.NewWordleScore(4, 6)
		testCompletion, _ := models.NewWordleCompletion(333333333, 123456789, score, yesterday)
		err := repo.Create(ctx, testCompletion)
		require.NoError(t, err)

		// Should not find it when querying for today
		completion, err := repo.GetByUserToday(ctx, 333333333, 0)
		require.NoError(t, err)
		assert.Nil(t, completion)
	})

	t.Run("different guild isolation", func(t *testing.T) {
		// Create completion in guild 123456789
		score, _ := models.NewWordleScore(2, 6)
		testCompletion, _ := models.NewWordleCompletion(444444444, 123456789, score, time.Now())
		err := repo.Create(ctx, testCompletion)
		require.NoError(t, err)

		// Try to retrieve from different guild
		repoOtherGuild := NewWordleCompletionRepositoryScoped(testDB.DB.Pool, 987654321)
		// Note: guildID parameter is ignored, repository uses its scoped guild ID
		completion, err := repoOtherGuild.GetByUserToday(ctx, 444444444, 0) // guildID param is ignored
		require.NoError(t, err)
		assert.Nil(t, completion)
	})
}

func TestWordleCompletionRepository_GetRecentCompletions(t *testing.T) {
	t.Parallel()
	testDB := testutil.SetupTestDatabase(t)

	repo := NewWordleCompletionRepositoryScoped(testDB.DB.Pool, 123456789)
	ctx := context.Background()

	t.Run("no completions", func(t *testing.T) {
		completions, err := repo.GetRecentCompletions(ctx, 555555555, 0, 10)
		require.NoError(t, err)
		assert.Empty(t, completions)
	})

	t.Run("multiple completions ordered by date", func(t *testing.T) {
		userID := int64(666666666)
		guildID := int64(123456789)

		// Create completions for different days
		now := time.Now()
		dates := []time.Time{
			now,
			now.AddDate(0, 0, -1),
			now.AddDate(0, 0, -2),
			now.AddDate(0, 0, -3),
			now.AddDate(0, 0, -4),
		}

		for i, date := range dates {
			score, _ := models.NewWordleScore(i+1, 6)
			completion, _ := models.NewWordleCompletion(userID, guildID, score, date)
			err := repo.Create(ctx, completion)
			require.NoError(t, err)
		}

		// Get recent completions
		completions, err := repo.GetRecentCompletions(ctx, userID, 0, 3)
		require.NoError(t, err)
		assert.Len(t, completions, 3)

		// Should be ordered by completed_at DESC (newest first)
		assert.Equal(t, 1, completions[0].Score.Guesses) // Today
		assert.Equal(t, 2, completions[1].Score.Guesses) // Yesterday
		assert.Equal(t, 3, completions[2].Score.Guesses) // 2 days ago
	})

	t.Run("limit parameter respected", func(t *testing.T) {
		userID := int64(777777777)
		guildID := int64(123456789)

		// Create 10 completions
		for i := 0; i < 10; i++ {
			date := time.Now().AddDate(0, 0, -i)
			score, _ := models.NewWordleScore((i%6)+1, 6)
			completion, _ := models.NewWordleCompletion(userID, guildID, score, date)
			err := repo.Create(ctx, completion)
			require.NoError(t, err)
		}

		// Test different limits
		limits := []int{1, 5, 10, 20}
		for _, limit := range limits {
			completions, err := repo.GetRecentCompletions(ctx, userID, 0, limit)
			require.NoError(t, err)
			expectedCount := limit
			if expectedCount > 10 {
				expectedCount = 10
			}
			assert.Len(t, completions, expectedCount, "limit %d should return %d completions", limit, expectedCount)
		}
	})

	t.Run("guild isolation", func(t *testing.T) {
		userID := int64(888888888)

		// Create completions in different guilds
		guilds := []int64{123456789, 987654321, 111111111}
		for _, guildID := range guilds {
			repoForGuild := NewWordleCompletionRepositoryScoped(testDB.DB.Pool, guildID)
			for i := 0; i < 3; i++ {
				date := time.Now().AddDate(0, 0, -i)
				score, _ := models.NewWordleScore(i+1, 6)
				completion, _ := models.NewWordleCompletion(userID, guildID, score, date)
				err := repoForGuild.Create(ctx, completion)
				require.NoError(t, err)
			}
		}

		// Each guild should only see its own completions
		for _, guildID := range guilds {
			repoForGuild := NewWordleCompletionRepositoryScoped(testDB.DB.Pool, guildID)
			completions, err := repoForGuild.GetRecentCompletions(ctx, userID, 0, 10)
			require.NoError(t, err)
			assert.Len(t, completions, 3)

			// Verify all completions belong to the correct guild
			for _, comp := range completions {
				assert.Equal(t, guildID, comp.GuildID)
			}
		}
	})
}

func TestWordleCompletionRepository_ScopedRepository(t *testing.T) {
	t.Parallel()
	testDB := testutil.SetupTestDatabase(t)
	ctx := context.Background()

	t.Run("scoped repository uses correct guild ID", func(t *testing.T) {
		guildID := int64(999999999)
		repo := NewWordleCompletionRepositoryScoped(testDB.DB.Pool, guildID)

		// Create a completion with a different guild ID in the model
		score, _ := models.NewWordleScore(3, 6)
		now := time.Now().UTC()
		completion, _ := models.NewWordleCompletion(123123123, 111111111, score, now)

		// The repository should use its scoped guild ID, not the model's
		err := repo.Create(ctx, completion)
		require.NoError(t, err)

		t.Logf("Created completion with ID %d, DiscordID %d, GuildID %d (should be %d)",
			completion.ID, completion.DiscordID, completion.GuildID, guildID)

		// Verify it was created with the repository's guild ID
		// Note: GetByUserToday uses r.guildID internally, ignoring the parameter
		retrieved, err := repo.GetByUserToday(ctx, 123123123, 0)
		require.NoError(t, err)
		require.NotNil(t, retrieved, "Expected to find completion for discord ID 123123123 in guild %d", guildID)
		assert.Equal(t, guildID, retrieved.GuildID)
	})
}

func TestWordleCompletionRepository_EdgeCases(t *testing.T) {
	t.Parallel()
	testDB := testutil.SetupTestDatabase(t)

	repo := NewWordleCompletionRepositoryScoped(testDB.DB.Pool, 123456789)
	ctx := context.Background()

	t.Run("handle different max guesses values", func(t *testing.T) {
		// Test with different valid max guesses (1-6)
		for maxGuesses := 1; maxGuesses <= 6; maxGuesses++ {
			userID := int64(100000000 + maxGuesses)
			score, err := models.NewWordleScore(maxGuesses, maxGuesses)
			require.NoError(t, err)

			now := time.Now().UTC()
			completion, err := models.NewWordleCompletion(userID, 123456789, score, now)
			require.NoError(t, err)

			err = repo.Create(ctx, completion)
			require.NoError(t, err)

			retrieved, err := repo.GetByUserToday(ctx, userID, 0)
			require.NoError(t, err)
			require.NotNil(t, retrieved, "Expected to find completion for user %d with max guesses %d", userID, maxGuesses)
			assert.Equal(t, maxGuesses, retrieved.Score.MaxGuesses)
		}
	})

	t.Run("zero limit returns empty slice", func(t *testing.T) {
		completions, err := repo.GetRecentCompletions(ctx, 999999999, 0, 0)
		require.NoError(t, err)
		assert.Empty(t, completions)
	})

	t.Run("completions at midnight boundary", func(t *testing.T) {
		userID := int64(234234234)

		// Use UTC for consistent timezone handling
		now := time.Now().UTC()

		// Create completion at 23:59:59 yesterday
		yesterday := now.AddDate(0, 0, -1)
		almostMidnight := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 23, 59, 59, 0, time.UTC)
		score1, _ := models.NewWordleScore(3, 6)
		completion1, _ := models.NewWordleCompletion(userID, 123456789, score1, almostMidnight)
		err := repo.Create(ctx, completion1)
		require.NoError(t, err)

		// Create completion at 00:00:01 today
		justAfterMidnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 1, 0, time.UTC)
		score2, _ := models.NewWordleScore(4, 6)
		completion2, _ := models.NewWordleCompletion(userID+1, 123456789, score2, justAfterMidnight)
		err = repo.Create(ctx, completion2)
		require.NoError(t, err)

		// Yesterday's completion should not be returned for today
		// Note: guildID parameter is ignored in scoped repository
		yesterdayResult, err := repo.GetByUserToday(ctx, userID, 0)
		require.NoError(t, err)
		assert.Nil(t, yesterdayResult)

		// Today's completion should be returned
		todayResult, err := repo.GetByUserToday(ctx, userID+1, 0)
		require.NoError(t, err)
		require.NotNil(t, todayResult, "Expected to find today's completion for user %d", userID+1)
		assert.Equal(t, userID+1, todayResult.DiscordID)
	})
}
