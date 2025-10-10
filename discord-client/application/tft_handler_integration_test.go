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
			QueueType:    "TFT_RANKED",
		}

		// Handle game start
		err := handler.HandleGameStarted(ctx, gameStarted)
		require.NoError(t, err)

		// Verify Discord post was made
		assert.Len(t, mockPoster.Posts, 1)
		post := mockPoster.Posts[0]
		assert.Equal(t, guildID, post.GuildID)
		assert.Contains(t, post.Title, summonerName)
		assert.Contains(t, post.Title, "Ranked TFT")
		assert.Len(t, post.Options, 4) // 4 placement range options

		// Verify TFT-specific options
		assert.Equal(t, "1-2", post.Options[0].Text)
		assert.Equal(t, "3-4", post.Options[1].Text)
		assert.Equal(t, "5-6", post.Options[2].Text)
		assert.Equal(t, "7-8", post.Options[3].Text)
		assert.Equal(t, float64(4.0), post.Options[0].Multiplier)
		assert.Equal(t, float64(4.0), post.Options[1].Multiplier)
		assert.Equal(t, float64(4.0), post.Options[2].Multiplier)
		assert.Equal(t, float64(4.0), post.Options[3].Multiplier)

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

	t.Run("Game End Resolves TFT House Wager - 3rd Place", func(t *testing.T) {
		// Don't use t.Parallel() in sub-tests when parent cleans up resources
		// TFT game end event - player gets 3rd place (3-4 range)
		gameEnded := dto.TFTGameEndedDTO{
			SummonerName:    summonerName,
			TagLine:         tagLine,
			GameID:          gameID,
			Placement:       3, // 3-4 range
			DurationSeconds: 1800, // 30 minutes
			QueueType:       "TFT_RANKED",
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

		// Verify the correct option won (3-4 option)
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
		assert.Equal(t, "3-4", winOption.OptionText)
	})
}

func TestTFTHandler_EndToEndFlow_6thPlace(t *testing.T) {
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
		QueueType:    "TFT_RANKED",
	}

	err := handler.HandleGameStarted(ctx, gameStarted)
	require.NoError(t, err)

	// Game end - player gets 6th place (5-6 range)
	gameEnded := dto.TFTGameEndedDTO{
		SummonerName:    summonerName,
		TagLine:         tagLine,
		GameID:          gameID,
		Placement:       6, // 5-6 range
		DurationSeconds: 1200, // 20 minutes
		QueueType:       "TFT_RANKED",
	}

	err = handler.HandleGameEnded(ctx, gameEnded)
	require.NoError(t, err)

	// Verify wager was resolved with 5-6 option
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

	// Verify the correct option won (5-6 option)
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
	assert.Equal(t, "5-6", winOption.OptionText)
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
		{"1st place", 1, "1-2"},
		{"2nd place", 2, "1-2"},
		{"3rd place", 3, "3-4"},
		{"4th place", 4, "3-4"},
		{"5th place", 5, "5-6"},
		{"6th place", 6, "5-6"},
		{"7th place", 7, "7-8"},
		{"8th place", 8, "7-8"},
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
				QueueType:    "TFT_RANKED",
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
				QueueType:       "TFT_RANKED",
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

