package repository

import (
	"context"
	"fmt"

	"gambler/discord-client/database"
	"gambler/discord-client/domain/entities"
	"gambler/discord-client/domain/interfaces"

	"github.com/jackc/pgx/v5"
)

// GroupWagerRepository implements all group wager related data access
type GroupWagerRepository struct {
	q       Queryable
	guildID int64
}

// NewGroupWagerRepository creates a new consolidated group wager repository
func NewGroupWagerRepository(db *database.DB) *GroupWagerRepository {
	return &GroupWagerRepository{q: db.Pool}
}

// newGroupWagerRepository creates a new group wager repository with a transaction and guild scope
func NewGroupWagerRepositoryScoped(tx Queryable, guildID int64) interfaces.GroupWagerRepository {
	return &GroupWagerRepository{
		q:       tx,
		guildID: guildID,
	}
}

// CreateWithOptions creates a new group wager with its options atomically
func (r *GroupWagerRepository) CreateWithOptions(ctx context.Context, wager *entities.GroupWager, options []*entities.GroupWagerOption) error {
	// Create the group wager
	query := `
		INSERT INTO group_wagers (
			creator_discord_id, guild_id, condition, state, wager_type, total_pot, 
			min_participants, message_id, channel_id, voting_period_minutes,
			voting_starts_at, voting_ends_at, external_id, external_system
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id, created_at
	`

	err := r.q.QueryRow(ctx, query,
		wager.CreatorDiscordID,
		r.guildID, // Use repository's guild scope
		wager.Condition,
		wager.State,
		wager.WagerType,
		wager.TotalPot,
		wager.MinParticipants,
		wager.MessageID,
		wager.ChannelID,
		wager.VotingPeriodMinutes,
		wager.VotingStartsAt,
		wager.VotingEndsAt,
		wager.GetExternalID(),
		wager.GetExternalSystem(),
	).Scan(&wager.ID, &wager.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create group wager: %w", err)
	}

	// Create options if provided
	if len(options) > 0 {
		optionQuery := `
			INSERT INTO group_wager_options (
				group_wager_id, option_text, option_order, total_amount, odds_multiplier
			)
			VALUES
		`

		var args []interface{}
		for i, option := range options {
			if i > 0 {
				optionQuery += ","
			}
			paramIndex := i * 5
			optionQuery += fmt.Sprintf(" ($%d, $%d, $%d, $%d, $%d)",
				paramIndex+1, paramIndex+2, paramIndex+3, paramIndex+4, paramIndex+5)

			args = append(args,
				wager.ID, // Use the newly created wager ID
				option.OptionText,
				option.OptionOrder,
				option.TotalAmount,
				option.OddsMultiplier,
			)
		}

		optionQuery += " RETURNING id, created_at"

		rows, err := r.q.Query(ctx, optionQuery, args...)
		if err != nil {
			return fmt.Errorf("failed to create group wager options: %w", err)
		}
		defer rows.Close()

		// Update the IDs and timestamps on the option objects
		i := 0
		for rows.Next() {
			if i >= len(options) {
				return fmt.Errorf("unexpected number of rows returned")
			}
			err := rows.Scan(&options[i].ID, &options[i].CreatedAt)
			if err != nil {
				return fmt.Errorf("failed to scan option ID: %w", err)
			}
			options[i].GroupWagerID = wager.ID
			i++
		}
	}

	return nil
}

// GetDetailByID retrieves a group wager with all its options and participants
func (r *GroupWagerRepository) GetDetailByID(ctx context.Context, id int64) (*entities.GroupWagerDetail, error) {
	// Get the wager
	wager, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if wager == nil {
		return nil, nil
	}

	// Get options
	options, err := r.getOptionsByGroupWager(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get options: %w", err)
	}

	// Get participants
	participants, err := r.getParticipantsByGroupWager(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get participants: %w", err)
	}

	return &entities.GroupWagerDetail{
		Wager:        wager,
		Options:      options,
		Participants: participants,
	}, nil
}

