package utils

import (
	"fmt"
)

// FormatShortNotation formats a number using short notation (e.g., 50k instead of 50000)
func FormatShortNotation(value int64) string {
	absValue := value
	sign := ""
	if value < 0 {
		absValue = -value
		sign = "-"
	}

	switch {
	case absValue >= 1_000_000_000_000:
		return fmt.Sprintf("%s%.2fT", sign, float64(absValue)/1_000_000_000_000)
	case absValue >= 1_000_000_000:
		return fmt.Sprintf("%s%.2fB", sign, float64(absValue)/1_000_000_000)
	case absValue >= 1_000_000:
		return fmt.Sprintf("%s%.2fM", sign, float64(absValue)/1_000_000)
	case absValue >= 10_000:
		// No decimal places between 10k and 1M
		return fmt.Sprintf("%s%dk", sign, absValue/1_000)
	case absValue >= 1_000:
		// One decimal place under 10k
		return fmt.Sprintf("%s%.1fk", sign, float64(absValue)/1_000)
	default:
		return fmt.Sprintf("%s%d", sign, absValue)
	}
}