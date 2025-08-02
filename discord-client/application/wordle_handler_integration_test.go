package application_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"gambler/discord-client/application"
	"gambler/discord-client/config"
	"gambler/discord-client/domain/events"
	"gambler/discord-client/infrastructure"
	"gambler/discord-client/domain/entities"
	"gambler/discord-client/repository/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockUserResolver implements application.UserResolver for testing
type mockUserResolver struct {
	nickToUserIDs map[string][]int64
}

func newMockUserResolver() *mockUserResolver {
	return &mockUserResolver{
		nickToUserIDs: make(map[string][]int64),
	}
}

func (m *mockUserResolver) ResolveUsersByNick(ctx context.Context, guildID int64, nickname string) ([]int64, error) {
	if users, ok := m.nickToUserIDs[nickname]; ok {
		return users, nil
	}
	return []int64{}, nil
}

func (m *mockUserResolver) addNickname(nickname string, userIDs ...int64) {
	m.nickToUserIDs[nickname] = userIDs
}

// createWordleMessage creates a Discord message event from the Wordle bot
func createWordleMessage(guildID, channelID string, content string) events.DiscordMessageEvent {
	cfg := config.Get()
	return events.DiscordMessageEvent{
		MessageID: fmt.Sprintf("test-message-%d", time.Now().UnixNano()),
		UserID:    cfg.WordleBotID,
		GuildID:   guildID,
		ChannelID: channelID,
		Content:   content,
	}
}

// setupWordleGuild creates guild settings with Wordle channel configured
func setupWordleGuild(t *testing.T, ctx context.Context, uowFactory application.UnitOfWorkFactory, guildID, channelID int64) {
	uow := uowFactory.CreateForGuild(guildID)
	require.NoError(t, uow.Begin(ctx))
	defer func() {
		if err := uow.Commit(); err != nil {
			t.Fatalf("Failed to commit guild setup: %v", err)
		}
	}()

	// Create guild settings with Wordle channel
	guildSettings := &entities.GuildSettings{
		GuildID:         guildID,
		WordleChannelID: &channelID,
	}

	_, err := uow.GuildSettingsRepository().GetOrCreateGuildSettings(ctx, guildID)
	require.NoError(t, err)

	// Update with Wordle channel
	err = uow.GuildSettingsRepository().UpdateGuildSettings(ctx, guildSettings)
	require.NoError(t, err)
}

func TestWordleHandler_EndToEndFlow(t *testing.T) {
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
	channelID := int64(67890)

	// Setup guild with Wordle channel
	setupWordleGuild(t, ctx, uowFactory, guildID, channelID)

	// Create mock user resolver
	mockResolver := newMockUserResolver()
	mockResolver.addNickname("Shid", 135678825894903808)
	mockResolver.addNickname("ChancellorLoaf", 232153861941362688)
	mockResolver.addNickname("Piplup", 141402190152597504)

	// Create Wordle handler
	handler := application.NewWordleHandler(uowFactory, mockResolver)

	t.Run("Process Wordle Results With Mixed Mentions", func(t *testing.T) {
		// Create Wordle message with both user ID and nickname mentions
		content := `**Your group is on a 70 day streak!** 🔥 Here are yesterday's results:
👑 3/6: <@133008606202167296> @Shid
4/6: <@232153861941362688> @ChancellorLoaf @Piplup
5/6: <@217883936964083713>`

		message := createWordleMessage(
			strconv.FormatInt(guildID, 10),
			strconv.FormatInt(channelID, 10),
			content,
		)

		// Handle the message
		err := handler.HandleDiscordMessage(ctx, message)
		require.NoError(t, err)

		// Verify users were created/updated with correct balances
		uow := uowFactory.CreateForGuild(guildID)
		require.NoError(t, uow.Begin(ctx))
		defer uow.Rollback()

		// Expected rewards based on guess count (from DailyAwardsService)
		expectedRewards := map[int64]int64{
			133008606202167296: 7000,  // 3/6
			135678825894903808: 7000,  // 3/6 (Shid)
			232153861941362688: 7000,  // 4/6
			141402190152597504: 7000,  // 4/6 (Piplup)
			217883936964083713: 5000,  // 5/6
		}

		for userID, expectedReward := range expectedRewards {
			user, err := uow.UserRepository().GetByDiscordID(ctx, userID)
			require.NoError(t, err)
			assert.Equal(t, expectedReward, user.Balance, "User %d should have balance %d", userID, expectedReward)

			// Verify WordleCompletion was created
			completion, err := uow.WordleCompletionRepo().GetByUserToday(ctx, userID, guildID)
			require.NoError(t, err)
			require.NotNil(t, completion)

			// Verify balance history with wordle_reward transaction type
			// This is the key test that would have caught the constraint violation
			history, err := uow.BalanceHistoryRepository().GetByUser(ctx, userID, 1)
			require.NoError(t, err)
			require.Len(t, history, 1)
			
			assert.Equal(t, entities.TransactionTypeWordleReward, history[0].TransactionType)
			assert.Equal(t, expectedReward, history[0].ChangeAmount)
			assert.Equal(t, int64(0), history[0].BalanceBefore)
			assert.Equal(t, expectedReward, history[0].BalanceAfter)
			assert.Equal(t, guildID, history[0].GuildID)
		}
	})
}

