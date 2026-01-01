package services_test

import (
	"context"
	"testing"
	"time"

	"gambler/discord-client/config"
	"gambler/discord-client/domain/entities"
	"gambler/discord-client/domain/interfaces"
	"gambler/discord-client/domain/services"
	"gambler/discord-client/domain/testhelpers"
	"gambler/discord-client/repository"
	"gambler/discord-client/repository/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	testGuildID = int64(999888777)
)

// lotteryTestContext holds all test dependencies
type lotteryTestContext struct {
	testDB             *testutil.TestDatabase
	userRepo           interfaces.UserRepository
	lotteryDrawRepo    interfaces.LotteryDrawRepository
	lotteryTicketRepo  interfaces.LotteryTicketRepository
	lotteryWinnerRepo  interfaces.LotteryWinnerRepository
	balanceHistoryRepo interfaces.BalanceHistoryRepository
	guildSettingsRepo  *repository.GuildSettingsRepository
	wagerRepo          interfaces.WagerRepository
	groupWagerRepo     interfaces.GroupWagerRepository
	eventPublisher     *testhelpers.MockEventPublisher
	lotteryService     interfaces.LotteryService
}

// setupLotteryIntegrationTest creates all repositories and the lottery service for integration tests
func setupLotteryIntegrationTest(t *testing.T, guildID int64) *lotteryTestContext {
	testDB := testutil.SetupTestDatabase(t)

	// Create repositories - lottery repos need scoped constructors
	userRepo := repository.NewUserRepositoryScoped(testDB.DB.Pool, guildID)
	lotteryDrawRepo := repository.NewLotteryDrawRepositoryScoped(testDB.DB.Pool, guildID)
	lotteryTicketRepo := repository.NewLotteryTicketRepositoryScoped(testDB.DB.Pool, guildID)
	lotteryWinnerRepo := repository.NewLotteryWinnerRepositoryScoped(testDB.DB.Pool, guildID)
	balanceHistoryRepo := repository.NewBalanceHistoryRepositoryScoped(testDB.DB.Pool, guildID)
	guildSettingsRepo := repository.NewGuildSettingsRepository(testDB.DB)
	wagerRepo := repository.NewWagerRepositoryScoped(testDB.DB.Pool, guildID)
	groupWagerRepo := repository.NewGroupWagerRepositoryScoped(testDB.DB.Pool, guildID)

	eventPublisher := &testhelpers.MockEventPublisher{}
	eventPublisher.On("Publish", mock.Anything).Return(nil)

	// Create service
	lotteryService := services.NewLotteryService(
		lotteryDrawRepo,
		lotteryTicketRepo,
		lotteryWinnerRepo,
		userRepo,
		wagerRepo,
		groupWagerRepo,
		balanceHistoryRepo,
		guildSettingsRepo,
		eventPublisher,
	)

	return &lotteryTestContext{
		testDB:             testDB,
		userRepo:           userRepo,
		lotteryDrawRepo:    lotteryDrawRepo,
		lotteryTicketRepo:  lotteryTicketRepo,
		lotteryWinnerRepo:  lotteryWinnerRepo,
		balanceHistoryRepo: balanceHistoryRepo,
		guildSettingsRepo:  guildSettingsRepo,
		wagerRepo:          wagerRepo,
		groupWagerRepo:     groupWagerRepo,
		eventPublisher:     eventPublisher,
		lotteryService:     lotteryService,
	}
}

// createGuildSettingsWithLottery creates guild settings with lottery configuration
func createGuildSettingsWithLottery(ctx context.Context, t *testing.T, repo *repository.GuildSettingsRepository, guildID int64, difficulty, ticketCost int64) *entities.GuildSettings {
	settings, err := repo.GetOrCreateGuildSettings(ctx, guildID)
	require.NoError(t, err)

	settings.LottoDifficulty = &difficulty
	settings.LottoTicketCost = &ticketCost

	err = repo.UpdateGuildSettings(ctx, settings)
	require.NoError(t, err)

	return settings
}

