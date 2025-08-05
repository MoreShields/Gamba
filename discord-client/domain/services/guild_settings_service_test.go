package services

import (
	"context"
	"errors"
	"testing"

	"gambler/discord-client/domain/entities"
	"gambler/discord-client/domain/testhelpers"

	"github.com/stretchr/testify/assert"
)

func TestGuildSettingsService_GetHighRollerRoleID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		guildID     int64
		setupMock   func(*testhelpers.MockGuildSettingsRepository)
		want        *int64
		wantErr     bool
		errContains string
	}{
		{
			name:    "successful retrieval of role ID",
			guildID: 123456789,
			setupMock: func(mockRepo *testhelpers.MockGuildSettingsRepository) {
				roleID := int64(987654321)
				settings := &entities.GuildSettings{
					GuildID:          123456789,
					HighRollerRoleID: &roleID,
				}
				mockRepo.On("GetOrCreateGuildSettings", context.Background(), int64(123456789)).Return(settings, nil)
			},
			want: func() *int64 { id := int64(987654321); return &id }(),
		},
		{
			name:    "role ID is nil (not configured)",
			guildID: 123456789,
			setupMock: func(mockRepo *testhelpers.MockGuildSettingsRepository) {
				settings := &entities.GuildSettings{
					GuildID:          123456789,
					HighRollerRoleID: nil,
				}
				mockRepo.On("GetOrCreateGuildSettings", context.Background(), int64(123456789)).Return(settings, nil)
			},
			want: nil,
		},
		{
			name:    "repository error",
			guildID: 123456789,
			setupMock: func(mockRepo *testhelpers.MockGuildSettingsRepository) {
				mockRepo.On("GetOrCreateGuildSettings", context.Background(), int64(123456789)).Return((*entities.GuildSettings)(nil), errors.New("database connection failed"))
			},
			want:        nil,
			wantErr:     true,
			errContains: "failed to get guild settings",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Setup
			ctx := context.Background()
			mockRepo := new(testhelpers.MockGuildSettingsRepository)
			tt.setupMock(mockRepo)

			service := NewGuildSettingsService(mockRepo)

			// Execute
			got, err := service.GetHighRollerRoleID(ctx, tt.guildID)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, got)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				if tt.want == nil {
					assert.Nil(t, got)
				} else {
					assert.NotNil(t, got)
					assert.Equal(t, *tt.want, *got)
				}
			}

			// Verify mock expectations
			mockRepo.AssertExpectations(t)
		})
	}
}