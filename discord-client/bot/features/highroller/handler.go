package highroller

import (
	"context"
	"fmt"
	"time"

	"gambler/discord-client/bot/common"
	"gambler/discord-client/domain/interfaces"
	"gambler/discord-client/domain/services"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

const (
	// DefaultHistoryLimit is the default number of purchase history items to show
	DefaultHistoryLimit = 10
	// MaxMemberFetchLimit is the maximum number of members to fetch from Discord
	MaxMemberFetchLimit = 1000
)

// handleBuy processes the /highroller buy command
func (f *Feature) handleBuy(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	// Get the offer amount from options
	options := i.ApplicationCommandData().Options[0].Options
	if len(options) == 0 {
		common.RespondWithError(s, i, "Amount parameter is required")
		return nil
	}

	offerAmount := options[0].IntValue()
	if offerAmount <= 0 {
		common.RespondWithError(s, i, "Offer amount must be positive")
		return nil
	}

	// Get guild and user IDs
	guildID, err := common.ParseGuildID(i.GuildID)
	if err != nil {
		log.Errorf("Failed to parse guild ID: %v", err)
		common.RespondWithError(s, i, "Failed to process command")
		return err
	}

	userID, err := common.ParseUserID(i.Member.User.ID)
	if err != nil {
		log.Errorf("Failed to parse user ID: %v", err)
		common.RespondWithError(s, i, "Failed to process command")
		return err
	}

	ctx := context.Background()

	// Process the purchase and update Discord role
	roleID, err := f.processPurchase(ctx, userID, guildID, offerAmount)
	if err != nil {
		// Handle specific error cases with user-friendly messages
		common.RespondWithError(s, i, err.Error())
		return nil
	}

	// Success! Send ephemeral success message
	embed := &discordgo.MessageEmbed{
		Title:       "",
		Description: fmt.Sprintf("**New <@&%d>!**\n\n<@%d> purchased the role for **%s bits**", roleID, userID, common.FormatBalance(offerAmount)),
		Color:       common.ColorSuccess,
	}

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})

	if err != nil {
		log.Errorf("Failed to respond to interaction: %v", err)
		return err
	}

	return nil
}

// handleInfo shows current high roller information and recent history
func (f *Feature) handleInfo(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	// Get guild ID
	guildID, err := common.ParseGuildID(i.GuildID)
	if err != nil {
		log.Errorf("Failed to parse guild ID: %v", err)
		common.RespondWithError(s, i, "Failed to process command")
		return err
	}

	ctx := context.Background()

	// Get current high roller info
	info, err := f.getCurrentHighRoller(ctx, guildID)
	if err != nil {
		log.Errorf("Failed to get high roller info: %v", err)
		common.RespondWithError(s, i, "Failed to get high roller information")
		return nil
	}

	// Get purchase history (last 5)
	history, err := f.getPurchaseHistory(ctx, guildID, 5)
	if err != nil {
		log.Errorf("Failed to get purchase history: %v", err)
		// Continue without history rather than failing completely
		history = []PurchaseHistoryItem{}
	}

	// Get the role name from Discord
	roleName := "High Roller" // Default fallback
	uow := f.uowFactory.CreateForGuild(guildID)
	if err := uow.Begin(ctx); err == nil {
		defer uow.Rollback()

		guildSettingsService := services.NewGuildSettingsService(uow.GuildSettingsRepository())
		if roleID, err := guildSettingsService.GetHighRollerRoleID(ctx, guildID); err == nil && roleID != nil {
			// Fetch the role from Discord
			if role, err := s.State.Role(i.GuildID, common.FormatDiscordID(*roleID)); err == nil {
				roleName = role.Name
			}
		}
		uow.Commit()
	}

	// Build embed
	embed := &discordgo.MessageEmbed{
		Title: fmt.Sprintf("👑 %s 👑", roleName),
		Color: common.ColorInfo,
	}

	if info.CurrentHolder != nil {
		// Use description for the current holder to make it more prominent
		embed.Description = fmt.Sprintf("# <@%d> - %s\n", info.CurrentHolder.DiscordID, common.FormatBalance(info.CurrentPrice))

		if info.LastPurchasedAt != nil {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   "Held Since",
				Value:  fmt.Sprintf("<t:%d:R>", info.LastPurchasedAt.Unix()),
				Inline: true,
			})
		}
	} else {
		embed.Description = "No one currently holds the high roller role."
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Starting Price",
			Value:  fmt.Sprintf("%s bits", common.FormatBalance(1)),
			Inline: true,
		})
	}

	// Add purchase history section with visual separation
	if len(history) > 0 {
		// Add a separator field
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "", // Zero-width space
			Value:  "───────────────────────",
			Inline: false,
		})

		var historyText string
		for _, purchase := range history {
			// Skip the current holder
			if info.CurrentHolder != nil && purchase.DiscordID == info.CurrentHolder.DiscordID {
				continue
			}
			historyText += fmt.Sprintf("<@%d> - **%s bits** - <t:%d:R>\n",
				purchase.DiscordID,
				common.FormatBalance(purchase.PurchasePrice),
				purchase.PurchasedAt.Unix(),
			)
		}
		// Only add history field if there's content
		if historyText != "" {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   "Recent Purchase History",
				Value:  historyText,
				Inline: false,
			})
		}
	}

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})

	if err != nil {
		log.Errorf("Failed to respond to interaction: %v", err)
		return err
	}

	return nil
}

