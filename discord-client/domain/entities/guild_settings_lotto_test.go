package entities

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGuildSettings_HasLottoChannel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		lottoChannelID *int64
		want           bool
	}{
		{
			name:           "has channel - valid ID",
			lottoChannelID: func() *int64 { id := int64(123456789); return &id }(),
			want:           true,
		},
		{
			name:           "no channel - nil",
			lottoChannelID: nil,
			want:           false,
		},
		{
			name:           "no channel - zero value",
			lottoChannelID: func() *int64 { id := int64(0); return &id }(),
			want:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gs := &GuildSettings{
				LottoChannelID: tt.lottoChannelID,
			}

			got := gs.HasLottoChannel()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGuildSettings_GetLottoChannelID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		lottoChannelID *int64
		want           int64
	}{
		{
			name:           "returns channel ID when set",
			lottoChannelID: func() *int64 { id := int64(123456789); return &id }(),
			want:           123456789,
		},
		{
			name:           "returns zero when nil",
			lottoChannelID: nil,
			want:           0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gs := &GuildSettings{
				LottoChannelID: tt.lottoChannelID,
			}

			got := gs.GetLottoChannelID()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGuildSettings_SetLottoChannel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		channelID *int64
	}{
		{
			name:      "set valid channel ID",
			channelID: func() *int64 { id := int64(123456789); return &id }(),
		},
		{
			name:      "clear channel by setting nil",
			channelID: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gs := &GuildSettings{}
			gs.SetLottoChannel(tt.channelID)

			if tt.channelID == nil {
				assert.Nil(t, gs.LottoChannelID)
			} else {
				assert.NotNil(t, gs.LottoChannelID)
				assert.Equal(t, *tt.channelID, *gs.LottoChannelID)
			}
		})
	}
}

func TestGuildSettings_GetLottoTicketCost(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		lottoTicketCost *int64
		want            int64
	}{
		{
			name:            "returns custom cost when set",
			lottoTicketCost: func() *int64 { c := int64(500); return &c }(),
			want:            500,
		},
		{
			name:            "returns default cost when nil",
			lottoTicketCost: nil,
			want:            DefaultLottoTicketCost,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gs := &GuildSettings{
				LottoTicketCost: tt.lottoTicketCost,
			}

			got := gs.GetLottoTicketCost()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGuildSettings_SetLottoTicketCost(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cost *int64
	}{
		{
			name: "set custom cost",
			cost: func() *int64 { c := int64(500); return &c }(),
		},
		{
			name: "reset to default by setting nil",
			cost: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gs := &GuildSettings{}
			gs.SetLottoTicketCost(tt.cost)

			if tt.cost == nil {
				assert.Nil(t, gs.LottoTicketCost)
			} else {
				assert.NotNil(t, gs.LottoTicketCost)
				assert.Equal(t, *tt.cost, *gs.LottoTicketCost)
			}
		})
	}
}

func TestGuildSettings_GetLottoDifficulty(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		lottoDifficulty *int64
		want            int64
	}{
		{
			name:            "returns custom difficulty when set",
			lottoDifficulty: func() *int64 { d := int64(10); return &d }(),
			want:            10,
		},
		{
			name:            "returns default difficulty when nil",
			lottoDifficulty: nil,
			want:            DefaultLottoDifficulty,
		},
		{
			name:            "returns minimum difficulty",
			lottoDifficulty: func() *int64 { d := int64(MinLottoDifficulty); return &d }(),
			want:            MinLottoDifficulty,
		},
		{
			name:            "returns maximum difficulty",
			lottoDifficulty: func() *int64 { d := int64(MaxLottoDifficulty); return &d }(),
			want:            MaxLottoDifficulty,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gs := &GuildSettings{
				LottoDifficulty: tt.lottoDifficulty,
			}

			got := gs.GetLottoDifficulty()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGuildSettings_SetLottoDifficulty(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		difficulty *int64
	}{
		{
			name:       "set custom difficulty",
			difficulty: func() *int64 { d := int64(12); return &d }(),
		},
		{
			name:       "reset to default by setting nil",
			difficulty: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gs := &GuildSettings{}
			gs.SetLottoDifficulty(tt.difficulty)

			if tt.difficulty == nil {
				assert.Nil(t, gs.LottoDifficulty)
			} else {
				assert.NotNil(t, gs.LottoDifficulty)
				assert.Equal(t, *tt.difficulty, *gs.LottoDifficulty)
			}
		})
	}
}

func TestGuildSettings_IsLottoEnabled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		lottoChannelID *int64
		want           bool
	}{
		{
			name:           "enabled when channel is set",
			lottoChannelID: func() *int64 { id := int64(123456789); return &id }(),
			want:           true,
		},
		{
			name:           "disabled when channel is nil",
			lottoChannelID: nil,
			want:           false,
		},
		{
			name:           "disabled when channel is zero",
			lottoChannelID: func() *int64 { id := int64(0); return &id }(),
			want:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gs := &GuildSettings{
				LottoChannelID: tt.lottoChannelID,
			}

			got := gs.IsLottoEnabled()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLottoConstants(t *testing.T) {
	t.Parallel()

	// Verify constants are set correctly
	assert.Equal(t, int64(1000), int64(DefaultLottoTicketCost))
	assert.Equal(t, int64(8), int64(DefaultLottoDifficulty))
	assert.Equal(t, int64(4), int64(MinLottoDifficulty))
	assert.Equal(t, int64(20), int64(MaxLottoDifficulty))

	// Verify min < default < max
	assert.Less(t, MinLottoDifficulty, DefaultLottoDifficulty)
	assert.Less(t, DefaultLottoDifficulty, MaxLottoDifficulty)
}
