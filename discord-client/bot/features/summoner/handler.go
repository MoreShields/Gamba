package summoner

import (
	"context"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"

	"gambler/discord-client/bot/common"
	"gambler/discord-client/service"
	summoner_pb "gambler/api/gen/go/services"
)

// handleWatchCommand handles the /summoner watch command
func (f *Feature) handleWatchCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()
	
	// Parse command options
	options := i.ApplicationCommandData().Options[0].Options // watch subcommand options
	var summonerName, region string
	
	for _, option := range options {
		switch option.Name {
		case "summoner_name":
			summonerName = option.StringValue()
		case "region":
			region = option.StringValue()
		}
	}
	
	// Default region to NA1 if not provided
	if region == "" {
		region = "NA1"
	}
	
	// Get guild ID from interaction
	guildID, err := strconv.ParseInt(i.GuildID, 10, 64)
	if err != nil {
		log.Errorf("Failed to parse guild ID %s: %v", i.GuildID, err)
		common.RespondWithError(s, i, "Invalid guild ID")
		return
	}
	
	log.Infof("Processing summoner watch request: %s in %s for guild %d", summonerName, region, guildID)
	
	// Step 1: Validate summoner with external service
	validateReq := &summoner_pb.StartTrackingSummonerRequest{
		SummonerName: summonerName,
		Region:       region,
		RequestedAt:  timestamppb.New(time.Now()),
	}
	
	validateResp, err := f.summonerClient.StartTrackingSummoner(ctx, validateReq)
	if err != nil {
		log.Errorf("Failed to validate summoner %s in %s: %v", summonerName, region, err)
		common.RespondWithError(s, i, "Failed to connect to summoner validation service. Please try again later.")
		return
	}
	
	// Check validation result
	if !validateResp.Success {
		errorMsg := f.mapValidationError(validateResp.ErrorCode, validateResp.ErrorMessage)
		log.Infof("Summoner validation failed for %s in %s: %s", summonerName, region, errorMsg)
		embed := createErrorEmbed(summonerName, region, errorMsg)
		common.RespondWithEmbed(s, i, embed, nil, false)
		return
	}
	
	log.Infof("Summoner %s in %s validated successfully", summonerName, region)
	
	// Step 2: Store in local database
	uow := f.uowFactory.CreateForGuild(guildID)
	if err := uow.Begin(ctx); err != nil {
		log.Errorf("Failed to begin transaction: %v", err)
		common.RespondWithError(s, i, "Database error occurred. Please try again.")
		return
	}
	defer uow.Rollback()
	
	// Create summoner watch service
	summonerWatchService := service.NewSummonerWatchService(uow.SummonerWatchRepository())
	
	// Add the watch
	watchDetail, err := summonerWatchService.AddWatch(ctx, guildID, summonerName, region)
	if err != nil {
		log.Errorf("Failed to add summoner watch for %s in %s: %v", summonerName, region, err)
		
		// Check if it's a duplicate watch error
		if isDuplicateWatchError(err) {
			embed := createAlreadyWatchingEmbed(summonerName, region)
			common.RespondWithEmbed(s, i, embed, nil, false)
		} else {
			common.RespondWithError(s, i, "Failed to save summoner watch. Please try again.")
		}
		return
	}
	
	// Commit transaction
	if err := uow.Commit(); err != nil {
		log.Errorf("Failed to commit transaction: %v", err)
		common.RespondWithError(s, i, "Failed to save summoner watch. Please try again.")
		return
	}
	
	log.Infof("Successfully added summoner watch for %s in %s for guild %d", summonerName, region, guildID)
	
	// Step 3: Send success response
	embed := createSuccessEmbed(watchDetail, validateResp.SummonerDetails)
	common.RespondWithEmbed(s, i, embed, nil, false)
}

// mapValidationError converts protobuf validation errors to user-friendly messages
func (f *Feature) mapValidationError(errorCode *summoner_pb.ValidationError, errorMessage *string) string {
	if errorCode == nil {
		if errorMessage != nil {
			return *errorMessage
		}
		return "Unknown validation error occurred"
	}
	
	switch *errorCode {
	case summoner_pb.ValidationError_VALIDATION_ERROR_SUMMONER_NOT_FOUND:
		return "Summoner not found. Please check the spelling and try again."
	case summoner_pb.ValidationError_VALIDATION_ERROR_INVALID_REGION:
		return "Invalid region. Please use a valid region like NA1, EUW1, KR, etc."
	case summoner_pb.ValidationError_VALIDATION_ERROR_API_ERROR:
		return "Riot API is currently unavailable. Please try again later."
	case summoner_pb.ValidationError_VALIDATION_ERROR_RATE_LIMITED:
		return "Too many requests. Please wait a moment and try again."
	case summoner_pb.ValidationError_VALIDATION_ERROR_ALREADY_TRACKED:
		return "This summoner is already being tracked by the service."
	case summoner_pb.ValidationError_VALIDATION_ERROR_INTERNAL_ERROR:
		return "Internal service error. Please try again later."
	default:
		if errorMessage != nil {
			return *errorMessage
		}
		return "Unknown validation error occurred"
	}
}

// isDuplicateWatchError checks if the error is due to a duplicate watch attempt
func isDuplicateWatchError(err error) bool {
	// This could be enhanced to check for specific database constraint errors
	// For now, we'll use a simple string check
	errStr := err.Error()
	return contains(errStr, "duplicate") || contains(errStr, "already exists") || contains(errStr, "unique")
}

// contains checks if a string contains a substring (case-insensitive helper)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || 
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || 
		containsHelper(s, substr))))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}