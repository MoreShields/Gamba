package repository

import (
	"context"
	"fmt"

	"gambler/discord-client/domain/entities"
)

// LotteryTicketRepository implements lottery ticket data access
type LotteryTicketRepository struct {
	q       Queryable
	guildID int64
}

// NewLotteryTicketRepositoryScoped creates a new lottery ticket repository with guild scope
func NewLotteryTicketRepositoryScoped(tx Queryable, guildID int64) *LotteryTicketRepository {
	return &LotteryTicketRepository{
		q:       tx,
		guildID: guildID,
	}
}

// CreateBatch creates multiple lottery tickets in a single batch insert
func (r *LotteryTicketRepository) CreateBatch(ctx context.Context, tickets []*entities.LotteryTicket) error {
	if len(tickets) == 0 {
		return nil
	}

	// Build batch insert query with parameterized values
	query := `
		INSERT INTO lottery_tickets (draw_id, guild_id, discord_id, ticket_number, purchase_price, balance_history_id)
		VALUES `

	values := make([]interface{}, 0, len(tickets)*6)
	for i, ticket := range tickets {
		if i > 0 {
			query += ", "
		}
		paramOffset := i * 6
		query += fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d)",
			paramOffset+1, paramOffset+2, paramOffset+3, paramOffset+4, paramOffset+5, paramOffset+6)
		values = append(values, ticket.DrawID, r.guildID, ticket.DiscordID,
			ticket.TicketNumber, ticket.PurchasePrice, ticket.BalanceHistoryID)
	}
	query += " RETURNING id, purchased_at"

	rows, err := r.q.Query(ctx, query, values...)
	if err != nil {
		return fmt.Errorf("failed to batch create lottery tickets: %w", err)
	}
	defer rows.Close()

	i := 0
	for rows.Next() {
		if err := rows.Scan(&tickets[i].ID, &tickets[i].PurchasedAt); err != nil {
			return fmt.Errorf("failed to scan ticket result: %w", err)
		}
		i++
	}

	return rows.Err()
}

// GetByUserForDraw returns all tickets for a user in a specific draw
func (r *LotteryTicketRepository) GetByUserForDraw(ctx context.Context, drawID, discordID int64) ([]*entities.LotteryTicket, error) {
	query := `
		SELECT id, draw_id, guild_id, discord_id, ticket_number, purchase_price, purchased_at, balance_history_id
		FROM lottery_tickets
		WHERE draw_id = $1 AND discord_id = $2
		ORDER BY ticket_number ASC
	`

	rows, err := r.q.Query(ctx, query, drawID, discordID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tickets for user %d in draw %d: %w", discordID, drawID, err)
	}
	defer rows.Close()

	var tickets []*entities.LotteryTicket
	for rows.Next() {
		var ticket entities.LotteryTicket
		err := rows.Scan(
			&ticket.ID,
			&ticket.DrawID,
			&ticket.GuildID,
			&ticket.DiscordID,
			&ticket.TicketNumber,
			&ticket.PurchasePrice,
			&ticket.PurchasedAt,
			&ticket.BalanceHistoryID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan lottery ticket: %w", err)
		}
		tickets = append(tickets, &ticket)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate lottery tickets: %w", err)
	}

	return tickets, nil
}

// GetWinningTickets returns all tickets matching the winning number
func (r *LotteryTicketRepository) GetWinningTickets(ctx context.Context, drawID, winningNumber int64) ([]*entities.LotteryTicket, error) {
	query := `
		SELECT id, draw_id, guild_id, discord_id, ticket_number, purchase_price, purchased_at, balance_history_id
		FROM lottery_tickets
		WHERE draw_id = $1 AND ticket_number = $2
	`

	rows, err := r.q.Query(ctx, query, drawID, winningNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get winning tickets for draw %d: %w", drawID, err)
	}
	defer rows.Close()

	var tickets []*entities.LotteryTicket
	for rows.Next() {
		var ticket entities.LotteryTicket
		err := rows.Scan(
			&ticket.ID,
			&ticket.DrawID,
			&ticket.GuildID,
			&ticket.DiscordID,
			&ticket.TicketNumber,
			&ticket.PurchasePrice,
			&ticket.PurchasedAt,
			&ticket.BalanceHistoryID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan winning lottery ticket: %w", err)
		}
		tickets = append(tickets, &ticket)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate winning lottery tickets: %w", err)
	}

	return tickets, nil
}

// CountTicketsForDraw returns the total number of tickets for a draw
func (r *LotteryTicketRepository) CountTicketsForDraw(ctx context.Context, drawID int64) (int64, error) {
	query := `SELECT COUNT(*) FROM lottery_tickets WHERE draw_id = $1`

	var count int64
	err := r.q.QueryRow(ctx, query, drawID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count tickets for draw %d: %w", drawID, err)
	}

	return count, nil
}

// GetParticipantSummary returns a summary of participants and their ticket counts
func (r *LotteryTicketRepository) GetParticipantSummary(ctx context.Context, drawID int64) ([]*entities.LotteryParticipantInfo, error) {
	query := `
		SELECT discord_id, COUNT(*) as ticket_count
		FROM lottery_tickets
		WHERE draw_id = $1
		GROUP BY discord_id
		ORDER BY ticket_count DESC
	`

	rows, err := r.q.Query(ctx, query, drawID)
	if err != nil {
		return nil, fmt.Errorf("failed to get participant summary for draw %d: %w", drawID, err)
	}
	defer rows.Close()

	var participants []*entities.LotteryParticipantInfo
	for rows.Next() {
		var p entities.LotteryParticipantInfo
		err := rows.Scan(&p.DiscordID, &p.TicketCount)
		if err != nil {
			return nil, fmt.Errorf("failed to scan participant info: %w", err)
		}
		participants = append(participants, &p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate participant summary: %w", err)
	}

	return participants, nil
}

// GetUsedNumbersByUser returns ticket numbers already used by a specific user in a draw
func (r *LotteryTicketRepository) GetUsedNumbersByUser(ctx context.Context, drawID, discordID int64) ([]int64, error) {
	query := `SELECT ticket_number FROM lottery_tickets WHERE draw_id = $1 AND discord_id = $2`

	rows, err := r.q.Query(ctx, query, drawID, discordID)
	if err != nil {
		return nil, fmt.Errorf("failed to get used numbers for draw %d: %w", drawID, err)
	}
	defer rows.Close()

	var numbers []int64
	for rows.Next() {
		var num int64
		if err := rows.Scan(&num); err != nil {
			return nil, fmt.Errorf("failed to scan ticket number: %w", err)
		}
		numbers = append(numbers, num)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate ticket numbers: %w", err)
	}

	return numbers, nil
}
