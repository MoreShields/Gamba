package service

import (
	"context"
	"fmt"

	"gambler/discord-client/models"
)

// guildSettingsService implements the GuildSettingsService interface
type guildSettingsService struct {
	guildSettingsRepo GuildSettingsRepository
}

// NewGuildSettingsService creates a new guild settings service
func NewGuildSettingsService(guildSettingsRepo GuildSettingsRepository) GuildSettingsService {
	return &guildSettingsService{
		guildSettingsRepo: guildSettingsRepo,
	}
}

// GetOrCreateSettings retrieves guild settings or creates default ones if not found
func (s *guildSettingsService) GetOrCreateSettings(ctx context.Context, guildID int64) (*models.GuildSettings, error) {

	// Get or create settings
	settings, err := s.guildSettingsRepo.GetOrCreateGuildSettings(ctx, guildID)
	if err != nil {
		return nil, fmt.Errorf("failed to get or create guild settings: %w", err)
	}

	return settings, nil
}

// UpdatePrimaryChannel updates the primary channel for a guild
func (s *guildSettingsService) UpdatePrimaryChannel(ctx context.Context, guildID int64, channelID int64) error {

	// Get existing settings
	settings, err := s.guildSettingsRepo.GetOrCreateGuildSettings(ctx, guildID)
	if err != nil {
		return fmt.Errorf("failed to get guild settings: %w", err)
	}

	// Update primary channel
	settings.PrimaryChannelID = &channelID

	// Save updated settings
	if err := s.guildSettingsRepo.UpdateGuildSettings(ctx, settings); err != nil {
		return fmt.Errorf("failed to update guild settings: %w", err)
	}

	return nil
}

// UpdateHighRollerRole updates the high roller role for a guild
func (s *guildSettingsService) UpdateHighRollerRole(ctx context.Context, guildID int64, roleID *int64) error {

	// Get existing settings
	settings, err := s.guildSettingsRepo.GetOrCreateGuildSettings(ctx, guildID)
	if err != nil {
		return fmt.Errorf("failed to get guild settings: %w", err)
	}

	// Update high roller role (can be nil to disable)
	settings.HighRollerRoleID = roleID

	// Save updated settings
	if err := s.guildSettingsRepo.UpdateGuildSettings(ctx, settings); err != nil {
		return fmt.Errorf("failed to update guild settings: %w", err)
	}

	return nil
}