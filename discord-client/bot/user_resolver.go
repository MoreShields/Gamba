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

// NameType represents the type of Discord name
type NameType int

const (
	NicknameType    NameType = iota // Server-specific nickname (highest priority)
	DisplayNameType                  // Global display name
	UsernameType                     // Username (lowest priority)
)

// String returns a human-readable name for the NameType
func (n NameType) String() string {
	switch n {
	case NicknameType:
		return "server nickname"
	case DisplayNameType:
		return "global display name"
	case UsernameType:
		return "username"
	default:
		return "unknown"
	}
}

// RateLimiter manages API call rate limiting
type RateLimiter struct {
	mutex       sync.Mutex
	lastCall    time.Time
	minInterval time.Duration
}

// Wait waits if necessary to respect rate limits
func (rl *RateLimiter) Wait(ctx context.Context) error {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	elapsed := time.Since(rl.lastCall)
	if elapsed < rl.minInterval {
		waitTime := rl.minInterval - elapsed
		log.Debugf("Rate limiting: waiting %v before next API call", waitTime)
		
		select {
		case <-time.After(waitTime):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	rl.lastCall = time.Now()
	return nil
}

// UserResolverImpl implements the UserResolver interface
type UserResolverImpl struct {
	session *discordgo.Session

	// Cache guild members to avoid repeated API calls
	memberCache map[int64][]*discordgo.Member
	cacheMutex  sync.RWMutex
	cacheExpiry map[int64]time.Time
	cacheTTL    time.Duration

	// Unified name cache
	nameCache      map[int64]map[NameType]map[string][]int64 // guildID -> nameType -> name -> userIDs
	nameCacheMutex sync.RWMutex

	// Rate limiting
	rateLimiter *RateLimiter
}

// NewUserResolver creates a new user resolver
func NewUserResolver(session *discordgo.Session) application.UserResolver {
	return &UserResolverImpl{
		session:     session,
		memberCache: make(map[int64][]*discordgo.Member),
		cacheExpiry: make(map[int64]time.Time),
		cacheTTL:    5 * time.Minute,
		nameCache:   make(map[int64]map[NameType]map[string][]int64),
		rateLimiter: &RateLimiter{
			minInterval: 1 * time.Second,
		},
	}
}

// ResolveUsersByNick finds Discord user IDs by their server nickname
func (r *UserResolverImpl) ResolveUsersByNick(ctx context.Context, guildID int64, nickname string) ([]int64, error) {
	normalizedNick := strings.TrimSpace(nickname)

	// Try cache first
	if userIDs := r.searchAllCacheTypes(guildID, normalizedNick); len(userIDs) > 0 {
		return userIDs, nil
	}

	// Refresh cache and try again
	if err := r.refreshMemberCache(ctx, guildID); err != nil {
		return nil, fmt.Errorf("failed to refresh member cache: %w", err)
	}

	userIDs := r.searchAllCacheTypes(guildID, normalizedNick)
	if len(userIDs) == 0 {
		log.Infof("No users found with nickname, display name, or username '%s' in guild %d", normalizedNick, guildID)
	}
	
	return userIDs, nil
}

// searchAllCacheTypes searches for a name in all cache types in priority order
func (r *UserResolverImpl) searchAllCacheTypes(guildID int64, name string) []int64 {
	searchOrder := []NameType{NicknameType, DisplayNameType, UsernameType}

	r.nameCacheMutex.RLock()
	defer r.nameCacheMutex.RUnlock()

	for _, nameType := range searchOrder {
		if userIDs := r.searchInCache(guildID, name, nameType); len(userIDs) > 0 {
			log.Infof("Found %d user(s) with %s '%s' in guild %d", 
				len(userIDs), nameType, name, guildID)
			return userIDs
		}
	}

	return []int64{}
}

// searchInCache searches for a name in a specific cache type (must be called with lock held)
func (r *UserResolverImpl) searchInCache(guildID int64, name string, nameType NameType) []int64 {
	if guildCache, exists := r.nameCache[guildID]; exists {
		if typeCache, exists := guildCache[nameType]; exists {
			if userIDs, found := typeCache[name]; found {
				return userIDs
			}
		}
	}
	return nil
}

// refreshMemberCache fetches members from Discord API and rebuilds the name cache
func (r *UserResolverImpl) refreshMemberCache(ctx context.Context, guildID int64) error {
	members, err := r.getGuildMembers(ctx, guildID)
	if err != nil {
		return err
	}

	r.buildNameCache(guildID, members)
	r.logCacheContents(guildID)
	
	return nil
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

	// Fetch from Discord API
	return r.fetchGuildMembersFromAPI(ctx, guildID)
}

// fetchGuildMembersFromAPI fetches all guild members from Discord API with pagination
func (r *UserResolverImpl) fetchGuildMembersFromAPI(ctx context.Context, guildID int64) ([]*discordgo.Member, error) {
	// Validate guild exists and we have access
	guildIDStr := strconv.FormatInt(guildID, 10)
	if _, err := r.session.Guild(guildIDStr); err != nil {
		return nil, fmt.Errorf("cannot access guild %d: %w", guildID, err)
	}

	var allMembers []*discordgo.Member
	after := ""
	
	for {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Apply rate limiting
		if err := r.rateLimiter.Wait(ctx); err != nil {
			return nil, err
		}

		// Fetch batch with retry
		batch, err := r.fetchMemberBatchWithRetry(ctx, guildIDStr, after)
		if err != nil {
			return nil, err
		}

		// Add valid members to result
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

		// Set 'after' for pagination
		if len(batch) > 0 && batch[len(batch)-1].User != nil {
			after = batch[len(batch)-1].User.ID
			log.Debugf("Fetching next batch for guild %d after user %s (current total: %d)", 
				guildID, after, len(allMembers))
		} else {
			log.Warnf("Unable to determine next pagination token, stopping at %d members", len(allMembers))
			break
		}
	}

	// Update cache
	r.updateMemberCache(guildID, allMembers)
	
	return allMembers, nil
}

// fetchMemberBatchWithRetry fetches a batch of members with exponential backoff retry
func (r *UserResolverImpl) fetchMemberBatchWithRetry(ctx context.Context, guildID string, after string) ([]*discordgo.Member, error) {
	maxRetries := 3
	
	for attempt := 0; attempt <= maxRetries; attempt++ {
		batch, err := r.session.GuildMembers(guildID, after, 1000)
		if err == nil {
			return batch, nil
		}

		// Check if it's a rate limit error
		if !isRateLimitError(err) {
			return nil, fmt.Errorf("failed to fetch guild members: %w", err)
		}

		// Don't retry if we've hit max retries
		if attempt >= maxRetries {
			return nil, fmt.Errorf("exceeded max retries for rate limit: %w", err)
		}

		// Exponential backoff
		waitTime := time.Duration(1<<uint(attempt)) * time.Second
		log.Warnf("Hit rate limit, waiting %v before retry %d/%d", waitTime, attempt+1, maxRetries)

		select {
		case <-time.After(waitTime):
			continue
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return nil, fmt.Errorf("unexpected retry loop exit")
}

// isRateLimitError checks if an error is a rate limit error
func isRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "429") || strings.Contains(errStr, "rate limit")
}

// updateMemberCache updates the member cache for a guild
func (r *UserResolverImpl) updateMemberCache(guildID int64, members []*discordgo.Member) {
	r.cacheMutex.Lock()
	r.memberCache[guildID] = members
	r.cacheExpiry[guildID] = time.Now().Add(r.cacheTTL)
	r.cacheMutex.Unlock()

	log.Debugf("Cached %d members for guild %d", len(members), guildID)
}

// buildNameCache builds the unified name cache from guild members
func (r *UserResolverImpl) buildNameCache(guildID int64, members []*discordgo.Member) {
	r.nameCacheMutex.Lock()
	defer r.nameCacheMutex.Unlock()

	// Initialize guild cache if needed
	if r.nameCache[guildID] == nil {
		r.nameCache[guildID] = make(map[NameType]map[string][]int64)
		for _, nameType := range []NameType{NicknameType, DisplayNameType, UsernameType} {
			r.nameCache[guildID][nameType] = make(map[string][]int64)
		}
	} else {
		// Clear existing caches
		for nameType := range r.nameCache[guildID] {
			r.nameCache[guildID][nameType] = make(map[string][]int64)
		}
	}

	stats := make(map[NameType]int)

	for _, member := range members {
		if member == nil || member.User == nil {
			continue
		}

		userID, err := strconv.ParseInt(member.User.ID, 10, 64)
		if err != nil {
			log.Warnf("Failed to parse user ID %s: %v", member.User.ID, err)
			continue
		}

		// Always add username
		r.addToNameCache(guildID, UsernameType, member.User.Username, userID)
		stats[UsernameType]++

		// Add display name if different from username
		if displayName := member.DisplayName(); displayName != "" && displayName != member.User.Username {
			r.addToNameCache(guildID, DisplayNameType, displayName, userID)
			stats[DisplayNameType]++
		}

		// Add server nickname if exists
		if member.Nick != "" {
			r.addToNameCache(guildID, NicknameType, member.Nick, userID)
			stats[NicknameType]++
		}
	}

	log.Infof("Built caches for guild %d: %d server nicknames, %d global display names, %d usernames",
		guildID, stats[NicknameType], stats[DisplayNameType], stats[UsernameType])
}

// addToNameCache adds a user ID to a specific name cache (must be called with lock held)
func (r *UserResolverImpl) addToNameCache(guildID int64, nameType NameType, name string, userID int64) {
	r.nameCache[guildID][nameType][name] = append(r.nameCache[guildID][nameType][name], userID)
}

// logCacheContents logs the complete contents of all name caches
func (r *UserResolverImpl) logCacheContents(guildID int64) {
	r.nameCacheMutex.RLock()
	defer r.nameCacheMutex.RUnlock()

	log.Infof("=== Cache contents for guild %d ===", guildID)

	nameTypes := []NameType{NicknameType, DisplayNameType, UsernameType}
	for _, nameType := range nameTypes {
		if guildCache, exists := r.nameCache[guildID]; exists {
			if typeCache, exists := guildCache[nameType]; exists && len(typeCache) > 0 {
				log.Infof("%s cache (%d entries):", nameType, len(typeCache))
				for name, userIDs := range typeCache {
					log.Infof("  '%s' -> %v", name, userIDs)
				}
			} else {
				log.Infof("%s cache: empty", nameType)
			}
		}
	}

	log.Infof("=== End cache contents ===")
}

// InvalidateCache removes cached members for a specific guild
func (r *UserResolverImpl) InvalidateCache(guildID int64) {
	// Clear member cache
	r.cacheMutex.Lock()
	delete(r.memberCache, guildID)
	delete(r.cacheExpiry, guildID)
	r.cacheMutex.Unlock()

	// Clear name cache
	r.nameCacheMutex.Lock()
	delete(r.nameCache, guildID)
	r.nameCacheMutex.Unlock()
}

// InvalidateAllCache removes all cached members
func (r *UserResolverImpl) InvalidateAllCache() {
	// Clear member cache
	r.cacheMutex.Lock()
	r.memberCache = make(map[int64][]*discordgo.Member)
	r.cacheExpiry = make(map[int64]time.Time)
	r.cacheMutex.Unlock()

	// Clear name cache
	r.nameCacheMutex.Lock()
	r.nameCache = make(map[int64]map[NameType]map[string][]int64)
	r.nameCacheMutex.Unlock()
}