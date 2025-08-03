package application_test

import (
	"context"
	"fmt"
	"testing"

	"gambler/discord-client/application"
	"gambler/discord-client/application/dto"
	"gambler/discord-client/domain/entities"
	"gambler/discord-client/infrastructure"
	"gambler/discord-client/repository/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTFTHandler_EndToEndFlow(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test database
	testDB := testutil.SetupTestDatabase(t)

	// Create no-op event publisher for integration tests
	noopPublisher := infrastructure.NewNoopEventPublisher()

	// Create UoW factory
	uowFactory := infrastructure.NewUnitOfWorkFactory(testDB.DB, noopPublisher)

	// Setup test data
	ctx := context.Background()
	guildID := int64(12345)
	summonerName := "TFTPlayer"
	tagLine := "NA1"
	gameID := "tft-game-123"

	// Setup guild and summoner watch with TFT channel
	setupTFTTestData(t, ctx, uowFactory, guildID, summonerName, tagLine)

	// Create mock Discord poster
	mockPoster := &application.MockDiscordPoster{}

	// Create TFT handler
	handler := application.NewTFTHandler(uowFactory, mockPoster)

	t.Run("Game Start Creates TFT House Wager", func(t *testing.T) {
		// Don't use t.Parallel() in sub-tests when parent cleans up resources
		// TFT game start event
		gameStarted := dto.TFTGameStartedDTO{
			SummonerName: summonerName,
			TagLine:      tagLine,
			GameID:       gameID,
			QueueType:    "RANKED_TFT",
		}

		// Handle game start
		err := handler.HandleGameStarted(ctx, gameStarted)
		require.NoError(t, err)

		// Verify Discord post was made
		assert.Len(t, mockPoster.Posts, 1)
		post := mockPoster.Posts[0]
		assert.Equal(t, guildID, post.GuildID)
		assert.Contains(t, post.Title, summonerName)
		assert.Contains(t, post.Title, "TFT Match")
		assert.Len(t, post.Options, 2) // Top 4/Bottom 4 options

		// Verify TFT-specific options
		assert.Equal(t, "Top 4", post.Options[0].Text)
		assert.Equal(t, "Bottom 4", post.Options[1].Text)
		assert.Equal(t, float64(2.0), post.Options[0].Multiplier)
		assert.Equal(t, float64(2.0), post.Options[1].Multiplier)

		// Verify wager was created in database
		uow := uowFactory.CreateForGuild(guildID)
		require.NoError(t, uow.Begin(ctx))
		defer uow.Rollback()

		externalRef := entities.ExternalReference{
			System: entities.SystemTFT,
			ID:     gameID,
		}

		wager, err := uow.GroupWagerRepository().GetByExternalReference(ctx, externalRef)
		require.NoError(t, err)
		require.NotNil(t, wager)

		assert.Equal(t, entities.GroupWagerTypeHouse, wager.WagerType)
		assert.Equal(t, entities.GroupWagerStateActive, wager.State)
		assert.Equal(t, guildID, wager.GuildID)
		assert.NotNil(t, wager.ExternalRef)
		assert.Equal(t, gameID, wager.ExternalRef.ID)
		assert.Equal(t, entities.SystemTFT, wager.ExternalRef.System)
	})

	t.Run("Game End Resolves TFT House Wager - Top 4", func(t *testing.T) {
		// Don't use t.Parallel() in sub-tests when parent cleans up resources
		// TFT game end event - player gets 3rd place (Top 4)
		gameEnded := dto.TFTGameEndedDTO{
			SummonerName:    summonerName,
			TagLine:         tagLine,
			GameID:          gameID,
			Placement:       3, // Top 4
			DurationSeconds: 1800, // 30 minutes
			QueueType:       "RANKED_TFT",
		}

		// Handle game end
		err := handler.HandleGameEnded(ctx, gameEnded)
		require.NoError(t, err)

		// Verify wager was resolved
		uow := uowFactory.CreateForGuild(guildID)
		require.NoError(t, uow.Begin(ctx))
		defer uow.Rollback()

		externalRef := entities.ExternalReference{
			System: entities.SystemTFT,
			ID:     gameID,
		}

		wager, err := uow.GroupWagerRepository().GetByExternalReference(ctx, externalRef)
		require.NoError(t, err)
		require.NotNil(t, wager)

		assert.Equal(t, entities.GroupWagerStateResolved, wager.State)
		assert.NotNil(t, wager.WinningOptionID)

		// Verify the correct option won (Top 4 option)
		detail, err := uow.GroupWagerRepository().GetDetailByID(ctx, wager.ID)
		require.NoError(t, err)
		require.NotNil(t, detail)

		var winOption *entities.GroupWagerOption
		for _, opt := range detail.Options {
			if opt.ID == *wager.WinningOptionID {
				winOption = opt
				break
			}
		}
		require.NotNil(t, winOption)
		assert.Equal(t, "Top 4", winOption.OptionText)
	})
}