// GetDetailByMessageID retrieves a group wager detail by its Discord message ID
func (r *GroupWagerRepository) GetDetailByMessageID(ctx context.Context, messageID int64) (*entities.GroupWagerDetail, error) {
	wager, err := r.GetByMessageID(ctx, messageID)
	if err != nil {
		return nil, err
	}
	if wager == nil {
		return nil, nil
	}

	return r.GetDetailByID(ctx, wager.ID)
}

// GetByID retrieves a group wager by its ID
func (r *GroupWagerRepository) GetByID(ctx context.Context, id int64) (*entities.GroupWager, error) {
	query := `
		SELECT 
			id, creator_discord_id, guild_id, condition, state, wager_type, resolver_discord_id,
			winning_option_id, total_pot, min_participants, message_id, 
			channel_id, voting_period_minutes, voting_starts_at, voting_ends_at,
			created_at, resolved_at, external_id, external_system
		FROM group_wagers
		WHERE id = $1
	`

	var wager entities.GroupWager
	var externalID, externalSystem *string

	err := r.q.QueryRow(ctx, query, id).Scan(
		&wager.ID,
		&wager.CreatorDiscordID,
		&wager.GuildID,
		&wager.Condition,
		&wager.State,
		&wager.WagerType,
		&wager.ResolverDiscordID,
		&wager.WinningOptionID,
		&wager.TotalPot,
		&wager.MinParticipants,
		&wager.MessageID,
		&wager.ChannelID,
		&wager.VotingPeriodMinutes,
		&wager.VotingStartsAt,
		&wager.VotingEndsAt,
		&wager.CreatedAt,
		&wager.ResolvedAt,
		&externalID,
		&externalSystem,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get group wager: %w", err)
	}

	// Set the external reference if both fields are present
	if externalID != nil && externalSystem != nil {
		wager.ExternalRef = &entities.ExternalReference{
			System: entities.ExternalSystem(*externalSystem),
			ID:     *externalID,
		}
	}

	return &wager, nil
}

// GetByMessageID retrieves a group wager by its Discord message ID
func (r *GroupWagerRepository) GetByMessageID(ctx context.Context, messageID int64) (*entities.GroupWager, error) {
	query := `
		SELECT 
			id, creator_discord_id, guild_id, condition, state, wager_type, resolver_discord_id,
			winning_option_id, total_pot, min_participants, message_id, 
			channel_id, voting_period_minutes, voting_starts_at, voting_ends_at,
			created_at, resolved_at, external_id, external_system
		FROM group_wagers
		WHERE message_id = $1
	`

	var wager entities.GroupWager
	var externalID, externalSystem *string

	err := r.q.QueryRow(ctx, query, messageID).Scan(
		&wager.ID,
		&wager.CreatorDiscordID,
		&wager.GuildID,
		&wager.Condition,
		&wager.State,
		&wager.WagerType,
		&wager.ResolverDiscordID,
		&wager.WinningOptionID,
		&wager.TotalPot,
		&wager.MinParticipants,
		&wager.MessageID,
		&wager.ChannelID,
		&wager.VotingPeriodMinutes,
		&wager.VotingStartsAt,
		&wager.VotingEndsAt,
		&wager.CreatedAt,
		&wager.ResolvedAt,
		&externalID,
		&externalSystem,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get group wager by message ID: %w", err)
	}

	// Set the external reference if both fields are present
	if externalID != nil && externalSystem != nil {
		wager.ExternalRef = &entities.ExternalReference{
			System: entities.ExternalSystem(*externalSystem),
			ID:     *externalID,
		}
	}

	return &wager, nil
}

