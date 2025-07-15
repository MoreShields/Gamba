package common

import (
	"strconv"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

// GetDisplayName returns the server-specific display name for a user
// Falls back to username if nickname is not set or if there's an error
func GetDisplayName(s *discordgo.Session, guildID, userID string) string {
	// Try to get guild member for server-specific nickname
	member, err := s.GuildMember(guildID, userID)
	if err == nil && member != nil {
		// Return nickname if set, otherwise username
		if member.Nick != "" {
			return member.Nick
		}
		if member.User != nil {
			return member.User.Username
		}
	}

	// Fallback to just getting the user
	user, err := s.User(userID)
	if err == nil && user != nil {
		return user.Username
	}

	return "Unknown"
}

// GetDisplayNameInt64 is a convenience wrapper that accepts int64 user IDs
func GetDisplayNameInt64(s *discordgo.Session, guildID string, userID int64) string {
	return GetDisplayName(s, guildID, strconv.FormatInt(userID, 10))
}

// ParseUserID converts a Discord user ID string to int64
func ParseUserID(userID string) (int64, error) {
	return strconv.ParseInt(userID, 10, 64)
}

// FormatUserID converts an int64 user ID to string
func FormatUserID(userID int64) string {
	return strconv.FormatInt(userID, 10)
}

// GetUserMention returns a Discord mention string for a user
func GetUserMention(userID int64) string {
	return "<@" + FormatUserID(userID) + ">"
}

// IsUserAdmin checks if a user has administrator permissions in a guild
func IsUserAdmin(s *discordgo.Session, guildID, userID string) bool {
	member, err := s.GuildMember(guildID, userID)
	if err != nil {
		log.Errorf("Failed to get guild member: %v", err)
		return false
	}

	// Check each role for admin permissions
	for _, roleID := range member.Roles {
		role, err := s.State.Role(guildID, roleID)
		if err != nil {
			continue
		}
		if role.Permissions&discordgo.PermissionAdministrator != 0 {
			return true
		}
	}

	return false
}
