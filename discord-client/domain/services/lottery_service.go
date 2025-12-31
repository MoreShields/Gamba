package services

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"time"

	"gambler/discord-client/domain/entities"
	"gambler/discord-client/domain/interfaces"
	"gambler/discord-client/domain/utils"

	log "github.com/sirupsen/logrus"
)

const (
	// denseThreshold is the number pool size below which we always enumerate available numbers.
	// 65536 (2^16) ensures enumeration is fast and memory-bounded even at 100% usage.
	denseThreshold = 1 << 16
)

// lotteryService implements business logic for lottery operations
type lotteryService struct {
	lotteryDrawRepo    interfaces.LotteryDrawRepository
	lotteryTicketRepo  interfaces.LotteryTicketRepository
	lotteryWinnerRepo  interfaces.LotteryWinnerRepository
	userRepo           interfaces.UserRepository
	wagerRepo          interfaces.WagerRepository
	groupWagerRepo     interfaces.GroupWagerRepository
	balanceHistoryRepo interfaces.BalanceHistoryRepository
	guildSettingsRepo  interfaces.GuildSettingsRepository
	eventPublisher     interfaces.EventPublisher
}

// NewLotteryService creates a new lottery service
func NewLotteryService(
	lotteryDrawRepo interfaces.LotteryDrawRepository,
	lotteryTicketRepo interfaces.LotteryTicketRepository,
	lotteryWinnerRepo interfaces.LotteryWinnerRepository,
	userRepo interfaces.UserRepository,
	wagerRepo interfaces.WagerRepository,
	groupWagerRepo interfaces.GroupWagerRepository,
	balanceHistoryRepo interfaces.BalanceHistoryRepository,
	guildSettingsRepo interfaces.GuildSettingsRepository,
	eventPublisher interfaces.EventPublisher,
) interfaces.LotteryService {
	return &lotteryService{
		lotteryDrawRepo:    lotteryDrawRepo,
		lotteryTicketRepo:  lotteryTicketRepo,
		lotteryWinnerRepo:  lotteryWinnerRepo,
		userRepo:           userRepo,
		wagerRepo:          wagerRepo,
		groupWagerRepo:     groupWagerRepo,
		balanceHistoryRepo: balanceHistoryRepo,
		guildSettingsRepo:  guildSettingsRepo,
		eventPublisher:     eventPublisher,
	}
}

// CalculateNextDrawTime calculates the next Friday 2pm UTC draw time
func (s *lotteryService) CalculateNextDrawTime() time.Time {
	now := time.Now().UTC()

	// Find next Friday
	daysUntilFriday := (time.Friday - now.Weekday() + 7) % 7
	if daysUntilFriday == 0 && now.Hour() >= 14 {
		// It's Friday but past 2pm, go to next Friday
		daysUntilFriday = 7
	}

	nextFriday := now.AddDate(0, 0, int(daysUntilFriday))
	return time.Date(nextFriday.Year(), nextFriday.Month(), nextFriday.Day(), 14, 0, 0, 0, time.UTC)
}

// GetOrCreateCurrentDraw gets the current open draw or creates one
func (s *lotteryService) GetOrCreateCurrentDraw(ctx context.Context, guildID int64) (*entities.LotteryDraw, error) {
	guildSettings, err := s.guildSettingsRepo.GetOrCreateGuildSettings(ctx, guildID)
	if err != nil {
		return nil, fmt.Errorf("failed to get guild settings: %w", err)
	}

	difficulty := guildSettings.GetLottoDifficulty()
	ticketCost := guildSettings.GetLottoTicketCost()
	nextDrawTime := s.CalculateNextDrawTime()

	draw, err := s.lotteryDrawRepo.GetOrCreateCurrentDraw(ctx, guildID, nextDrawTime, difficulty, ticketCost)
	if err != nil {
		return nil, fmt.Errorf("failed to get or create lottery draw: %w", err)
	}

	return draw, nil
}