// GetByExternalReference retrieves a group wager by its external reference
func (r *GroupWagerRepository) GetByExternalReference(ctx context.Context, ref entities.ExternalReference) (*entities.GroupWager, error) {
	query := `
		SELECT 
			id, creator_discord_id, guild_id, condition, state, wager_type, resolver_discord_id,
			winning_option_id, total_pot, min_participants, message_id, 
			channel_id, voting_period_minutes, voting_starts_at, voting_ends_at,
			created_at, resolved_at, external_id, external_system
		FROM group_wagers
		WHERE external_id = $1 AND external_system = $2 AND guild_id = $3
	`

	var wager entities.GroupWager
	var externalID, externalSystem *string

	err := r.q.QueryRow(ctx, query, ref.ID, string(ref.System), r.guildID).Scan(
		&wager.ID,
		&wager.CreatorDiscordID,
		&wager.GuildID,
		&wager.Condition,
		&wager.State,
		&wager.WagerType,
		&wager.ResolverDiscordID,
		&wager.WinningOptionID,
		&wager.TotalPot,
		&wager.MinParticipants,
		&wager.MessageID,
		&wager.ChannelID,
		&wager.VotingPeriodMinutes,
		&wager.VotingStartsAt,
		&wager.VotingEndsAt,
		&wager.CreatedAt,
		&wager.ResolvedAt,
		&externalID,
		&externalSystem,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get group wager by external reference: %w", err)
	}

	// Set the external reference if both fields are present
	if externalID != nil && externalSystem != nil {
		wager.ExternalRef = &entities.ExternalReference{
			System: entities.ExternalSystem(*externalSystem),
			ID:     *externalID,
		}
	}

	return &wager, nil
}

// Update updates a group wager's state and related fields
func (r *GroupWagerRepository) Update(ctx context.Context, wager *entities.GroupWager) error {
	query := `
		UPDATE group_wagers
		SET state = $2, resolver_discord_id = $3, winning_option_id = $4,
		    total_pot = $5, resolved_at = $6, message_id = $7, channel_id = $8,
		    voting_period_minutes = $9, voting_starts_at = $10, voting_ends_at = $11,
		    external_id = $12, external_system = $13
		WHERE id = $1
	`

	result, err := r.q.Exec(ctx, query,
		wager.ID,
		wager.State,
		wager.ResolverDiscordID,
		wager.WinningOptionID,
		wager.TotalPot,
		wager.ResolvedAt,
		wager.MessageID,
		wager.ChannelID,
		wager.VotingPeriodMinutes,
		wager.VotingStartsAt,
		wager.VotingEndsAt,
		wager.GetExternalID(),
		wager.GetExternalSystem(),
	)

	if err != nil {
		return fmt.Errorf("failed to update group wager: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("group wager not found")
	}

	return nil
}

// GetActiveByUser returns all active group wagers where the user is participating
func (r *GroupWagerRepository) GetActiveByUser(ctx context.Context, discordID int64) ([]*entities.GroupWager, error) {
	query := `
		SELECT DISTINCT
			gw.id, gw.creator_discord_id, gw.guild_id, gw.condition, gw.state, gw.wager_type, gw.resolver_discord_id,
			gw.winning_option_id, gw.total_pot, gw.min_participants, gw.message_id, 
			gw.channel_id, gw.voting_period_minutes, gw.voting_starts_at, gw.voting_ends_at,
			gw.created_at, gw.resolved_at
		FROM group_wagers gw
		JOIN group_wager_participants gwp ON gwp.group_wager_id = gw.id
		WHERE gwp.discord_id = $1 AND gw.state = 'active' AND gw.guild_id = $2
		ORDER BY gw.created_at DESC
	`

	rows, err := r.q.Query(ctx, query, discordID, r.guildID)
	if err != nil {
		return nil, fmt.Errorf("failed to query active group wagers: %w", err)
	}
	defer rows.Close()

	var wagers []*entities.GroupWager
	for rows.Next() {
		var wager entities.GroupWager
		err := rows.Scan(
			&wager.ID,
			&wager.CreatorDiscordID,
			&wager.GuildID,
			&wager.Condition,
			&wager.State,
			&wager.WagerType,
			&wager.ResolverDiscordID,
			&wager.WinningOptionID,
			&wager.TotalPot,
			&wager.MinParticipants,
			&wager.MessageID,
			&wager.ChannelID,
			&wager.VotingPeriodMinutes,
			&wager.VotingStartsAt,
			&wager.VotingEndsAt,
			&wager.CreatedAt,
			&wager.ResolvedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan group wager: %w", err)
		}
		wagers = append(wagers, &wager)
	}

	return wagers, nil
}

// GetAll returns all group wagers with optional state filter
func (r *GroupWagerRepository) GetAll(ctx context.Context, state *entities.GroupWagerState) ([]*entities.GroupWager, error) {
	var query string
	var args []interface{}

	if state != nil {
		query = `
			SELECT 
				id, creator_discord_id, guild_id, condition, state, wager_type, resolver_discord_id,
				winning_option_id, total_pot, min_participants, message_id, 
				channel_id, voting_period_minutes, voting_starts_at, voting_ends_at,
				created_at, resolved_at
			FROM group_wagers
			WHERE state = $1 AND guild_id = $2
			ORDER BY created_at DESC
		`
		args = append(args, *state, r.guildID)
	} else {
		query = `
			SELECT 
				id, creator_discord_id, guild_id, condition, state, wager_type, resolver_discord_id,
				winning_option_id, total_pot, min_participants, message_id, 
				channel_id, voting_period_minutes, voting_starts_at, voting_ends_at,
				created_at, resolved_at
			FROM group_wagers
			WHERE guild_id = $1
			ORDER BY created_at DESC
		`
		args = append(args, r.guildID)
	}

	rows, err := r.q.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query group wagers: %w", err)
	}
	defer rows.Close()

	var wagers []*entities.GroupWager
	for rows.Next() {
		var wager entities.GroupWager
		err := rows.Scan(
			&wager.ID,
			&wager.CreatorDiscordID,
			&wager.GuildID,
			&wager.Condition,
			&wager.State,
			&wager.WagerType,
			&wager.ResolverDiscordID,
			&wager.WinningOptionID,
			&wager.TotalPot,
			&wager.MinParticipants,
			&wager.MessageID,
			&wager.ChannelID,
			&wager.VotingPeriodMinutes,
			&wager.VotingStartsAt,
			&wager.VotingEndsAt,
			&wager.CreatedAt,
			&wager.ResolvedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan group wager: %w", err)
		}
		wagers = append(wagers, &wager)
	}

	return wagers, nil
}

