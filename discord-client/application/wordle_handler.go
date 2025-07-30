package application

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"gambler/discord-client/config"
	"gambler/discord-client/events"
	"gambler/discord-client/models"
	"gambler/discord-client/service"
	"github.com/jackc/pgx/v5"

	log "github.com/sirupsen/logrus"
)

// WordleHandler processes Discord messages from the Wordle bot
type WordleHandler interface {
	HandleDiscordMessage(ctx context.Context, event interface{}) error
}

// wordleHandler implements the WordleHandler interface
type wordleHandler struct {
	uowFactory UnitOfWorkFactory
}

// NewWordleHandler creates a new WordleHandler
func NewWordleHandler(uowFactory UnitOfWorkFactory) WordleHandler {
	return &wordleHandler{
		uowFactory: uowFactory,
	}
}

// HandleDiscordMessage processes Discord messages and awards bits for Wordle completions
func (h *wordleHandler) HandleDiscordMessage(ctx context.Context, event interface{}) error {
	m, err := AssertEventType[events.DiscordMessageEvent](event, "DiscordMessageEvent")
	if err != nil {
		return err
	}

	// Check if message is from the Wordle bot
	cfg := config.Get()
	if m.UserID != cfg.WordleBotID {
		return nil
	}

	log.WithFields(log.Fields{
		"message_id": m.MessageID,
		"channel_id": m.ChannelID,
		"guild_id":   m.GuildID,
		"content":    m.Content,
	}).Debug("Processing Wordle bot message")

	// Parse guild ID
	guildID, err := strconv.ParseInt(m.GuildID, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse guild ID: %w", err)
	}

	// Parse Wordle results from the message
	results, err := parseWordleResults(m.Content)
	if err != nil {
		log.WithError(err).WithField("message_id", m.MessageID).Error("Failed to parse Wordle results")
		return nil // Don't return error to avoid retries on parsing issues
	}

	if len(results) == 0 {
		log.WithField("message_id", m.MessageID).Debug("No Wordle results found in message")
		return nil
	}

	log.WithFields(log.Fields{
		"result_count": len(results),
		"guild_id":     guildID,
	}).Info("Processing Wordle results")

	// Process each result
	for _, result := range results {
		if err := h.processWordleResult(ctx, result, guildID); err != nil {
			log.WithError(err).WithFields(log.Fields{
				"user_id":     result.UserID,
				"guess_count": result.GuessCount,
			}).Error("Failed to process Wordle result")
			// Continue processing other results
		}
	}

	return nil
}

// processWordleResult handles a single Wordle completion
func (h *wordleHandler) processWordleResult(ctx context.Context, result WordleResult, guildID int64) error {
	// Parse user ID
	userID, err := strconv.ParseInt(result.UserID, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse user ID: %w", err)
	}

	// Create unit of work for this guild
	uow := h.uowFactory.CreateForGuild(guildID)
	if err := uow.Begin(ctx); err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer uow.Rollback()

	// Check for duplicate completion today
	existingCompletion, err := uow.WordleCompletionRepo().GetByUserToday(ctx, userID, guildID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("failed to check existing completion: %w", err)
	}
	if existingCompletion != nil {
		log.WithFields(log.Fields{
			"user_id":  userID,
			"guild_id": guildID,
		}).Debug("User already has a Wordle completion for today")
		return nil
	}

	// Create WordleScore
	score, err := models.NewWordleScore(result.GuessCount, result.MaxGuesses)
	if err != nil {
		return fmt.Errorf("failed to create WordleScore: %w", err)
	}

	// Create WordleCompletion
	completion, err := models.NewWordleCompletion(userID, guildID, score, time.Now())
	if err != nil {
		return fmt.Errorf("failed to create WordleCompletion: %w", err)
	}

	// Save completion
	if err := uow.WordleCompletionRepo().Create(ctx, completion); err != nil {
		return fmt.Errorf("failed to save WordleCompletion: %w", err)
	}

	// Calculate reward including streak bonus
	rewardService := service.NewWordleRewardService(uow.WordleCompletionRepo(), 0) // baseReward not used with new payout structure
	reward, err := rewardService.CalculateReward(ctx, userID, guildID, score)
	if err != nil {
		return fmt.Errorf("failed to calculate reward: %w", err)
	}

	// Get or create user
	user, err := uow.UserRepository().GetByDiscordID(ctx, userID)
	if err != nil {
		// User doesn't exist, create with initial balance as the reward
		user, err = uow.UserRepository().Create(ctx, userID, fmt.Sprintf("User%d", userID), reward)
		if err != nil {
			return fmt.Errorf("failed to create user: %w", err)
		}
	} else {
		// Update user balance
		newBalance := user.Balance + reward
		if err := uow.UserRepository().UpdateBalance(ctx, userID, newBalance); err != nil {
			return fmt.Errorf("failed to update user balance: %w", err)
		}
		user.Balance = newBalance
	}

	// Record balance history
	balanceHistory := &models.BalanceHistory{
		DiscordID:       userID,
		GuildID:         guildID,
		BalanceBefore:   user.Balance - reward,
		BalanceAfter:    user.Balance,
		ChangeAmount:    reward,
		TransactionType: models.TransactionTypeWordleReward,
		TransactionMetadata: map[string]any{
			"guess_count":  result.GuessCount,
			"max_guesses":  result.MaxGuesses,
			"final_reward": reward,
		},
		CreatedAt: time.Now(),
	}

	if err := uow.BalanceHistoryRepository().Record(ctx, balanceHistory); err != nil {
		return fmt.Errorf("failed to record balance history: %w", err)
	}

	// Commit transaction
	if err := uow.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.WithFields(log.Fields{
		"user_id":     userID,
		"guild_id":    guildID,
		"guess_count": result.GuessCount,
		"reward":      reward,
		"new_balance": user.Balance,
	}).Info("Successfully processed Wordle completion")

	return nil
}