// PurchaseTickets buys lottery tickets for a user
func (s *lotteryService) PurchaseTickets(ctx context.Context, discordID, guildID int64, quantity int) (*interfaces.LotteryPurchaseResult, error) {
	if quantity <= 0 {
		return nil, errors.New("quantity must be positive")
	}

	// Get current draw
	draw, err := s.GetOrCreateCurrentDraw(ctx, guildID)
	if err != nil {
		return nil, err
	}

	// Check if tickets can still be purchased
	if !draw.CanPurchaseTickets() {
		return nil, errors.New("tickets can no longer be purchased for this draw")
	}

	// Calculate total cost using draw's captured ticket cost
	ticketCost := draw.TicketCost
	totalCost := ticketCost * int64(quantity)

	// Get user
	user, err := s.userRepo.GetByDiscordID(ctx, discordID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, errors.New("user not found")
	}

	// Calculate available balance
	availableBalance, err := s.calculateAvailableBalance(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate available balance: %w", err)
	}

	if availableBalance < totalCost {
		return nil, fmt.Errorf("insufficient balance: have %d available, need %d", availableBalance, totalCost)
	}

	// Get numbers this user already has for this draw (to avoid duplicates)
	usedNumbers, err := s.lotteryTicketRepo.GetUsedNumbersByUser(ctx, draw.ID, discordID)
	if err != nil {
		return nil, fmt.Errorf("failed to get used numbers: %w", err)
	}

	// Check if user can get the requested quantity of unique numbers
	totalNumbers := draw.GetTotalNumbers()
	userAvailableNumbers := totalNumbers - int64(len(usedNumbers))
	if int64(quantity) > userAvailableNumbers {
		return nil, fmt.Errorf("only %d more unique numbers available", userAvailableNumbers)
	}

	// Build set of user's existing numbers for quick lookup
	usedSet := make(map[int64]bool)
	for _, num := range usedNumbers {
		usedSet[num] = true
	}

	// Generate unique ticket numbers
	ticketNumbers, err := s.generateUniqueNumbers(draw.GetTotalNumbers(), usedSet, quantity)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ticket numbers: %w", err)
	}

	// Update user balance
	newBalance := user.Balance - totalCost
	if err := s.userRepo.UpdateBalance(ctx, user.DiscordID, newBalance); err != nil {
		return nil, fmt.Errorf("failed to update user balance: %w", err)
	}

	// Record balance history
	history := &entities.BalanceHistory{
		DiscordID:       discordID,
		GuildID:         guildID,
		BalanceBefore:   user.Balance,
		BalanceAfter:    newBalance,
		ChangeAmount:    -totalCost,
		TransactionType: entities.TransactionTypeLottoTicket,
		TransactionMetadata: map[string]interface{}{
			"draw_id":        draw.ID,
			"quantity":       quantity,
			"ticket_cost":    ticketCost,
			"ticket_numbers": ticketNumbers,
		},
	}
	if err := utils.RecordBalanceChange(ctx, s.balanceHistoryRepo, s.eventPublisher, history); err != nil {
		return nil, fmt.Errorf("failed to record balance change: %w", err)
	}

	// Create ticket records
	tickets := make([]*entities.LotteryTicket, 0, quantity)
	for _, num := range ticketNumbers {
		ticket := &entities.LotteryTicket{
			DrawID:           draw.ID,
			GuildID:          guildID,
			DiscordID:        discordID,
			TicketNumber:     num,
			PurchasePrice:    ticketCost,
			BalanceHistoryID: history.ID,
		}
		tickets = append(tickets, ticket)
	}

	// Batch insert all tickets at once
	if err := s.lotteryTicketRepo.CreateBatch(ctx, tickets); err != nil {
		return nil, fmt.Errorf("failed to create tickets: %w", err)
	}

	// Increment pot
	if err := s.lotteryDrawRepo.IncrementPot(ctx, draw.ID, totalCost); err != nil {
		return nil, fmt.Errorf("failed to increment pot: %w", err)
	}

	// Refresh draw to get updated pot
	draw, err = s.lotteryDrawRepo.GetByID(ctx, draw.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh draw: %w", err)
	}

	return &interfaces.LotteryPurchaseResult{
		Tickets:    tickets,
		TotalCost:  totalCost,
		NewBalance: newBalance,
		Draw:       draw,
	}, nil
}