func TestLotteryPurchaseTickets_Integration(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Verify config is set up
	cfg := config.Get()
	require.NotNil(t, cfg)

	ctx := context.Background()

	t.Run("purchase_single_ticket", func(t *testing.T) {
		t.Parallel()

		guildID := int64(100001)
		tc := setupLotteryIntegrationTest(t, guildID)

		// Setup: Create guild settings with lottery enabled
		ticketCost := int64(1000)
		difficulty := int64(8) // 256 possible numbers
		createGuildSettingsWithLottery(ctx, t, tc.guildSettingsRepo, guildID, difficulty, ticketCost)

		// Create user with initial balance
		user, err := tc.userRepo.Create(ctx, 111111, "buyer1", 100000)
		require.NoError(t, err)

		// Purchase single ticket
		result, err := tc.lotteryService.PurchaseTickets(ctx, user.DiscordID, guildID, 1)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Verify result structure
		assert.Len(t, result.Tickets, 1)
		assert.Equal(t, ticketCost, result.TotalCost)
		assert.Equal(t, int64(100000-ticketCost), result.NewBalance)

		// Verify ticket was created in database
		tickets, err := tc.lotteryTicketRepo.GetByUserForDraw(ctx, result.Draw.ID, user.DiscordID)
		require.NoError(t, err)
		assert.Len(t, tickets, 1)
		assert.Equal(t, ticketCost, tickets[0].PurchasePrice)

		// Verify pot was incremented
		draw, err := tc.lotteryDrawRepo.GetByID(ctx, result.Draw.ID)
		require.NoError(t, err)
		assert.Equal(t, ticketCost, draw.TotalPot)

		// Verify user balance was deducted
		updatedUser, err := tc.userRepo.GetByDiscordID(ctx, user.DiscordID)
		require.NoError(t, err)
		assert.Equal(t, int64(100000-ticketCost), updatedUser.Balance)

		// Verify balance history was recorded
		history, err := tc.balanceHistoryRepo.GetByUser(ctx, user.DiscordID, 10)
		require.NoError(t, err)
		require.Len(t, history, 1)
		assert.Equal(t, entities.TransactionTypeLottoTicket, history[0].TransactionType)
		assert.Equal(t, -ticketCost, history[0].ChangeAmount)
	})

	t.Run("purchase_multiple_tickets", func(t *testing.T) {
		t.Parallel()

		guildID := int64(100002)
		tc := setupLotteryIntegrationTest(t, guildID)

		// Setup
		ticketCost := int64(500)
		quantity := 5
		createGuildSettingsWithLottery(ctx, t, tc.guildSettingsRepo, guildID, 8, ticketCost)

		user, err := tc.userRepo.Create(ctx, 222222, "buyer2", 100000)
		require.NoError(t, err)

		// Purchase multiple tickets
		result, err := tc.lotteryService.PurchaseTickets(ctx, user.DiscordID, guildID, quantity)
		require.NoError(t, err)

		// Verify correct number of tickets
		assert.Len(t, result.Tickets, quantity)
		assert.Equal(t, ticketCost*int64(quantity), result.TotalCost)

		// Verify all ticket numbers are unique
		ticketNumbers := make(map[int64]bool)
		for _, ticket := range result.Tickets {
			assert.False(t, ticketNumbers[ticket.TicketNumber], "duplicate ticket number found")
			ticketNumbers[ticket.TicketNumber] = true
		}

		// Verify pot reflects total purchase
		draw, err := tc.lotteryDrawRepo.GetByID(ctx, result.Draw.ID)
		require.NoError(t, err)
		assert.Equal(t, ticketCost*int64(quantity), draw.TotalPot)

		// Verify database has all tickets
		dbTickets, err := tc.lotteryTicketRepo.GetByUserForDraw(ctx, draw.ID, user.DiscordID)
		require.NoError(t, err)
		assert.Len(t, dbTickets, quantity)
	})

	t.Run("multiple_users_purchase_tickets", func(t *testing.T) {
		t.Parallel()

		guildID := int64(100003)
		tc := setupLotteryIntegrationTest(t, guildID)

		// Setup
		ticketCost := int64(1000)
		createGuildSettingsWithLottery(ctx, t, tc.guildSettingsRepo, guildID, 8, ticketCost)

		user1, err := tc.userRepo.Create(ctx, 333333, "buyer3", 100000)
		require.NoError(t, err)
		user2, err := tc.userRepo.Create(ctx, 444444, "buyer4", 100000)
		require.NoError(t, err)

		// User 1 buys 3 tickets
		result1, err := tc.lotteryService.PurchaseTickets(ctx, user1.DiscordID, guildID, 3)
		require.NoError(t, err)

		// User 2 buys 2 tickets
		result2, err := tc.lotteryService.PurchaseTickets(ctx, user2.DiscordID, guildID, 2)
		require.NoError(t, err)

		// Verify they share the same draw
		assert.Equal(t, result1.Draw.ID, result2.Draw.ID)

		// Verify pot has combined amounts
		draw, err := tc.lotteryDrawRepo.GetByID(ctx, result1.Draw.ID)
		require.NoError(t, err)
		assert.Equal(t, ticketCost*5, draw.TotalPot) // 3 + 2 tickets

		// Verify each user has correct number of tickets
		user1Tickets, err := tc.lotteryTicketRepo.GetByUserForDraw(ctx, draw.ID, user1.DiscordID)
		require.NoError(t, err)
		assert.Len(t, user1Tickets, 3)

		user2Tickets, err := tc.lotteryTicketRepo.GetByUserForDraw(ctx, draw.ID, user2.DiscordID)
		require.NoError(t, err)
		assert.Len(t, user2Tickets, 2)

		// Verify each user's balance is correctly deducted
		updatedUser1, err := tc.userRepo.GetByDiscordID(ctx, user1.DiscordID)
		require.NoError(t, err)
		assert.Equal(t, int64(100000-3*ticketCost), updatedUser1.Balance)

		updatedUser2, err := tc.userRepo.GetByDiscordID(ctx, user2.DiscordID)
		require.NoError(t, err)
		assert.Equal(t, int64(100000-2*ticketCost), updatedUser2.Balance)
	})

	t.Run("insufficient_balance_rejected", func(t *testing.T) {
		t.Parallel()

		guildID := int64(100004)
		tc := setupLotteryIntegrationTest(t, guildID)

		// Setup with high ticket cost
		ticketCost := int64(10000)
		createGuildSettingsWithLottery(ctx, t, tc.guildSettingsRepo, guildID, 8, ticketCost)

		// Create user with low balance
		user, err := tc.userRepo.Create(ctx, 555555, "pooruser", 5000)
		require.NoError(t, err)

		// Attempt to purchase - should fail
		result, err := tc.lotteryService.PurchaseTickets(ctx, user.DiscordID, guildID, 1)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "insufficient balance")

		// Verify user balance unchanged
		updatedUser, err := tc.userRepo.GetByDiscordID(ctx, user.DiscordID)
		require.NoError(t, err)
		assert.Equal(t, int64(5000), updatedUser.Balance)
	})

	t.Run("respects_locked_wager_amounts", func(t *testing.T) {
		t.Parallel()

		guildID := int64(100005)
		tc := setupLotteryIntegrationTest(t, guildID)

		// Setup
		ticketCost := int64(1000)
		createGuildSettingsWithLottery(ctx, t, tc.guildSettingsRepo, guildID, 8, ticketCost)

		// Create user with 10000 balance
		user, err := tc.userRepo.Create(ctx, 666666, "wageruser", 10000)
		require.NoError(t, err)

		// Create target user for the wager (required by foreign key constraint)
		_, err = tc.userRepo.Create(ctx, 777777, "targetuser", 10000)
		require.NoError(t, err)

		// Create an active wager that locks 8000 of the user's balance
		msgID := int64(123456)
		chID := int64(789012)
		wager := &entities.Wager{
			GuildID:           guildID,
			ProposerDiscordID: user.DiscordID,
			TargetDiscordID:   777777, // another user
			Amount:            8000,
			State:             entities.WagerStateVoting,
			Condition:         "test condition",
			MessageID:         &msgID,
			ChannelID:         &chID,
		}
		err = tc.wagerRepo.Create(ctx, wager)
		require.NoError(t, err)

		// User has 10000 balance but 8000 is locked, so only 2000 available
		// Trying to buy 3 tickets at 1000 each (3000 total) should fail
		result, err := tc.lotteryService.PurchaseTickets(ctx, user.DiscordID, guildID, 3)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "insufficient balance")

		// But buying 2 tickets (2000 total) should work
		result, err = tc.lotteryService.PurchaseTickets(ctx, user.DiscordID, guildID, 2)
		require.NoError(t, err)
		assert.Len(t, result.Tickets, 2)
	})
}