func TestTFTHandler_EndToEndFlow_Bottom4(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test database
	testDB := testutil.SetupTestDatabase(t)

	// Create no-op event publisher for integration tests
	noopPublisher := infrastructure.NewNoopEventPublisher()

	// Create UoW factory
	uowFactory := infrastructure.NewUnitOfWorkFactory(testDB.DB, noopPublisher)

	// Setup test data
	ctx := context.Background()
	guildID := int64(54321)
	summonerName := "TFTPlayer2"
	tagLine := "EUW1"
	gameID := "tft-game-456"

	// Setup guild and summoner watch
	setupTFTTestData(t, ctx, uowFactory, guildID, summonerName, tagLine)

	// Create mock Discord poster
	mockPoster := &application.MockDiscordPoster{}

	// Create TFT handler
	handler := application.NewTFTHandler(uowFactory, mockPoster)

	// Game start
	gameStarted := dto.TFTGameStartedDTO{
		SummonerName: summonerName,
		TagLine:      tagLine,
		GameID:       gameID,
		QueueType:    "RANKED_TFT",
	}

	err := handler.HandleGameStarted(ctx, gameStarted)
	require.NoError(t, err)

	// Game end - player gets 6th place (Bottom 4)
	gameEnded := dto.TFTGameEndedDTO{
		SummonerName:    summonerName,
		TagLine:         tagLine,
		GameID:          gameID,
		Placement:       6, // Bottom 4
		DurationSeconds: 1200, // 20 minutes
		QueueType:       "RANKED_TFT",
	}

	err = handler.HandleGameEnded(ctx, gameEnded)
	require.NoError(t, err)

	// Verify wager was resolved with Bottom 4 option
	uow := uowFactory.CreateForGuild(guildID)
	require.NoError(t, uow.Begin(ctx))
	defer uow.Rollback()

	externalRef := entities.ExternalReference{
		System: entities.SystemTFT,
		ID:     gameID,
	}

	wager, err := uow.GroupWagerRepository().GetByExternalReference(ctx, externalRef)
	require.NoError(t, err)
	require.NotNil(t, wager)

	assert.Equal(t, entities.GroupWagerStateResolved, wager.State)
	assert.NotNil(t, wager.WinningOptionID)

	// Verify the correct option won (Bottom 4 option)
	detail, err := uow.GroupWagerRepository().GetDetailByID(ctx, wager.ID)
	require.NoError(t, err)
	require.NotNil(t, detail)

	var winOption *entities.GroupWagerOption
	for _, opt := range detail.Options {
		if opt.ID == *wager.WinningOptionID {
			winOption = opt
			break
		}
	}
	require.NotNil(t, winOption)
	assert.Equal(t, "Bottom 4", winOption.OptionText)
}

