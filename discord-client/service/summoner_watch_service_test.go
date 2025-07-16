package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"gambler/discord-client/models"

	"github.com/stretchr/testify/assert"
)

func TestSummonerWatchService_AddWatch_Success(t *testing.T) {
	ctx := context.Background()

	// Setup mocks
	mockRepo := new(MockSummonerWatchRepository)
	service := NewSummonerWatchService(mockRepo)

	expectedWatch := &models.SummonerWatchDetail{
		WatchID:      1,
		GuildID:      12345,
		WatchedAt:    time.Now(),
		SummonerID:   100,
		SummonerName: "TestSummoner",
		Region:       "NA1",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Mock expectations
	mockRepo.On("CreateWatch", ctx, int64(12345), "TestSummoner", "NA1").Return(expectedWatch, nil)

	// Execute
	result, err := service.AddWatch(ctx, 12345, "TestSummoner", "NA1")

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, expectedWatch, result)
	mockRepo.AssertExpectations(t)
}

func TestSummonerWatchService_AddWatch_InvalidSummonerName(t *testing.T) {
	testCases := []struct {
		name         string
		summonerName string
		expectedErr  string
	}{
		{
			name:         "empty summoner name",
			summonerName: "",
			expectedErr:  "summoner name cannot be empty",
		},
		{
			name:         "whitespace only summoner name",
			summonerName: "   ",
			expectedErr:  "summoner name cannot be empty",
		},
		{
			name:         "too short summoner name",
			summonerName: "ab",
			expectedErr:  "summoner name must be between 3 and 16 characters",
		},
		{
			name:         "too long summoner name",
			summonerName: "ThisNameIsTooLongForLol",
			expectedErr:  "summoner name must be between 3 and 16 characters",
		},
		{
			name:         "invalid characters",
			summonerName: "Test@Summoner",
			expectedErr:  "summoner name contains invalid characters",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			mockRepo := new(MockSummonerWatchRepository)
			service := NewSummonerWatchService(mockRepo)

			// Execute
			result, err := service.AddWatch(ctx, 12345, tc.summonerName, "NA1")

			// Assert
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tc.expectedErr)
			assert.Nil(t, result)
			mockRepo.AssertNotCalled(t, "CreateWatch")
		})
	}
}

func TestSummonerWatchService_AddWatch_InvalidRegion(t *testing.T) {
	testCases := []struct {
		name        string
		region      string
		expectedErr string
	}{
		{
			name:        "empty region",
			region:      "",
			expectedErr: "region cannot be empty",
		},
		{
			name:        "invalid region",
			region:      "INVALID",
			expectedErr: "invalid region 'INVALID'",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			mockRepo := new(MockSummonerWatchRepository)
			service := NewSummonerWatchService(mockRepo)

			// Execute
			result, err := service.AddWatch(ctx, 12345, "TestSummoner", tc.region)

			// Assert
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tc.expectedErr)
			assert.Nil(t, result)
			mockRepo.AssertNotCalled(t, "CreateWatch")
		})
	}
}

func TestSummonerWatchService_AddWatch_LowercaseRegion(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockSummonerWatchRepository)
	service := NewSummonerWatchService(mockRepo)

	expectedWatch := &models.SummonerWatchDetail{
		WatchID:      1,
		GuildID:      12345,
		WatchedAt:    time.Now(),
		SummonerID:   100,
		SummonerName: "TestSummoner",
		Region:       "NA1", // Should be normalized to uppercase
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Mock expectations - should call with uppercase region
	mockRepo.On("CreateWatch", ctx, int64(12345), "TestSummoner", "NA1").Return(expectedWatch, nil)

	// Execute with lowercase region
	result, err := service.AddWatch(ctx, 12345, "TestSummoner", "na1")

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, expectedWatch, result)
	mockRepo.AssertExpectations(t)
}