func TestLotteryDrawConduct_Integration(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	cfg := config.Get()
	require.NotNil(t, cfg)

	ctx := context.Background()

	t.Run("single_winner_gets_full_pot", func(t *testing.T) {
		t.Parallel()

		guildID := int64(200001)
		tc := setupLotteryIntegrationTest(t, guildID)

		// Setup with very low difficulty (2 = 4 possible numbers)
		// This makes it guaranteed that one of the 4 tickets will win
		ticketCost := int64(1000)
		difficulty := int64(2) // Only 4 possible numbers: 0, 1, 2, 3
		createGuildSettingsWithLottery(ctx, t, tc.guildSettingsRepo, guildID, difficulty, ticketCost)

		// Create user and buy all 4 possible tickets to guarantee a winner
		user, err := tc.userRepo.Create(ctx, 111111, "winner", 100000)
		require.NoError(t, err)

		result, err := tc.lotteryService.PurchaseTickets(ctx, user.DiscordID, guildID, 4)
		require.NoError(t, err)
		require.Len(t, result.Tickets, 4)

		draw := result.Draw
		potAmount := draw.TotalPot
		initialBalance := int64(100000) - potAmount // Balance after purchase

		// Conduct the draw
		drawResult, err := tc.lotteryService.ConductDraw(ctx, draw)
		require.NoError(t, err)
		require.NotNil(t, drawResult)

		// Verify there's exactly one winner (the only participant)
		assert.False(t, drawResult.RolledOver)
		assert.Len(t, drawResult.Winners, 1)
		assert.Equal(t, user.DiscordID, drawResult.Winners[0].DiscordID)
		assert.Equal(t, potAmount, drawResult.PotAmount)

		// Verify winner balance was updated
		updatedUser, err := tc.userRepo.GetByDiscordID(ctx, user.DiscordID)
		require.NoError(t, err)
		assert.Equal(t, initialBalance+potAmount, updatedUser.Balance)

		// Verify draw is marked completed
		completedDraw, err := tc.lotteryDrawRepo.GetByID(ctx, draw.ID)
		require.NoError(t, err)
		assert.True(t, completedDraw.IsCompleted())
		assert.NotNil(t, completedDraw.WinningNumber)

		// Verify winner record was created
		winners, err := tc.lotteryWinnerRepo.GetByDrawID(ctx, draw.ID)
		require.NoError(t, err)
		require.Len(t, winners, 1)
		assert.Equal(t, user.DiscordID, winners[0].DiscordID)
		assert.Equal(t, potAmount, winners[0].WinningAmount)

		// Verify balance history for win
		history, err := tc.balanceHistoryRepo.GetByUser(ctx, user.DiscordID, 10)
		require.NoError(t, err)
		// Should have ticket purchase and win
		var hasWinEntry bool
		for _, h := range history {
			if h.TransactionType == entities.TransactionTypeLottoWin {
				hasWinEntry = true
				assert.Equal(t, potAmount, h.ChangeAmount)
			}
		}
		assert.True(t, hasWinEntry, "expected lotto_win balance history entry")
	})

	t.Run("multiple_winners_split_pot", func(t *testing.T) {
		t.Parallel()

		guildID := int64(200002)
		tc := setupLotteryIntegrationTest(t, guildID)

		// Use difficulty of 2 (4 numbers) and have 2 users each buy 2 tickets
		ticketCost := int64(1000)
		difficulty := int64(2)
		createGuildSettingsWithLottery(ctx, t, tc.guildSettingsRepo, guildID, difficulty, ticketCost)

		user1, err := tc.userRepo.Create(ctx, 111111, "player1", 100000)
		require.NoError(t, err)
		user2, err := tc.userRepo.Create(ctx, 222222, "player2", 100000)
		require.NoError(t, err)

		// Each user buys 2 tickets - together they cover all 4 numbers
		result1, err := tc.lotteryService.PurchaseTickets(ctx, user1.DiscordID, guildID, 2)
		require.NoError(t, err)
		result2, err := tc.lotteryService.PurchaseTickets(ctx, user2.DiscordID, guildID, 2)
		require.NoError(t, err)

		draw := result1.Draw
		totalPot := draw.TotalPot + ticketCost*2 // After second user's purchase

		// Refresh draw to get correct pot
		draw, err = tc.lotteryDrawRepo.GetByID(ctx, draw.ID)
		require.NoError(t, err)
		totalPot = draw.TotalPot

		// Check if both users happen to have the same winning number
		// (This is a probabilistic test - we check the outcome is valid)
		allTickets := append(result1.Tickets, result2.Tickets...)
		ticketsByNumber := make(map[int64][]*entities.LotteryTicket)
		for _, ticket := range allTickets {
			ticketsByNumber[ticket.TicketNumber] = append(ticketsByNumber[ticket.TicketNumber], ticket)
		}
		_ = ticketsByNumber // Used for debugging if needed

		// Conduct draw
		drawResult, err := tc.lotteryService.ConductDraw(ctx, draw)
		require.NoError(t, err)

		// We should have at least one winner since all numbers are covered
		assert.False(t, drawResult.RolledOver)
		assert.GreaterOrEqual(t, len(drawResult.Winners), 1)

		// If there are multiple winners, verify pot was split
		if len(drawResult.Winners) > 1 {
			winningsPerWinner := totalPot / int64(len(drawResult.Winners))
			for _, winner := range drawResult.Winners {
				winnerRecords, err := tc.lotteryWinnerRepo.GetByDrawID(ctx, draw.ID)
				require.NoError(t, err)
				for _, record := range winnerRecords {
					if record.DiscordID == winner.DiscordID {
						assert.Equal(t, winningsPerWinner, record.WinningAmount)
					}
				}
			}
		}
	})

	t.Run("no_winner_rollover", func(t *testing.T) {
		t.Parallel()

		guildID := int64(200003)
		tc := setupLotteryIntegrationTest(t, guildID)

		// Setup
		ticketCost := int64(1000)
		difficulty := int64(8)
		createGuildSettingsWithLottery(ctx, t, tc.guildSettingsRepo, guildID, difficulty, ticketCost)

		// Create a draw with a pot but no tickets (simulating a scenario where tickets exist but none match)
		// We'll manually create a draw with some pot
		draw, err := tc.lotteryService.GetOrCreateCurrentDraw(ctx, guildID)
		require.NoError(t, err)

		// Add some pot manually (simulating previous rollover)
		err = tc.lotteryDrawRepo.IncrementPot(ctx, draw.ID, 10000)
		require.NoError(t, err)

		draw, err = tc.lotteryDrawRepo.GetByID(ctx, draw.ID)
		require.NoError(t, err)
		initialPot := draw.TotalPot

		// Conduct draw with no tickets - guaranteed rollover
		drawResult, err := tc.lotteryService.ConductDraw(ctx, draw)
		require.NoError(t, err)

		assert.True(t, drawResult.RolledOver)
		assert.Empty(t, drawResult.Winners)
		assert.Equal(t, initialPot, drawResult.PotAmount)

		// Verify next draw was created with carried pot
		assert.NotNil(t, drawResult.NextDraw)

		// Query the next draw from database directly to verify pot was carried over
		// (in-memory state may not reflect final DB state)
		nextDrawFromDB, err := tc.lotteryDrawRepo.GetByID(ctx, drawResult.NextDraw.ID)
		require.NoError(t, err)
		assert.Equal(t, initialPot, nextDrawFromDB.TotalPot)

		// Verify original draw is completed
		completedDraw, err := tc.lotteryDrawRepo.GetByID(ctx, draw.ID)
		require.NoError(t, err)
		assert.True(t, completedDraw.IsCompleted())
	})

	t.Run("already_completed_draw_rejected", func(t *testing.T) {
		t.Parallel()

		guildID := int64(200004)
		tc := setupLotteryIntegrationTest(t, guildID)

		// Setup
		ticketCost := int64(1000)
		difficulty := int64(2)
		createGuildSettingsWithLottery(ctx, t, tc.guildSettingsRepo, guildID, difficulty, ticketCost)

		user, err := tc.userRepo.Create(ctx, 111111, "player", 100000)
		require.NoError(t, err)

		// Buy tickets and conduct draw once
		result, err := tc.lotteryService.PurchaseTickets(ctx, user.DiscordID, guildID, 4)
		require.NoError(t, err)
		draw := result.Draw

		_, err = tc.lotteryService.ConductDraw(ctx, draw)
		require.NoError(t, err)

		// Try to conduct the same draw again - should fail
		// Refresh draw from DB first
		completedDraw, err := tc.lotteryDrawRepo.GetByID(ctx, draw.ID)
		require.NoError(t, err)

		secondResult, err := tc.lotteryService.ConductDraw(ctx, completedDraw)
		assert.Error(t, err)
		assert.Nil(t, secondResult)
		assert.Contains(t, err.Error(), "already completed")
	})
}