func TestWordleHandler_NicknameResolution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Set up test config
	config.SetTestConfig(config.NewTestConfig())
	defer config.ResetConfig()

	// Setup test database
	testDB := testutil.SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	// Create no-op event publisher
	noopPublisher := infrastructure.NewNoopEventPublisher()
	uowFactory := infrastructure.NewUnitOfWorkFactory(testDB.DB, noopPublisher)

	ctx := context.Background()
	guildID := int64(22222)
	channelID := int64(33333)

	setupWordleGuild(t, ctx, uowFactory, guildID, channelID)

	// Create mock resolver with duplicate nicknames
	mockResolver := newMockUserResolver()
	mockResolver.addNickname("PopularNick", 111111111, 222222222) // Two users with same nickname
	mockResolver.addNickname("UniqueNick", 333333333)

	handler := application.NewWordleHandler(uowFactory, mockResolver)

	t.Run("Multiple Users With Same Nickname", func(t *testing.T) {
		content := `Today's results:
3/6: @PopularNick
4/6: @UniqueNick
5/6: @UnknownNick`

		message := createWordleMessage(
			strconv.FormatInt(guildID, 10),
			strconv.FormatInt(channelID, 10),
			content,
		)

		err := handler.HandleDiscordMessage(ctx, message)
		require.NoError(t, err)

		// Verify both users with PopularNick got rewards
		uow := uowFactory.CreateForGuild(guildID)
		require.NoError(t, uow.Begin(ctx))
		defer uow.Rollback()

		// Both users with PopularNick should get 3/6 reward (7000)
		for _, userID := range []int64{111111111, 222222222} {
			user, err := uow.UserRepository().GetByDiscordID(ctx, userID)
			require.NoError(t, err)
			assert.Equal(t, int64(7000), user.Balance)
		}

		// UniqueNick user should get 4/6 reward (7000)
		user, err := uow.UserRepository().GetByDiscordID(ctx, 333333333)
		require.NoError(t, err)
		assert.Equal(t, int64(7000), user.Balance)

		// UnknownNick should be ignored (no error)
	})
}

func TestWordleHandler_DuplicatePrevention(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config.SetTestConfig(config.NewTestConfig())
	defer config.ResetConfig()

	testDB := testutil.SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	noopPublisher := infrastructure.NewNoopEventPublisher()
	uowFactory := infrastructure.NewUnitOfWorkFactory(testDB.DB, noopPublisher)

	ctx := context.Background()
	guildID := int64(44444)
	channelID := int64(55555)

	setupWordleGuild(t, ctx, uowFactory, guildID, channelID)

	mockResolver := newMockUserResolver()
	mockResolver.addNickname("DupeUser", 777777777)

	handler := application.NewWordleHandler(uowFactory, mockResolver)

	// First message - should process
	content1 := `Today's results:
3/6: <@777777777>`

	message1 := createWordleMessage(
		strconv.FormatInt(guildID, 10),
		strconv.FormatInt(channelID, 10),
		content1,
	)

	err := handler.HandleDiscordMessage(ctx, message1)
	require.NoError(t, err)

	// Second message with same user (by nickname) - should be ignored
	content2 := `Later results:
2/6: @DupeUser`

	message2 := createWordleMessage(
		strconv.FormatInt(guildID, 10),
		strconv.FormatInt(channelID, 10),
		content2,
	)

	err = handler.HandleDiscordMessage(ctx, message2)
	require.NoError(t, err)

	// Verify user still has original balance
	uow := uowFactory.CreateForGuild(guildID)
	require.NoError(t, uow.Begin(ctx))
	defer uow.Rollback()

	user, err := uow.UserRepository().GetByDiscordID(ctx, 777777777)
	require.NoError(t, err)
	assert.Equal(t, int64(7000), user.Balance) // Still 3/6 reward (7000), not 2/6

	// Verify only one completion exists
	history, err := uow.BalanceHistoryRepository().GetByUser(ctx, 777777777, 10)
	require.NoError(t, err)
	assert.Len(t, history, 1)
}