func TestSummonerWatchService_AddWatch_ValidRegions(t *testing.T) {
	validRegions := []string{"NA1", "EUW1", "EUN1", "KR", "BR1", "LA1", "LA2", "OC1", "RU", "TR1", "JP1", "PH2", "SG2", "TH2", "TW2", "VN2"}

	for _, region := range validRegions {
		t.Run("valid_region_"+region, func(t *testing.T) {
			ctx := context.Background()
			mockRepo := new(MockSummonerWatchRepository)
			service := NewSummonerWatchService(mockRepo)

			expectedWatch := &models.SummonerWatchDetail{
				WatchID:      1,
				GuildID:      12345,
				WatchedAt:    time.Now(),
				SummonerID:   100,
				SummonerName: "TestSummoner",
				Region:       region,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			}

			mockRepo.On("CreateWatch", ctx, int64(12345), "TestSummoner", region).Return(expectedWatch, nil)

			// Execute
			result, err := service.AddWatch(ctx, 12345, "TestSummoner", region)

			// Assert
			assert.NoError(t, err)
			assert.Equal(t, expectedWatch, result)
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestSummonerWatchService_AddWatch_RepositoryError(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockSummonerWatchRepository)
	service := NewSummonerWatchService(mockRepo)

	repoErr := errors.New("repository error")
	mockRepo.On("CreateWatch", ctx, int64(12345), "TestSummoner", "NA1").Return(nil, repoErr)

	// Execute
	result, err := service.AddWatch(ctx, 12345, "TestSummoner", "NA1")

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create summoner watch")
	assert.Nil(t, result)
	mockRepo.AssertExpectations(t)
}

func TestSummonerWatchService_RemoveWatch_Success(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockSummonerWatchRepository)
	service := NewSummonerWatchService(mockRepo)

	existingWatch := &models.SummonerWatchDetail{
		WatchID:      1,
		GuildID:      12345,
		SummonerName: "TestSummoner",
		Region:       "NA1",
	}

	// Mock expectations
	mockRepo.On("GetWatch", ctx, int64(12345), "TestSummoner", "NA1").Return(existingWatch, nil)
	mockRepo.On("DeleteWatch", ctx, int64(12345), "TestSummoner", "NA1").Return(nil)

	// Execute
	err := service.RemoveWatch(ctx, 12345, "TestSummoner", "NA1")

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestSummonerWatchService_RemoveWatch_NotFound(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockSummonerWatchRepository)
	service := NewSummonerWatchService(mockRepo)

	// Mock expectations - watch doesn't exist
	mockRepo.On("GetWatch", ctx, int64(12345), "TestSummoner", "NA1").Return(nil, errors.New("not found"))

	// Execute
	err := service.RemoveWatch(ctx, 12345, "TestSummoner", "NA1")

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "summoner watch not found")
	mockRepo.AssertNotCalled(t, "DeleteWatch")
}

func TestSummonerWatchService_RemoveWatch_ValidationError(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockSummonerWatchRepository)
	service := NewSummonerWatchService(mockRepo)

	// Execute with invalid summoner name
	err := service.RemoveWatch(ctx, 12345, "", "NA1")

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "summoner name cannot be empty")
	mockRepo.AssertNotCalled(t, "GetWatch")
	mockRepo.AssertNotCalled(t, "DeleteWatch")
}

func TestSummonerWatchService_RemoveWatch_DeleteError(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockSummonerWatchRepository)
	service := NewSummonerWatchService(mockRepo)

	existingWatch := &models.SummonerWatchDetail{
		WatchID:      1,
		GuildID:      12345,
		SummonerName: "TestSummoner",
		Region:       "NA1",
	}

	deleteErr := errors.New("delete failed")

	// Mock expectations
	mockRepo.On("GetWatch", ctx, int64(12345), "TestSummoner", "NA1").Return(existingWatch, nil)
	mockRepo.On("DeleteWatch", ctx, int64(12345), "TestSummoner", "NA1").Return(deleteErr)

	// Execute
	err := service.RemoveWatch(ctx, 12345, "TestSummoner", "NA1")

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to remove summoner watch")
	mockRepo.AssertExpectations(t)
}