func TestTFTHandler_PlacementBoundaries(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testCases := []struct {
		name              string
		placement         int32
		expectedWinOption string
	}{
		{"1st place", 1, "Top 4"},
		{"4th place exactly", 4, "Top 4"},
		{"5th place", 5, "Bottom 4"},
		{"8th place", 8, "Bottom 4"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// Setup test database
			testDB := testutil.SetupTestDatabase(t)
			defer testDB.Cleanup(t)

			// Create no-op event publisher for integration tests
			noopPublisher := infrastructure.NewNoopEventPublisher()

			// Create UoW factory
			uowFactory := infrastructure.NewUnitOfWorkFactory(testDB.DB, noopPublisher)

			// Setup test data
			ctx := context.Background()
			guildID := int64(77777 + int64(tc.placement)) // Unique guild per test
			summonerName := "BoundaryPlayer"
			tagLine := "TEST"
			gameID := fmt.Sprintf("tft-boundary-%d", tc.placement)

			// Setup guild and summoner watch
			setupTFTTestData(t, ctx, uowFactory, guildID, summonerName, tagLine)

			// Create mock Discord poster
			mockPoster := &application.MockDiscordPoster{}

			// Create TFT handler
			handler := application.NewTFTHandler(uowFactory, mockPoster)

			// Game start
			gameStarted := dto.TFTGameStartedDTO{
				SummonerName: summonerName,
				TagLine:      tagLine,
				GameID:       gameID,
				QueueType:    "RANKED_TFT",
			}

			err := handler.HandleGameStarted(ctx, gameStarted)
			require.NoError(t, err)

			// Game end with specific placement
			gameEnded := dto.TFTGameEndedDTO{
				SummonerName:    summonerName,
				TagLine:         tagLine,
				GameID:          gameID,
				Placement:       tc.placement,
				DurationSeconds: 1500,
				QueueType:       "RANKED_TFT",
			}

			err = handler.HandleGameEnded(ctx, gameEnded)
			require.NoError(t, err)

			// Verify correct winner selection
			uow := uowFactory.CreateForGuild(guildID)
			require.NoError(t, uow.Begin(ctx))
			defer uow.Rollback()

			externalRef := entities.ExternalReference{
				System: entities.SystemTFT,
				ID:     gameID,
			}

			wager, err := uow.GroupWagerRepository().GetByExternalReference(ctx, externalRef)
			require.NoError(t, err)
			require.NotNil(t, wager)

			assert.Equal(t, entities.GroupWagerStateResolved, wager.State)
			assert.NotNil(t, wager.WinningOptionID)

			// Verify the correct option won
			detail, err := uow.GroupWagerRepository().GetDetailByID(ctx, wager.ID)
			require.NoError(t, err)
			require.NotNil(t, detail)

			var winOption *entities.GroupWagerOption
			for _, opt := range detail.Options {
				if opt.ID == *wager.WinningOptionID {
					winOption = opt
					break
				}
			}
			require.NotNil(t, winOption)
			assert.Equal(t, tc.expectedWinOption, winOption.OptionText)
		})
	}
}

func TestTFTHandler_NoCancellationLogic(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test database
	testDB := testutil.SetupTestDatabase(t)

	// Create no-op event publisher for integration tests
	noopPublisher := infrastructure.NewNoopEventPublisher()

	// Create UoW factory
	uowFactory := infrastructure.NewUnitOfWorkFactory(testDB.DB, noopPublisher)

	// Setup test data
	ctx := context.Background()
	guildID := int64(88888)
	summonerName := "ShortGamePlayer"
	tagLine := "NA1"
	gameID := "tft-short-game"

	// Setup guild and summoner watch
	setupTFTTestData(t, ctx, uowFactory, guildID, summonerName, tagLine)

	// Create mock Discord poster
	mockPoster := &application.MockDiscordPoster{}

	// Create TFT handler
	handler := application.NewTFTHandler(uowFactory, mockPoster)

	// Game start
	gameStarted := dto.TFTGameStartedDTO{
		SummonerName: summonerName,
		TagLine:      tagLine,
		GameID:       gameID,
		QueueType:    "RANKED_TFT",
	}

	err := handler.HandleGameStarted(ctx, gameStarted)
	require.NoError(t, err)

	// Verify wager was created
	uow := uowFactory.CreateForGuild(guildID)
	require.NoError(t, uow.Begin(ctx))
	defer uow.Rollback()

	externalRef := entities.ExternalReference{
		System: entities.SystemTFT,
		ID:     gameID,
	}

	wager, err := uow.GroupWagerRepository().GetByExternalReference(ctx, externalRef)
	require.NoError(t, err)
	require.NotNil(t, wager)
	require.Equal(t, entities.GroupWagerStateActive, wager.State)

	// Game end - very short duration (would be cancelled in LoL but not in TFT)
	gameEnded := dto.TFTGameEndedDTO{
		SummonerName:    summonerName,
		TagLine:         tagLine,
		GameID:          gameID,
		Placement:       7, // Bottom 4
		DurationSeconds: 300, // 5 minutes - would trigger cancellation in LoL
		QueueType:       "RANKED_TFT",
	}

	err = handler.HandleGameEnded(ctx, gameEnded)
	require.NoError(t, err)

	// Verify wager was RESOLVED, not cancelled (TFT has no 10-minute cancellation logic)
	uow = uowFactory.CreateForGuild(guildID)
	require.NoError(t, uow.Begin(ctx))
	defer uow.Rollback()

	wager, err = uow.GroupWagerRepository().GetByExternalReference(ctx, externalRef)
	require.NoError(t, err)
	require.NotNil(t, wager)
	assert.Equal(t, entities.GroupWagerStateResolved, wager.State)
	assert.NotNil(t, wager.WinningOptionID)

	// Verify the correct option won (Bottom 4 for placement 7)
	detail, err := uow.GroupWagerRepository().GetDetailByID(ctx, wager.ID)
	require.NoError(t, err)
	require.NotNil(t, detail)

	var winOption *entities.GroupWagerOption
	for _, opt := range detail.Options {
		if opt.ID == *wager.WinningOptionID {
			winOption = opt
			break
		}
	}
	require.NotNil(t, winOption)
	assert.Equal(t, "Bottom 4", winOption.OptionText)
}

