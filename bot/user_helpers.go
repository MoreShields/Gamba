package bot

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
	log.Infof("user id %s", userID)
	log.Infof("member %+v", member)
	log.Infof("guild %s", guildID)
	log.Infof("err %s", err)
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
