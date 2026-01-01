package debug

import (
	"context"
	"fmt"
	"strconv"

	"gambler/discord-client/domain/entities"
)

// addAdminCommands adds admin commands to the shell
func (s *Shell) addAdminCommands() {
	adminCommands := map[string]Command{
		"adjust-balance": {
			Handler:     s.handleAdjustBalance,
			Description: "Adjust balance by amount (+/-)",
			Usage:       "adjust-balance [guild_id] <user_id> <+/-amount>",
			Category:    "admin",
		},
		"admin-transfer": {
			Handler:     s.handleAdminTransfer,
			Description: "Transfer bits between users (admin)",
			Usage:       "admin-transfer [guild_id] <from_user_id> <to_user_id> <amount>",
			Category:    "admin",
		},
		"reset-all-2026": {
			Handler:     s.handleResetAll2026,
			Description: "Reset all guild balances to 1 bit for 2026",
			Usage:       "reset-all-2026 [guild_id]",
			Category:    "admin",
		},
	}

	// Merge admin commands into main commands map
	for name, cmd := range adminCommands {
		s.commands[name] = cmd
	}
}

// handleUpdateBalance sets a user's balance to a specific amount
func (s *Shell) handleUpdateBalance(shell *Shell, args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("usage: update-balance <guild_id> <user_id> <amount>")
	}

	guildID, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid guild ID: %w", err)
	}

	userID, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	newBalance, err := strconv.ParseInt(args[2], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid amount: %w", err)
	}

	if newBalance < 0 {
		return fmt.Errorf("balance cannot be negative")
	}

	ctx := context.Background()
	uow := s.uowFactory.CreateForGuild(guildID)
	if err := uow.Begin(ctx); err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer uow.Rollback()

	// Get current user info
	user, err := uow.UserRepository().GetByDiscordID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Show current vs new balance
	fmt.Printf("\n💰 Balance Update for User %d:\n", userID)
	fmt.Printf("   Current: %s bits\n", formatNumber(user.Balance))
	fmt.Printf("   New:     %s bits\n", formatNumber(newBalance))
	fmt.Printf("   Change:  %s bits\n", formatSignedNumber(newBalance-user.Balance))

	// Confirm action
	if !s.confirmAction("Update user balance?") {
		return nil
	}

	// Update balance
	if err := uow.UserRepository().UpdateBalance(ctx, userID, newBalance); err != nil {
		return fmt.Errorf("failed to update balance: %w", err)
	}

	// Record balance history
	history := &entities.BalanceHistory{
		DiscordID:       userID,
		GuildID:         guildID,
		BalanceBefore:   user.Balance,
		BalanceAfter:    newBalance,
		ChangeAmount:    newBalance - user.Balance,
		TransactionType: entities.TransactionTypeTransferIn,
		TransactionMetadata: map[string]any{
			"admin":  "true",
			"source": "debug_shell",
			"reason": "manual_update",
		},
	}

	if err := uow.BalanceHistoryRepository().Record(ctx, history); err != nil {
		return fmt.Errorf("failed to record balance history: %w", err)
	}

	// Commit transaction
	if err := uow.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Log admin action
	s.logAdminAction("update_balance", map[string]interface{}{
		"guild_id":       guildID,
		"user_id":        userID,
		"old_balance":    user.Balance,
		"new_balance":    newBalance,
		"change_amount":  newBalance - user.Balance,
	})

	s.printSuccess(fmt.Sprintf("Balance updated successfully for user %d", userID))
	return nil
}