// Participant operations

// SaveParticipant creates or updates a participant entry based on whether ID is set
func (r *GroupWagerRepository) SaveParticipant(ctx context.Context, participant *entities.GroupWagerParticipant) error {
	if participant.ID > 0 {
		// Update existing participant
		query := `
			UPDATE group_wager_participants
			SET option_id = $2, amount = $3, updated_at = CURRENT_TIMESTAMP
			WHERE id = $1
			RETURNING updated_at
		`

		err := r.q.QueryRow(ctx, query,
			participant.ID,
			participant.OptionID,
			participant.Amount,
		).Scan(&participant.UpdatedAt)

		if err != nil {
			return fmt.Errorf("failed to update group wager participant: %w", err)
		}
	} else {
		// Create new participant
		query := `
			INSERT INTO group_wager_participants (
				group_wager_id, discord_id, option_id, amount
			)
			VALUES ($1, $2, $3, $4)
			RETURNING id, created_at, updated_at
		`

		err := r.q.QueryRow(ctx, query,
			participant.GroupWagerID,
			participant.DiscordID,
			participant.OptionID,
			participant.Amount,
		).Scan(&participant.ID, &participant.CreatedAt, &participant.UpdatedAt)

		if err != nil {
			return fmt.Errorf("failed to create group wager participant: %w", err)
		}
	}

	return nil
}

