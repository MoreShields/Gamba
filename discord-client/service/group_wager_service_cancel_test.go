package service

import (
	"errors"
	"gambler/discord-client/models"
	"testing"

	"github.com/stretchr/testify/mock"
)

func TestGroupWagerService_CancelGroupWager(t *testing.T) {
	fixture := NewGroupWagerTestFixture(t)
	
	creatorID := int64(123)
	resolverID := int64(999)
	unauthorizedID := int64(456)
	
	tests := []struct {
		name          string
		groupWagerID  int64
		cancellerID   *int64
		setupMocks    func(*TestMocks, *MockHelper)
		expectedError string
	}{
		{
			name:         "successful cancellation by creator",
			groupWagerID: 1,
			cancellerID:  &creatorID,
			setupMocks: func(mocks *TestMocks, helper *MockHelper) {
				wager := &models.GroupWager{
					ID:               1,
					CreatorDiscordID: &creatorID,
					State:            models.GroupWagerStateActive,
					MessageID:        789,
					ChannelID:        456,
				}
				helper.ExpectWagerLookup(1, wager)
				mocks.GroupWagerRepo.On("Update", helper.ctx, mock.MatchedBy(func(w *models.GroupWager) bool {
					return w.ID == 1 && w.State == models.GroupWagerStateCancelled
				})).Return(nil)
				helper.ExpectEventPublish("events.GroupWagerStateChangeEvent")
			},
		},
		{
			name:         "successful cancellation by resolver",
			groupWagerID: 1,
			cancellerID:  &resolverID, // Different from creator
			setupMocks: func(mocks *TestMocks, helper *MockHelper) {
				wager := &models.GroupWager{
					ID:               1,
					CreatorDiscordID: &creatorID, // Different from canceller
					State:            models.GroupWagerStateActive,
					MessageID:        789,
					ChannelID:        456,
				}
				helper.ExpectWagerLookup(1, wager)
				mocks.GroupWagerRepo.On("Update", helper.ctx, mock.MatchedBy(func(w *models.GroupWager) bool {
					return w.ID == 1 && w.State == models.GroupWagerStateCancelled
				})).Return(nil)
				helper.ExpectEventPublish("events.GroupWagerStateChangeEvent")
			},
		},
		{
			name:         "group wager not found",
			groupWagerID: 1,
			cancellerID:  &creatorID,
			setupMocks: func(mocks *TestMocks, helper *MockHelper) {
				helper.ExpectWagerNotFound(1)
			},
			expectedError: "group wager not found",
		},
		{
			name:         "database error",
			groupWagerID: 1,
			cancellerID:  &creatorID,
			setupMocks: func(mocks *TestMocks, helper *MockHelper) {
				mocks.GroupWagerRepo.On("GetDetailByID", helper.ctx, int64(1)).Return(nil, errors.New("db error"))
			},
			expectedError: "failed to get group wager detail: db error",
		},
		{
			name:         "successful cancellation of pending_resolution wager by creator",
			groupWagerID: 1,
			cancellerID:  &creatorID,
			setupMocks: func(mocks *TestMocks, helper *MockHelper) {
				wager := &models.GroupWager{
					ID:               1,
					CreatorDiscordID: &creatorID,
					State:            models.GroupWagerStatePendingResolution,
					MessageID:        789,
					ChannelID:        456,
				}
				helper.ExpectWagerLookup(1, wager)
				mocks.GroupWagerRepo.On("Update", helper.ctx, mock.MatchedBy(func(w *models.GroupWager) bool {
					return w.ID == 1 && w.State == models.GroupWagerStateCancelled
				})).Return(nil)
				helper.ExpectEventPublish("events.GroupWagerStateChangeEvent")
			},
		},
		{
			name:         "cannot cancel resolved wager",
			groupWagerID: 1,
			cancellerID:  &creatorID,
			setupMocks: func(mocks *TestMocks, helper *MockHelper) {
				wager := &models.GroupWager{
					ID:               1,
					CreatorDiscordID: &creatorID,
					State:            models.GroupWagerStateResolved,
				}
				helper.ExpectWagerLookup(1, wager)
			},
			expectedError: "can only cancel active or pending resolution group wagers",
		},
		{
			name:         "cannot cancel cancelled wager",
			groupWagerID: 1,
			cancellerID:  &creatorID,
			setupMocks: func(mocks *TestMocks, helper *MockHelper) {
				wager := &models.GroupWager{
					ID:               1,
					CreatorDiscordID: &creatorID,
					State:            models.GroupWagerStateCancelled,
				}
				helper.ExpectWagerLookup(1, wager)
			},
			expectedError: "can only cancel active or pending resolution group wagers",
		},
		{
			name:         "unauthorized canceller",
			groupWagerID: 1,
			cancellerID:  &unauthorizedID, // Not creator
			setupMocks: func(mocks *TestMocks, helper *MockHelper) {
				wager := &models.GroupWager{
					ID:               1,
					CreatorDiscordID: &creatorID, // Different from canceller
					State:            models.GroupWagerStateActive,
				}
				helper.ExpectWagerLookup(1, wager)
			},
			expectedError: "only the creator or a resolver can cancel a group wager",
		},
		{
			name:         "update failure",
			groupWagerID: 1,
			cancellerID:  &creatorID,
			setupMocks: func(mocks *TestMocks, helper *MockHelper) {
				wager := &models.GroupWager{
					ID:               1,
					CreatorDiscordID: &creatorID,
					State:            models.GroupWagerStateActive,
				}
				helper.ExpectWagerLookup(1, wager)
				mocks.GroupWagerRepo.On("Update", helper.ctx, mock.Anything).Return(errors.New("update error"))
			},
			expectedError: "failed to update group wager: update error",
		},
		{
			name:         "successful system cancellation",
			groupWagerID: 1,
			cancellerID:  nil, // System cancellation
			setupMocks: func(mocks *TestMocks, helper *MockHelper) {
				wager := &models.GroupWager{
					ID:               1,
					CreatorDiscordID: nil, // System created wager
					State:            models.GroupWagerStateActive,
					MessageID:        789,
					ChannelID:        456,
				}
				helper.ExpectWagerLookup(1, wager)
				mocks.GroupWagerRepo.On("Update", helper.ctx, mock.MatchedBy(func(w *models.GroupWager) bool {
					return w.ID == 1 && w.State == models.GroupWagerStateCancelled
				})).Return(nil)
				helper.ExpectEventPublish("events.GroupWagerStateChangeEvent")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset fixture for each test
			fixture.Reset()
			
			// Set resolvers if testing resolver cancellation
			if tt.cancellerID != nil && *tt.cancellerID == resolverID {
				fixture.SetResolvers(resolverID)
			} else if tt.name == "unauthorized canceller" {
				fixture.SetResolvers() // Clear resolvers
			}
			
			// Setup mocks
			tt.setupMocks(fixture.Mocks, fixture.Helper)

			// Execute test
			err := fixture.Service.CancelGroupWager(fixture.Ctx, tt.groupWagerID, tt.cancellerID)

			// Assert
			if tt.expectedError != "" {
				fixture.Assertions.AssertValidationError(err, tt.expectedError)
			} else {
				fixture.Assertions.AssertNoError(err)
			}

			// Verify all expectations were met
			fixture.AssertAllMocks()
		})
	}
}