// handleAdjustBalance adjusts a user's balance by a specific amount
func (s *Shell) handleAdjustBalance(shell *Shell, args []string) error {
	var guildID, userID int64
	var adjustment int64
	var err error

	// Check if we have a default guild
	if s.currentGuild != 0 && len(args) == 2 {
		// Use default guild: adjust-balance <user_id> <amount>
		guildID = s.currentGuild
		userID, err = strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid user ID: %w", err)
		}
		adjustment, err = strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid amount: %w", err)
		}
	} else if len(args) >= 3 {
		// Full syntax: adjust-balance <guild_id> <user_id> <amount>
		guildID, err = strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid guild ID: %w", err)
		}
		userID, err = strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid user ID: %w", err)
		}
		adjustment, err = strconv.ParseInt(args[2], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid amount: %w", err)
		}
	} else {
		if s.currentGuild == 0 {
			return fmt.Errorf("usage: adjust-balance <guild_id> <user_id> <+/-amount>\nOr set a guild with 'guild <id>' and use: adjust-balance <user_id> <+/-amount>")
		}
		return fmt.Errorf("usage: adjust-balance <user_id> <+/-amount>")
	}

	ctx := context.Background()
	uow := s.uowFactory.CreateForGuild(guildID)
	if err := uow.Begin(ctx); err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer uow.Rollback()

	// Get current user info
	user, err := uow.UserRepository().GetByDiscordID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	newBalance := user.Balance + adjustment
	if newBalance < 0 {
		return fmt.Errorf("adjustment would result in negative balance")
	}

	// Show adjustment details
	fmt.Printf("\n💰 Balance Adjustment:\n")
	fmt.Printf("   Guild:      %d\n", guildID)
	fmt.Printf("   User:       %d\n", userID)
	fmt.Printf("   Current:    %s bits\n", formatNumber(user.Balance))
	fmt.Printf("   Adjustment: %s bits\n", formatSignedNumber(adjustment))
	fmt.Printf("   New:        %s bits\n", formatNumber(newBalance))

	// Confirm action
	if !s.confirmAction("Adjust user balance?") {
		return nil
	}

	// Update balance
	if err := uow.UserRepository().UpdateBalance(ctx, userID, newBalance); err != nil {
		return fmt.Errorf("failed to update balance: %w", err)
	}

	// Determine transaction type
	transactionType := entities.TransactionTypeTransferIn
	if adjustment < 0 {
		transactionType = entities.TransactionTypeTransferOut
	}

	// Record balance history
	history := &entities.BalanceHistory{
		DiscordID:       userID,
		GuildID:         guildID,
		BalanceBefore:   user.Balance,
		BalanceAfter:    newBalance,
		ChangeAmount:    adjustment,
		TransactionType: transactionType,
		TransactionMetadata: map[string]any{
			"admin":  "true",
			"source": "debug_shell",
			"reason": "manual_adjustment",
		},
	}

	if err := uow.BalanceHistoryRepository().Record(ctx, history); err != nil {
		return fmt.Errorf("failed to record balance history: %w", err)
	}

	// Commit transaction
	if err := uow.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Log admin action
	s.logAdminAction("adjust_balance", map[string]interface{}{
		"guild_id":       guildID,
		"user_id":        userID,
		"old_balance":    user.Balance,
		"new_balance":    newBalance,
		"adjustment":     adjustment,
	})

	s.printSuccess(fmt.Sprintf("Balance adjusted successfully for user %d", userID))
	return nil
}

// handleResetUser resets a user to the starting balance
func (s *Shell) handleResetUser(shell *Shell, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: reset-user <guild_id> <user_id>")
	}

	guildID, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid guild ID: %w", err)
	}

	userID, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	const startingBalance int64 = 100000 // 100k starting balance

	ctx := context.Background()
	uow := s.uowFactory.CreateForGuild(guildID)
	if err := uow.Begin(ctx); err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer uow.Rollback()

	// Get current user info
	user, err := uow.UserRepository().GetByDiscordID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Show reset details
	fmt.Printf("\n🔄 Reset User %d:\n", userID)
	fmt.Printf("   Current: %s bits\n", formatNumber(user.Balance))
	fmt.Printf("   Reset:   %s bits (starting balance)\n", formatNumber(startingBalance))

	// Confirm action
	if !s.confirmAction("Reset user to starting balance?") {
		return nil
	}

	// Update balance
	if err := uow.UserRepository().UpdateBalance(ctx, userID, startingBalance); err != nil {
		return fmt.Errorf("failed to update balance: %w", err)
	}

	// Record balance history
	history := &entities.BalanceHistory{
		DiscordID:       userID,
		GuildID:         guildID,
		BalanceBefore:   user.Balance,
		BalanceAfter:    startingBalance,
		ChangeAmount:    startingBalance - user.Balance,
		TransactionType: entities.TransactionTypeTransferIn,
		TransactionMetadata: map[string]any{
			"admin":  "true",
			"source": "debug_shell",
			"reason": "user_reset",
		},
	}

	if err := uow.BalanceHistoryRepository().Record(ctx, history); err != nil {
		return fmt.Errorf("failed to record balance history: %w", err)
	}

	// Commit transaction
	if err := uow.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Log admin action
	s.logAdminAction("reset_user", map[string]interface{}{
		"guild_id":       guildID,
		"user_id":        userID,
		"old_balance":    user.Balance,
		"new_balance":    startingBalance,
	})

	s.printSuccess(fmt.Sprintf("User %d reset to starting balance", userID))
	return nil
}

