package entities

import "time"

// Lottery configuration defaults
const (
	DefaultLottoTicketCost = 1000
	DefaultLottoDifficulty = 8 // 2^8 = 256 possible numbers
	MinLottoDifficulty     = 4
	MaxLottoDifficulty     = 20
)

// GuildSettings represents per-guild configuration settings
type GuildSettings struct {
	GuildID                     int64      `db:"guild_id"`
	PrimaryChannelID            *int64     `db:"primary_channel_id"`              // Nullable - channel for gamba updates
	LolChannelID                *int64     `db:"lol_channel_id"`                  // Nullable - channel for LOL updates
	TftChannelID                *int64     `db:"tft_channel_id"`                  // Nullable - channel for TFT updates
	WordleChannelID             *int64     `db:"wordle_channel_id"`               // Nullable - channel for Wordle results
	HighRollerRoleID            *int64     `db:"high_roller_role_id"`             // Nullable - role ID for high roller (NULL = disabled)
	HighRollerTrackingStartTime *time.Time `db:"high_roller_tracking_start_time"` // Nullable - when to start tracking durations
	LottoChannelID              *int64     `db:"lotto_channel_id"`                // Nullable - channel for lottery messages
	LottoTicketCost             *int64     `db:"lotto_ticket_cost"`               // Nullable - ticket cost in bits (default: 1000)
	LottoDifficulty             *int64     `db:"lotto_difficulty"`                // Nullable - number of bits for ticket numbers (default: 8)
}

// HasPrimaryChannel checks if a primary channel is configured
func (gs *GuildSettings) HasPrimaryChannel() bool {
	return gs.PrimaryChannelID != nil && *gs.PrimaryChannelID > 0
}

// HasLolChannel checks if a LOL channel is configured
func (gs *GuildSettings) HasLolChannel() bool {
	return gs.LolChannelID != nil && *gs.LolChannelID > 0
}

// HasTftChannel checks if a TFT channel is configured
func (gs *GuildSettings) HasTftChannel() bool {
	return gs.TftChannelID != nil && *gs.TftChannelID > 0
}

// HasWordleChannel checks if a Wordle channel is configured
func (gs *GuildSettings) HasWordleChannel() bool {
	return gs.WordleChannelID != nil && *gs.WordleChannelID > 0
}

// HasHighRollerRole checks if a high roller role is configured
func (gs *GuildSettings) HasHighRollerRole() bool {
	return gs.HighRollerRoleID != nil && *gs.HighRollerRoleID > 0
}

// SetPrimaryChannel sets the primary channel ID
func (gs *GuildSettings) SetPrimaryChannel(channelID *int64) {
	gs.PrimaryChannelID = channelID
}

// SetLolChannel sets the LOL channel ID
func (gs *GuildSettings) SetLolChannel(channelID *int64) {
	gs.LolChannelID = channelID
}

// SetTftChannel sets the TFT channel ID
func (gs *GuildSettings) SetTftChannel(channelID *int64) {
	gs.TftChannelID = channelID
}

// SetWordleChannel sets the Wordle channel ID
func (gs *GuildSettings) SetWordleChannel(channelID *int64) {
	gs.WordleChannelID = channelID
}

// SetHighRollerRole sets the high roller role ID
func (gs *GuildSettings) SetHighRollerRole(roleID *int64) {
	gs.HighRollerRoleID = roleID
}

// HasHighRollerTrackingStartTime checks if a tracking start time is configured
func (gs *GuildSettings) HasHighRollerTrackingStartTime() bool {
	return gs.HighRollerTrackingStartTime != nil
}

// SetHighRollerTrackingStartTime sets the high roller tracking start time
func (gs *GuildSettings) SetHighRollerTrackingStartTime(startTime *time.Time) {
	gs.HighRollerTrackingStartTime = startTime
}

// HasLottoChannel checks if a lottery channel is configured
func (gs *GuildSettings) HasLottoChannel() bool {
	return gs.LottoChannelID != nil && *gs.LottoChannelID > 0
}

// GetLottoChannelID returns the lottery channel ID or 0 if not set
func (gs *GuildSettings) GetLottoChannelID() int64 {
	if gs.LottoChannelID != nil {
		return *gs.LottoChannelID
	}
	return 0
}

// SetLottoChannel sets the lottery channel ID
func (gs *GuildSettings) SetLottoChannel(channelID *int64) {
	gs.LottoChannelID = channelID
}

// GetLottoTicketCost returns the lottery ticket cost or default if not set
func (gs *GuildSettings) GetLottoTicketCost() int64 {
	if gs.LottoTicketCost != nil {
		return *gs.LottoTicketCost
	}
	return DefaultLottoTicketCost
}

// SetLottoTicketCost sets the lottery ticket cost
func (gs *GuildSettings) SetLottoTicketCost(cost *int64) {
	gs.LottoTicketCost = cost
}

// GetLottoDifficulty returns the lottery difficulty or default if not set
func (gs *GuildSettings) GetLottoDifficulty() int64 {
	if gs.LottoDifficulty != nil {
		return *gs.LottoDifficulty
	}
	return DefaultLottoDifficulty
}

// SetLottoDifficulty sets the lottery difficulty
func (gs *GuildSettings) SetLottoDifficulty(difficulty *int64) {
	gs.LottoDifficulty = difficulty
}

// IsLottoEnabled returns true if lottery is configured (channel is set)
func (gs *GuildSettings) IsLottoEnabled() bool {
	return gs.HasLottoChannel()
}
