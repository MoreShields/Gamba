package common

import (
	"testing"
)


func TestFormatBalanceCompact(t *testing.T) {
	tests := []struct {
		name     string
		balance  int64
		expected string
	}{
		{"Less than 1k", 999, "999"},
		{"Exactly 1k", 1000, "1k"},
		{"1.5k", 1500, "1.5k"},
		{"10k", 10000, "10k"},
		{"100k", 100000, "100k"},
		{"213.9k", 213901, "213.9k"},
		{"1M", 1000000, "1M"},
		{"1.5M", 1500000, "1.5M"},
		{"10M", 10000000, "10M"},
		{"100M", 100000000, "100M"},
		{"1B", 1000000000, "1B"},
		{"1.5B", 1500000000, "1.5B"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatBalanceCompact(tt.balance)
			if result != tt.expected {
				t.Errorf("FormatBalanceCompact(%d) = %s; want %s", tt.balance, result, tt.expected)
			}
		})
	}
}