// GetUserTickets returns user's tickets for the current draw
func (s *lotteryService) GetUserTickets(ctx context.Context, discordID, guildID int64) ([]*entities.LotteryTicket, error) {
	draw, err := s.lotteryDrawRepo.GetCurrentOpenDraw(ctx, guildID)
	if err != nil {
		return nil, fmt.Errorf("failed to get current draw: %w", err)
	}
	if draw == nil {
		return nil, nil
	}

	tickets, err := s.lotteryTicketRepo.GetByUserForDraw(ctx, draw.ID, discordID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user tickets: %w", err)
	}

	return tickets, nil
}

// GetDrawInfo returns full draw info for display
func (s *lotteryService) GetDrawInfo(ctx context.Context, guildID int64) (*interfaces.LotteryDrawInfo, error) {
	draw, err := s.GetOrCreateCurrentDraw(ctx, guildID)
	if err != nil {
		return nil, err
	}

	ticketCount, err := s.lotteryTicketRepo.CountTicketsForDraw(ctx, draw.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to count tickets: %w", err)
	}

	participants, err := s.lotteryTicketRepo.GetParticipantSummary(ctx, draw.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get participant summary: %w", err)
	}

	return &interfaces.LotteryDrawInfo{
		Draw:         draw,
		TicketCount:  ticketCount,
		Participants: participants,
		TicketCost:   draw.TicketCost,
	}, nil
}

