package entities

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLotteryDraw_IsCompleted(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		completedAt *time.Time
		want        bool
	}{
		{
			name:        "draw not completed - nil completedAt",
			completedAt: nil,
			want:        false,
		},
		{
			name:        "draw completed - has completedAt",
			completedAt: func() *time.Time { t := time.Now(); return &t }(),
			want:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			draw := &LotteryDraw{
				CompletedAt: tt.completedAt,
			}

			got := draw.IsCompleted()
			assert.Equal(t, tt.want, got)
		})
	}
}


func TestLotteryDraw_CanPurchaseTickets(t *testing.T) {
	t.Parallel()

	now := time.Now()
	futureTime := now.Add(1 * time.Hour)
	pastTime := now.Add(-1 * time.Hour)

	tests := []struct {
		name        string
		completedAt *time.Time
		drawTime    time.Time
		want        bool
	}{
		{
			name:        "can purchase - not completed and before draw time",
			completedAt: nil,
			drawTime:    futureTime,
			want:        true,
		},
		{
			name:        "cannot purchase - already completed",
			completedAt: &now,
			drawTime:    futureTime,
			want:        false,
		},
		{
			name:        "cannot purchase - past draw time",
			completedAt: nil,
			drawTime:    pastTime,
			want:        false,
		},
		{
			name:        "cannot purchase - completed and past draw time",
			completedAt: &now,
			drawTime:    pastTime,
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			draw := &LotteryDraw{
				CompletedAt: tt.completedAt,
				DrawTime:    tt.drawTime,
			}

			got := draw.CanPurchaseTickets()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLotteryDraw_GetMaxNumber(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		difficulty int64
		want       int64
	}{
		{
			name:       "difficulty 4 - max 15",
			difficulty: 4,
			want:       15, // 2^4 - 1 = 15
		},
		{
			name:       "difficulty 8 - max 255",
			difficulty: 8,
			want:       255, // 2^8 - 1 = 255
		},
		{
			name:       "difficulty 10 - max 1023",
			difficulty: 10,
			want:       1023, // 2^10 - 1 = 1023
		},
		{
			name:       "difficulty 16 - max 65535",
			difficulty: 16,
			want:       65535, // 2^16 - 1 = 65535
		},
		{
			name:       "difficulty 20 - max 1048575",
			difficulty: 20,
			want:       1048575, // 2^20 - 1 = 1048575
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			draw := &LotteryDraw{
				Difficulty: tt.difficulty,
			}

			got := draw.GetMaxNumber()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLotteryDraw_GetTotalNumbers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		difficulty int64
		want       int64
	}{
		{
			name:       "difficulty 4 - 16 total numbers",
			difficulty: 4,
			want:       16, // 2^4 = 16
		},
		{
			name:       "difficulty 8 - 256 total numbers",
			difficulty: 8,
			want:       256, // 2^8 = 256
		},
		{
			name:       "difficulty 10 - 1024 total numbers",
			difficulty: 10,
			want:       1024, // 2^10 = 1024
		},
		{
			name:       "difficulty 16 - 65536 total numbers",
			difficulty: 16,
			want:       65536, // 2^16 = 65536
		},
		{
			name:       "difficulty 20 - 1048576 total numbers",
			difficulty: 20,
			want:       1048576, // 2^20 = 1048576
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			draw := &LotteryDraw{
				Difficulty: tt.difficulty,
			}

			got := draw.GetTotalNumbers()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLotteryDraw_GenerateWinningNumber(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		difficulty int64
	}{
		{
			name:       "difficulty 4 - generates number in range [0, 15]",
			difficulty: 4,
		},
		{
			name:       "difficulty 8 - generates number in range [0, 255]",
			difficulty: 8,
		},
		{
			name:       "difficulty 10 - generates number in range [0, 1023]",
			difficulty: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			draw := &LotteryDraw{
				Difficulty: tt.difficulty,
			}

			// Generate multiple numbers to verify range
			maxNum := draw.GetMaxNumber()
			for i := 0; i < 100; i++ {
				num, err := draw.GenerateWinningNumber()
				assert.NoError(t, err)
				assert.GreaterOrEqual(t, num, int64(0))
				assert.LessOrEqual(t, num, maxNum)
			}
		})
	}
}

func TestLotteryDraw_Complete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		winningNumber int64
	}{
		{
			name:          "complete with winning number 42",
			winningNumber: 42,
		},
		{
			name:          "complete with winning number 0",
			winningNumber: 0,
		},
		{
			name:          "complete with winning number 255",
			winningNumber: 255,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			draw := &LotteryDraw{
				ID:          1,
				GuildID:     123456789,
				Difficulty:  8,
				TicketCost:  1000,
				DrawTime:    time.Now(),
				TotalPot:    10000,
				CompletedAt: nil,
			}

			beforeComplete := time.Now()
			draw.Complete(tt.winningNumber)
			afterComplete := time.Now()

			// Verify winning number is set
			assert.NotNil(t, draw.WinningNumber)
			assert.Equal(t, tt.winningNumber, *draw.WinningNumber)

			// Verify completedAt is set within expected range
			assert.NotNil(t, draw.CompletedAt)
			assert.True(t, draw.CompletedAt.After(beforeComplete) || draw.CompletedAt.Equal(beforeComplete))
			assert.True(t, draw.CompletedAt.Before(afterComplete) || draw.CompletedAt.Equal(afterComplete))

			// Verify IsCompleted returns true
			assert.True(t, draw.IsCompleted())
		})
	}
}

func TestLotteryDraw_SetMessage(t *testing.T) {
	t.Parallel()

	draw := &LotteryDraw{
		ID:      1,
		GuildID: 123456789,
	}

	channelID := int64(111222333)
	messageID := int64(444555666)

	draw.SetMessage(channelID, messageID)

	assert.NotNil(t, draw.ChannelID)
	assert.Equal(t, channelID, *draw.ChannelID)
	assert.NotNil(t, draw.MessageID)
	assert.Equal(t, messageID, *draw.MessageID)
}

func TestLotteryDraw_HasMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		channelID *int64
		messageID *int64
		want      bool
	}{
		{
			name:      "has message - both IDs set",
			channelID: func() *int64 { id := int64(111222333); return &id }(),
			messageID: func() *int64 { id := int64(444555666); return &id }(),
			want:      true,
		},
		{
			name:      "no message - channelID nil",
			channelID: nil,
			messageID: func() *int64 { id := int64(444555666); return &id }(),
			want:      false,
		},
		{
			name:      "no message - messageID nil",
			channelID: func() *int64 { id := int64(111222333); return &id }(),
			messageID: nil,
			want:      false,
		},
		{
			name:      "no message - both nil",
			channelID: nil,
			messageID: nil,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			draw := &LotteryDraw{
				ChannelID: tt.channelID,
				MessageID: tt.messageID,
			}

			got := draw.HasMessage()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFormatBinaryNumber(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		number     int64
		difficulty int64
		want       string
	}{
		{
			name:       "number 0 with difficulty 4",
			number:     0,
			difficulty: 4,
			want:       "0000",
		},
		{
			name:       "number 5 with difficulty 4",
			number:     5,
			difficulty: 4,
			want:       "0101",
		},
		{
			name:       "number 15 with difficulty 4",
			number:     15,
			difficulty: 4,
			want:       "1111",
		},
		{
			name:       "number 42 with difficulty 8",
			number:     42,
			difficulty: 8,
			want:       "00101010",
		},
		{
			name:       "number 255 with difficulty 8",
			number:     255,
			difficulty: 8,
			want:       "11111111",
		},
		{
			name:       "number 0 with difficulty 8",
			number:     0,
			difficulty: 8,
			want:       "00000000",
		},
		{
			name:       "number 1023 with difficulty 10",
			number:     1023,
			difficulty: 10,
			want:       "1111111111",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := FormatBinaryNumber(tt.number, tt.difficulty)
			assert.Equal(t, tt.want, got)
		})
	}
}