// handleAdminTransfer performs an admin transfer between users
func (s *Shell) handleAdminTransfer(shell *Shell, args []string) error {
	var guildID, fromUserID, toUserID, amount int64
	var err error

	// Check if we have a default guild
	if s.currentGuild != 0 && len(args) == 3 {
		// Use default guild: admin-transfer <from_user_id> <to_user_id> <amount>
		guildID = s.currentGuild
		fromUserID, err = strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid from user ID: %w", err)
		}
		toUserID, err = strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid to user ID: %w", err)
		}
		amount, err = strconv.ParseInt(args[2], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid amount: %w", err)
		}
	} else if len(args) >= 4 {
		// Full syntax: admin-transfer <guild_id> <from_user_id> <to_user_id> <amount>
		guildID, err = strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid guild ID: %w", err)
		}
		fromUserID, err = strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid from user ID: %w", err)
		}
		toUserID, err = strconv.ParseInt(args[2], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid to user ID: %w", err)
		}
		amount, err = strconv.ParseInt(args[3], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid amount: %w", err)
		}
	} else {
		if s.currentGuild == 0 {
			return fmt.Errorf("usage: admin-transfer <guild_id> <from_user_id> <to_user_id> <amount>\nOr set a guild with 'guild <id>' and use: admin-transfer <from_user_id> <to_user_id> <amount>")
		}
		return fmt.Errorf("usage: admin-transfer <from_user_id> <to_user_id> <amount>")
	}

	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}

	if fromUserID == toUserID {
		return fmt.Errorf("cannot transfer to same user")
	}

	ctx := context.Background()
	uow := s.uowFactory.CreateForGuild(guildID)
	if err := uow.Begin(ctx); err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer uow.Rollback()

	// Get both users
	fromUser, err := uow.UserRepository().GetByDiscordID(ctx, fromUserID)
	if err != nil {
		return fmt.Errorf("failed to get from user: %w", err)
	}

	toUser, err := uow.UserRepository().GetByDiscordID(ctx, toUserID)
	if err != nil {
		return fmt.Errorf("failed to get to user: %w", err)
	}

	// Show transfer details
	fmt.Printf("\n💸 Admin Transfer:\n")
	fmt.Printf("   Guild:  %d\n", guildID)
	fmt.Printf("   From:   User %d (Balance: %s bits)\n", fromUserID, formatNumber(fromUser.Balance))
	fmt.Printf("   To:     User %d (Balance: %s bits)\n", toUserID, formatNumber(toUser.Balance))
	fmt.Printf("   Amount: %s bits\n", formatNumber(amount))

	if fromUser.Balance < amount {
		s.printWarning("Source user has insufficient balance")
	}

	// Confirm action
	if !s.confirmAction("Perform admin transfer?") {
		return nil
	}

	// Check if we need to force the transfer
	forceTransfer := false
	if fromUser.Balance < amount {
		if !s.confirmAction("Force transfer despite insufficient balance?") {
			return nil
		}
		forceTransfer = true
	}

	// Update balances manually
	newFromBalance := fromUser.Balance - amount
	newToBalance := toUser.Balance + amount

	if err := uow.UserRepository().UpdateBalance(ctx, fromUserID, newFromBalance); err != nil {
		return fmt.Errorf("failed to update from user balance: %w", err)
	}

	if err := uow.UserRepository().UpdateBalance(ctx, toUserID, newToBalance); err != nil {
		return fmt.Errorf("failed to update to user balance: %w", err)
	}

	// Record balance history for both users
	fromHistory := &entities.BalanceHistory{
		DiscordID:       fromUserID,
		GuildID:         guildID,
		BalanceBefore:   fromUser.Balance,
		BalanceAfter:    newFromBalance,
		ChangeAmount:    -amount,
		TransactionType: entities.TransactionTypeTransferOut,
		TransactionMetadata: map[string]any{
			"admin":        "true",
			"source":       "debug_shell",
			"forced":       fmt.Sprintf("%v", forceTransfer),
			"recipient_id": toUserID,
		},
	}

	toHistory := &entities.BalanceHistory{
		DiscordID:       toUserID,
		GuildID:         guildID,
		BalanceBefore:   toUser.Balance,
		BalanceAfter:    newToBalance,
		ChangeAmount:    amount,
		TransactionType: entities.TransactionTypeTransferIn,
		TransactionMetadata: map[string]any{
			"admin":     "true",
			"source":    "debug_shell",
			"forced":    fmt.Sprintf("%v", forceTransfer),
			"sender_id": fromUserID,
		},
	}

	if err := uow.BalanceHistoryRepository().Record(ctx, fromHistory); err != nil {
		return fmt.Errorf("failed to record from user history: %w", err)
	}

	if err := uow.BalanceHistoryRepository().Record(ctx, toHistory); err != nil {
		return fmt.Errorf("failed to record to user history: %w", err)
	}

	// Commit transaction
	if err := uow.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Log admin action
	s.logAdminAction("admin_transfer", map[string]interface{}{
		"guild_id":      guildID,
		"from_user_id":  fromUserID,
		"to_user_id":    toUserID,
		"amount":        amount,
		"forced":        forceTransfer,
	})

	s.printSuccess(fmt.Sprintf("Transfer of %s bits completed successfully", formatNumber(amount)))
	return nil
}