// ConductDraw processes the draw - selects winner, transfers pot
func (s *lotteryService) ConductDraw(ctx context.Context, draw *entities.LotteryDraw) (*interfaces.LotteryDrawResult, error) {
	// Check if already completed (idempotency guard)
	if draw.IsCompleted() {
		return nil, errors.New("draw already completed")
	}

	// Lock the draw for update
	lockedDraw, err := s.lotteryDrawRepo.GetByIDForUpdate(ctx, draw.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to lock draw: %w", err)
	}
	if lockedDraw == nil {
		return nil, errors.New("draw not found")
	}
	if lockedDraw.IsCompleted() {
		return nil, errors.New("draw already completed")
	}

	// Generate winning number using draw's stored difficulty
	winningNumber, err := lockedDraw.GenerateWinningNumber()
	if err != nil {
		return nil, fmt.Errorf("failed to generate winning number: %w", err)
	}

	// Find winning tickets
	winningTickets, err := s.lotteryTicketRepo.GetWinningTickets(ctx, draw.ID, winningNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get winning tickets: %w", err)
	}

	result := &interfaces.LotteryDrawResult{
		WinningNumber: winningNumber,
		PotAmount:     lockedDraw.TotalPot,
		RolledOver:    len(winningTickets) == 0,
	}

	if len(winningTickets) > 0 {
		// Build map of winner ID -> their winning tickets
		winnerTicketsMap := make(map[int64][]*entities.LotteryTicket)
		for _, ticket := range winningTickets {
			winnerTicketsMap[ticket.DiscordID] = append(winnerTicketsMap[ticket.DiscordID], ticket)
		}

		// Calculate winnings per winner
		winningsPerWinner := lockedDraw.TotalPot / int64(len(winnerTicketsMap))

		// Process each winner
		var winners []*entities.User
		for winnerID, winnerTickets := range winnerTicketsMap {
			user, err := s.userRepo.GetByDiscordID(ctx, winnerID)
			if err != nil {
				return nil, fmt.Errorf("failed to get winner user: %w", err)
			}
			if user == nil {
				continue
			}

			// Update winner balance
			newBalance := user.Balance + winningsPerWinner
			if err := s.userRepo.UpdateBalance(ctx, user.DiscordID, newBalance); err != nil {
				return nil, fmt.Errorf("failed to update winner balance: %w", err)
			}

			// Record balance history
			history := &entities.BalanceHistory{
				DiscordID:       winnerID,
				GuildID:         draw.GuildID,
				BalanceBefore:   user.Balance,
				BalanceAfter:    newBalance,
				ChangeAmount:    winningsPerWinner,
				TransactionType: entities.TransactionTypeLottoWin,
				TransactionMetadata: map[string]interface{}{
					"draw_id":        draw.ID,
					"winning_number": winningNumber,
					"pot_amount":     lockedDraw.TotalPot,
					"winner_count":   len(winnerTicketsMap),
				},
			}
			if err := utils.RecordBalanceChange(ctx, s.balanceHistoryRepo, s.eventPublisher, history); err != nil {
				return nil, fmt.Errorf("failed to record winner balance change: %w", err)
			}

			// Create LotteryWinner record (use first winning ticket for reference)
			lotteryWinner := &entities.LotteryWinner{
				DrawID:           draw.ID,
				DiscordID:        winnerID,
				TicketID:         winnerTickets[0].ID,
				WinningAmount:    winningsPerWinner,
				BalanceHistoryID: history.ID,
			}
			if err := s.lotteryWinnerRepo.Create(ctx, lotteryWinner); err != nil {
				return nil, fmt.Errorf("failed to create lottery winner record: %w", err)
			}

			user.Balance = newBalance
			winners = append(winners, user)
		}

		result.Winners = winners
		lockedDraw.Complete(winningNumber)
	} else {
		// No winner - rollover
		result.RolledOver = true
		lockedDraw.Complete(winningNumber)
	}

	// Always create next draw
	guildSettings, err := s.guildSettingsRepo.GetOrCreateGuildSettings(ctx, draw.GuildID)
	if err != nil {
		log.WithError(err).WithField("guildID", draw.GuildID).Error("failed to get guild settings for next draw")
	} else {
		nextDrawTime := s.CalculateNextDrawTime()
		nextDraw, err := s.lotteryDrawRepo.GetOrCreateCurrentDraw(
			ctx,
			draw.GuildID,
			nextDrawTime,
			guildSettings.GetLottoDifficulty(),
			guildSettings.GetLottoTicketCost(),
		)
		if err != nil {
			log.WithError(err).WithFields(log.Fields{
				"guildID": draw.GuildID,
			}).Error("failed to create next draw")
		} else if nextDraw != nil {
			// Transfer pot to next draw only on rollover
			if result.RolledOver {
				if err := s.lotteryDrawRepo.IncrementPot(ctx, nextDraw.ID, lockedDraw.TotalPot); err != nil {
					log.WithError(err).WithFields(log.Fields{
						"guildID":    draw.GuildID,
						"nextDrawID": nextDraw.ID,
						"potAmount":  lockedDraw.TotalPot,
					}).Error("failed to increment pot for rollover, pot is lost")
				} else {
					nextDraw.TotalPot += lockedDraw.TotalPot
				}
			}
			result.NextDraw = nextDraw
		}
	}

	// Update draw record
	if err := s.lotteryDrawRepo.Update(ctx, lockedDraw); err != nil {
		return nil, fmt.Errorf("failed to update draw: %w", err)
	}

	return result, nil
}

// SetDrawMessage updates draw with Discord message/channel IDs
func (s *lotteryService) SetDrawMessage(ctx context.Context, drawID, channelID, messageID int64) error {
	draw, err := s.lotteryDrawRepo.GetByID(ctx, drawID)
	if err != nil {
		return fmt.Errorf("failed to get draw: %w", err)
	}
	if draw == nil {
		return errors.New("draw not found")
	}

	draw.SetMessage(channelID, messageID)
	if err := s.lotteryDrawRepo.Update(ctx, draw); err != nil {
		return fmt.Errorf("failed to update draw message: %w", err)
	}

	return nil
}

