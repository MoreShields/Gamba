package service

import (
	"context"
	"errors"
	"gambler/discord-client/config"
	"gambler/discord-client/models"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGroupWagerService_CancelGroupWager(t *testing.T) {
	creatorID := int64(123)
	
	tests := []struct {
		name          string
		groupWagerID  int64
		cancellerID   int64
		setupMocks    func(*MockGroupWagerRepository, *MockEventPublisher)
		expectedError string
	}{
		{
			name:         "successful cancellation by creator",
			groupWagerID: 1,
			cancellerID:  123,
			setupMocks: func(repo *MockGroupWagerRepository, eventPub *MockEventPublisher) {
				wager := &models.GroupWager{
					ID:               1,
					CreatorDiscordID: &creatorID,
					State:            models.GroupWagerStateActive,
					MessageID:        789,
					ChannelID:        456,
				}
				repo.On("GetByID", mock.Anything, int64(1)).Return(wager, nil)
				repo.On("Update", mock.Anything, mock.MatchedBy(func(w *models.GroupWager) bool {
					return w.ID == 1 && w.State == models.GroupWagerStateCancelled
				})).Return(nil)
				eventPub.On("Publish", mock.Anything).Return()
			},
		},
		{
			name:         "successful cancellation by resolver",
			groupWagerID: 1,
			cancellerID:  999, // Different from creator
			setupMocks: func(repo *MockGroupWagerRepository, eventPub *MockEventPublisher) {
				// Set up config with resolver
				config.Get().ResolverDiscordIDs = []int64{999}

				wager := &models.GroupWager{
					ID:               1,
					CreatorDiscordID: &creatorID, // Different from canceller
					State:            models.GroupWagerStateActive,
					MessageID:        789,
					ChannelID:        456,
				}
				repo.On("GetByID", mock.Anything, int64(1)).Return(wager, nil)
				repo.On("Update", mock.Anything, mock.MatchedBy(func(w *models.GroupWager) bool {
					return w.ID == 1 && w.State == models.GroupWagerStateCancelled
				})).Return(nil)
				eventPub.On("Publish", mock.Anything).Return()
			},
		},
		{
			name:         "group wager not found",
			groupWagerID: 1,
			cancellerID:  123,
			setupMocks: func(repo *MockGroupWagerRepository, eventPub *MockEventPublisher) {
				repo.On("GetByID", mock.Anything, int64(1)).Return(nil, nil)
			},
			expectedError: "group wager not found",
		},
		{
			name:         "database error",
			groupWagerID: 1,
			cancellerID:  123,
			setupMocks: func(repo *MockGroupWagerRepository, eventPub *MockEventPublisher) {
				repo.On("GetByID", mock.Anything, int64(1)).Return(nil, errors.New("db error"))
			},
			expectedError: "failed to get group wager: db error",
		},
		{
			name:         "successful cancellation of pending_resolution wager by creator",
			groupWagerID: 1,
			cancellerID:  123,
			setupMocks: func(repo *MockGroupWagerRepository, eventPub *MockEventPublisher) {
				wager := &models.GroupWager{
					ID:               1,
					CreatorDiscordID: &creatorID,
					State:            models.GroupWagerStatePendingResolution,
					MessageID:        789,
					ChannelID:        456,
				}
				repo.On("GetByID", mock.Anything, int64(1)).Return(wager, nil)
				repo.On("Update", mock.Anything, mock.MatchedBy(func(w *models.GroupWager) bool {
					return w.ID == 1 && w.State == models.GroupWagerStateCancelled
				})).Return(nil)
				eventPub.On("Publish", mock.Anything).Return()
			},
		},
		{
			name:         "cannot cancel resolved wager",
			groupWagerID: 1,
			cancellerID:  123,
			setupMocks: func(repo *MockGroupWagerRepository, eventPub *MockEventPublisher) {
				wager := &models.GroupWager{
					ID:               1,
					CreatorDiscordID: &creatorID,
					State:            models.GroupWagerStateResolved,
				}
				repo.On("GetByID", mock.Anything, int64(1)).Return(wager, nil)
			},
			expectedError: "can only cancel active or pending resolution group wagers",
		},
		{
			name:         "cannot cancel cancelled wager",
			groupWagerID: 1,
			cancellerID:  123,
			setupMocks: func(repo *MockGroupWagerRepository, eventPub *MockEventPublisher) {
				wager := &models.GroupWager{
					ID:               1,
					CreatorDiscordID: &creatorID,
					State:            models.GroupWagerStateCancelled,
				}
				repo.On("GetByID", mock.Anything, int64(1)).Return(wager, nil)
			},
			expectedError: "can only cancel active or pending resolution group wagers",
		},
		{
			name:         "unauthorized canceller",
			groupWagerID: 1,
			cancellerID:  456, // Not creator
			setupMocks: func(repo *MockGroupWagerRepository, eventPub *MockEventPublisher) {
				// Clear resolvers
				config.Get().ResolverDiscordIDs = []int64{}

				wager := &models.GroupWager{
					ID:               1,
					CreatorDiscordID: &creatorID, // Different from canceller
					State:            models.GroupWagerStateActive,
				}
				repo.On("GetByID", mock.Anything, int64(1)).Return(wager, nil)
			},
			expectedError: "only the creator or a resolver can cancel a group wager",
		},
		{
			name:         "update failure",
			groupWagerID: 1,
			cancellerID:  123,
			setupMocks: func(repo *MockGroupWagerRepository, eventPub *MockEventPublisher) {
				wager := &models.GroupWager{
					ID:               1,
					CreatorDiscordID: &creatorID,
					State:            models.GroupWagerStateActive,
				}
				repo.On("GetByID", mock.Anything, int64(1)).Return(wager, nil)
				repo.On("Update", mock.Anything, mock.Anything).Return(errors.New("update error"))
			},
			expectedError: "failed to update group wager: update error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mocks
			mockGroupWagerRepo := new(MockGroupWagerRepository)
			mockUserRepo := new(MockUserRepository)
			mockBalanceHistoryRepo := new(MockBalanceHistoryRepository)
			mockEventPublisher := new(MockEventPublisher)

			// Setup mocks
			tt.setupMocks(mockGroupWagerRepo, mockEventPublisher)

			// Create service
			service := NewGroupWagerService(
				mockGroupWagerRepo,
				mockUserRepo,
				mockBalanceHistoryRepo,
				mockEventPublisher,
			)

			// Call method
			err := service.CancelGroupWager(context.Background(), tt.groupWagerID, tt.cancellerID)

			// Assert
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}

			// Verify all expectations were met
			mockGroupWagerRepo.AssertExpectations(t)
			mockEventPublisher.AssertExpectations(t)
		})
	}
}
