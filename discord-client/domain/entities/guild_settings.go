package entities

// GuildSettings represents per-guild configuration settings
type GuildSettings struct {
	GuildID          int64  `db:"guild_id"`
	PrimaryChannelID *int64 `db:"primary_channel_id"` // Nullable - channel for gamba updates
	LolChannelID     *int64 `db:"lol_channel_id"`     // Nullable - channel for LOL updates
	WordleChannelID  *int64 `db:"wordle_channel_id"`  // Nullable - channel for Wordle results
	HighRollerRoleID *int64 `db:"high_roller_role_id"` // Nullable - role ID for high roller (NULL = disabled)
}

// HasPrimaryChannel checks if a primary channel is configured
func (gs *GuildSettings) HasPrimaryChannel() bool {
	return gs.PrimaryChannelID != nil && *gs.PrimaryChannelID > 0
}

// HasLolChannel checks if a LOL channel is configured
func (gs *GuildSettings) HasLolChannel() bool {
	return gs.LolChannelID != nil && *gs.LolChannelID > 0
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

// SetWordleChannel sets the Wordle channel ID
func (gs *GuildSettings) SetWordleChannel(channelID *int64) {
	gs.WordleChannelID = channelID
}

// SetHighRollerRole sets the high roller role ID
func (gs *GuildSettings) SetHighRollerRole(roleID *int64) {
	gs.HighRollerRoleID = roleID
}