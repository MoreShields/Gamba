package entities

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLotteryTicket_IsWinner(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		ticketNumber  int64
		winningNumber int64
		want          bool
	}{
		{
			name:          "ticket is winner - matching number",
			ticketNumber:  42,
			winningNumber: 42,
			want:          true,
		},
		{
			name:          "ticket is not winner - different number",
			ticketNumber:  42,
			winningNumber: 43,
			want:          false,
		},
		{
			name:          "ticket is winner - zero number",
			ticketNumber:  0,
			winningNumber: 0,
			want:          true,
		},
		{
			name:          "ticket is not winner - zero ticket, non-zero winning",
			ticketNumber:  0,
			winningNumber: 1,
			want:          false,
		},
		{
			name:          "ticket is winner - max number for difficulty 8",
			ticketNumber:  255,
			winningNumber: 255,
			want:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ticket := &LotteryTicket{
				TicketNumber: tt.ticketNumber,
			}

			got := ticket.IsWinner(tt.winningNumber)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLotteryTicket_FormatBinary(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		ticketNumber int64
		difficulty   int64
		want         string
	}{
		{
			name:         "format number 0 with difficulty 4",
			ticketNumber: 0,
			difficulty:   4,
			want:         "0000",
		},
		{
			name:         "format number 5 with difficulty 4",
			ticketNumber: 5,
			difficulty:   4,
			want:         "0101",
		},
		{
			name:         "format number 15 with difficulty 4",
			ticketNumber: 15,
			difficulty:   4,
			want:         "1111",
		},
		{
			name:         "format number 42 with difficulty 8",
			ticketNumber: 42,
			difficulty:   8,
			want:         "00101010",
		},
		{
			name:         "format number 255 with difficulty 8",
			ticketNumber: 255,
			difficulty:   8,
			want:         "11111111",
		},
		{
			name:         "format number 128 with difficulty 8",
			ticketNumber: 128,
			difficulty:   8,
			want:         "10000000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ticket := &LotteryTicket{
				TicketNumber: tt.ticketNumber,
			}

			got := ticket.FormatBinary(tt.difficulty)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLotteryTicket_FullStructure(t *testing.T) {
	t.Parallel()

	// Test that all fields can be set and retrieved correctly
	now := time.Now()

	ticket := &LotteryTicket{
		ID:               1,
		DrawID:           100,
		GuildID:          123456789,
		DiscordID:        987654321,
		TicketNumber:     42,
		PurchasePrice:    1000,
		PurchasedAt:      now,
		BalanceHistoryID: 500,
	}

	assert.Equal(t, int64(1), ticket.ID)
	assert.Equal(t, int64(100), ticket.DrawID)
	assert.Equal(t, int64(123456789), ticket.GuildID)
	assert.Equal(t, int64(987654321), ticket.DiscordID)
	assert.Equal(t, int64(42), ticket.TicketNumber)
	assert.Equal(t, int64(1000), ticket.PurchasePrice)
	assert.Equal(t, now, ticket.PurchasedAt)
	assert.Equal(t, int64(500), ticket.BalanceHistoryID)
}

func TestLotteryParticipantInfo_Structure(t *testing.T) {
	t.Parallel()

	// Test that LotteryParticipantInfo can be created and used correctly
	info := &LotteryParticipantInfo{
		DiscordID:   123456789,
		TicketCount: 5,
	}

	assert.Equal(t, int64(123456789), info.DiscordID)
	assert.Equal(t, int64(5), info.TicketCount)
}
