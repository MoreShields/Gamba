package services

import (
	"context"
	"fmt"
	"strings"

	"gambler/discord-client/domain/entities"
	"gambler/discord-client/domain/interfaces"
)

type summonerWatchService struct {
	summonerWatchRepo interfaces.SummonerWatchRepository
}

// NewSummonerWatchService creates a new summoner watch service
func NewSummonerWatchService(summonerWatchRepo interfaces.SummonerWatchRepository) interfaces.SummonerWatchService {
	return &summonerWatchService{
		summonerWatchRepo: summonerWatchRepo,
	}
}

// AddWatch creates a new summoner watch for a guild
func (s *summonerWatchService) AddWatch(ctx context.Context, guildID int64, summonerName, tagLine string) (*entities.SummonerWatchDetail, error) {
	// Validate inputs
	if err := s.validateSummonerName(summonerName); err != nil {
		return nil, err
	}

	if err := s.validateTagLine(tagLine); err != nil {
		return nil, err
	}

	// Normalize inputs to lowercase for case-insensitive storage
	normalizedSummonerName := strings.ToLower(strings.TrimSpace(summonerName))
	normalizedTagLine := strings.ToLower(strings.TrimSpace(tagLine))

	// Create the watch - repository handles upsert of summoner and watch relationship
	watch, err := s.summonerWatchRepo.CreateWatch(ctx, guildID, normalizedSummonerName, normalizedTagLine)
	if err != nil {
		return nil, fmt.Errorf("failed to create summoner watch: %w", err)
	}

	return watch, nil
}

// RemoveWatch removes a summoner watch for a guild
func (s *summonerWatchService) RemoveWatch(ctx context.Context, guildID int64, summonerName, tagLine string) error {

	// Validate inputs
	if err := s.validateSummonerName(summonerName); err != nil {
		return err
	}

	if err := s.validateTagLine(tagLine); err != nil {
		return err
	}

	// Normalize inputs to lowercase for case-insensitive comparison
	normalizedSummonerName := strings.ToLower(strings.TrimSpace(summonerName))
	normalizedTagLine := strings.ToLower(strings.TrimSpace(tagLine))

	// Check if watch exists before attempting to delete
	_, err := s.summonerWatchRepo.GetWatch(ctx, guildID, normalizedSummonerName, normalizedTagLine)
	if err != nil {
		return fmt.Errorf("summoner watch not found")
	}

	// Delete the watch
	err = s.summonerWatchRepo.DeleteWatch(ctx, guildID, normalizedSummonerName, normalizedTagLine)
	if err != nil {
		return fmt.Errorf("failed to remove summoner watch: %w", err)
	}

	return nil
}

// ListWatches returns all summoner watches for a specific guild
func (s *summonerWatchService) ListWatches(ctx context.Context, guildID int64) ([]*entities.SummonerWatchDetail, error) {
	watches, err := s.summonerWatchRepo.GetWatchesByGuild(ctx, guildID)
	if err != nil {
		return nil, fmt.Errorf("failed to get guild watches: %w", err)
	}

	return watches, nil
}

// validateSummonerName validates the format of a summoner name
func (s *summonerWatchService) validateSummonerName(summonerName string) error {
	if summonerName == "" {
		return fmt.Errorf("summoner name cannot be empty")
	}

	// Trim whitespace
	trimmed := strings.TrimSpace(summonerName)
	if trimmed == "" {
		return fmt.Errorf("summoner name cannot be empty")
	}

	// Check length - LoL summoner names are typically 3-16 characters
	if len(trimmed) < 3 || len(trimmed) > 16 {
		return fmt.Errorf("summoner name must be between 3 and 16 characters")
	}

	// Check for valid characters - alphanumeric, spaces, and common special characters
	for _, char := range trimmed {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == ' ' || char == '_') {
			return fmt.Errorf("summoner name contains invalid characters. Only letters, numbers, spaces, and underscores are allowed")
		}
	}

	return nil
}

// validateTagLine validates the Riot ID tag line format
func (s *summonerWatchService) validateTagLine(tagLine string) error {
	if tagLine == "" {
		return fmt.Errorf("tag line cannot be empty")
	}

	// Trim whitespace
	trimmed := strings.TrimSpace(tagLine)
	if trimmed == "" {
		return fmt.Errorf("tag line cannot be empty")
	}

	// Check length - Riot tag lines are typically 3-5 characters
	if len(trimmed) < 2 || len(trimmed) > 5 {
		return fmt.Errorf("tag line must be between 2 and 5 characters")
	}

	// Check for valid characters - alphanumeric only
	for _, char := range trimmed {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9')) {
			return fmt.Errorf("tag line contains invalid characters. Only letters and numbers are allowed")
		}
	}

	return nil
}
