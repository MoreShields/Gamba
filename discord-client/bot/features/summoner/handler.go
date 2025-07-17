package summoner

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"

	"gambler/discord-client/bot/common"
	summoner_pb "gambler/discord-client/proto/services"
	"gambler/discord-client/service"
)

// handleWatchCommand handles the /summoner watch command
func (f *Feature) handleWatchCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()

	// Parse command options
	options := i.ApplicationCommandData().Options[0].Options // watch subcommand options
	var gameName, tagLine string

	for _, option := range options {
		switch option.Name {
		case "game_name":
			gameName = strings.TrimSpace(option.StringValue())
		case "tag":
			tagLine = strings.TrimSpace(option.StringValue())
		}
	}

	// Validate required values
	if gameName == "" || tagLine == "" {
		log.Infof("Missing required parameters: gameName=%s, tagLine=%s", gameName, tagLine)
		common.RespondWithError(s, i, "Both game name and tag are required")
		return
	}

	// Get guild ID from interaction
	guildID, err := strconv.ParseInt(i.GuildID, 10, 64)
	if err != nil {
		log.Errorf("Failed to parse guild ID %s: %v", i.GuildID, err)
		common.RespondWithError(s, i, "Invalid guild ID")
		return
	}

	log.Infof("Processing summoner watch request: %s#%s for guild %d", gameName, tagLine, guildID)

	// Step 1: Validate summoner with external service
	// Note: We now pass separate game_name and tag_line fields
	validateReq := &summoner_pb.StartTrackingSummonerRequest{
		GameName:    gameName,
		TagLine:     tagLine,
		RequestedAt: timestamppb.New(time.Now()),
	}

	validateResp, err := f.summonerClient.StartTrackingSummoner(ctx, validateReq)
	if err != nil {
		log.Errorf("Failed to validate summoner %s#%s: %v", gameName, tagLine, err)
		common.RespondWithError(s, i, "Failed to connect to summoner validation service. Please try again later.")
		return
	}

	// Check validation result
	if !validateResp.Success {
		errorMsg := f.mapValidationError(validateResp.ErrorCode, validateResp.ErrorMessage)
		// Check if it's a duplicate watch error
		log.Infof("Summoner validation failed for %s#%s: %s", gameName, tagLine, errorMsg)
		if *validateResp.ErrorCode == summoner_pb.ValidationError_VALIDATION_ERROR_ALREADY_TRACKED {
			embed := createAlreadyWatchingEmbed(gameName, tagLine)
			common.RespondWithEmbed(s, i, embed, nil, true)
		} else {
			common.RespondWithError(s, i, "Failed to save summoner watch. Please try again.")
			embed := createErrorEmbed(gameName, tagLine, errorMsg)
			common.RespondWithEmbed(s, i, embed, nil, false)
		}
	}

	log.Infof("Summoner %s#%s validated successfully", gameName, tagLine)

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

	// Add the watch - use the validated game name and tag line
	watchDetail, err := summonerWatchService.AddWatch(ctx, guildID, validateResp.SummonerDetails.GameName, tagLine)
	if err != nil {
		log.Errorf("Failed to add summoner watch for %s#%s: %v", gameName, tagLine, err)
		common.RespondWithError(s, i, "Failed to save summoner watch. Please try again.")
		return
	}

	// Commit transaction
	if err := uow.Commit(); err != nil {
		log.Errorf("Failed to commit transaction: %v", err)
		common.RespondWithError(s, i, "Failed to save summoner watch. Please try again.")
		return
	}

	log.Infof("Successfully added summoner watch for %s#%s for guild %d", gameName, tagLine, guildID)

	// Step 3: Send success response
	embed := createSuccessEmbed(watchDetail, validateResp.SummonerDetails)
	common.RespondWithEmbed(s, i, embed, nil, false)
}

// handleUnwatchCommand handles the /summoner unwatch command
func (f *Feature) handleUnwatchCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()

	// Parse command options
	options := i.ApplicationCommandData().Options[0].Options // unwatch subcommand options
	var gameName, tagLine string

	for _, option := range options {
		switch option.Name {
		case "game_name":
			gameName = strings.TrimSpace(option.StringValue())
		case "tag":
			tagLine = strings.TrimSpace(option.StringValue())
		}
	}

	// Validate required values
	if gameName == "" || tagLine == "" {
		log.Infof("Missing required parameters: gameName=%s, tagLine=%s", gameName, tagLine)
		common.RespondWithError(s, i, "Both game name and tag are required")
		return
	}

	// Get guild ID from interaction
	guildID, err := strconv.ParseInt(i.GuildID, 10, 64)
	if err != nil {
		log.Errorf("Failed to parse guild ID %s: %v", i.GuildID, err)
		common.RespondWithError(s, i, "Invalid guild ID")
		return
	}

	log.Infof("Processing summoner unwatch request: %s#%s for guild %d", gameName, tagLine, guildID)

	// Remove from local database
	uow := f.uowFactory.CreateForGuild(guildID)
	if err := uow.Begin(ctx); err != nil {
		log.Errorf("Failed to begin transaction: %v", err)
		common.RespondWithError(s, i, "Database error occurred. Please try again.")
		return
	}
	defer uow.Rollback()

	// Create summoner watch service
	summonerWatchService := service.NewSummonerWatchService(uow.SummonerWatchRepository())

	// Remove the watch using the parsed name and tag line
	err = summonerWatchService.RemoveWatch(ctx, guildID, gameName, tagLine)
	if err != nil {
		log.Errorf("Failed to remove summoner watch for %s#%s: %v", gameName, tagLine, err)

		// Check if it's a "not found" error
		if strings.Contains(err.Error(), "not found") {
			embed := createNotWatchingEmbed(gameName, tagLine)
			common.RespondWithEmbed(s, i, embed, nil, false)
		} else {
			common.RespondWithError(s, i, "Failed to remove summoner watch. Please try again.")
		}
		return
	}

	// Commit transaction
	if err := uow.Commit(); err != nil {
		log.Errorf("Failed to commit transaction: %v", err)
		common.RespondWithError(s, i, "Failed to remove summoner watch. Please try again.")
		return
	}

	log.Infof("Successfully removed summoner watch for %s#%s for guild %d", gameName, tagLine, guildID)

	// Step 3: Send success response
	embed := createUnwatchSuccessEmbed(gameName, tagLine)
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
	case summoner_pb.ValidationError_VALIDATION_ERROR_NOT_TRACKED:
		return "This summoner is not being tracked."
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
