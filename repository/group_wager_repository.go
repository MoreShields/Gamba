package repository

import (
	"context"
	"fmt"

	"gambler/database"
	"gambler/models"
	"gambler/service"
	"github.com/jackc/pgx/v5"
)

// GroupWagerRepository implements all group wager related data access
type GroupWagerRepository struct {
	q queryable
}

// NewGroupWagerRepository creates a new consolidated group wager repository
func NewGroupWagerRepository(db *database.DB) *GroupWagerRepository {
	return &GroupWagerRepository{q: db.Pool}
}

// newGroupWagerRepositoryWithTx creates a new group wager repository with a transaction
func newGroupWagerRepositoryWithTx(tx queryable) service.GroupWagerRepository {
	return &GroupWagerRepository{q: tx}
}

// CreateWithOptions creates a new group wager with its options atomically
func (r *GroupWagerRepository) CreateWithOptions(ctx context.Context, wager *models.GroupWager, options []*models.GroupWagerOption) error {
	// Create the group wager
	query := `
		INSERT INTO group_wagers (
			creator_discord_id, condition, state, total_pot, 
			min_participants, message_id, channel_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at
	`

	err := r.q.QueryRow(ctx, query,
		wager.CreatorDiscordID,
		wager.Condition,
		wager.State,
		wager.TotalPot,
		wager.MinParticipants,
		wager.MessageID,
		wager.ChannelID,
	).Scan(&wager.ID, &wager.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create group wager: %w", err)
	}

	// Create options if provided
	if len(options) > 0 {
		optionQuery := `
			INSERT INTO group_wager_options (
				group_wager_id, option_text, option_order, total_amount
			)
			VALUES
		`

		var args []interface{}
		for i, option := range options {
			if i > 0 {
				optionQuery += ","
			}
			paramIndex := i * 4
			optionQuery += fmt.Sprintf(" ($%d, $%d, $%d, $%d)",
				paramIndex+1, paramIndex+2, paramIndex+3, paramIndex+4)

			args = append(args,
				wager.ID, // Use the newly created wager ID
				option.OptionText,
				option.OptionOrder,
				option.TotalAmount,
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
func (r *GroupWagerRepository) GetDetailByID(ctx context.Context, id int64) (*models.GroupWagerDetail, error) {
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

	return &models.GroupWagerDetail{
		Wager:        wager,
		Options:      options,
		Participants: participants,
	}, nil
}

// GetDetailByMessageID retrieves a group wager detail by its Discord message ID
func (r *GroupWagerRepository) GetDetailByMessageID(ctx context.Context, messageID int64) (*models.GroupWagerDetail, error) {
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
func (r *GroupWagerRepository) GetByID(ctx context.Context, id int64) (*models.GroupWager, error) {
	query := `
		SELECT 
			id, creator_discord_id, condition, state, resolver_discord_id,
			winning_option_id, total_pot, min_participants, message_id, 
			channel_id, created_at, resolved_at
		FROM group_wagers
		WHERE id = $1
	`

	var wager models.GroupWager
	err := r.q.QueryRow(ctx, query, id).Scan(
		&wager.ID,
		&wager.CreatorDiscordID,
		&wager.Condition,
		&wager.State,
		&wager.ResolverDiscordID,
		&wager.WinningOptionID,
		&wager.TotalPot,
		&wager.MinParticipants,
		&wager.MessageID,
		&wager.ChannelID,
		&wager.CreatedAt,
		&wager.ResolvedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get group wager: %w", err)
	}

	return &wager, nil
}

// GetByMessageID retrieves a group wager by its Discord message ID
func (r *GroupWagerRepository) GetByMessageID(ctx context.Context, messageID int64) (*models.GroupWager, error) {
	query := `
		SELECT 
			id, creator_discord_id, condition, state, resolver_discord_id,
			winning_option_id, total_pot, min_participants, message_id, 
			channel_id, created_at, resolved_at
		FROM group_wagers
		WHERE message_id = $1
	`

	var wager models.GroupWager
	err := r.q.QueryRow(ctx, query, messageID).Scan(
		&wager.ID,
		&wager.CreatorDiscordID,
		&wager.Condition,
		&wager.State,
		&wager.ResolverDiscordID,
		&wager.WinningOptionID,
		&wager.TotalPot,
		&wager.MinParticipants,
		&wager.MessageID,
		&wager.ChannelID,
		&wager.CreatedAt,
		&wager.ResolvedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get group wager by message ID: %w", err)
	}

	return &wager, nil
}

// Update updates a group wager's state and related fields
func (r *GroupWagerRepository) Update(ctx context.Context, wager *models.GroupWager) error {
	query := `
		UPDATE group_wagers
		SET state = $2, resolver_discord_id = $3, winning_option_id = $4,
		    total_pot = $5, resolved_at = $6, message_id = $7, channel_id = $8
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
func (r *GroupWagerRepository) GetActiveByUser(ctx context.Context, discordID int64) ([]*models.GroupWager, error) {
	query := `
		SELECT DISTINCT
			gw.id, gw.creator_discord_id, gw.condition, gw.state, gw.resolver_discord_id,
			gw.winning_option_id, gw.total_pot, gw.min_participants, gw.message_id, 
			gw.channel_id, gw.created_at, gw.resolved_at
		FROM group_wagers gw
		JOIN group_wager_participants gwp ON gwp.group_wager_id = gw.id
		WHERE gwp.discord_id = $1 AND gw.state = 'active'
		ORDER BY gw.created_at DESC
	`

	rows, err := r.q.Query(ctx, query, discordID)
	if err != nil {
		return nil, fmt.Errorf("failed to query active group wagers: %w", err)
	}
	defer rows.Close()

	var wagers []*models.GroupWager
	for rows.Next() {
		var wager models.GroupWager
		err := rows.Scan(
			&wager.ID,
			&wager.CreatorDiscordID,
			&wager.Condition,
			&wager.State,
			&wager.ResolverDiscordID,
			&wager.WinningOptionID,
			&wager.TotalPot,
			&wager.MinParticipants,
			&wager.MessageID,
			&wager.ChannelID,
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
func (r *GroupWagerRepository) GetAll(ctx context.Context, state *models.GroupWagerState) ([]*models.GroupWager, error) {
	var query string
	var args []interface{}

	if state != nil {
		query = `
			SELECT 
				id, creator_discord_id, condition, state, resolver_discord_id,
				winning_option_id, total_pot, min_participants, message_id, 
				channel_id, created_at, resolved_at
			FROM group_wagers
			WHERE state = $1
			ORDER BY created_at DESC
		`
		args = append(args, *state)
	} else {
		query = `
			SELECT 
				id, creator_discord_id, condition, state, resolver_discord_id,
				winning_option_id, total_pot, min_participants, message_id, 
				channel_id, created_at, resolved_at
			FROM group_wagers
			ORDER BY created_at DESC
		`
	}

	rows, err := r.q.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query group wagers: %w", err)
	}
	defer rows.Close()

	var wagers []*models.GroupWager
	for rows.Next() {
		var wager models.GroupWager
		err := rows.Scan(
			&wager.ID,
			&wager.CreatorDiscordID,
			&wager.Condition,
			&wager.State,
			&wager.ResolverDiscordID,
			&wager.WinningOptionID,
			&wager.TotalPot,
			&wager.MinParticipants,
			&wager.MessageID,
			&wager.ChannelID,
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
func (r *GroupWagerRepository) SaveParticipant(ctx context.Context, participant *models.GroupWagerParticipant) error {
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
func (r *GroupWagerRepository) GetParticipant(ctx context.Context, groupWagerID int64, discordID int64) (*models.GroupWagerParticipant, error) {
	query := `
		SELECT 
			id, group_wager_id, discord_id, option_id, amount,
			payout_amount, balance_history_id, created_at, updated_at
		FROM group_wager_participants
		WHERE group_wager_id = $1 AND discord_id = $2
	`

	var participant models.GroupWagerParticipant
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
func (r *GroupWagerRepository) GetActiveParticipationsByUser(ctx context.Context, discordID int64) ([]*models.GroupWagerParticipant, error) {
	query := `
		SELECT 
			gwp.id, gwp.group_wager_id, gwp.discord_id, gwp.option_id, gwp.amount,
			gwp.payout_amount, gwp.balance_history_id, gwp.created_at, gwp.updated_at
		FROM group_wager_participants gwp
		JOIN group_wagers gw ON gw.id = gwp.group_wager_id
		WHERE gwp.discord_id = $1 AND gw.state = 'active'
		ORDER BY gwp.created_at DESC
	`

	rows, err := r.q.Query(ctx, query, discordID)
	if err != nil {
		return nil, fmt.Errorf("failed to query active group wager participants: %w", err)
	}
	defer rows.Close()

	var participants []*models.GroupWagerParticipant
	for rows.Next() {
		var participant models.GroupWagerParticipant
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
func (r *GroupWagerRepository) UpdateParticipantPayouts(ctx context.Context, participants []*models.GroupWagerParticipant) error {
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

// Internal helper methods

// getOptionsByGroupWager returns all options for a group wager
func (r *GroupWagerRepository) getOptionsByGroupWager(ctx context.Context, groupWagerID int64) ([]*models.GroupWagerOption, error) {
	query := `
		SELECT 
			id, group_wager_id, option_text, option_order, 
			total_amount, created_at
		FROM group_wager_options
		WHERE group_wager_id = $1
		ORDER BY option_order
	`

	rows, err := r.q.Query(ctx, query, groupWagerID)
	if err != nil {
		return nil, fmt.Errorf("failed to query group wager options: %w", err)
	}
	defer rows.Close()

	var options []*models.GroupWagerOption
	for rows.Next() {
		var option models.GroupWagerOption
		err := rows.Scan(
			&option.ID,
			&option.GroupWagerID,
			&option.OptionText,
			&option.OptionOrder,
			&option.TotalAmount,
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
func (r *GroupWagerRepository) getParticipantsByGroupWager(ctx context.Context, groupWagerID int64) ([]*models.GroupWagerParticipant, error) {
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

	var participants []*models.GroupWagerParticipant
	for rows.Next() {
		var participant models.GroupWagerParticipant
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