// GetParticipant returns a participant entry for a specific user in a group wager
func (r *GroupWagerRepository) GetParticipant(ctx context.Context, groupWagerID int64, discordID int64) (*entities.GroupWagerParticipant, error) {
	query := `
		SELECT 
			id, group_wager_id, discord_id, option_id, amount,
			payout_amount, balance_history_id, created_at, updated_at
		FROM group_wager_participants
		WHERE group_wager_id = $1 AND discord_id = $2
	`

	var participant entities.GroupWagerParticipant
	err := r.q.QueryRow(ctx, query, groupWagerID, discordID).Scan(
		&participant.ID,
		&participant.GroupWagerID,
		&participant.DiscordID,
		&participant.OptionID,
		&participant.Amount,
		&participant.PayoutAmount,
		&participant.BalanceHistoryID,
		&participant.CreatedAt,
		&participant.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get group wager participant: %w", err)
	}

	return &participant, nil
}

// GetActiveParticipationsByUser returns all active group wager participations for a user
func (r *GroupWagerRepository) GetActiveParticipationsByUser(ctx context.Context, discordID int64) ([]*entities.GroupWagerParticipant, error) {
	query := `
		SELECT 
			gwp.id, gwp.group_wager_id, gwp.discord_id, gwp.option_id, gwp.amount,
			gwp.payout_amount, gwp.balance_history_id, gwp.created_at, gwp.updated_at
		FROM group_wager_participants gwp
		JOIN group_wagers gw ON gw.id = gwp.group_wager_id
		WHERE gwp.discord_id = $1 AND gw.state = 'active' AND gw.guild_id = $2
		ORDER BY gwp.created_at DESC
	`

	rows, err := r.q.Query(ctx, query, discordID, r.guildID)
	if err != nil {
		return nil, fmt.Errorf("failed to query active group wager participants: %w", err)
	}
	defer rows.Close()

	var participants []*entities.GroupWagerParticipant
	for rows.Next() {
		var participant entities.GroupWagerParticipant
		err := rows.Scan(
			&participant.ID,
			&participant.GroupWagerID,
			&participant.DiscordID,
			&participant.OptionID,
			&participant.Amount,
			&participant.PayoutAmount,
			&participant.BalanceHistoryID,
			&participant.CreatedAt,
			&participant.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan group wager participant: %w", err)
		}
		participants = append(participants, &participant)
	}

	return participants, nil
}


// UpdateParticipantPayouts updates payout amounts and balance history IDs for multiple participants
func (r *GroupWagerRepository) UpdateParticipantPayouts(ctx context.Context, participants []*entities.GroupWagerParticipant) error {
	if len(participants) == 0 {
		return nil
	}

	// Use a batch update approach
	for _, participant := range participants {
		query := `
			UPDATE group_wager_participants
			SET payout_amount = $2, balance_history_id = $3
			WHERE id = $1
		`

		_, err := r.q.Exec(ctx, query,
			participant.ID,
			participant.PayoutAmount,
			participant.BalanceHistoryID,
		)
		if err != nil {
			return fmt.Errorf("failed to update participant payout: %w", err)
		}
	}

	return nil
}

// Option operations

