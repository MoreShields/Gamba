package models

// GuildSettings represents per-guild configuration settings
type GuildSettings struct {
	GuildID          int64  `db:"guild_id"`
	PrimaryChannelID *int64 `db:"primary_channel_id"`  // Nullable - channel for gamba updates
	LolChannelID     *int64 `db:"lol_channel_id"`      // Nullable - channel for LOL updates
	HighRollerRoleID *int64 `db:"high_roller_role_id"` // Nullable - role ID for high roller (NULL = disabled)
}