func TestTFTHandler_MultipleGuilds(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test database
	testDB := testutil.SetupTestDatabase(t)

	// Create no-op event publisher for integration tests
	noopPublisher := infrastructure.NewNoopEventPublisher()

	// Create UoW factory
	uowFactory := infrastructure.NewUnitOfWorkFactory(testDB.DB, noopPublisher)

	// Setup test data
	ctx := context.Background()
	guild1ID := int64(11111)
	guild2ID := int64(22222)
	summonerName := "SharedTFTPlayer"
	tagLine := "NA1"
	gameID := "tft-shared-game"

	// Setup multiple guilds watching the same summoner
	setupTFTTestData(t, ctx, uowFactory, guild1ID, summonerName, tagLine)
	setupTFTTestData(t, ctx, uowFactory, guild2ID, summonerName, tagLine)

	// Create mock Discord poster
	mockPoster := &application.MockDiscordPoster{}

	// Create TFT handler
	handler := application.NewTFTHandler(uowFactory, mockPoster)

	// Game start - should create wagers for both guilds
	gameStarted := dto.TFTGameStartedDTO{
		SummonerName: summonerName,
		TagLine:      tagLine,
		GameID:       gameID,
		QueueType:    "RANKED_TFT",
	}

	err := handler.HandleGameStarted(ctx, gameStarted)
	require.NoError(t, err)

	// Verify Discord posts were made for both guilds
	assert.Len(t, mockPoster.Posts, 2)

	// Verify both wagers exist in their respective guilds
	externalRef := entities.ExternalReference{
		System: entities.SystemTFT,
		ID:     gameID,
	}

	// Check guild 1
	uow1 := uowFactory.CreateForGuild(guild1ID)
	require.NoError(t, uow1.Begin(ctx))
	defer uow1.Rollback()

	wager1, err := uow1.GroupWagerRepository().GetByExternalReference(ctx, externalRef)
	require.NoError(t, err)
	require.NotNil(t, wager1)
	assert.Equal(t, guild1ID, wager1.GuildID)

	// Check guild 2
	uow2 := uowFactory.CreateForGuild(guild2ID)
	require.NoError(t, uow2.Begin(ctx))
	defer uow2.Rollback()

	wager2, err := uow2.GroupWagerRepository().GetByExternalReference(ctx, externalRef)
	require.NoError(t, err)
	require.NotNil(t, wager2)
	assert.Equal(t, guild2ID, wager2.GuildID)

	// Game end - should resolve both wagers
	gameEnded := dto.TFTGameEndedDTO{
		SummonerName:    summonerName,
		TagLine:         tagLine,
		GameID:          gameID,
		Placement:       2, // Top 4
		DurationSeconds: 1500,
		QueueType:       "RANKED_TFT",
	}

	err = handler.HandleGameEnded(ctx, gameEnded)
	require.NoError(t, err)

	// Verify both wagers are resolved
	wager1, err = uow1.GroupWagerRepository().GetByExternalReference(ctx, externalRef)
	require.NoError(t, err)
	assert.Equal(t, entities.GroupWagerStateResolved, wager1.State)

	wager2, err = uow2.GroupWagerRepository().GetByExternalReference(ctx, externalRef)
	require.NoError(t, err)
	assert.Equal(t, entities.GroupWagerStateResolved, wager2.State)
}

func TestTFTHandler_NoWatchingGuilds(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test database
	testDB := testutil.SetupTestDatabase(t)

	// Create no-op event publisher for integration tests
	noopPublisher := infrastructure.NewNoopEventPublisher()

	// Create UoW factory
	uowFactory := infrastructure.NewUnitOfWorkFactory(testDB.DB, noopPublisher)

	ctx := context.Background()

	mockPoster := &application.MockDiscordPoster{}
	handler := application.NewTFTHandler(uowFactory, mockPoster)

	// Game start for unwatched summoner
	gameStarted := dto.TFTGameStartedDTO{
		SummonerName: "UnwatchedTFTPlayer",
		TagLine:      "NA1",
		GameID:       "tft-unwatched-game",
		QueueType:    "RANKED_TFT",
	}

	err := handler.HandleGameStarted(ctx, gameStarted)
	require.NoError(t, err)

	// No Discord posts should be made
	assert.Len(t, mockPoster.Posts, 0)

	// Game end should also handle gracefully
	gameEnded := dto.TFTGameEndedDTO{
		SummonerName:    "UnwatchedTFTPlayer",
		TagLine:         "NA1",
		GameID:          "tft-unwatched-game",
		Placement:       1,
		DurationSeconds: 1800,
		QueueType:       "RANKED_TFT",
	}

	err = handler.HandleGameEnded(ctx, gameEnded)
	require.NoError(t, err)
}