func TestLotteryDrawInfo_Integration(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	cfg := config.Get()
	require.NotNil(t, cfg)

	ctx := context.Background()

	t.Run("returns_correct_ticket_count", func(t *testing.T) {
		t.Parallel()

		guildID := int64(300001)
		tc := setupLotteryIntegrationTest(t, guildID)

		// Setup
		ticketCost := int64(1000)
		createGuildSettingsWithLottery(ctx, t, tc.guildSettingsRepo, guildID, 8, ticketCost)

		user1, err := tc.userRepo.Create(ctx, 111111, "player1", 100000)
		require.NoError(t, err)
		user2, err := tc.userRepo.Create(ctx, 222222, "player2", 100000)
		require.NoError(t, err)

		// User1 buys 3 tickets
		_, err = tc.lotteryService.PurchaseTickets(ctx, user1.DiscordID, guildID, 3)
		require.NoError(t, err)

		// User2 buys 5 tickets
		_, err = tc.lotteryService.PurchaseTickets(ctx, user2.DiscordID, guildID, 5)
		require.NoError(t, err)

		// Get draw info
		drawInfo, err := tc.lotteryService.GetDrawInfo(ctx, guildID)
		require.NoError(t, err)

		assert.Equal(t, int64(8), drawInfo.TicketCount)
		assert.Equal(t, ticketCost*8, drawInfo.Draw.TotalPot)
	})

	t.Run("returns_participant_summary", func(t *testing.T) {
		t.Parallel()

		guildID := int64(300002)
		tc := setupLotteryIntegrationTest(t, guildID)

		// Setup
		ticketCost := int64(500)
		createGuildSettingsWithLottery(ctx, t, tc.guildSettingsRepo, guildID, 8, ticketCost)

		user1, err := tc.userRepo.Create(ctx, 111111, "player1", 100000)
		require.NoError(t, err)
		user2, err := tc.userRepo.Create(ctx, 222222, "player2", 100000)
		require.NoError(t, err)
		user3, err := tc.userRepo.Create(ctx, 333333, "player3", 100000)
		require.NoError(t, err)

		// Different users buy different amounts
		_, err = tc.lotteryService.PurchaseTickets(ctx, user1.DiscordID, guildID, 2)
		require.NoError(t, err)
		_, err = tc.lotteryService.PurchaseTickets(ctx, user2.DiscordID, guildID, 5)
		require.NoError(t, err)
		_, err = tc.lotteryService.PurchaseTickets(ctx, user3.DiscordID, guildID, 1)
		require.NoError(t, err)

		// Get draw info
		drawInfo, err := tc.lotteryService.GetDrawInfo(ctx, guildID)
		require.NoError(t, err)

		assert.Len(t, drawInfo.Participants, 3)

		// Create map for easier assertion
		participantTickets := make(map[int64]int64)
		for _, p := range drawInfo.Participants {
			participantTickets[p.DiscordID] = p.TicketCount
		}

		assert.Equal(t, int64(2), participantTickets[user1.DiscordID])
		assert.Equal(t, int64(5), participantTickets[user2.DiscordID])
		assert.Equal(t, int64(1), participantTickets[user3.DiscordID])
	})

	t.Run("creates_draw_if_none_exists", func(t *testing.T) {
		t.Parallel()

		guildID := int64(300003)
		tc := setupLotteryIntegrationTest(t, guildID)

		// Setup guild settings but don't create any draw
		createGuildSettingsWithLottery(ctx, t, tc.guildSettingsRepo, guildID, 8, 1000)

		// Verify no draws exist
		existingDraw, err := tc.lotteryDrawRepo.GetCurrentOpenDraw(ctx, guildID)
		require.NoError(t, err)
		assert.Nil(t, existingDraw)

		// GetDrawInfo should create a new draw
		drawInfo, err := tc.lotteryService.GetDrawInfo(ctx, guildID)
		require.NoError(t, err)
		require.NotNil(t, drawInfo)
		require.NotNil(t, drawInfo.Draw)

		// Verify draw now exists in database
		createdDraw, err := tc.lotteryDrawRepo.GetCurrentOpenDraw(ctx, guildID)
		require.NoError(t, err)
		assert.NotNil(t, createdDraw)
		assert.Equal(t, drawInfo.Draw.ID, createdDraw.ID)

		// Verify draw has correct settings
		assert.Equal(t, int64(8), createdDraw.Difficulty)
		assert.Equal(t, int64(1000), createdDraw.TicketCost)
		assert.Equal(t, guildID, createdDraw.GuildID)
	})
}

