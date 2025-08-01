package debug

import (
	"fmt"
	"strings"
)

// formatNumber formats a number with thousands separators
func formatNumber(n int64) string {
	if n < 0 {
		return "-" + formatNumber(-n)
	}

	str := fmt.Sprintf("%d", n)
	result := ""
	
	for i, digit := range str {
		if i > 0 && (len(str)-i)%3 == 0 {
			result += ","
		}
		result += string(digit)
	}
	
	return result
}

// formatSignedNumber formats a number with sign and thousands separators
func formatSignedNumber(n int64) string {
	if n > 0 {
		return "+" + formatNumber(n)
	}
	return formatNumber(n)
}

// truncateString truncates a string to a maximum length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// padRight pads a string to the right with spaces
func padRight(s string, length int) string {
	if len(s) >= length {
		return s
	}
	return s + strings.Repeat(" ", length-len(s))
}

// padLeft pads a string to the left with spaces
func padLeft(s string, length int) string {
	if len(s) >= length {
		return s
	}
	return strings.Repeat(" ", length-len(s)) + s
}

// centerString centers a string within a given width
func centerString(s string, width int) string {
	if len(s) >= width {
		return s
	}
	
	leftPad := (width - len(s)) / 2
	rightPad := width - len(s) - leftPad
	
	return strings.Repeat(" ", leftPad) + s + strings.Repeat(" ", rightPad)
}

// formatTable formats data as a simple ASCII table
func formatTable(headers []string, rows [][]string) string {
	if len(headers) == 0 || len(rows) == 0 {
		return ""
	}

	// Calculate column widths
	colWidths := make([]int, len(headers))
	for i, header := range headers {
		colWidths[i] = len(header)
	}
	
	for _, row := range rows {
		for i, cell := range row {
			if i < len(colWidths) && len(cell) > colWidths[i] {
				colWidths[i] = len(cell)
			}
		}
	}

	// Build separator line
	separator := "+"
	for _, width := range colWidths {
		separator += strings.Repeat("-", width+2) + "+"
	}

	// Build table
	var result strings.Builder
	
	// Header
	result.WriteString(separator + "\n")
	result.WriteString("|")
	for i, header := range headers {
		result.WriteString(" " + padRight(header, colWidths[i]) + " |")
	}
	result.WriteString("\n" + separator + "\n")

	// Rows
	for _, row := range rows {
		result.WriteString("|")
		for i, cell := range row {
			if i < len(colWidths) {
				result.WriteString(" " + padRight(cell, colWidths[i]) + " |")
			}
		}
		result.WriteString("\n")
	}
	result.WriteString(separator)

	return result.String()
}

// colorText returns text with ANSI color codes
func colorText(text string, color string) string {
	colors := map[string]string{
		"red":     "\033[31m",
		"green":   "\033[32m",
		"yellow":  "\033[33m",
		"blue":    "\033[34m",
		"magenta": "\033[35m",
		"cyan":    "\033[36m",
		"white":   "\033[37m",
		"reset":   "\033[0m",
	}

	if code, ok := colors[color]; ok {
		return code + text + colors["reset"]
	}
	return text
}

// formatDuration formats a duration in seconds to a human-readable string
func formatDuration(seconds int64) string {
	if seconds < 60 {
		return fmt.Sprintf("%d seconds", seconds)
	}
	
	minutes := seconds / 60
	if minutes < 60 {
		return fmt.Sprintf("%d minutes", minutes)
	}
	
	hours := minutes / 60
	if hours < 24 {
		return fmt.Sprintf("%d hours", hours)
	}
	
	days := hours / 24
	return fmt.Sprintf("%d days", days)
}

// progressBar creates a simple ASCII progress bar
func progressBar(current, total int64, width int) string {
	if total == 0 {
		return strings.Repeat("─", width)
	}

	percentage := float64(current) / float64(total)
	filled := int(percentage * float64(width))
	
	bar := strings.Repeat("█", filled)
	bar += strings.Repeat("░", width-filled)
	
	return fmt.Sprintf("[%s] %.1f%%", bar, percentage*100)
}