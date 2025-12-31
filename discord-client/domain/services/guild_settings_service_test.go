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

func TestGuildSettingsService_UpdateLottoChannel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		guildID     int64
		channelID   *int64
		setupMock   func(*testhelpers.MockGuildSettingsRepository)
		wantErr     bool
		errContains string
	}{
		{
			name:      "successful channel update",
			guildID:   123456789,
			channelID: func() *int64 { id := int64(111222333); return &id }(),
			setupMock: func(mockRepo *testhelpers.MockGuildSettingsRepository) {
				settings := &entities.GuildSettings{GuildID: 123456789}
				mockRepo.On("GetOrCreateGuildSettings", context.Background(), int64(123456789)).Return(settings, nil)
				mockRepo.On("UpdateGuildSettings", context.Background(), settings).Return(nil)
			},
			wantErr: false,
		},
		{
			name:      "clear channel by setting nil",
			guildID:   123456789,
			channelID: nil,
			setupMock: func(mockRepo *testhelpers.MockGuildSettingsRepository) {
				existingChannelID := int64(111222333)
				settings := &entities.GuildSettings{
					GuildID:        123456789,
					LottoChannelID: &existingChannelID,
				}
				mockRepo.On("GetOrCreateGuildSettings", context.Background(), int64(123456789)).Return(settings, nil)
				mockRepo.On("UpdateGuildSettings", context.Background(), settings).Return(nil)
			},
			wantErr: false,
		},
		{
			name:      "get settings error",
			guildID:   123456789,
			channelID: func() *int64 { id := int64(111222333); return &id }(),
			setupMock: func(mockRepo *testhelpers.MockGuildSettingsRepository) {
				mockRepo.On("GetOrCreateGuildSettings", context.Background(), int64(123456789)).Return((*entities.GuildSettings)(nil), errors.New("database error"))
			},
			wantErr:     true,
			errContains: "failed to get guild settings",
		},
		{
			name:      "update settings error",
			guildID:   123456789,
			channelID: func() *int64 { id := int64(111222333); return &id }(),
			setupMock: func(mockRepo *testhelpers.MockGuildSettingsRepository) {
				settings := &entities.GuildSettings{GuildID: 123456789}
				mockRepo.On("GetOrCreateGuildSettings", context.Background(), int64(123456789)).Return(settings, nil)
				mockRepo.On("UpdateGuildSettings", context.Background(), settings).Return(errors.New("database error"))
			},
			wantErr:     true,
			errContains: "failed to update guild settings",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			mockRepo := new(testhelpers.MockGuildSettingsRepository)
			tt.setupMock(mockRepo)

			service := NewGuildSettingsService(mockRepo)

			err := service.UpdateLottoChannel(ctx, tt.guildID, tt.channelID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestGuildSettingsService_UpdateLottoTicketCost(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		guildID     int64
		cost        *int64
		setupMock   func(*testhelpers.MockGuildSettingsRepository)
		wantErr     bool
		errContains string
	}{
		{
			name:    "successful cost update",
			guildID: 123456789,
			cost:    func() *int64 { c := int64(500); return &c }(),
			setupMock: func(mockRepo *testhelpers.MockGuildSettingsRepository) {
				settings := &entities.GuildSettings{GuildID: 123456789}
				mockRepo.On("GetOrCreateGuildSettings", context.Background(), int64(123456789)).Return(settings, nil)
				mockRepo.On("UpdateGuildSettings", context.Background(), settings).Return(nil)
			},
			wantErr: false,
		},
		{
			name:    "reset to default by setting nil",
			guildID: 123456789,
			cost:    nil,
			setupMock: func(mockRepo *testhelpers.MockGuildSettingsRepository) {
				existingCost := int64(500)
				settings := &entities.GuildSettings{
					GuildID:         123456789,
					LottoTicketCost: &existingCost,
				}
				mockRepo.On("GetOrCreateGuildSettings", context.Background(), int64(123456789)).Return(settings, nil)
				mockRepo.On("UpdateGuildSettings", context.Background(), settings).Return(nil)
			},
			wantErr: false,
		},
		{
			name:    "zero cost rejected",
			guildID: 123456789,
			cost:    func() *int64 { c := int64(0); return &c }(),
			setupMock: func(mockRepo *testhelpers.MockGuildSettingsRepository) {
				// No mock setup needed - validation fails before repository call
			},
			wantErr:     true,
			errContains: "ticket cost must be positive",
		},
		{
			name:    "negative cost rejected",
			guildID: 123456789,
			cost:    func() *int64 { c := int64(-100); return &c }(),
			setupMock: func(mockRepo *testhelpers.MockGuildSettingsRepository) {
				// No mock setup needed - validation fails before repository call
			},
			wantErr:     true,
			errContains: "ticket cost must be positive",
		},
		{
			name:    "get settings error",
			guildID: 123456789,
			cost:    func() *int64 { c := int64(500); return &c }(),
			setupMock: func(mockRepo *testhelpers.MockGuildSettingsRepository) {
				mockRepo.On("GetOrCreateGuildSettings", context.Background(), int64(123456789)).Return((*entities.GuildSettings)(nil), errors.New("database error"))
			},
			wantErr:     true,
			errContains: "failed to get guild settings",
		},
		{
			name:    "update settings error",
			guildID: 123456789,
			cost:    func() *int64 { c := int64(500); return &c }(),
			setupMock: func(mockRepo *testhelpers.MockGuildSettingsRepository) {
				settings := &entities.GuildSettings{GuildID: 123456789}
				mockRepo.On("GetOrCreateGuildSettings", context.Background(), int64(123456789)).Return(settings, nil)
				mockRepo.On("UpdateGuildSettings", context.Background(), settings).Return(errors.New("database error"))
			},
			wantErr:     true,
			errContains: "failed to update guild settings",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			mockRepo := new(testhelpers.MockGuildSettingsRepository)
			tt.setupMock(mockRepo)

			service := NewGuildSettingsService(mockRepo)

			err := service.UpdateLottoTicketCost(ctx, tt.guildID, tt.cost)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestGuildSettingsService_UpdateLottoDifficulty(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		guildID     int64
		difficulty  *int64
		setupMock   func(*testhelpers.MockGuildSettingsRepository)
		wantErr     bool
		errContains string
	}{
		{
			name:       "successful difficulty update - minimum value",
			guildID:    123456789,
			difficulty: func() *int64 { d := int64(entities.MinLottoDifficulty); return &d }(),
			setupMock: func(mockRepo *testhelpers.MockGuildSettingsRepository) {
				settings := &entities.GuildSettings{GuildID: 123456789}
				mockRepo.On("GetOrCreateGuildSettings", context.Background(), int64(123456789)).Return(settings, nil)
				mockRepo.On("UpdateGuildSettings", context.Background(), settings).Return(nil)
			},
			wantErr: false,
		},
		{
			name:       "successful difficulty update - maximum value",
			guildID:    123456789,
			difficulty: func() *int64 { d := int64(entities.MaxLottoDifficulty); return &d }(),
			setupMock: func(mockRepo *testhelpers.MockGuildSettingsRepository) {
				settings := &entities.GuildSettings{GuildID: 123456789}
				mockRepo.On("GetOrCreateGuildSettings", context.Background(), int64(123456789)).Return(settings, nil)
				mockRepo.On("UpdateGuildSettings", context.Background(), settings).Return(nil)
			},
			wantErr: false,
		},
		{
			name:       "successful difficulty update - middle value",
			guildID:    123456789,
			difficulty: func() *int64 { d := int64(10); return &d }(),
			setupMock: func(mockRepo *testhelpers.MockGuildSettingsRepository) {
				settings := &entities.GuildSettings{GuildID: 123456789}
				mockRepo.On("GetOrCreateGuildSettings", context.Background(), int64(123456789)).Return(settings, nil)
				mockRepo.On("UpdateGuildSettings", context.Background(), settings).Return(nil)
			},
			wantErr: false,
		},
		{
			name:       "reset to default by setting nil",
			guildID:    123456789,
			difficulty: nil,
			setupMock: func(mockRepo *testhelpers.MockGuildSettingsRepository) {
				existingDifficulty := int64(10)
				settings := &entities.GuildSettings{
					GuildID:         123456789,
					LottoDifficulty: &existingDifficulty,
				}
				mockRepo.On("GetOrCreateGuildSettings", context.Background(), int64(123456789)).Return(settings, nil)
				mockRepo.On("UpdateGuildSettings", context.Background(), settings).Return(nil)
			},
			wantErr: false,
		},
		{
			name:       "difficulty below minimum rejected",
			guildID:    123456789,
			difficulty: func() *int64 { d := int64(entities.MinLottoDifficulty - 1); return &d }(),
			setupMock: func(mockRepo *testhelpers.MockGuildSettingsRepository) {
				// No mock setup needed - validation fails before repository call
			},
			wantErr:     true,
			errContains: "difficulty must be between",
		},
		{
			name:       "difficulty above maximum rejected",
			guildID:    123456789,
			difficulty: func() *int64 { d := int64(entities.MaxLottoDifficulty + 1); return &d }(),
			setupMock: func(mockRepo *testhelpers.MockGuildSettingsRepository) {
				// No mock setup needed - validation fails before repository call
			},
			wantErr:     true,
			errContains: "difficulty must be between",
		},
		{
			name:       "negative difficulty rejected",
			guildID:    123456789,
			difficulty: func() *int64 { d := int64(-5); return &d }(),
			setupMock: func(mockRepo *testhelpers.MockGuildSettingsRepository) {
				// No mock setup needed - validation fails before repository call
			},
			wantErr:     true,
			errContains: "difficulty must be between",
		},
		{
			name:       "get settings error",
			guildID:    123456789,
			difficulty: func() *int64 { d := int64(10); return &d }(),
			setupMock: func(mockRepo *testhelpers.MockGuildSettingsRepository) {
				mockRepo.On("GetOrCreateGuildSettings", context.Background(), int64(123456789)).Return((*entities.GuildSettings)(nil), errors.New("database error"))
			},
			wantErr:     true,
			errContains: "failed to get guild settings",
		},
		{
			name:       "update settings error",
			guildID:    123456789,
			difficulty: func() *int64 { d := int64(10); return &d }(),
			setupMock: func(mockRepo *testhelpers.MockGuildSettingsRepository) {
				settings := &entities.GuildSettings{GuildID: 123456789}
				mockRepo.On("GetOrCreateGuildSettings", context.Background(), int64(123456789)).Return(settings, nil)
				mockRepo.On("UpdateGuildSettings", context.Background(), settings).Return(errors.New("database error"))
			},
			wantErr:     true,
			errContains: "failed to update guild settings",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			mockRepo := new(testhelpers.MockGuildSettingsRepository)
			tt.setupMock(mockRepo)

			service := NewGuildSettingsService(mockRepo)

			err := service.UpdateLottoDifficulty(ctx, tt.guildID, tt.difficulty)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}