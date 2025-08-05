package services

import (
	"context"
	"fmt"

	"gambler/discord-client/domain/entities"
	"gambler/discord-client/domain/interfaces"
)

// guildSettingsService implements the GuildSettingsService interface
type guildSettingsService struct {
	guildSettingsRepo interfaces.GuildSettingsRepository
}

// NewGuildSettingsService creates a new guild settings service
func NewGuildSettingsService(guildSettingsRepo interfaces.GuildSettingsRepository) interfaces.GuildSettingsService {
	return &guildSettingsService{
		guildSettingsRepo: guildSettingsRepo,
	}
}

// GetOrCreateSettings retrieves guild settings or creates default ones if not found
func (s *guildSettingsService) GetOrCreateSettings(ctx context.Context, guildID int64) (*entities.GuildSettings, error) {

	// Get or create settings
	settings, err := s.guildSettingsRepo.GetOrCreateGuildSettings(ctx, guildID)
	if err != nil {
		return nil, fmt.Errorf("failed to get or create guild settings: %w", err)
	}

	return settings, nil
}

// UpdatePrimaryChannel updates the primary channel for a guild
func (s *guildSettingsService) UpdatePrimaryChannel(ctx context.Context, guildID int64, channelID *int64) error {

	// Get existing settings
	settings, err := s.guildSettingsRepo.GetOrCreateGuildSettings(ctx, guildID)
	if err != nil {
		return fmt.Errorf("failed to get guild settings: %w", err)
	}

	// Update primary channel (can be nil to disable)
	settings.PrimaryChannelID = channelID

	// Save updated settings
	if err := s.guildSettingsRepo.UpdateGuildSettings(ctx, settings); err != nil {
		return fmt.Errorf("failed to update guild settings: %w", err)
	}

	return nil
}

// UpdateLolChannel updates the LOL channel for a guild
func (s *guildSettingsService) UpdateLolChannel(ctx context.Context, guildID int64, channelID *int64) error {

	// Get existing settings
	settings, err := s.guildSettingsRepo.GetOrCreateGuildSettings(ctx, guildID)
	if err != nil {
		return fmt.Errorf("failed to get guild settings: %w", err)
	}

	// Update LOL channel (can be nil to disable)
	settings.LolChannelID = channelID

	// Save updated settings
	if err := s.guildSettingsRepo.UpdateGuildSettings(ctx, settings); err != nil {
		return fmt.Errorf("failed to update guild settings: %w", err)
	}

	return nil
}

// UpdateTftChannel updates the TFT channel for a guild
func (s *guildSettingsService) UpdateTftChannel(ctx context.Context, guildID int64, channelID *int64) error {

	// Get existing settings
	settings, err := s.guildSettingsRepo.GetOrCreateGuildSettings(ctx, guildID)
	if err != nil {
		return fmt.Errorf("failed to get guild settings: %w", err)
	}

	// Update TFT channel (can be nil to disable)
	settings.TftChannelID = channelID

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

// UpdateWordleChannel updates the Wordle channel for a guild
func (s *guildSettingsService) UpdateWordleChannel(ctx context.Context, guildID int64, channelID *int64) error {

	// Get existing settings
	settings, err := s.guildSettingsRepo.GetOrCreateGuildSettings(ctx, guildID)
	if err != nil {
		return fmt.Errorf("failed to get guild settings: %w", err)
	}

	// Update Wordle channel (can be nil to disable)
	settings.WordleChannelID = channelID

	// Save updated settings
	if err := s.guildSettingsRepo.UpdateGuildSettings(ctx, settings); err != nil {
		return fmt.Errorf("failed to update guild settings: %w", err)
	}

	return nil
}

// GetHighRollerRoleID returns the high roller role ID for a guild
func (s *guildSettingsService) GetHighRollerRoleID(ctx context.Context, guildID int64) (*int64, error) {
	settings, err := s.guildSettingsRepo.GetOrCreateGuildSettings(ctx, guildID)
	if err != nil {
		return nil, fmt.Errorf("failed to get guild settings: %w", err)
	}

	return settings.HighRollerRoleID, nil
}