// processPurchase handles the high roller purchase logic
func (f *Feature) processPurchase(ctx context.Context, discordID, guildID, offerAmount int64) (int64, error) {
	// Create guild-scoped unit of work
	uow := f.uowFactory.CreateForGuild(guildID)
	if err := uow.Begin(ctx); err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer uow.Rollback()

	// Create services
	highRollerService := services.NewHighRollerService(
		uow.HighRollerPurchaseRepository(),
		uow.UserRepository(),
		uow.WagerRepository(),
		uow.GroupWagerRepository(),
		uow.BalanceHistoryRepository(),
		uow.GuildSettingsRepository(),
		uow.EventBus(),
	)

	guildSettingsService := services.NewGuildSettingsService(
		uow.GuildSettingsRepository(),
	)

	// Process purchase (validates and updates database)
	err := highRollerService.PurchaseHighRollerRole(ctx, discordID, guildID, offerAmount)
	if err != nil {
		return 0, err // Return service errors directly (validation, insufficient balance, etc.)
	}

	// Get high roller role ID for this guild
	roleID, err := guildSettingsService.GetHighRollerRoleID(ctx, guildID)
	if err != nil {
		return 0, fmt.Errorf("failed to get high roller role ID: %w", err)
	}

	if roleID == nil {
		return 0, fmt.Errorf("high roller role not configured for this guild")
	}

	// Update Discord role BEFORE committing transaction
	err = f.updateDiscordRole(ctx, guildID, discordID, *roleID)
	if err != nil {
		// Transaction will rollback automatically via defer
		return 0, fmt.Errorf("failed to update Discord role: %w", err)
	}

	// Only commit if Discord update succeeded
	if err := uow.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return *roleID, nil
}

// getCurrentHighRoller returns information about the current high roller
func (f *Feature) getCurrentHighRoller(ctx context.Context, guildID int64) (*interfaces.HighRollerInfo, error) {
	uow := f.uowFactory.CreateForGuild(guildID)
	if err := uow.Begin(ctx); err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer uow.Rollback()

	// Create service
	highRollerService := services.NewHighRollerService(
		uow.HighRollerPurchaseRepository(),
		uow.UserRepository(),
		uow.WagerRepository(),
		uow.GroupWagerRepository(),
		uow.BalanceHistoryRepository(),
		uow.GuildSettingsRepository(),
		uow.EventBus(),
	)

	// Get current high roller info
	info, err := highRollerService.GetCurrentHighRoller(ctx, guildID)
	if err != nil {
		return nil, err
	}

	// Read-only operation, commit to release lock
	if err := uow.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return info, nil
}

// getPurchaseHistory returns the purchase history for a guild
func (f *Feature) getPurchaseHistory(ctx context.Context, guildID int64, limit int) ([]PurchaseHistoryItem, error) {
	uow := f.uowFactory.CreateForGuild(guildID)
	if err := uow.Begin(ctx); err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer uow.Rollback()

	// Get purchase history
	purchases, err := uow.HighRollerPurchaseRepository().GetPurchaseHistory(ctx, guildID, limit)
	if err != nil {
		return nil, err
	}

	// Convert to display items
	var items []PurchaseHistoryItem
	for _, purchase := range purchases {
		user, err := uow.UserRepository().GetByDiscordID(ctx, purchase.DiscordID)
		if err != nil {
			// User might have been deleted, skip
			continue
		}

		items = append(items, PurchaseHistoryItem{
			Username:      user.Username,
			DiscordID:     user.DiscordID,
			PurchasePrice: purchase.PurchasePrice,
			PurchasedAt:   purchase.PurchasedAt,
		})
	}

	// Read-only operation, commit to release lock
	if err := uow.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return items, nil
}

// updateDiscordRole updates the high roller role for a guild
func (f *Feature) updateDiscordRole(ctx context.Context, guildID, newHolderDiscordID, roleID int64) error {
	guildIDStr := common.FormatDiscordID(guildID)
	roleIDStr := common.FormatDiscordID(roleID)
	newHolderDiscordIDStr := common.FormatDiscordID(newHolderDiscordID)

	// Get all guild members with the high roller role
	members, err := f.session.GuildMembers(guildIDStr, "", MaxMemberFetchLimit)
	if err != nil {
		return fmt.Errorf("failed to get guild members: %w", err)
	}

	// Find who currently has the role
	var currentHolders []string
	for _, member := range members {
		for _, memberRoleID := range member.Roles {
			if memberRoleID == roleIDStr {
				currentHolders = append(currentHolders, member.User.ID)
				break
			}
		}
	}

	// Remove role from anyone who shouldn't have it
	for _, holderID := range currentHolders {
		if holderID != newHolderDiscordIDStr {
			if err := f.session.GuildMemberRoleRemove(guildIDStr, holderID, roleIDStr); err != nil {
				log.Errorf("Failed to remove high roller role from user %s: %v", holderID, err)
				// Continue with other removals even if one fails
			} else {
				log.Infof("Removed high roller role from user %s", holderID)
			}
		}
	}

	// Add role to the new holder if they don't have it
	hasRole := false
	for _, holderID := range currentHolders {
		if holderID == newHolderDiscordIDStr {
			hasRole = true
			break
		}
	}

	if !hasRole {
		if err := f.session.GuildMemberRoleAdd(guildIDStr, newHolderDiscordIDStr, roleIDStr); err != nil {
			return fmt.Errorf("failed to add high roller role to user %s: %w", newHolderDiscordIDStr, err)
		}
		log.Infof("Added high roller role to user %s", newHolderDiscordIDStr)
	}

	return nil
}

// PurchaseHistoryItem represents a purchase history entry
type PurchaseHistoryItem struct {
	Username      string
	DiscordID     int64
	PurchasePrice int64
	PurchasedAt   time.Time
}