func TestWordleHandler_ChannelFiltering(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config.SetTestConfig(config.NewTestConfig())
	defer config.ResetConfig()

	testDB := testutil.SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	noopPublisher := infrastructure.NewNoopEventPublisher()
	uowFactory := infrastructure.NewUnitOfWorkFactory(testDB.DB, noopPublisher)

	ctx := context.Background()
	guildID := int64(66666)
	wordleChannelID := int64(77777)
	otherChannelID := int64(88888)

	setupWordleGuild(t, ctx, uowFactory, guildID, wordleChannelID)

	mockResolver := newMockUserResolver()
	handler := application.NewWordleHandler(uowFactory, mockResolver)

	content := `Today's results:
3/6: <@999999999>`

	t.Run("Message From Configured Channel", func(t *testing.T) {
		message := createWordleMessage(
			strconv.FormatInt(guildID, 10),
			strconv.FormatInt(wordleChannelID, 10),
			content,
		)

		err := handler.HandleDiscordMessage(ctx, message)
		require.NoError(t, err)

		// Verify user was created
		uow := uowFactory.CreateForGuild(guildID)
		require.NoError(t, uow.Begin(ctx))
		defer uow.Rollback()

		user, err := uow.UserRepository().GetByDiscordID(ctx, 999999999)
		require.NoError(t, err)
		assert.Equal(t, int64(7000), user.Balance)
	})

	t.Run("Message From Different Channel", func(t *testing.T) {
		message := createWordleMessage(
			strconv.FormatInt(guildID, 10),
			strconv.FormatInt(otherChannelID, 10),
			content,
		)

		err := handler.HandleDiscordMessage(ctx, message)
		require.NoError(t, err)

		// Message should be ignored - no new completions
		uow := uowFactory.CreateForGuild(guildID)
		require.NoError(t, uow.Begin(ctx))
		defer uow.Rollback()

		history, err := uow.BalanceHistoryRepository().GetByUser(ctx, 999999999, 10)
		require.NoError(t, err)
		assert.Len(t, history, 1) // Still only one from the first test
	})

	t.Run("Guild Without Wordle Channel", func(t *testing.T) {
		noWordleGuildID := int64(101010)
		
		// Create guild without Wordle channel
		uow := uowFactory.CreateForGuild(noWordleGuildID)
		require.NoError(t, uow.Begin(ctx))
		_, err := uow.GuildSettingsRepository().GetOrCreateGuildSettings(ctx, noWordleGuildID)
		require.NoError(t, err)
		require.NoError(t, uow.Commit())

		message := createWordleMessage(
			strconv.FormatInt(noWordleGuildID, 10),
			"123456",
			content,
		)

		err = handler.HandleDiscordMessage(ctx, message)
		require.NoError(t, err)
		// Message should be ignored completely
	})
}

func TestWordleHandler_InvalidMessages(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config.SetTestConfig(config.NewTestConfig())
	defer config.ResetConfig()

	testDB := testutil.SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	noopPublisher := infrastructure.NewNoopEventPublisher()
	uowFactory := infrastructure.NewUnitOfWorkFactory(testDB.DB, noopPublisher)

	ctx := context.Background()
	guildID := int64(131313)
	channelID := int64(141414)

	setupWordleGuild(t, ctx, uowFactory, guildID, channelID)

	mockResolver := newMockUserResolver()
	handler := application.NewWordleHandler(uowFactory, mockResolver)

	testCases := []struct {
		name    string
		content string
		userID  string // Override Wordle bot ID to test non-bot messages
	}{
		{
			name:    "Non-Wordle Bot Message",
			content: "3/6: <@123123123>",
			userID:  "987654321", // Not the Wordle bot
		},
		{
			name:    "No Score Pattern",
			content: "Just a regular message with <@123123123>",
		},
		{
			name:    "Score Without Users",
			content: "3/6: Just text, no user mentions",
		},
		{
			name:    "Invalid Score Format",
			content: "X/6: <@123123123>",
		},
		{
			name:    "Empty Message",
			content: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			userID := config.Get().WordleBotID
			if tc.userID != "" {
				userID = tc.userID
			}

			message := events.DiscordMessageEvent{
				MessageID: fmt.Sprintf("test-%s", tc.name),
				UserID:    userID,
				GuildID:   strconv.FormatInt(guildID, 10),
				ChannelID: strconv.FormatInt(channelID, 10),
				Content:   tc.content,
			}

			err := handler.HandleDiscordMessage(ctx, message)
			require.NoError(t, err) // Should not error, just ignore invalid messages
		})
	}

	// Verify no users were created from invalid messages
	uow := uowFactory.CreateForGuild(guildID)
	require.NoError(t, uow.Begin(ctx))
	defer uow.Rollback()

	// Check that user 123123123 was never created
	user, err := uow.UserRepository().GetByDiscordID(ctx, 123123123)
	assert.NoError(t, err)
	assert.Nil(t, user) // Should not exist
}