// handleResetAll2026 resets all user balances in a guild to 1 bit for the new year
func (s *Shell) handleResetAll2026(shell *Shell, args []string) error {
	var guildID int64
	var err error

	// Check if we have a default guild or explicit guild_id
	if len(args) >= 1 {
		guildID, err = strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid guild ID: %w", err)
		}
	} else if s.currentGuild != 0 {
		guildID = s.currentGuild
	} else {
		return fmt.Errorf("usage: reset-all-2026 <guild_id>\nOr set a guild with 'guild <id>' first")
	}

	const newBalance int64 = 1

	ctx := context.Background()
	uow := s.uowFactory.CreateForGuild(guildID)
	if err := uow.Begin(ctx); err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer uow.Rollback()

	// Get all users in the guild
	users, err := uow.UserRepository().GetAll(ctx)
	if err != nil {
		return fmt.Errorf("failed to get users: %w", err)
	}

	if len(users) == 0 {
		s.printWarning("No users found in this guild")
		return nil
	}

	// Calculate total bits being reset
	var totalBits int64
	usersToReset := 0
	for _, user := range users {
		if user.Balance != newBalance {
			totalBits += user.Balance
			usersToReset++
		}
	}

	// Show summary
	fmt.Printf("\n🎆 2026 New Year Balance Reset:\n")
	fmt.Printf("   Guild:         %d\n", guildID)
	fmt.Printf("   Total users:   %d\n", len(users))
	fmt.Printf("   Users to reset: %d\n", usersToReset)
	fmt.Printf("   Total bits:    %s → %d bits\n", formatNumber(totalBits), usersToReset)

	if usersToReset == 0 {
		s.printWarning("All users already have balance of 1 bit")
		return nil
	}

	// Confirm action
	if !s.confirmAction("Reset ALL user balances to 1 bit?") {
		return nil
	}

	// Reset each user
	resetCount := 0
	for _, user := range users {
		if user.Balance == newBalance {
			continue // Skip users already at 1 bit
		}

		changeAmount := newBalance - user.Balance

		// Update balance
		if err := uow.UserRepository().UpdateBalance(ctx, user.DiscordID, newBalance); err != nil {
			return fmt.Errorf("failed to update balance for user %d: %w", user.DiscordID, err)
		}

		// Record balance history
		history := &entities.BalanceHistory{
			DiscordID:       user.DiscordID,
			GuildID:         guildID,
			BalanceBefore:   user.Balance,
			BalanceAfter:    newBalance,
			ChangeAmount:    changeAmount,
			TransactionType: entities.TransactionTypeTransferOut,
			TransactionMetadata: map[string]any{
				"admin":  "true",
				"source": "debug_shell",
				"reason": "2026_new_year_reset",
			},
		}

		if err := uow.BalanceHistoryRepository().Record(ctx, history); err != nil {
			return fmt.Errorf("failed to record balance history for user %d: %w", user.DiscordID, err)
		}

		resetCount++
	}

	// Commit transaction
	if err := uow.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Log admin action
	s.logAdminAction("reset_all_2026", map[string]interface{}{
		"guild_id":    guildID,
		"users_reset": resetCount,
		"total_bits":  totalBits,
	})

	s.printSuccess(fmt.Sprintf("Reset %d users to 1 bit for 2026", resetCount))
	return nil
}