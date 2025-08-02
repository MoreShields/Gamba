package repository

import (
	"context"
	"testing"

	"gambler/discord-client/domain/entities"
	"gambler/discord-client/repository/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWagerRepository_UpdateMessageIDs(t *testing.T) {
	testDB := testutil.SetupTestDatabase(t)
	ctx := context.Background()
	testGuildID := int64(1018733499869577296)

	// Create test users
	userRepo := NewUserRepository(testDB.DB)
	proposer, err := userRepo.Create(ctx, 123456789, "proposer", 100000)
	require.NoError(t, err)
	target, err := userRepo.Create(ctx, 987654321, "target", 100000)
	require.NoError(t, err)

	// Create test wager
	wagerRepo := NewWagerRepositoryScoped(testDB.DB.Pool, testGuildID)
	wager := &entities.Wager{
		ProposerDiscordID: proposer.DiscordID,
		TargetDiscordID:   target.DiscordID,
		GuildID:           testGuildID,
		Amount:            1000,
		Condition:         "Test wager condition",
		State:             entities.WagerStateProposed,
		MessageID:         nil,
		ChannelID:         nil,
	}

	err = wagerRepo.Create(ctx, wager)
	require.NoError(t, err)
	require.NotEqual(t, int64(0), wager.ID)

	// Update with message IDs
	testMessageID := int64(1394399185394077766)
	testChannelID := int64(1018733499869577296)

	wager.MessageID = &testMessageID
	wager.ChannelID = &testChannelID

	err = wagerRepo.Update(ctx, wager)
	require.NoError(t, err)

	// Fetch the wager back and verify message IDs were saved
	savedWager, err := wagerRepo.GetByID(ctx, wager.ID)
	require.NoError(t, err)
	require.NotNil(t, savedWager)

	// This test should pass once we fix the Update method
	assert.NotNil(t, savedWager.MessageID, "MessageID should not be nil after update")
	assert.NotNil(t, savedWager.ChannelID, "ChannelID should not be nil after update")

	if savedWager.MessageID != nil {
		assert.Equal(t, testMessageID, *savedWager.MessageID, "MessageID should match")
	}
	if savedWager.ChannelID != nil {
		assert.Equal(t, testChannelID, *savedWager.ChannelID, "ChannelID should match")
	}
}
