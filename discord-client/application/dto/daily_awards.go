package dto

import "time"

// DailyAwardsPostDTO contains all information needed to post daily awards to Discord
type DailyAwardsPostDTO struct {
	GuildID   int64
	ChannelID int64
	Summary   *DailyAwardsSummaryDTO
}

// DailyAwardsSummaryDTO represents the awards summary for posting
type DailyAwardsSummaryDTO struct {
	GuildID         int64
	Date            time.Time
	Awards          []DailyAwardDTO
	TotalPayout     int64
	TotalServerBits int64
}

// DailyAwardDTO represents a single award
type DailyAwardDTO struct {
	Type      string
	DiscordID int64
	Reward    int64
	Details   string
}

// GuildChannelInfo contains channel information for a guild
type GuildChannelInfo struct {
	GuildID          int64
	PrimaryChannelID *int64
}