func TestGroupWagerService_CancelGroupWager_SystemUser(t *testing.T) {
	fixture := NewGroupWagerTestFixture(t)
	resolverID := int64(999)

	t.Run("system can cancel own wager", func(t *testing.T) {
		fixture.Reset()
		fixture.SetResolvers(resolverID)

		// Setup mocks
		systemWager := &models.GroupWager{
			ID:               1,
			CreatorDiscordID: nil, // System user
			State:            models.GroupWagerStateActive,
			MessageID:        789,
			ChannelID:        456,
		}
		fixture.Helper.ExpectWagerLookup(1, systemWager)
		fixture.Mocks.GroupWagerRepo.On("Update", fixture.Ctx, mock.MatchedBy(func(gw *models.GroupWager) bool {
			return gw.State == models.GroupWagerStateCancelled
		})).Return(nil)
		fixture.Helper.ExpectEventPublish("events.GroupWagerStateChangeEvent")

		// Execute
		err := fixture.Service.CancelGroupWager(fixture.Ctx, 1, nil) // System cancelling

		// Assert
		fixture.Assertions.AssertNoError(err)
		fixture.AssertAllMocks()
	})

	t.Run("resolver can cancel system wager", func(t *testing.T) {
		fixture.Reset()
		fixture.SetResolvers(resolverID)

		// Setup mocks
		systemWager := &models.GroupWager{
			ID:               2,
			CreatorDiscordID: nil, // System user
			State:            models.GroupWagerStateActive,
			MessageID:        789,
			ChannelID:        456,
		}
		fixture.Helper.ExpectWagerLookup(2, systemWager)
		fixture.Mocks.GroupWagerRepo.On("Update", fixture.Ctx, mock.MatchedBy(func(gw *models.GroupWager) bool {
			return gw.State == models.GroupWagerStateCancelled
		})).Return(nil)
		fixture.Helper.ExpectEventPublish("events.GroupWagerStateChangeEvent")

		// Execute
		err := fixture.Service.CancelGroupWager(fixture.Ctx, 2, &resolverID) // Resolver cancelling

		// Assert
		fixture.Assertions.AssertNoError(err)
		fixture.AssertAllMocks()
	})

	t.Run("regular user cannot cancel system wager", func(t *testing.T) {
		fixture.Reset()
		fixture.SetResolvers() // Clear resolvers

		// Setup mocks
		systemWager := &models.GroupWager{
			ID:               3,
			CreatorDiscordID: nil, // System user
			State:            models.GroupWagerStateActive,
		}
		fixture.Helper.ExpectWagerLookup(3, systemWager)

		// Execute
		regularUserID := int64(12345)
		err := fixture.Service.CancelGroupWager(fixture.Ctx, 3, &regularUserID) // Regular user trying to cancel

		// Assert
		fixture.Assertions.AssertValidationError(err, "only the creator or a resolver can cancel")
		fixture.AssertAllMocks()
	})
}
