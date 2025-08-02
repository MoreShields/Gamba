package application_test

import (
	"context"
	"testing"

	"gambler/discord-client/application"
	"gambler/discord-client/application/dto"
	"gambler/discord-client/config"
	"gambler/discord-client/infrastructure"
	"gambler/discord-client/domain/entities"
	"gambler/discord-client/repository/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoLHandler_EndToEndFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Set up test config
	config.SetTestConfig(config.NewTestConfig())
	defer config.ResetConfig()

	// Setup test database
	testDB := testutil.SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	// Create no-op event publisher for integration tests
	noopPublisher := infrastructure.NewNoopEventPublisher()

	// Create UoW factory
	uowFactory := infrastructure.NewUnitOfWorkFactory(testDB.DB, noopPublisher)

	// Setup test data
	ctx := context.Background()
	guildID := int64(12345)
	summonerName := "TestPlayer"
	tagLine := "NA1"
	gameID := "test-game-123"

	// Setup guild and summoner watch
	setupTestData(t, ctx, uowFactory, guildID, summonerName, tagLine)

	// Create mock Discord poster
	mockPoster := &application.MockDiscordPoster{}

	// Create LoL handler
	handler := application.NewLoLHandler(uowFactory, mockPoster)

	t.Run("Game Start Creates House Wager", func(t *testing.T) {
		// Game start event
		gameStarted := dto.GameStartedDTO{
			SummonerName: summonerName,
			TagLine:      tagLine,
			GameID:       gameID,
			QueueType:    "RANKED_SOLO_5x5",
		}

		// Handle game start
		err := handler.HandleGameStarted(ctx, gameStarted)
		require.NoError(t, err)

		// Verify Discord post was made
		assert.Len(t, mockPoster.Posts, 1)
		post := mockPoster.Posts[0]
		assert.Equal(t, guildID, post.GuildID)
		assert.Contains(t, post.Title, summonerName)
		assert.Len(t, post.Options, 2) // Win/Loss options

		// Verify wager was created in database
		uow := uowFactory.CreateForGuild(guildID)
		require.NoError(t, uow.Begin(ctx))
		defer uow.Rollback()

		externalRef := entities.ExternalReference{
			System: entities.SystemLeagueOfLegends,
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
		assert.Equal(t, entities.SystemLeagueOfLegends, wager.ExternalRef.System)
	})

	t.Run("Game End Resolves House Wager - Win", func(t *testing.T) {
		// Game end event - player wins
		gameEnded := dto.GameEndedDTO{
			SummonerName:    summonerName,
			TagLine:         tagLine,
			GameID:          gameID,
			Won:             true,
			DurationSeconds: 1800, // 30 minutes
		}

		// Handle game end
		err := handler.HandleGameEnded(ctx, gameEnded)
		require.NoError(t, err)

		// Verify wager was resolved
		uow := uowFactory.CreateForGuild(guildID)
		require.NoError(t, uow.Begin(ctx))
		defer uow.Rollback()

		externalRef := entities.ExternalReference{
			System: entities.SystemLeagueOfLegends,
			ID:     gameID,
		}

		wager, err := uow.GroupWagerRepository().GetByExternalReference(ctx, externalRef)
		require.NoError(t, err)
		require.NotNil(t, wager)

		assert.Equal(t, entities.GroupWagerStateResolved, wager.State)
		assert.NotNil(t, wager.WinningOptionID)

		// Verify the correct option won (Win option)
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
		assert.Equal(t, "Win", winOption.OptionText)
	})
}

func TestLoLHandler_EndToEndFlow_Loss(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Set up test config
	config.SetTestConfig(config.NewTestConfig())
	defer config.ResetConfig()

	// Setup test database
	testDB := testutil.SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	// Create no-op event publisher for integration tests
	noopPublisher := infrastructure.NewNoopEventPublisher()

	// Create UoW factory
	uowFactory := infrastructure.NewUnitOfWorkFactory(testDB.DB, noopPublisher)

	// Setup test data
	ctx := context.Background()
	guildID := int64(54321)
	summonerName := "TestPlayer2"
	tagLine := "EUW1"
	gameID := "test-game-456"

	// Setup guild and summoner watch
	setupTestData(t, ctx, uowFactory, guildID, summonerName, tagLine)

	// Create mock Discord poster
	mockPoster := &application.MockDiscordPoster{}

	// Create LoL handler
	handler := application.NewLoLHandler(uowFactory, mockPoster)

	// Game start
	gameStarted := dto.GameStartedDTO{
		SummonerName: summonerName,
		TagLine:      tagLine,
		GameID:       gameID,
		QueueType:    "RANKED_SOLO_5x5",
	}

	err := handler.HandleGameStarted(ctx, gameStarted)
	require.NoError(t, err)

	// Game end - player loses
	gameEnded := dto.GameEndedDTO{
		SummonerName:    summonerName,
		TagLine:         tagLine,
		GameID:          gameID,
		Won:             false,
		DurationSeconds: 1200, // 20 minutes
	}

	err = handler.HandleGameEnded(ctx, gameEnded)
	require.NoError(t, err)

	// Verify wager was resolved with Loss option
	uow := uowFactory.CreateForGuild(guildID)
	require.NoError(t, uow.Begin(ctx))
	defer uow.Rollback()

	externalRef := entities.ExternalReference{
		System: entities.SystemLeagueOfLegends,
		ID:     gameID,
	}

	wager, err := uow.GroupWagerRepository().GetByExternalReference(ctx, externalRef)
	require.NoError(t, err)
	require.NotNil(t, wager)

	assert.Equal(t, entities.GroupWagerStateResolved, wager.State)
	assert.NotNil(t, wager.WinningOptionID)

	// Verify the correct option won (Loss option)
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
	assert.Equal(t, "Loss", winOption.OptionText)
}

func TestLoLHandler_MultipleGuilds(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Set up test config
	config.SetTestConfig(config.NewTestConfig())
	defer config.ResetConfig()

	// Setup test database
	testDB := testutil.SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	// Create no-op event publisher for integration tests
	noopPublisher := infrastructure.NewNoopEventPublisher()

	// Create UoW factory
	uowFactory := infrastructure.NewUnitOfWorkFactory(testDB.DB, noopPublisher)

	// Setup test data
	ctx := context.Background()
	guild1ID := int64(11111)
	guild2ID := int64(22222)
	summonerName := "SharedPlayer"
	tagLine := "NA1"
	gameID := "test-game-shared"

	// Setup multiple guilds watching the same summoner
	setupTestData(t, ctx, uowFactory, guild1ID, summonerName, tagLine)
	setupTestData(t, ctx, uowFactory, guild2ID, summonerName, tagLine)

	// Create mock Discord poster
	mockPoster := &application.MockDiscordPoster{}

	// Create LoL handler
	handler := application.NewLoLHandler(uowFactory, mockPoster)

	// Game start - should create wagers for both guilds
	gameStarted := dto.GameStartedDTO{
		SummonerName: summonerName,
		TagLine:      tagLine,
		GameID:       gameID,
		QueueType:    "RANKED_SOLO_5x5",
	}

	err := handler.HandleGameStarted(ctx, gameStarted)
	require.NoError(t, err)

	// Verify Discord posts were made for both guilds
	assert.Len(t, mockPoster.Posts, 2)

	// Verify both wagers exist in their respective guilds
	externalRef := entities.ExternalReference{
		System: entities.SystemLeagueOfLegends,
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
	gameEnded := dto.GameEndedDTO{
		SummonerName:    summonerName,
		TagLine:         tagLine,
		GameID:          gameID,
		Won:             true,
		DurationSeconds: 1500,
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

func TestLoLHandler_NoWatchingGuilds(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Set up test config
	config.SetTestConfig(config.NewTestConfig())
	defer config.ResetConfig()

	// Setup test database
	testDB := testutil.SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	// Create no-op event publisher for integration tests
	noopPublisher := infrastructure.NewNoopEventPublisher()

	// Create UoW factory
	uowFactory := infrastructure.NewUnitOfWorkFactory(testDB.DB, noopPublisher)

	ctx := context.Background()

	mockPoster := &application.MockDiscordPoster{}
	handler := application.NewLoLHandler(uowFactory, mockPoster)

	// Game start for unwatched summoner
	gameStarted := dto.GameStartedDTO{
		SummonerName: "UnwatchedPlayer",
		TagLine:      "NA1",
		GameID:       "test-game-nowatcher",
		QueueType:    "RANKED_SOLO_5x5",
	}

	err := handler.HandleGameStarted(ctx, gameStarted)
	require.NoError(t, err)

	// No Discord posts should be made
	assert.Len(t, mockPoster.Posts, 0)

	// Game end should also handle gracefully
	gameEnded := dto.GameEndedDTO{
		SummonerName:    "UnwatchedPlayer",
		TagLine:         "NA1",
		GameID:          "test-game-nowatcher",
		Won:             true,
		DurationSeconds: 1000,
	}

	err = handler.HandleGameEnded(ctx, gameEnded)
	require.NoError(t, err)
}

// setupTestData creates the necessary guild settings and summoner watch for testing
func setupTestData(t *testing.T, ctx context.Context, uowFactory application.UnitOfWorkFactory, guildID int64, summonerName, tagLine string) {
	uow := uowFactory.CreateForGuild(guildID)
	require.NoError(t, uow.Begin(ctx))
	defer func() {
		if err := uow.Commit(); err != nil {
			t.Fatalf("Failed to commit test data setup: %v", err)
		}
	}()

	// Create guild settings with LoL channel
	guildSettings := &entities.GuildSettings{
		GuildID:      guildID,
		LolChannelID: func() *int64 { id := int64(999999); return &id }(), // Mock channel ID
	}

	_, err := uow.GuildSettingsRepository().GetOrCreateGuildSettings(ctx, guildID)
	require.NoError(t, err)

	// Update with LoL channel
	err = uow.GuildSettingsRepository().UpdateGuildSettings(ctx, guildSettings)
	require.NoError(t, err)

	// Create summoner watch
	_, err = uow.SummonerWatchRepository().CreateWatch(ctx, guildID, summonerName, tagLine)
	require.NoError(t, err)
}
func TestLoLHandler_ForfeitRemake(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Set up test config
	config.SetTestConfig(config.NewTestConfig())
	defer config.ResetConfig()

	// Setup test database
	testDB := testutil.SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	// Create no-op event publisher for integration tests
	noopPublisher := infrastructure.NewNoopEventPublisher()

	// Create UoW factory
	uowFactory := infrastructure.NewUnitOfWorkFactory(testDB.DB, noopPublisher)

	// Setup test data
	ctx := context.Background()
	guildID := int64(77777)
	summonerName := "ForfeitPlayer"
	tagLine := "NA1"
	gameID := "test-game-forfeit"

	// Setup guild and summoner watch
	setupTestData(t, ctx, uowFactory, guildID, summonerName, tagLine)

	// Create mock Discord poster
	mockPoster := &application.MockDiscordPoster{}

	// Create LoL handler
	handler := application.NewLoLHandler(uowFactory, mockPoster)

	// Game start
	gameStarted := dto.GameStartedDTO{
		SummonerName: summonerName,
		TagLine:      tagLine,
		GameID:       gameID,
		QueueType:    "RANKED_SOLO_5x5",
	}

	err := handler.HandleGameStarted(ctx, gameStarted)
	require.NoError(t, err)

	// Verify wager was created
	uow := uowFactory.CreateForGuild(guildID)
	require.NoError(t, uow.Begin(ctx))

	externalRef := entities.ExternalReference{
		System: entities.SystemLeagueOfLegends,
		ID:     gameID,
	}

	wager, err := uow.GroupWagerRepository().GetByExternalReference(ctx, externalRef)
	require.NoError(t, err)
	require.NotNil(t, wager)
	require.Equal(t, entities.GroupWagerStateActive, wager.State)
	uow.Rollback()

	// Game end - neither win nor loss (forfeit/remake)
	gameEnded := dto.GameEndedDTO{
		SummonerName:    summonerName,
		TagLine:         tagLine,
		GameID:          gameID,
		Won:             false, // Neither win nor loss indicates forfeit/remake
		DurationSeconds: 300,   // 5 minutes - typical forfeit time
	}

	err = handler.HandleGameEnded(ctx, gameEnded)
	require.NoError(t, err)

	// Verify wager was cancelled
	uow = uowFactory.CreateForGuild(guildID)
	require.NoError(t, uow.Begin(ctx))
	defer uow.Rollback()

	wager, err = uow.GroupWagerRepository().GetByExternalReference(ctx, externalRef)
	require.NoError(t, err)
	require.NotNil(t, wager)
	assert.Equal(t, entities.GroupWagerStateCancelled, wager.State)
}
