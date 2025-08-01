package bot

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"gambler/discord-client/application"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

// UserResolverImpl implements the UserResolver interface
type UserResolverImpl struct {
	session *discordgo.Session

	// Cache guild members to avoid repeated API calls
	memberCache map[int64][]*discordgo.Member
	cacheMutex  sync.RWMutex
	cacheExpiry map[int64]time.Time
	cacheTTL    time.Duration
	
	// Rate limit handling
	rateLimitMutex sync.Mutex
	lastAPICall    time.Time
	minAPIInterval time.Duration
}

// NewUserResolver creates a new user resolver
func NewUserResolver(session *discordgo.Session) application.UserResolver {
	return &UserResolverImpl{
		session:        session,
		memberCache:    make(map[int64][]*discordgo.Member),
		cacheExpiry:    make(map[int64]time.Time),
		cacheTTL:       5 * time.Minute, // Cache for 5 minutes
		minAPIInterval: 1 * time.Second,  // Minimum 1 second between API calls
	}
}

// ResolveUsersByNick finds Discord user IDs by their server nickname
func (r *UserResolverImpl) ResolveUsersByNick(ctx context.Context, guildID int64, nickname string) ([]int64, error) {
	// Get guild members (from cache or API)
	members, err := r.getGuildMembers(ctx, guildID)
	if err != nil {
		return nil, fmt.Errorf("failed to get guild members: %w", err)
	}

	// Search for users with matching nickname
	var userIDs []int64
	normalizedNick := strings.TrimSpace(nickname)

	for _, member := range members {
		// Skip members with nil User (defensive programming)
		if member == nil || member.User == nil {
			continue
		}
		
		// Check if nickname matches (case-sensitive to match Discord behavior)
		// Also check username if no server nickname is set
		memberNick := member.Nick
		if memberNick == "" {
			memberNick = member.User.Username
		}
		
		if memberNick == normalizedNick {
			userID, err := strconv.ParseInt(member.User.ID, 10, 64)
			if err != nil {
				log.Warnf("Failed to parse user ID %s: %v", member.User.ID, err)
				continue
			}
			userIDs = append(userIDs, userID)
		}
	}

	if len(userIDs) == 0 {
		log.Debugf("No users found with nickname '%s' in guild %d", nickname, guildID)
	} else {
		log.Debugf("Found %d user(s) with nickname '%s' in guild %d", len(userIDs), nickname, guildID)
	}

	return userIDs, nil
}

// getGuildMembers retrieves guild members from cache or Discord API
func (r *UserResolverImpl) getGuildMembers(ctx context.Context, guildID int64) ([]*discordgo.Member, error) {
	// Check cache first
	r.cacheMutex.RLock()
	members, exists := r.memberCache[guildID]
	expiry, hasExpiry := r.cacheExpiry[guildID]
	r.cacheMutex.RUnlock()

	// Return cached members if not expired
	if exists && hasExpiry && time.Now().Before(expiry) {
		return members, nil
	}

	// Validate guild exists and we have access
	guildIDStr := strconv.FormatInt(guildID, 10)
	_, err := r.session.Guild(guildIDStr)
	if err != nil {
		return nil, fmt.Errorf("cannot access guild %d: %w", guildID, err)
	}

	// Fetch from Discord API with rate limiting
	var allMembers []*discordgo.Member
	after := ""
	retryCount := 0
	maxRetries := 3

	for {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Apply rate limiting
		r.rateLimitMutex.Lock()
		timeSinceLastCall := time.Since(r.lastAPICall)
		if timeSinceLastCall < r.minAPIInterval {
			waitTime := r.minAPIInterval - timeSinceLastCall
			r.rateLimitMutex.Unlock()
			
			log.Debugf("Rate limiting: waiting %v before next API call", waitTime)
			select {
			case <-time.After(waitTime):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
			r.rateLimitMutex.Lock()
		}
		r.lastAPICall = time.Now()
		r.rateLimitMutex.Unlock()

		// Fetch batch of members (Discord limits to 1000 per request)
		batch, err := r.session.GuildMembers(guildIDStr, after, 1000)
		if err != nil {
			// Check if it's a rate limit error
			if strings.Contains(err.Error(), "429") || strings.Contains(err.Error(), "rate limit") {
				retryCount++
				if retryCount > maxRetries {
					return nil, fmt.Errorf("exceeded max retries for rate limit: %w", err)
				}
				
				// Exponential backoff: 2^retry seconds
				waitTime := time.Duration(1<<retryCount) * time.Second
				log.Warnf("Hit rate limit, waiting %v before retry %d/%d", waitTime, retryCount, maxRetries)
				
				select {
				case <-time.After(waitTime):
					continue // Retry the same request
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			}
			return nil, fmt.Errorf("failed to fetch guild members: %w", err)
		}
		
		// Reset retry count on successful request
		retryCount = 0

		// Filter out members with nil User (shouldn't happen but be defensive)
		for _, member := range batch {
			if member != nil && member.User != nil {
				allMembers = append(allMembers, member)
			}
		}

		// Check if we got all members
		if len(batch) < 1000 {
			log.Debugf("Fetched all %d members for guild %d", len(allMembers), guildID)
			break
		}

		// Set 'after' to the last member ID for pagination
		if len(batch) > 0 && batch[len(batch)-1].User != nil {
			after = batch[len(batch)-1].User.ID
			log.Debugf("Fetching next batch for guild %d after user %s (current total: %d)", guildID, after, len(allMembers))
		} else {
			// Shouldn't happen, but break to avoid infinite loop
			log.Warnf("Unable to determine next pagination token, stopping at %d members", len(allMembers))
			break
		}
	}

	// Update cache
	r.cacheMutex.Lock()
	r.memberCache[guildID] = allMembers
	r.cacheExpiry[guildID] = time.Now().Add(r.cacheTTL)
	r.cacheMutex.Unlock()

	log.Debugf("Cached %d members for guild %d", len(allMembers), guildID)

	return allMembers, nil
}

// InvalidateCache removes cached members for a specific guild
func (r *UserResolverImpl) InvalidateCache(guildID int64) {
	r.cacheMutex.Lock()
	delete(r.memberCache, guildID)
	delete(r.cacheExpiry, guildID)
	r.cacheMutex.Unlock()
}

// InvalidateAllCache removes all cached members
func (r *UserResolverImpl) InvalidateAllCache() {
	r.cacheMutex.Lock()
	r.memberCache = make(map[int64][]*discordgo.Member)
	r.cacheExpiry = make(map[int64]time.Time)
	r.cacheMutex.Unlock()
}