// calculateAvailableBalance calculates user's available balance considering pending wagers
func (s *lotteryService) calculateAvailableBalance(ctx context.Context, user *entities.User) (int64, error) {
	// Get user's active wagers
	activeWagers, err := s.wagerRepo.GetActiveByUser(ctx, user.DiscordID)
	if err != nil {
		return 0, fmt.Errorf("failed to get active wagers: %w", err)
	}

	var lockedAmount int64
	for _, wager := range activeWagers {
		if wager.State == entities.WagerStateProposed || wager.State == entities.WagerStateVoting {
			if wager.ProposerDiscordID == user.DiscordID || wager.TargetDiscordID == user.DiscordID {
				lockedAmount += wager.Amount
			}
		}
	}

	// Get active group wager participations
	participations, err := s.groupWagerRepo.GetActiveParticipationsByUser(ctx, user.DiscordID)
	if err != nil {
		return 0, fmt.Errorf("failed to get group wager participations: %w", err)
	}

	for _, participation := range participations {
		lockedAmount += participation.Amount
	}

	return user.Balance - lockedAmount, nil
}

// generateUniqueNumbers selects N unique ticket numbers not in usedSet.
// Uses a hybrid approach to handle both small ranges (where users may own
// a large fraction of numbers) and large ranges (where enumeration is impractical).
//
// This design enables the lottery to start with easy odds (small number pools
// where users can purchase many tickets) and scale to harder odds (larger pools)
// without errors at either extreme:
//   - Small pools: Users can buy up to 100% of available numbers reliably
//   - Large pools: Memory-efficient random generation with negligible collision
func (s *lotteryService) generateUniqueNumbers(totalNumbers int64, usedSet map[int64]bool, count int) ([]int64, error) {
	available := int(totalNumbers) - len(usedSet)
	if count > available {
		return nil, fmt.Errorf("not enough available numbers: need %d, have %d", count, available)
	}

	usedRatio := float64(len(usedSet)) / float64(totalNumbers)

	// Use dense enumeration for small ranges or high usage ratios
	if usedRatio > 0.5 || totalNumbers <= denseThreshold {
		return s.generateFromAvailablePool(totalNumbers, usedSet, count)
	}

	// Use sparse retry for large ranges with low usage
	return s.generateWithRetry(totalNumbers, usedSet, count)
}

// generateFromAvailablePool enumerates available numbers and samples randomly.
// Used when the number pool is small or heavily utilized.
func (s *lotteryService) generateFromAvailablePool(totalNumbers int64, usedSet map[int64]bool, count int) ([]int64, error) {
	// Build list of available numbers
	available := make([]int64, 0, int(totalNumbers)-len(usedSet))
	for i := int64(0); i < totalNumbers; i++ {
		if !usedSet[i] {
			available = append(available, i)
		}
	}

	// Fisher-Yates shuffle (partial - only shuffle first `count` elements)
	for i := 0; i < count; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(available)-i)))
		if err != nil {
			return nil, fmt.Errorf("random generation failed: %w", err)
		}
		j := i + int(n.Int64())
		available[i], available[j] = available[j], available[i]
	}

	return available[:count], nil
}

// generateWithRetry uses random generation with collision checking.
// Efficient for large ranges where collision probability is low.
func (s *lotteryService) generateWithRetry(totalNumbers int64, usedSet map[int64]bool, count int) ([]int64, error) {
	result := make([]int64, 0, count)
	localUsed := make(map[int64]bool, len(usedSet)+count)
	for k, v := range usedSet {
		localUsed[k] = v
	}

	for len(result) < count {
		n, err := rand.Int(rand.Reader, big.NewInt(totalNumbers))
		if err != nil {
			return nil, fmt.Errorf("random generation failed: %w", err)
		}
		num := n.Int64()
		if !localUsed[num] {
			result = append(result, num)
			localUsed[num] = true
		}
	}

	return result, nil
}
