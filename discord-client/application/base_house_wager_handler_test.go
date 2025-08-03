package application

import (
	"testing"
	"time"

	"gambler/discord-client/application/dto"
	"gambler/discord-client/domain/entities"

	"github.com/stretchr/testify/assert"
)

func TestBaseHouseWagerHandler_BuildHouseWagerDTO(t *testing.T) {
	t.Parallel()
	
	// Create handler for testing
	handler := NewBaseHouseWagerHandler(nil, nil)

	tests := []struct {
		name     string
		detail   *entities.GroupWagerDetail
		expected dto.HouseWagerPostDTO
	}{
		{
			name: "basic wager with title only",
			detail: &entities.GroupWagerDetail{
				Wager: &entities.GroupWager{
					ID:           123,
					GuildID:      12345,
					ChannelID:    999,
					Condition:    "TestPlayer - **TFT Match**",
					State:        entities.GroupWagerStateActive,
					TotalPot:     500,
					VotingEndsAt: func() *time.Time { t := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC); return &t }(),
				},
				Options: []*entities.GroupWagerOption{
					{
						ID:             1,
						OptionText:     "Top 4",
						OptionOrder:    1,
						OddsMultiplier: 2.0,
						TotalAmount:    250,
					},
					{
						ID:             2,
						OptionText:     "Bottom 4",
						OptionOrder:    2,
						OddsMultiplier: 2.0,
						TotalAmount:    250,
					},
				},
				Participants: []*entities.GroupWagerParticipant{
					{
						DiscordID: 111111111111111111,
						OptionID:  1,
						Amount:    100,
					},
					{
						DiscordID: 222222222222222222,
						OptionID:  2,
						Amount:    150,
					},
				},
			},
			expected: dto.HouseWagerPostDTO{
				GuildID:      12345,
				ChannelID:    999,
				WagerID:      123,
				Title:        "TestPlayer - **TFT Match**",
				Description:  "",
				State:        "active",
				TotalPot:     500,
				VotingEndsAt: func() *time.Time { t := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC); return &t }(),
				Options: []dto.WagerOptionDTO{
					{
						ID:          1,
						Text:        "Top 4",
						Order:       1,
						Multiplier:  2.0,
						TotalAmount: 250,
					},
					{
						ID:          2,
						Text:        "Bottom 4",
						Order:       2,
						Multiplier:  2.0,
						TotalAmount: 250,
					},
				},
				Participants: []dto.ParticipantDTO{
					{
						DiscordID: 111111111111111111,
						OptionID:  1,
						Amount:    100,
					},
					{
						DiscordID: 222222222222222222,
						OptionID:  2,
						Amount:    150,
					},
				},
			},
		},
		{
			name: "wager with title and description",
			detail: &entities.GroupWagerDetail{
				Wager: &entities.GroupWager{
					ID:           456,
					GuildID:      54321,
					ChannelID:    888,
					Condition:    "ProPlayer - **LoL Match**\nRanked Solo Queue - Diamond+",
					State:        entities.GroupWagerStateResolved,
					TotalPot:     1000,
					VotingEndsAt: func() *time.Time { t := time.Date(2024, 2, 20, 18, 30, 0, 0, time.UTC); return &t }(),
				},
				Options: []*entities.GroupWagerOption{
					{
						ID:             1,
						OptionText:     "Win",
						OptionOrder:    1,
						OddsMultiplier: 1.8,
						TotalAmount:    600,
					},
					{
						ID:             2,
						OptionText:     "Loss",
						OptionOrder:    2,
						OddsMultiplier: 2.2,
						TotalAmount:    400,
					},
				},
				Participants: []*entities.GroupWagerParticipant{
					{
						DiscordID: 333333333333333333,
						OptionID:  1,
						Amount:    200,
					},
					{
						DiscordID: 444444444444444444,
						OptionID:  1,
						Amount:    400,
					},
					{
						DiscordID: 555555555555555555,
						OptionID:  2,
						Amount:    400,
					},
				},
			},
			expected: dto.HouseWagerPostDTO{
				GuildID:      54321,
				ChannelID:    888,
				WagerID:      456,
				Title:        "ProPlayer - **LoL Match**",
				Description:  "Ranked Solo Queue - Diamond+",
				State:        "resolved",
				TotalPot:     1000,
				VotingEndsAt: func() *time.Time { t := time.Date(2024, 2, 20, 18, 30, 0, 0, time.UTC); return &t }(),
				Options: []dto.WagerOptionDTO{
					{
						ID:          1,
						Text:        "Win",
						Order:       1,
						Multiplier:  1.8,
						TotalAmount: 600,
					},
					{
						ID:          2,
						Text:        "Loss",
						Order:       2,
						Multiplier:  2.2,
						TotalAmount: 400,
					},
				},
				Participants: []dto.ParticipantDTO{
					{
						DiscordID: 333333333333333333,
						OptionID:  1,
						Amount:    200,
					},
					{
						DiscordID: 444444444444444444,
						OptionID:  1,
						Amount:    400,
					},
					{
						DiscordID: 555555555555555555,
						OptionID:  2,
						Amount:    400,
					},
				},
			},
		},
		{
			name: "cancelled wager with no participants",
			detail: &entities.GroupWagerDetail{
				Wager: &entities.GroupWager{
					ID:           789,
					GuildID:      99999,
					ChannelID:    777,
					Condition:    "ShortGamePlayer - **TFT Match**",
					State:        entities.GroupWagerStateCancelled,
					TotalPot:     0,
					VotingEndsAt: func() *time.Time { t := time.Date(2024, 3, 10, 14, 0, 0, 0, time.UTC); return &t }(),
				},
				Options: []*entities.GroupWagerOption{
					{
						ID:             1,
						OptionText:     "Top 4",
						OptionOrder:    1,
						OddsMultiplier: 2.0,
						TotalAmount:    0,
					},
					{
						ID:             2,
						OptionText:     "Bottom 4",
						OptionOrder:    2,
						OddsMultiplier: 2.0,
						TotalAmount:    0,
					},
				},
				Participants: []*entities.GroupWagerParticipant{},
			},
			expected: dto.HouseWagerPostDTO{
				GuildID:      99999,
				ChannelID:    777,
				WagerID:      789,
				Title:        "ShortGamePlayer - **TFT Match**",
				Description:  "",
				State:        "cancelled",
				TotalPot:     0,
				VotingEndsAt: func() *time.Time { t := time.Date(2024, 3, 10, 14, 0, 0, 0, time.UTC); return &t }(),
				Options: []dto.WagerOptionDTO{
					{
						ID:          1,
						Text:        "Top 4",
						Order:       1,
						Multiplier:  2.0,
						TotalAmount: 0,
					},
					{
						ID:          2,
						Text:        "Bottom 4",
						Order:       2,
						Multiplier:  2.0,
						TotalAmount: 0,
					},
				},
				Participants: []dto.ParticipantDTO{},
			},
		},
		{
			name: "wager with multiline description",
			detail: &entities.GroupWagerDetail{
				Wager: &entities.GroupWager{
					ID:           999,
					GuildID:      11111,
					ChannelID:    666,
					Condition:    "ComplexPlayer - **TFT Match**\nSet 12 Ranked\nChallenger MMR\nExpected 20+ minute game",
					State:        entities.GroupWagerStateActive,
					TotalPot:     2500,
					VotingEndsAt: func() *time.Time { t := time.Date(2024, 4, 5, 20, 15, 0, 0, time.UTC); return &t }(),
				},
				Options: []*entities.GroupWagerOption{
					{
						ID:             1,
						OptionText:     "Top 4",
						OptionOrder:    1,
						OddsMultiplier: 1.9,
						TotalAmount:    1300,
					},
					{
						ID:             2,
						OptionText:     "Bottom 4",
						OptionOrder:    2,
						OddsMultiplier: 2.1,
						TotalAmount:    1200,
					},
				},
				Participants: []*entities.GroupWagerParticipant{
					{
						DiscordID: 777777777777777777,
						OptionID:  1,
						Amount:    500,
					},
					{
						DiscordID: 888888888888888888,
						OptionID:  1,
						Amount:    800,
					},
					{
						DiscordID: 999999999999999999,
						OptionID:  2,
						Amount:    1200,
					},
				},
			},
			expected: dto.HouseWagerPostDTO{
				GuildID:      11111,
				ChannelID:    666,
				WagerID:      999,
				Title:        "ComplexPlayer - **TFT Match**",
				Description:  "Set 12 Ranked\nChallenger MMR\nExpected 20+ minute game",
				State:        "active",
				TotalPot:     2500,
				VotingEndsAt: func() *time.Time { t := time.Date(2024, 4, 5, 20, 15, 0, 0, time.UTC); return &t }(),
				Options: []dto.WagerOptionDTO{
					{
						ID:          1,
						Text:        "Top 4",
						Order:       1,
						Multiplier:  1.9,
						TotalAmount: 1300,
					},
					{
						ID:          2,
						Text:        "Bottom 4",
						Order:       2,
						Multiplier:  2.1,
						TotalAmount: 1200,
					},
				},
				Participants: []dto.ParticipantDTO{
					{
						DiscordID: 777777777777777777,
						OptionID:  1,
						Amount:    500,
					},
					{
						DiscordID: 888888888888888888,
						OptionID:  1,
						Amount:    800,
					},
					{
						DiscordID: 999999999999999999,
						OptionID:  2,
						Amount:    1200,
					},
				},
			},
		},
		{
			name: "empty condition handling",
			detail: &entities.GroupWagerDetail{
				Wager: &entities.GroupWager{
					ID:           100,
					GuildID:      22222,
					ChannelID:    555,
					Condition:    "",
					State:        entities.GroupWagerStateActive,
					TotalPot:     0,
					VotingEndsAt: func() *time.Time { t := time.Date(2024, 5, 1, 10, 0, 0, 0, time.UTC); return &t }(),
				},
				Options:      []*entities.GroupWagerOption{},
				Participants: []*entities.GroupWagerParticipant{},
			},
			expected: dto.HouseWagerPostDTO{
				GuildID:      22222,
				ChannelID:    555,
				WagerID:      100,
				Title:        "",
				Description:  "",
				State:        "active",
				TotalPot:     0,
				VotingEndsAt: func() *time.Time { t := time.Date(2024, 5, 1, 10, 0, 0, 0, time.UTC); return &t }(),
				Options:      []dto.WagerOptionDTO{},
				Participants: []dto.ParticipantDTO{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			
			result := handler.BuildHouseWagerDTO(tt.detail)
			
			// Assert all fields match
			assert.Equal(t, tt.expected.GuildID, result.GuildID)
			assert.Equal(t, tt.expected.ChannelID, result.ChannelID)
			assert.Equal(t, tt.expected.WagerID, result.WagerID)
			assert.Equal(t, tt.expected.Title, result.Title)
			assert.Equal(t, tt.expected.Description, result.Description)
			assert.Equal(t, tt.expected.State, result.State)
			assert.Equal(t, tt.expected.TotalPot, result.TotalPot)
			assert.Equal(t, tt.expected.VotingEndsAt, result.VotingEndsAt)
			
			// Assert options match
			assert.Len(t, result.Options, len(tt.expected.Options))
			for i, expectedOption := range tt.expected.Options {
				if i < len(result.Options) {
					assert.Equal(t, expectedOption.ID, result.Options[i].ID)
					assert.Equal(t, expectedOption.Text, result.Options[i].Text)
					assert.Equal(t, expectedOption.Order, result.Options[i].Order)
					assert.Equal(t, expectedOption.Multiplier, result.Options[i].Multiplier)
					assert.Equal(t, expectedOption.TotalAmount, result.Options[i].TotalAmount)
				}
			}
			
			// Assert participants match
			assert.Len(t, result.Participants, len(tt.expected.Participants))
			for i, expectedParticipant := range tt.expected.Participants {
				if i < len(result.Participants) {
					assert.Equal(t, expectedParticipant.DiscordID, result.Participants[i].DiscordID)
					assert.Equal(t, expectedParticipant.OptionID, result.Participants[i].OptionID)
					assert.Equal(t, expectedParticipant.Amount, result.Participants[i].Amount)
				}
			}
			
			// Use overall struct comparison for final verification
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBaseHouseWagerHandler_BuildHouseWagerDTO_ConditionParsing(t *testing.T) {
	t.Parallel()
	
	handler := NewBaseHouseWagerHandler(nil, nil)

	tests := []struct {
		name              string
		condition         string
		expectedTitle     string
		expectedDescription string
	}{
		{
			name:              "title only",
			condition:         "Player - **TFT Match**",
			expectedTitle:     "Player - **TFT Match**",
			expectedDescription: "",
		},
		{
			name:              "title with single line description",
			condition:         "Player - **TFT Match**\nRanked Game",
			expectedTitle:     "Player - **TFT Match**",
			expectedDescription: "Ranked Game",
		},
		{
			name:              "title with multiline description",
			condition:         "Player - **TFT Match**\nSet 12 Ranked\nChallenger MMR",
			expectedTitle:     "Player - **TFT Match**",
			expectedDescription: "Set 12 Ranked\nChallenger MMR",
		},
		{
			name:              "empty condition",
			condition:         "",
			expectedTitle:     "",
			expectedDescription: "",
		},
		{
			name:              "only newline",
			condition:         "\n",
			expectedTitle:     "",
			expectedDescription: "",
		},
		{
			name:              "newline at start",
			condition:         "\nDescription only",
			expectedTitle:     "",
			expectedDescription: "Description only",
		},
		{
			name:              "multiple consecutive newlines",
			condition:         "Title\n\n\nDescription with gaps",
			expectedTitle:     "Title",
			expectedDescription: "\n\nDescription with gaps",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			
			detail := &entities.GroupWagerDetail{
				Wager: &entities.GroupWager{
					ID:           1,
					GuildID:      1,
					ChannelID:    1,
					Condition:    tt.condition,
					State:        entities.GroupWagerStateActive,
					TotalPot:     0,
					VotingEndsAt: func() *time.Time { t := time.Now(); return &t }(),
				},
				Options:      []*entities.GroupWagerOption{},
				Participants: []*entities.GroupWagerParticipant{},
			}
			
			result := handler.BuildHouseWagerDTO(detail)
			
			assert.Equal(t, tt.expectedTitle, result.Title)
			assert.Equal(t, tt.expectedDescription, result.Description)
		})
	}
}