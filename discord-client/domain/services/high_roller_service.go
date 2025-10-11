package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gambler/discord-client/domain/entities"
	"gambler/discord-client/domain/interfaces"
	"gambler/discord-client/domain/utils"
)

// highRollerService implements business logic for high roller role purchases
type highRollerService struct {
	highRollerRepo     interfaces.HighRollerPurchaseRepository
	userRepo           interfaces.UserRepository
	wagerRepo          interfaces.WagerRepository
	groupWagerRepo     interfaces.GroupWagerRepository
	balanceHistoryRepo interfaces.BalanceHistoryRepository
	guildSettingsRepo  interfaces.GuildSettingsRepository
	eventPublisher     interfaces.EventPublisher
}

// NewHighRollerService creates a new high roller service
func NewHighRollerService(
	highRollerRepo interfaces.HighRollerPurchaseRepository,
	userRepo interfaces.UserRepository,
	wagerRepo interfaces.WagerRepository,
	groupWagerRepo interfaces.GroupWagerRepository,
	balanceHistoryRepo interfaces.BalanceHistoryRepository,
	guildSettingsRepo interfaces.GuildSettingsRepository,
	eventPublisher interfaces.EventPublisher,
) interfaces.HighRollerService {
	return &highRollerService{
		highRollerRepo:     highRollerRepo,
		userRepo:           userRepo,
		wagerRepo:          wagerRepo,
		groupWagerRepo:     groupWagerRepo,
		balanceHistoryRepo: balanceHistoryRepo,
		guildSettingsRepo:  guildSettingsRepo,
		eventPublisher:     eventPublisher,
	}
}

// GetCurrentHighRoller returns information about the current high roller
func (s *highRollerService) GetCurrentHighRoller(ctx context.Context, guildID int64) (*interfaces.HighRollerInfo, error) {
	// Get the latest purchase
	latestPurchase, err := s.highRollerRepo.GetLatestPurchase(ctx, guildID)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest purchase: %w", err)
	}

	info := &interfaces.HighRollerInfo{
		CurrentPrice: 0,
	}

	// If there's a purchase, get the user and price info
	if latestPurchase != nil {
		user, err := s.userRepo.GetByDiscordID(ctx, latestPurchase.DiscordID)
		if err != nil {
			return nil, fmt.Errorf("failed to get high roller user: %w", err)
		}
		info.CurrentHolder = user
		info.CurrentPrice = latestPurchase.PurchasePrice
		info.LastPurchasedAt = &latestPurchase.PurchasedAt

		// Get guild settings to check for tracking start time
		guildSettings, _ := s.guildSettingsRepo.GetOrCreateGuildSettings(ctx, guildID)

		// Calculate duration if tracking start time is set
		if guildSettings != nil && guildSettings.HasHighRollerTrackingStartTime() {
			trackingStartTime := *guildSettings.HighRollerTrackingStartTime

			// Calculate the total duration for the current holder
			duration, err := s.highRollerRepo.GetUserTotalDurationSince(ctx, guildID, latestPurchase.DiscordID, trackingStartTime)
			if err == nil {
				info.CurrentHolderDuration = duration
			}
		}
	}

	return info, nil
}

// PurchaseHighRollerRole processes a high roller role purchase
func (s *highRollerService) PurchaseHighRollerRole(ctx context.Context, discordID, guildID, offerAmount int64) error {
	// Validate offer amount
	if offerAmount <= 0 {
		return errors.New("offer amount must be positive")
	}

	// Get current high roller info
	currentInfo, err := s.GetCurrentHighRoller(ctx, guildID)
	if err != nil {
		return fmt.Errorf("failed to get current high roller: %w", err)
	}

	// Check if user is already the high roller
	if currentInfo.CurrentHolder != nil && currentInfo.CurrentHolder.DiscordID == discordID {
		return errors.New("you already hold the high roller role")
	}

	// Validate offer is higher than current price
	if offerAmount <= currentInfo.CurrentPrice {
		return fmt.Errorf("offer must be greater than current price of %d bits", currentInfo.CurrentPrice)
	}

	// Get user
	user, err := s.userRepo.GetByDiscordID(ctx, discordID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Calculate available balance (current balance minus pending wagers)
	availableBalance, err := s.calculateAvailableBalance(ctx, user)
	if err != nil {
		return fmt.Errorf("failed to calculate available balance: %w", err)
	}

	// Check if user has sufficient balance
	if availableBalance < offerAmount {
		return fmt.Errorf("insufficient balance: available %d bits, need %d bits", availableBalance, offerAmount)
	}

	// Initialize tracking start time if not set (first purchase in this guild)
	guildSettings, err := s.guildSettingsRepo.GetOrCreateGuildSettings(ctx, guildID)
	if err == nil && guildSettings != nil && !guildSettings.HasHighRollerTrackingStartTime() {
		now := time.Now()
		guildSettings.SetHighRollerTrackingStartTime(&now)
		_ = s.guildSettingsRepo.UpdateGuildSettings(ctx, guildSettings)
	}

	// Update user balance
	user.Balance -= offerAmount
	if err := s.userRepo.UpdateBalance(ctx, user.DiscordID, user.Balance); err != nil {
		return fmt.Errorf("failed to update user balance: %w", err)
	}

	// Record balance change
	history := &entities.BalanceHistory{
		DiscordID:       discordID,
		GuildID:         guildID,
		BalanceBefore:   user.Balance + offerAmount,
		BalanceAfter:    user.Balance,
		ChangeAmount:    -offerAmount,
		TransactionType: entities.TransactionTypeHighRollerPurchase,
		TransactionMetadata: map[string]interface{}{
			"purchase_price": offerAmount,
		},
	}
	if err := utils.RecordBalanceChange(ctx, s.balanceHistoryRepo, s.eventPublisher, history); err != nil {
		return fmt.Errorf("failed to record balance change: %w", err)
	}

	// Create purchase record
	purchase := &entities.HighRollerPurchase{
		GuildID:       guildID,
		DiscordID:     discordID,
		PurchasePrice: offerAmount,
		PurchasedAt:   time.Now(),
	}
	if err := s.highRollerRepo.CreatePurchase(ctx, purchase); err != nil {
		return fmt.Errorf("failed to create purchase record: %w", err)
	}

	return nil
}

// calculateAvailableBalance calculates user's available balance considering pending wagers
func (s *highRollerService) calculateAvailableBalance(ctx context.Context, user *entities.User) (int64, error) {
	// Get user's active wagers
	activeWagers, err := s.wagerRepo.GetActiveByUser(ctx, user.DiscordID)
	if err != nil {
		return 0, fmt.Errorf("failed to get active wagers: %w", err)
	}

	// Calculate total amount locked in wagers
	var lockedAmount int64
	for _, wager := range activeWagers {
		// Only count wagers that are not yet resolved
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

	// Add group wager amounts
	for _, participation := range participations {
		lockedAmount += participation.Amount
	}

	return user.Balance - lockedAmount, nil
}