// UpdateOptionTotal updates an option's total amount
func (r *GroupWagerRepository) UpdateOptionTotal(ctx context.Context, optionID int64, totalAmount int64) error {
	query := `
		UPDATE group_wager_options
		SET total_amount = $2
		WHERE id = $1
	`

	result, err := r.q.Exec(ctx, query, optionID, totalAmount)
	if err != nil {
		return fmt.Errorf("failed to update group wager option: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("group wager option not found")
	}

	return nil
}

// UpdateOptionOdds updates an option's odds multiplier
func (r *GroupWagerRepository) UpdateOptionOdds(ctx context.Context, optionID int64, oddsMultiplier float64) error {
	query := `
		UPDATE group_wager_options
		SET odds_multiplier = $2
		WHERE id = $1
	`

	result, err := r.q.Exec(ctx, query, optionID, oddsMultiplier)
	if err != nil {
		return fmt.Errorf("failed to update group wager option odds: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("group wager option not found")
	}

	return nil
}

// UpdateAllOptionOdds updates odds multipliers for all options in a group wager
func (r *GroupWagerRepository) UpdateAllOptionOdds(ctx context.Context, groupWagerID int64, oddsMultipliers map[int64]float64) error {
	if len(oddsMultipliers) == 0 {
		return nil
	}

	// Update each option's odds individually
	// This is safe because odds updates are not critical for consistency
	// and the service layer should handle any coordination needed
	for optionID, odds := range oddsMultipliers {
		query := `
			UPDATE group_wager_options
			SET odds_multiplier = $2
			WHERE id = $1 AND group_wager_id = $3
		`

		result, err := r.q.Exec(ctx, query, optionID, odds, groupWagerID)
		if err != nil {
			return fmt.Errorf("failed to update odds for option %d: %w", optionID, err)
		}

		if result.RowsAffected() == 0 {
			return fmt.Errorf("group wager option %d not found in wager %d", optionID, groupWagerID)
		}
	}

	return nil
}

// Internal helper methods

// getOptionsByGroupWager returns all options for a group wager
func (r *GroupWagerRepository) getOptionsByGroupWager(ctx context.Context, groupWagerID int64) ([]*entities.GroupWagerOption, error) {
	query := `
		SELECT 
			id, group_wager_id, option_text, option_order, 
			total_amount, odds_multiplier, created_at
		FROM group_wager_options
		WHERE group_wager_id = $1
		ORDER BY option_order
	`

	rows, err := r.q.Query(ctx, query, groupWagerID)
	if err != nil {
		return nil, fmt.Errorf("failed to query group wager options: %w", err)
	}
	defer rows.Close()

	var options []*entities.GroupWagerOption
	for rows.Next() {
		var option entities.GroupWagerOption
		err := rows.Scan(
			&option.ID,
			&option.GroupWagerID,
			&option.OptionText,
			&option.OptionOrder,
			&option.TotalAmount,
			&option.OddsMultiplier,
			&option.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan group wager option: %w", err)
		}
		options = append(options, &option)
	}

	return options, nil
}

// getParticipantsByGroupWager returns all participants for a group wager
func (r *GroupWagerRepository) getParticipantsByGroupWager(ctx context.Context, groupWagerID int64) ([]*entities.GroupWagerParticipant, error) {
	query := `
		SELECT 
			id, group_wager_id, discord_id, option_id, amount,
			payout_amount, balance_history_id, created_at, updated_at
		FROM group_wager_participants
		WHERE group_wager_id = $1
		ORDER BY created_at
	`

	rows, err := r.q.Query(ctx, query, groupWagerID)
	if err != nil {
		return nil, fmt.Errorf("failed to query group wager participants: %w", err)
	}
	defer rows.Close()

	var participants []*entities.GroupWagerParticipant
	for rows.Next() {
		var participant entities.GroupWagerParticipant
		err := rows.Scan(
			&participant.ID,
			&participant.GroupWagerID,
			&participant.DiscordID,
			&participant.OptionID,
			&participant.Amount,
			&participant.PayoutAmount,
			&participant.BalanceHistoryID,
			&participant.CreatedAt,
			&participant.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan group wager participant: %w", err)
		}
		participants = append(participants, &participant)
	}

	return participants, nil
}

// GetGroupWagerPredictions returns all group wager predictions for resolved wagers in the guild
// Can optionally filter by external system (pass nil for all wagers)
func (r *GroupWagerRepository) GetGroupWagerPredictions(ctx context.Context, externalSystem *entities.ExternalSystem) ([]*entities.GroupWagerPrediction, error) {
	query := `
		SELECT 
			gwp.discord_id,
			gwp.group_wager_id,
			gwp.option_id,
			gwo.option_text,
			gw.winning_option_id,
			gwp.amount,
			gwp.option_id = gw.winning_option_id AS was_correct,
			gwp.payout_amount,
			gw.external_system,
			gw.external_id
		FROM group_wager_participants gwp
		JOIN group_wagers gw ON gw.id = gwp.group_wager_id
		JOIN group_wager_options gwo ON gwo.id = gwp.option_id
		WHERE gw.state = 'resolved' 
		AND gw.winning_option_id IS NOT NULL
		AND gw.guild_id = $1
	`

	args := []interface{}{r.guildID}

	// Add external system filter if specified
	if externalSystem != nil {
		query += " AND gw.external_system = $2"
		args = append(args, *externalSystem)
	}

	query += " ORDER BY gwp.discord_id, gwp.created_at"

	rows, err := r.q.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query group wager predictions: %w", err)
	}
	defer rows.Close()

	var predictions []*entities.GroupWagerPrediction
	for rows.Next() {
		var prediction entities.GroupWagerPrediction
		var externalSystem *string
		var externalID *string

		err := rows.Scan(
			&prediction.DiscordID,
			&prediction.GroupWagerID,
			&prediction.OptionID,
			&prediction.OptionText,
			&prediction.WinningOptionID,
			&prediction.Amount,
			&prediction.WasCorrect,
			&prediction.PayoutAmount,
			&externalSystem,
			&externalID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan group wager prediction: %w", err)
		}

		// Convert external system string to ExternalSystem type
		if externalSystem != nil {
			system := entities.ExternalSystem(*externalSystem)
			prediction.ExternalSystem = &system
		}
		prediction.ExternalID = externalID

		predictions = append(predictions, &prediction)
	}

	return predictions, nil
}

// GetExpiredActiveWagers returns all active group wagers where voting period has expired
func (r *GroupWagerRepository) GetExpiredActiveWagers(ctx context.Context) ([]*entities.GroupWager, error) {
	query := `
		SELECT 
			id, creator_discord_id, guild_id, condition, state, wager_type, resolver_discord_id,
			winning_option_id, total_pot, min_participants, message_id, 
			channel_id, voting_period_minutes, voting_starts_at, voting_ends_at,
			created_at, resolved_at, external_id, external_system
		FROM group_wagers
		WHERE state = 'active' 
		AND voting_ends_at IS NOT NULL 
		AND voting_ends_at < CURRENT_TIMESTAMP
		AND guild_id = $1
		ORDER BY voting_ends_at ASC
	`

	rows, err := r.q.Query(ctx, query, r.guildID)
	if err != nil {
		return nil, fmt.Errorf("failed to query expired active wagers: %w", err)
	}
	defer rows.Close()

	var wagers []*entities.GroupWager
	for rows.Next() {
		var wager entities.GroupWager
		var externalID, externalSystem *string

		err := rows.Scan(
			&wager.ID,
			&wager.CreatorDiscordID,
			&wager.GuildID,
			&wager.Condition,
			&wager.State,
			&wager.WagerType,
			&wager.ResolverDiscordID,
			&wager.WinningOptionID,
			&wager.TotalPot,
			&wager.MinParticipants,
			&wager.MessageID,
			&wager.ChannelID,
			&wager.VotingPeriodMinutes,
			&wager.VotingStartsAt,
			&wager.VotingEndsAt,
			&wager.CreatedAt,
			&wager.ResolvedAt,
			&externalID,
			&externalSystem,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan expired active wager: %w", err)
		}

		// Set the external reference if both fields are present
		if externalID != nil && externalSystem != nil {
			wager.ExternalRef = &entities.ExternalReference{
				System: entities.ExternalSystem(*externalSystem),
				ID:     *externalID,
			}
		}

		wagers = append(wagers, &wager)
	}

	return wagers, nil
}

// GetWagersPendingResolution returns all group wagers in pending_resolution state
func (r *GroupWagerRepository) GetWagersPendingResolution(ctx context.Context) ([]*entities.GroupWager, error) {
	query := `
		SELECT 
			id, creator_discord_id, guild_id, condition, state, wager_type, resolver_discord_id,
			winning_option_id, total_pot, min_participants, message_id, 
			channel_id, voting_period_minutes, voting_starts_at, voting_ends_at,
			created_at, resolved_at
		FROM group_wagers
		WHERE state = 'pending_resolution' AND guild_id = $1
		ORDER BY voting_ends_at ASC
	`

	rows, err := r.q.Query(ctx, query, r.guildID)
	if err != nil {
		return nil, fmt.Errorf("failed to query wagers pending resolution: %w", err)
	}
	defer rows.Close()

	var wagers []*entities.GroupWager
	for rows.Next() {
		var wager entities.GroupWager
		err := rows.Scan(
			&wager.ID,
			&wager.CreatorDiscordID,
			&wager.GuildID,
			&wager.Condition,
			&wager.State,
			&wager.WagerType,
			&wager.ResolverDiscordID,
			&wager.WinningOptionID,
			&wager.TotalPot,
			&wager.MinParticipants,
			&wager.MessageID,
			&wager.ChannelID,
			&wager.VotingPeriodMinutes,
			&wager.VotingStartsAt,
			&wager.VotingEndsAt,
			&wager.CreatedAt,
			&wager.ResolvedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan wager pending resolution: %w", err)
		}
		wagers = append(wagers, &wager)
	}

	return wagers, nil
}

// GetStats returns group wager statistics for a user
func (r *GroupWagerRepository) GetStats(ctx context.Context, discordID int64) (*entities.GroupWagerStats, error) {
	// Get participation stats
	participationQuery := `
		SELECT 
			COUNT(DISTINCT gwp.group_wager_id) as total_group_wagers,
			COUNT(DISTINCT CASE WHEN gw.state = 'resolved' AND gw.winning_option_id = gwp.option_id THEN gw.id END) as total_won,
			COALESCE(SUM(CASE WHEN gw.state = 'resolved' AND gw.winning_option_id = gwp.option_id AND gwp.payout_amount IS NOT NULL THEN gwp.payout_amount ELSE 0 END), 0) as total_won_amount
		FROM group_wager_participants gwp
		JOIN group_wagers gw ON gw.id = gwp.group_wager_id
		WHERE gwp.discord_id = $1 AND gw.guild_id = $2`

	var totalGroupWagers, totalWon int
	var totalWonAmount int64

	err := r.q.QueryRow(ctx, participationQuery, discordID, r.guildID).Scan(
		&totalGroupWagers,
		&totalWon,
		&totalWonAmount,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get participation stats: %w", err)
	}

	// Get creation stats
	creationQuery := `
		SELECT COUNT(*) 
		FROM group_wagers 
		WHERE creator_discord_id = $1 AND guild_id = $2`

	var totalProposed int
	err = r.q.QueryRow(ctx, creationQuery, discordID, r.guildID).Scan(&totalProposed)
	if err != nil {
		return nil, fmt.Errorf("failed to get creation stats: %w", err)
	}

	return &entities.GroupWagerStats{
		TotalGroupWagers: totalGroupWagers,
		TotalProposed:    totalProposed,
		TotalWon:         totalWon,
		TotalWonAmount:   totalWonAmount,
	}, nil
}

// GetGuildsWithActiveWagers returns all guild IDs that have active group wagers
func (r *GroupWagerRepository) GetGuildsWithActiveWagers(ctx context.Context) ([]int64, error) {
	query := `
		SELECT DISTINCT guild_id 
		FROM group_wagers 
		WHERE state = 'active' 
		  AND voting_ends_at IS NOT NULL 
		  AND voting_ends_at < CURRENT_TIMESTAMP
		ORDER BY guild_id
	`

	rows, err := r.q.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query guilds with active wagers: %w", err)
	}
	defer rows.Close()

	var guildIDs []int64
	for rows.Next() {
		var guildID int64
		err := rows.Scan(&guildID)
		if err != nil {
			return nil, fmt.Errorf("failed to scan guild ID: %w", err)
		}
		guildIDs = append(guildIDs, guildID)
	}

	return guildIDs, nil
}