func TestTFTHandler_MissingTFTChannel(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test database
	testDB := testutil.SetupTestDatabase(t)

	// Create no-op event publisher for integration tests
	noopPublisher := infrastructure.NewNoopEventPublisher()

	// Create UoW factory
	uowFactory := infrastructure.NewUnitOfWorkFactory(testDB.DB, noopPublisher)

	// Setup test data
	ctx := context.Background()
	guildID := int64(99999)
	summonerName := "NoChannelPlayer"
	tagLine := "NA1"
	gameID := "tft-no-channel-game"

	// Setup guild without TFT channel
	setupTFTTestDataWithoutChannel(t, ctx, uowFactory, guildID, summonerName, tagLine)

	// Create mock Discord poster
	mockPoster := &application.MockDiscordPoster{}

	// Create TFT handler
	handler := application.NewTFTHandler(uowFactory, mockPoster)

	// Game start - should not create wager due to missing TFT channel
	gameStarted := dto.TFTGameStartedDTO{
		SummonerName: summonerName,
		TagLine:      tagLine,
		GameID:       gameID,
		QueueType:    "RANKED_TFT",
	}

	err := handler.HandleGameStarted(ctx, gameStarted)
	require.NoError(t, err) // Handler should not error, just skip the guild

	// No Discord posts should be made
	assert.Len(t, mockPoster.Posts, 0)
}

// setupTFTTestData creates the necessary guild settings and summoner watch for TFT testing
func setupTFTTestData(t *testing.T, ctx context.Context, uowFactory application.UnitOfWorkFactory, guildID int64, summonerName, tagLine string) {
	uow := uowFactory.CreateForGuild(guildID)
	require.NoError(t, uow.Begin(ctx))
	defer func() {
		if err := uow.Commit(); err != nil {
			t.Fatalf("Failed to commit test data setup: %v", err)
		}
	}()

	// Create guild settings with TFT channel
	guildSettings := &entities.GuildSettings{
		GuildID:      guildID,
		TftChannelID: func() *int64 { id := int64(888888); return &id }(), // Mock TFT channel ID
	}

	_, err := uow.GuildSettingsRepository().GetOrCreateGuildSettings(ctx, guildID)
	require.NoError(t, err)

	// Update with TFT channel
	err = uow.GuildSettingsRepository().UpdateGuildSettings(ctx, guildSettings)
	require.NoError(t, err)

	// Verify the TFT channel was set
	updatedSettings, err := uow.GuildSettingsRepository().GetOrCreateGuildSettings(ctx, guildID)
	require.NoError(t, err)
	require.NotNil(t, updatedSettings.TftChannelID, "TFT channel ID should be set after update")
	require.Equal(t, int64(888888), *updatedSettings.TftChannelID, "TFT channel ID should match expected value")

	// Create summoner watch
	_, err = uow.SummonerWatchRepository().CreateWatch(ctx, guildID, summonerName, tagLine)
	require.NoError(t, err)
}

// setupTFTTestDataWithoutChannel creates guild settings without TFT channel for negative testing
func setupTFTTestDataWithoutChannel(t *testing.T, ctx context.Context, uowFactory application.UnitOfWorkFactory, guildID int64, summonerName, tagLine string) {
	uow := uowFactory.CreateForGuild(guildID)
	require.NoError(t, uow.Begin(ctx))
	defer func() {
		if err := uow.Commit(); err != nil {
			t.Fatalf("Failed to commit test data setup: %v", err)
		}
	}()

	// Create guild settings WITHOUT TFT channel
	guildSettings := &entities.GuildSettings{
		GuildID:      guildID,
		TftChannelID: nil, // No TFT channel configured
	}

	_, err := uow.GuildSettingsRepository().GetOrCreateGuildSettings(ctx, guildID)
	require.NoError(t, err)

	// Update without TFT channel
	err = uow.GuildSettingsRepository().UpdateGuildSettings(ctx, guildSettings)
	require.NoError(t, err)

	// Create summoner watch
	_, err = uow.SummonerWatchRepository().CreateWatch(ctx, guildID, summonerName, tagLine)
	require.NoError(t, err)
}