func TestSummonerWatchService_ListWatches_Success(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockSummonerWatchRepository)
	service := NewSummonerWatchService(mockRepo)

	expectedWatches := []*models.SummonerWatchDetail{
		{
			WatchID:      1,
			GuildID:      12345,
			SummonerName: "Summoner1",
			Region:       "NA1",
		},
		{
			WatchID:      2,
			GuildID:      12345,
			SummonerName: "Summoner2",
			Region:       "EUW1",
		},
	}

	// Mock expectations
	mockRepo.On("GetWatchesByGuild", ctx, int64(12345)).Return(expectedWatches, nil)

	// Execute
	result, err := service.ListWatches(ctx, 12345)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, expectedWatches, result)
	mockRepo.AssertExpectations(t)
}

func TestSummonerWatchService_ListWatches_Empty(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockSummonerWatchRepository)
	service := NewSummonerWatchService(mockRepo)

	// Mock expectations - empty list
	mockRepo.On("GetWatchesByGuild", ctx, int64(12345)).Return([]*models.SummonerWatchDetail{}, nil)

	// Execute
	result, err := service.ListWatches(ctx, 12345)

	// Assert
	assert.NoError(t, err)
	assert.Empty(t, result)
	mockRepo.AssertExpectations(t)
}

func TestSummonerWatchService_ListWatches_RepositoryError(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockSummonerWatchRepository)
	service := NewSummonerWatchService(mockRepo)

	repoErr := errors.New("repository error")
	mockRepo.On("GetWatchesByGuild", ctx, int64(12345)).Return(nil, repoErr)

	// Execute
	result, err := service.ListWatches(ctx, 12345)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get guild watches")
	assert.Nil(t, result)
	mockRepo.AssertExpectations(t)
}

func TestSummonerWatchService_GetWatchDetails_Success(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockSummonerWatchRepository)
	service := NewSummonerWatchService(mockRepo)

	expectedWatch := &models.SummonerWatchDetail{
		WatchID:      1,
		GuildID:      12345,
		SummonerName: "TestSummoner",
		Region:       "NA1",
	}

	// Mock expectations
	mockRepo.On("GetWatch", ctx, int64(12345), "TestSummoner", "NA1").Return(expectedWatch, nil)

	// Execute
	result, err := service.GetWatchDetails(ctx, 12345, "TestSummoner", "NA1")

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, expectedWatch, result)
	mockRepo.AssertExpectations(t)
}

func TestSummonerWatchService_GetWatchDetails_NotFound(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockSummonerWatchRepository)
	service := NewSummonerWatchService(mockRepo)

	// Mock expectations - watch doesn't exist
	mockRepo.On("GetWatch", ctx, int64(12345), "TestSummoner", "NA1").Return(nil, errors.New("not found"))

	// Execute
	result, err := service.GetWatchDetails(ctx, 12345, "TestSummoner", "NA1")

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "summoner watch not found")
	assert.Nil(t, result)
	mockRepo.AssertExpectations(t)
}

func TestSummonerWatchService_GetWatchDetails_ValidationError(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockSummonerWatchRepository)
	service := NewSummonerWatchService(mockRepo)

	// Execute with invalid region
	result, err := service.GetWatchDetails(ctx, 12345, "TestSummoner", "INVALID")

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid region")
	assert.Nil(t, result)
	mockRepo.AssertNotCalled(t, "GetWatch")
}

func TestSummonerWatchService_ValidateSummonerName_ValidNames(t *testing.T) {
	service := &summonerWatchService{}

	validNames := []string{
		"TestSummoner",
		"Test123",
		"Test_User",
		"Sum With Space",
		"abc",
		"1234567890123456", // 16 characters
	}

	for _, name := range validNames {
		t.Run("valid_name_"+name, func(t *testing.T) {
			err := service.validateSummonerName(name)
			assert.NoError(t, err)
		})
	}
}

func TestSummonerWatchService_ValidateRegion_CaseInsensitive(t *testing.T) {
	service := &summonerWatchService{}

	// Should pass for lowercase since validation converts to uppercase internally
	err := service.validateRegion("na1")
	assert.NoError(t, err)

	// Should pass for uppercase
	err = service.validateRegion("NA1")
	assert.NoError(t, err)

	// Should fail for invalid region
	err = service.validateRegion("INVALID")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid region 'INVALID'")
}