func TestLotteryEdgeCases_Integration(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	cfg := config.Get()
	require.NotNil(t, cfg)

	ctx := context.Background()

	t.Run("multiple_users_purchase_tickets_sequentially", func(t *testing.T) {
		t.Parallel()

		guildID := int64(400001)
		tc := setupLotteryIntegrationTest(t, guildID)

		// Setup with small number range
		ticketCost := int64(100)
		difficulty := int64(4) // 16 possible numbers
		createGuildSettingsWithLottery(ctx, t, tc.guildSettingsRepo, guildID, difficulty, ticketCost)

		// Create 4 users
		users := make([]*entities.User, 4)
		for i := 0; i < 4; i++ {
			user, err := tc.userRepo.Create(ctx, int64(100000+i), "player", 100000)
			require.NoError(t, err)
			users[i] = user
		}

		// Each user buys 4 tickets sequentially
		for _, user := range users {
			_, err := tc.lotteryService.PurchaseTickets(ctx, user.DiscordID, guildID, 4)
			require.NoError(t, err)
		}

		// Verify all 16 tickets have unique numbers
		draw, err := tc.lotteryDrawRepo.GetCurrentOpenDraw(ctx, guildID)
		require.NoError(t, err)

		count, err := tc.lotteryTicketRepo.CountTicketsForDraw(ctx, draw.ID)
		require.NoError(t, err)
		assert.Equal(t, int64(16), count)

		// Verify pot is correct
		assert.Equal(t, ticketCost*16, draw.TotalPot)
	})

	t.Run("ticket_number_exhaustion", func(t *testing.T) {
		t.Parallel()

		guildID := int64(400002)
		tc := setupLotteryIntegrationTest(t, guildID)

		// Very low difficulty - only 4 possible numbers
		ticketCost := int64(100)
		difficulty := int64(2) // Only 4 numbers: 0, 1, 2, 3
		createGuildSettingsWithLottery(ctx, t, tc.guildSettingsRepo, guildID, difficulty, ticketCost)

		user, err := tc.userRepo.Create(ctx, 111111, "buyer", 100000)
		require.NoError(t, err)

		// Buy all 4 tickets
		_, err = tc.lotteryService.PurchaseTickets(ctx, user.DiscordID, guildID, 4)
		require.NoError(t, err)

		// Try to buy one more - should fail
		result, err := tc.lotteryService.PurchaseTickets(ctx, user.DiscordID, guildID, 1)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "only 0 more unique numbers available")
	})

	t.Run("balance_history_metadata_correct", func(t *testing.T) {
		t.Parallel()

		guildID := int64(400003)
		tc := setupLotteryIntegrationTest(t, guildID)

		// Setup
		ticketCost := int64(1000)
		createGuildSettingsWithLottery(ctx, t, tc.guildSettingsRepo, guildID, 8, ticketCost)

		user, err := tc.userRepo.Create(ctx, 111111, "buyer", 100000)
		require.NoError(t, err)

		// Buy tickets
		result, err := tc.lotteryService.PurchaseTickets(ctx, user.DiscordID, guildID, 3)
		require.NoError(t, err)

		// Get balance history
		history, err := tc.balanceHistoryRepo.GetByUser(ctx, user.DiscordID, 10)
		require.NoError(t, err)
		require.Len(t, history, 1)

		// Verify metadata
		metadata := history[0].TransactionMetadata
		require.NotNil(t, metadata)

		// Check metadata contains expected fields
		drawID, ok := metadata["draw_id"]
		assert.True(t, ok, "metadata should contain draw_id")
		assert.Equal(t, float64(result.Draw.ID), drawID) // JSON unmarshals to float64

		quantity, ok := metadata["quantity"]
		assert.True(t, ok, "metadata should contain quantity")
		assert.Equal(t, float64(3), quantity)

		ticketNumbers, ok := metadata["ticket_numbers"]
		assert.True(t, ok, "metadata should contain ticket_numbers")
		ticketNumSlice, ok := ticketNumbers.([]interface{})
		assert.True(t, ok)
		assert.Len(t, ticketNumSlice, 3)
	})

	t.Run("draw_time_is_friday_2pm_utc", func(t *testing.T) {
		t.Parallel()

		guildID := int64(400004)
		tc := setupLotteryIntegrationTest(t, guildID)

		// Setup
		createGuildSettingsWithLottery(ctx, t, tc.guildSettingsRepo, guildID, 8, 1000)

		// Get or create draw
		draw, err := tc.lotteryService.GetOrCreateCurrentDraw(ctx, guildID)
		require.NoError(t, err)

		// Verify draw time is a Friday at 2pm UTC
		assert.Equal(t, time.Friday, draw.DrawTime.Weekday())
		assert.Equal(t, 14, draw.DrawTime.Hour())
		assert.Equal(t, 0, draw.DrawTime.Minute())
		assert.Equal(t, 0, draw.DrawTime.Second())
		assert.Equal(t, time.UTC, draw.DrawTime.Location())
	})
}
