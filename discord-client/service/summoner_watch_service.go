package service

import (
	"context"
	"fmt"
	"strings"

	"gambler/discord-client/models"
)

type summonerWatchService struct {
	summonerWatchRepo SummonerWatchRepository
}

// NewSummonerWatchService creates a new summoner watch service
func NewSummonerWatchService(summonerWatchRepo SummonerWatchRepository) SummonerWatchService {
	return &summonerWatchService{
		summonerWatchRepo: summonerWatchRepo,
	}
}

// AddWatch creates a new summoner watch for a guild
func (s *summonerWatchService) AddWatch(ctx context.Context, guildID int64, summonerName, region string) (*models.SummonerWatchDetail, error) {
	// Validate inputs
	if err := s.validateSummonerName(summonerName); err != nil {
		return nil, err
	}

	// Normalize region to uppercase for validation and storage
	normalizedRegion := strings.ToUpper(region)
	if err := s.validateRegion(normalizedRegion); err != nil {
		return nil, err
	}

	// Create the watch - repository handles upsert of summoner and watch relationship
	watch, err := s.summonerWatchRepo.CreateWatch(ctx, guildID, summonerName, normalizedRegion)
	if err != nil {
		return nil, fmt.Errorf("failed to create summoner watch: %w", err)
	}

	return watch, nil
}

// RemoveWatch removes a summoner watch for a guild
func (s *summonerWatchService) RemoveWatch(ctx context.Context, guildID int64, summonerName, region string) error {
	// Validate inputs
	if err := s.validateSummonerName(summonerName); err != nil {
		return err
	}

	// Normalize region to uppercase for validation and storage
	normalizedRegion := strings.ToUpper(region)
	if err := s.validateRegion(normalizedRegion); err != nil {
		return err
	}

	// Check if watch exists before attempting to delete
	_, err := s.summonerWatchRepo.GetWatch(ctx, guildID, summonerName, normalizedRegion)
	if err != nil {
		return fmt.Errorf("summoner watch not found")
	}

	// Delete the watch
	err = s.summonerWatchRepo.DeleteWatch(ctx, guildID, summonerName, normalizedRegion)
	if err != nil {
		return fmt.Errorf("failed to remove summoner watch: %w", err)
	}

	return nil
}

// ListWatches returns all summoner watches for a specific guild
func (s *summonerWatchService) ListWatches(ctx context.Context, guildID int64) ([]*models.SummonerWatchDetail, error) {
	watches, err := s.summonerWatchRepo.GetWatchesByGuild(ctx, guildID)
	if err != nil {
		return nil, fmt.Errorf("failed to get guild watches: %w", err)
	}

	return watches, nil
}

// GetWatchDetails retrieves a specific summoner watch for a guild
func (s *summonerWatchService) GetWatchDetails(ctx context.Context, guildID int64, summonerName, region string) (*models.SummonerWatchDetail, error) {
	// Validate inputs
	if err := s.validateSummonerName(summonerName); err != nil {
		return nil, err
	}

	// Normalize region to uppercase for validation and storage
	normalizedRegion := strings.ToUpper(region)
	if err := s.validateRegion(normalizedRegion); err != nil {
		return nil, err
	}

	watch, err := s.summonerWatchRepo.GetWatch(ctx, guildID, summonerName, normalizedRegion)
	if err != nil {
		return nil, fmt.Errorf("summoner watch not found")
	}

	return watch, nil
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

// validateRegion validates the LoL region code
func (s *summonerWatchService) validateRegion(region string) error {
	if region == "" {
		return fmt.Errorf("region cannot be empty")
	}

	// Valid LoL regions
	validRegions := map[string]bool{
		"NA1":  true, // North America
		"EUW1": true, // Europe West
		"EUN1": true, // Europe Nordic & East
		"KR":   true, // Korea
		"BR1":  true, // Brazil
		"LA1":  true, // Latin America North
		"LA2":  true, // Latin America South
		"OC1":  true, // Oceania
		"RU":   true, // Russia
		"TR1":  true, // Turkey
		"JP1":  true, // Japan
		"PH2":  true, // Philippines
		"SG2":  true, // Singapore
		"TH2":  true, // Thailand
		"TW2":  true, // Taiwan
		"VN2":  true, // Vietnam
	}

	upperRegion := strings.ToUpper(region)
	if !validRegions[upperRegion] {
		return fmt.Errorf("invalid region '%s'. Valid regions are: NA1, EUW1, EUN1, KR, BR1, LA1, LA2, OC1, RU, TR1, JP1, PH2, SG2, TH2, TW2, VN2", region)
	}

	return nil
}