func TestTFTHandler_DoubleUpPlacement(t *testing.T) {
	testCases := []struct {
		name              string
		queueType         string
		placement         int32
		expectedOptions   []string
		expectedWinOption string
	}{
		// Double Up games (4 teams)
		{"Double Up 1st", "TFT_NORMAL_DOUBLE_UP", 1, []string{"1", "2", "3", "4"}, "1"},
		{"Double Up 2nd", "TFT_NORMAL_DOUBLE_UP", 2, []string{"1", "2", "3", "4"}, "2"},
		{"Double Up 3rd", "TFT_RANKED_DOUBLE_UP", 3, []string{"1", "2", "3", "4"}, "3"},
		{"Double Up 4th", "TFT_RANKED_DOUBLE_UP", 4, []string{"1", "2", "3", "4"}, "4"},
		// Regular TFT for comparison
		{"Regular TFT 5th", "TFT_RANKED", 5, []string{"1-2", "3-4", "5-6", "7-8"}, "5-6"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if testing.Short() {
				t.Skip("Skipping integration test in short mode")
			}

			// Setup test database
			testDB := testutil.SetupTestDatabase(t)
			defer testDB.Cleanup(t)

			noopPublisher := infrastructure.NewNoopEventPublisher()
			uowFactory := infrastructure.NewUnitOfWorkFactory(testDB.DB, noopPublisher)

			ctx := context.Background()
			guildID := int64(88888 + int64(tc.placement))
			summonerName := "DoubleUpTester"
			tagLine := "DUO"
			gameID := fmt.Sprintf("double-up-test-%d", tc.placement)

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
				QueueType:    tc.queueType,
			}

			err := handler.HandleGameStarted(ctx, gameStarted)
			require.NoError(t, err)

			// Verify wager has correct options
			require.Equal(t, 1, len(mockPoster.Posts), "Should create exactly one wager")
			wagerDTO := mockPoster.Posts[0]
			
			assert.Equal(t, len(tc.expectedOptions), len(wagerDTO.Options))
			for i, opt := range wagerDTO.Options {
				assert.Equal(t, tc.expectedOptions[i], opt.Text)
			}

			// Game end with placement
			gameEnded := dto.TFTGameEndedDTO{
				SummonerName:    summonerName,
				TagLine:         tagLine,
				GameID:          gameID,
				Placement:       tc.placement,
				DurationSeconds: 1200,
				QueueType:       tc.queueType,
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

			var winOption *entities.GroupWagerOption
			for _, opt := range detail.Options {
				if opt.ID == *wager.WinningOptionID {
					winOption = opt
					break
				}
			}
			require.NotNil(t, winOption)
			assert.Equal(t, tc.expectedWinOption, winOption.OptionText, 
				"Expected option %s to win for placement %d in %s", 
				tc.expectedWinOption, tc.placement, tc.queueType)
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
		QueueType:    "TFT_RANKED",
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
		Placement:       7, // 7-8 range
		DurationSeconds: 300, // 5 minutes - would trigger cancellation in LoL
		QueueType:       "TFT_RANKED",
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

	// Verify the correct option won (7-8 for placement 7)
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
	assert.Equal(t, "7-8", winOption.OptionText)
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
		QueueType:    "TFT_RANKED",
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
		Placement:       2, // 1-2 range
		DurationSeconds: 1500,
		QueueType:       "TFT_RANKED",
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
		QueueType:    "TFT_RANKED",
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
		QueueType:       "TFT_RANKED",
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
		QueueType:    "TFT_RANKED",
	}

	err := handler.HandleGameStarted(ctx, gameStarted)
	require.NoError(t, err) // Handler should not error, just skip the guild

	// No Discord posts should be made
	assert.Len(t, mockPoster.Posts, 0)
}

func TestTFTHandler_UnknownQueueType_DropsEvent(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test database
	testDB := testutil.SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	// Create no-op event publisher for integration tests
	noopPublisher := infrastructure.NewNoopEventPublisher()

	// Create UoW factory
	uowFactory := infrastructure.NewUnitOfWorkFactory(testDB.DB, noopPublisher)

	// Setup test data
	ctx := context.Background()
	guildID := int64(88888)
	summonerName := "UnknownTFTQueuePlayer"
	tagLine := "NA1"
	gameID := "tft-unknown-queue"

	// Setup guild and summoner watch
	setupTFTTestData(t, ctx, uowFactory, guildID, summonerName, tagLine)

	// Create mock Discord poster
	mockPoster := &application.MockDiscordPoster{}

	// Create TFT handler
	handler := application.NewTFTHandler(uowFactory, mockPoster)

	// Game start with unknown queue type
	gameStarted := dto.TFTGameStartedDTO{
		SummonerName: summonerName,
		TagLine:      tagLine,
		GameID:       gameID,
		QueueType:    "TFT_UNKNOWN_MODE", // Unknown queue type
	}

	err := handler.HandleGameStarted(ctx, gameStarted)
	require.NoError(t, err) // Should not return error, just drop silently

	// Verify no Discord posts were made
	assert.Len(t, mockPoster.Posts, 0)

	// Verify no wager was created in database
	uow := uowFactory.CreateForGuild(guildID)
	require.NoError(t, uow.Begin(ctx))
	defer uow.Rollback()

	externalRef := entities.ExternalReference{
		System: entities.SystemTFT,
		ID:     gameID,
	}

	wager, err := uow.GroupWagerRepository().GetByExternalReference(ctx, externalRef)
	require.NoError(t, err)
	assert.Nil(t, wager) // Wager should not exist for unknown queue type
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