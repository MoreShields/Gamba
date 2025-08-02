package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatShortNotation(t *testing.T) {
	tests := []struct {
		name     string
		value    int64
		expected string
	}{
		{
			name:     "zero",
			value:    0,
			expected: "0",
		},
		{
			name:     "small positive",
			value:    999,
			expected: "999",
		},
		{
			name:     "exactly 1k",
			value:    1000,
			expected: "1.0k",
		},
		{
			name:     "1.5k",
			value:    1500,
			expected: "1.5k",
		},
		{
			name:     "9.9k",
			value:    9900,
			expected: "9.9k",
		},
		{
			name:     "10k",
			value:    10000,
			expected: "10k",
		},
		{
			name:     "15k",
			value:    15000,
			expected: "15k",
		},
		{
			name:     "50k",
			value:    50000,
			expected: "50k",
		},
		{
			name:     "103k",
			value:    103000,
			expected: "103k",
		},
		{
			name:     "999k",
			value:    999000,
			expected: "999k",
		},
		{
			name:     "exactly 1M",
			value:    1000000,
			expected: "1.00M",
		},
		{
			name:     "1.25M",
			value:    1250000,
			expected: "1.25M",
		},
		{
			name:     "1.5M",
			value:    1500000,
			expected: "1.50M",
		},
		{
			name:     "999M",
			value:    999000000,
			expected: "999.00M",
		},
		{
			name:     "exactly 1B",
			value:    1000000000,
			expected: "1.00B",
		},
		{
			name:     "1.5B",
			value:    1500000000,
			expected: "1.50B",
		},
		{
			name:     "exactly 1T",
			value:    1000000000000,
			expected: "1.00T",
		},
		{
			name:     "negative small",
			value:    -500,
			expected: "-500",
		},
		{
			name:     "negative 1k",
			value:    -1000,
			expected: "-1.0k",
		},
		{
			name:     "negative 50k",
			value:    -50000,
			expected: "-50k",
		},
		{
			name:     "negative 1M",
			value:    -1000000,
			expected: "-1.00M",
		},
		{
			name:     "fractional thousands",
			value:    1234,
			expected: "1.2k",
		},
		{
			name:     "fractional millions",
			value:    1234567,
			expected: "1.23M",
		},
		{
			name:     "fractional billions",
			value:    1234567890,
			expected: "1.23B",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatShortNotation(tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}