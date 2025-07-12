package service

import (
	"context"
	"fmt"

	"gambler/models"
)

// guildSettingsService implements the GuildSettingsService interface
type guildSettingsService struct {
	uowFactory UnitOfWorkFactory
}

// NewGuildSettingsService creates a new guild settings service
func NewGuildSettingsService(uowFactory UnitOfWorkFactory) GuildSettingsService {
	return &guildSettingsService{
		uowFactory: uowFactory,
	}
}

// GetOrCreateSettings retrieves guild settings or creates default ones if not found
func (s *guildSettingsService) GetOrCreateSettings(ctx context.Context, guildID int64) (*models.GuildSettings, error) {
	// Create unit of work
	uow := s.uowFactory.Create()
	if err := uow.Begin(ctx); err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer uow.Rollback() // No-op if already committed

	// Get or create settings
	settings, err := uow.GuildSettingsRepository().GetOrCreateGuildSettings(ctx, guildID)
	if err != nil {
		return nil, fmt.Errorf("failed to get or create guild settings: %w", err)
	}

	// Commit the transaction (in case new settings were created)
	if err := uow.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return settings, nil
}

// UpdatePrimaryChannel updates the primary channel for a guild
func (s *guildSettingsService) UpdatePrimaryChannel(ctx context.Context, guildID int64, channelID int64) error {
	// Create unit of work
	uow := s.uowFactory.Create()
	if err := uow.Begin(ctx); err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer uow.Rollback() // No-op if already committed

	// Get existing settings
	settings, err := uow.GuildSettingsRepository().GetOrCreateGuildSettings(ctx, guildID)
	if err != nil {
		return fmt.Errorf("failed to get guild settings: %w", err)
	}

	// Update primary channel
	settings.PrimaryChannelID = &channelID

	// Save updated settings
	if err := uow.GuildSettingsRepository().UpdateGuildSettings(ctx, settings); err != nil {
		return fmt.Errorf("failed to update guild settings: %w", err)
	}

	// Commit the transaction
	if err := uow.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// UpdateHighRollerRole updates the high roller role for a guild
func (s *guildSettingsService) UpdateHighRollerRole(ctx context.Context, guildID int64, roleID *int64) error {
	// Create unit of work
	uow := s.uowFactory.Create()
	if err := uow.Begin(ctx); err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer uow.Rollback() // No-op if already committed

	// Get existing settings
	settings, err := uow.GuildSettingsRepository().GetOrCreateGuildSettings(ctx, guildID)
	if err != nil {
		return fmt.Errorf("failed to get guild settings: %w", err)
	}

	// Update high roller role (can be nil to disable)
	settings.HighRollerRoleID = roleID

	// Save updated settings
	if err := uow.GuildSettingsRepository().UpdateGuildSettings(ctx, settings); err != nil {
		return fmt.Errorf("failed to update guild settings: %w", err)
	}

	// Commit the transaction
	if err := uow.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}