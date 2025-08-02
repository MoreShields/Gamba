package services

import (
	"gambler/discord-client/domain/testhelpers"
	"context"
	"errors"
	"testing"
	"time"

	"gambler/discord-client/domain/entities"

	"github.com/stretchr/testify/assert"
)

func TestSummonerWatchService_AddWatch_Success(t *testing.T) {
	ctx := context.Background()

	// Setup mocks
	mockRepo := new(testhelpers.MockSummonerWatchRepository)
	service := NewSummonerWatchService(mockRepo)

	expectedWatch := &entities.SummonerWatchDetail{
		GuildID:      12345,
		WatchedAt:    time.Now(),
		SummonerName: "testsummoner",
		TagLine:      "gamba",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Mock expectations
	mockRepo.On("CreateWatch", ctx, int64(12345), "testsummoner", "gamba").Return(expectedWatch, nil)

	// Execute
	result, err := service.AddWatch(ctx, 12345, "TestSummoner", "gamba")

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
			mockRepo := new(testhelpers.MockSummonerWatchRepository)
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

func TestSummonerWatchService_AddWatch_InvalidTagLine(t *testing.T) {
	testCases := []struct {
		name        string
		tagLine     string
		expectedErr string
	}{
		{
			name:        "empty tagLine",
			tagLine:     "",
			expectedErr: "tag line cannot be empty",
		},
		{
			name:        "invalid tagLine",
			tagLine:     "INVALID",
			expectedErr: "tag line must be between 2 and 5 characters",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			mockRepo := new(testhelpers.MockSummonerWatchRepository)
			service := NewSummonerWatchService(mockRepo)

			// Execute
			result, err := service.AddWatch(ctx, 12345, "TestSummoner", tc.tagLine)

			// Assert
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tc.expectedErr)
			assert.Nil(t, result)
			mockRepo.AssertNotCalled(t, "CreateWatch")
		})
	}
}

func TestSummonerWatchService_AddWatch_LowercaseTagLine(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(testhelpers.MockSummonerWatchRepository)
	service := NewSummonerWatchService(mockRepo)

	expectedWatch := &entities.SummonerWatchDetail{
		GuildID:      12345,
		WatchedAt:    time.Now(),
		SummonerName: "testsummoner",
		TagLine:      "na1",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Mock expectations - should call with lowercase tagLine as is
	mockRepo.On("CreateWatch", ctx, int64(12345), "testsummoner", "na1").Return(expectedWatch, nil)

	// Execute with lowercase tagLine
	result, err := service.AddWatch(ctx, 12345, "TestSummoner", "na1")

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, expectedWatch, result)
	mockRepo.AssertExpectations(t)
}

func TestSummonerWatchService_AddWatch_RepositoryError(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(testhelpers.MockSummonerWatchRepository)
	service := NewSummonerWatchService(mockRepo)

	repoErr := errors.New("repository error")
	mockRepo.On("CreateWatch", ctx, int64(12345), "testsummoner", "gamba").Return(nil, repoErr)

	// Execute
	result, err := service.AddWatch(ctx, 12345, "TestSummoner", "gamba")

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create summoner watch")
	assert.Nil(t, result)
	mockRepo.AssertExpectations(t)
}

func TestSummonerWatchService_RemoveWatch_Success(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(testhelpers.MockSummonerWatchRepository)
	service := NewSummonerWatchService(mockRepo)

	existingWatch := &entities.SummonerWatchDetail{
		GuildID:      12345,
		SummonerName: "TestSummoner",
		TagLine:      "NA1",
	}

	// Mock expectations
	mockRepo.On("GetWatch", ctx, int64(12345), "testsummoner", "na1").Return(existingWatch, nil)
	mockRepo.On("DeleteWatch", ctx, int64(12345), "testsummoner", "na1").Return(nil)

	// Execute
	err := service.RemoveWatch(ctx, 12345, "TestSummoner", "NA1")

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestSummonerWatchService_RemoveWatch_NotFound(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(testhelpers.MockSummonerWatchRepository)
	service := NewSummonerWatchService(mockRepo)

	// Mock expectations - watch doesn't exist
	mockRepo.On("GetWatch", ctx, int64(12345), "testsummoner", "na1").Return(nil, errors.New("not found"))

	// Execute
	err := service.RemoveWatch(ctx, 12345, "TestSummoner", "NA1")

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "summoner watch not found")
	mockRepo.AssertNotCalled(t, "DeleteWatch")
}

func TestSummonerWatchService_RemoveWatch_ValidationError(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(testhelpers.MockSummonerWatchRepository)
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
	mockRepo := new(testhelpers.MockSummonerWatchRepository)
	service := NewSummonerWatchService(mockRepo)

	existingWatch := &entities.SummonerWatchDetail{
		GuildID:      12345,
		SummonerName: "testsummoner",
		TagLine:      "NA1",
	}

	deleteErr := errors.New("delete failed")

	// Mock expectations
	mockRepo.On("GetWatch", ctx, int64(12345), "testsummoner", "na1").Return(existingWatch, nil)
	mockRepo.On("DeleteWatch", ctx, int64(12345), "testsummoner", "na1").Return(deleteErr)

	// Execute
	err := service.RemoveWatch(ctx, 12345, "TestSummoner", "NA1")

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to remove summoner watch")
	mockRepo.AssertExpectations(t)
}

func TestSummonerWatchService_ListWatches_Success(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(testhelpers.MockSummonerWatchRepository)
	service := NewSummonerWatchService(mockRepo)

	expectedWatches := []*entities.SummonerWatchDetail{
		{
			GuildID:      12345,
			SummonerName: "Summoner1",
			TagLine:      "NA1",
		},
		{
			GuildID:      12345,
			SummonerName: "Summoner2",
			TagLine:      "EUW1",
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
	mockRepo := new(testhelpers.MockSummonerWatchRepository)
	service := NewSummonerWatchService(mockRepo)

	// Mock expectations - empty list
	mockRepo.On("GetWatchesByGuild", ctx, int64(12345)).Return([]*entities.SummonerWatchDetail{}, nil)

	// Execute
	result, err := service.ListWatches(ctx, 12345)

	// Assert
	assert.NoError(t, err)
	assert.Empty(t, result)
	mockRepo.AssertExpectations(t)
}

func TestSummonerWatchService_ListWatches_RepositoryError(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(testhelpers.MockSummonerWatchRepository)
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

func TestSummonerWatchService_ValidateTagLine(t *testing.T) {
	service := &summonerWatchService{}

	// Valid tag lines
	validCases := []string{"gamba", "test", "123", "AB", "xyz"}
	for _, tagLine := range validCases {
		err := service.validateTagLine(tagLine)
		assert.NoError(t, err, "tag line %s should be valid", tagLine)
	}

	// Invalid cases
	invalidCases := []struct {
		tagLine string
		error   string
	}{
		{"", "tag line cannot be empty"},
		{"a", "tag line must be between 2 and 5 characters"},       // too short
		{"toolong", "tag line must be between 2 and 5 characters"}, // too long
		{"test!", "tag line contains invalid characters"},          // special char
		{"te st", "tag line contains invalid characters"},          // space
	}

	for _, tc := range invalidCases {
		err := service.validateTagLine(tc.tagLine)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), tc.error, "tag line %s should fail with expected error", tc.tagLine)
	}
}
