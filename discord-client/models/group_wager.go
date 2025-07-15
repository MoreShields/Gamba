package models

import (
	"time"
)

// GroupWagerState represents the state of a group wager
type GroupWagerState string

const (
	GroupWagerStateActive            GroupWagerState = "active"
	GroupWagerStatePendingResolution GroupWagerState = "pending_resolution"
	GroupWagerStateResolved          GroupWagerState = "resolved"
	GroupWagerStateCancelled         GroupWagerState = "cancelled"
)

// GroupWager represents a multi-participant wager with multiple outcome options
type GroupWager struct {
	ID                  int64           `db:"id"`
	CreatorDiscordID    int64           `db:"creator_discord_id"`
	GuildID             int64           `db:"guild_id"`
	Condition           string          `db:"condition"`
	State               GroupWagerState `db:"state"`
	ResolverDiscordID   *int64          `db:"resolver_discord_id"`
	WinningOptionID     *int64          `db:"winning_option_id"`
	TotalPot            int64           `db:"total_pot"`
	MinParticipants     int             `db:"min_participants"`
	VotingPeriodMinutes int             `db:"voting_period_minutes"`
	VotingStartsAt      *time.Time      `db:"voting_starts_at"`
	VotingEndsAt        *time.Time      `db:"voting_ends_at"`
	MessageID           int64           `db:"message_id"`
	ChannelID           int64           `db:"channel_id"`
	CreatedAt           time.Time       `db:"created_at"`
	ResolvedAt          *time.Time      `db:"resolved_at"`
}

// GroupWagerOption represents a possible outcome for a group wager
type GroupWagerOption struct {
	ID           int64     `db:"id"`
	GroupWagerID int64     `db:"group_wager_id"`
	OptionText   string    `db:"option_text"`
	OptionOrder  int16     `db:"option_order"`
	TotalAmount  int64     `db:"total_amount"`
	CreatedAt    time.Time `db:"created_at"`
}

// GroupWagerParticipant represents a user's participation in a group wager
type GroupWagerParticipant struct {
	ID               int64     `db:"id"`
	GroupWagerID     int64     `db:"group_wager_id"`
	DiscordID        int64     `db:"discord_id"`
	OptionID         int64     `db:"option_id"`
	Amount           int64     `db:"amount"`
	PayoutAmount     *int64    `db:"payout_amount"`
	BalanceHistoryID *int64    `db:"balance_history_id"`
	CreatedAt        time.Time `db:"created_at"`
	UpdatedAt        time.Time `db:"updated_at"`
}

// GroupWagerDetail combines a group wager with its options and participants
type GroupWagerDetail struct {
	Wager        *GroupWager
	Options      []*GroupWagerOption
	Participants []*GroupWagerParticipant
}

// GroupWagerResult represents the outcome of a group wager resolution
type GroupWagerResult struct {
	GroupWager    *GroupWager
	WinningOption *GroupWagerOption
	Winners       []*GroupWagerParticipant
	Losers        []*GroupWagerParticipant
	TotalPot      int64
	PayoutDetails map[int64]int64 // Discord ID -> payout amount
}

// IsActive checks if the group wager is in an active state
func (gw *GroupWager) IsActive() bool {
	return gw.State == GroupWagerStateActive
}

// IsPendingResolution checks if the group wager is awaiting resolution
func (gw *GroupWager) IsPendingResolution() bool {
	return gw.State == GroupWagerStatePendingResolution
}

// IsResolved checks if the group wager has been resolved
func (gw *GroupWager) IsResolved() bool {
	return gw.State == GroupWagerStateResolved
}

// IsVotingPeriodActive checks if voting period is currently active
func (gw *GroupWager) IsVotingPeriodActive() bool {
	if gw.State != GroupWagerStateActive || gw.VotingEndsAt == nil {
		return false
	}
	return time.Now().Before(*gw.VotingEndsAt)
}

// IsVotingPeriodExpired checks if voting period has expired
func (gw *GroupWager) IsVotingPeriodExpired() bool {
	if gw.State != GroupWagerStateActive || gw.VotingEndsAt == nil {
		return false
	}
	return time.Now().After(*gw.VotingEndsAt)
}

// CanAcceptBets checks if the group wager can still accept bets
func (gw *GroupWager) CanAcceptBets() bool {
	return gw.IsActive() && gw.IsVotingPeriodActive()
}

// HasMinimumParticipants checks if the wager has enough participants
func (gw *GroupWager) HasMinimumParticipants(participantCount int) bool {
	return participantCount >= gw.MinParticipants
}

// CalculateMultiplier calculates the potential payout multiplier for an option
func (o *GroupWagerOption) CalculateMultiplier(totalPot int64) float64 {
	if o.TotalAmount == 0 {
		return 0
	}
	return float64(totalPot) / float64(o.TotalAmount)
}

// CalculatePayout calculates the payout for a participant based on their contribution
func (p *GroupWagerParticipant) CalculatePayout(winningOptionTotal int64, totalPot int64) int64 {
	if winningOptionTotal == 0 {
		return 0
	}
	return (p.Amount * totalPot) / winningOptionTotal
}

// GetParticipantsByOption groups participants by their chosen option
func (gwd *GroupWagerDetail) GetParticipantsByOption() map[int64][]*GroupWagerParticipant {
	result := make(map[int64][]*GroupWagerParticipant)
	for _, participant := range gwd.Participants {
		result[participant.OptionID] = append(result[participant.OptionID], participant)
	}
	return result
}

// HasMultipleOptionsWithParticipants checks if at least 2 options have participants
func (gwd *GroupWagerDetail) HasMultipleOptionsWithParticipants() bool {
	participantsByOption := gwd.GetParticipantsByOption()
	optionsWithParticipants := 0

	for _, participants := range participantsByOption {
		if len(participants) > 0 {
			optionsWithParticipants++
		}
	}

	return optionsWithParticipants >= 2
}
