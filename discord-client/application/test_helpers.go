package application

import (
	"context"

	"gambler/discord-client/application/dto"
)

// MockDiscordPoster implements DiscordPoster for testing
type MockDiscordPoster struct {
	Posts []dto.HouseWagerPostDTO
	Error error
}

func (m *MockDiscordPoster) PostHouseWager(ctx context.Context, dto dto.HouseWagerPostDTO) (*PostResult, error) {
	if m.Error != nil {
		return nil, m.Error
	}

	m.Posts = append(m.Posts, dto)

	return &PostResult{
		MessageID: 123456789, // Mock Discord message ID
		ChannelID: dto.ChannelID,
	}, nil
}

// UpdateHouseWager mock implementation
func (m *MockDiscordPoster) UpdateHouseWager(ctx context.Context, messageID, channelID int64, dto dto.HouseWagerPostDTO) error {
	if m.Error != nil {
		return m.Error
	}
	// For tests, we don't need to track updates, just return success
	return nil
}

// UpdateGroupWager mock implementation
func (m *MockDiscordPoster) UpdateGroupWager(ctx context.Context, messageID, channelID int64, detail interface{}) error {
	if m.Error != nil {
		return m.Error
	}
	// For tests, we don't need to track updates, just return success
	return nil
}

// PostDailyAwards mock implementation
func (m *MockDiscordPoster) PostDailyAwards(ctx context.Context, dto dto.DailyAwardsPostDTO) error {
	if m.Error != nil {
		return m.Error
	}
	// For tests, we don't need to track daily awards posts, just return success
	return nil
}
