package service

import (
	"time"
)

// GetNextResetTime calculates the next daily limit reset time based on the configured hour
func GetNextResetTime(resetHour int) time.Time {
	now := time.Now().UTC()
	resetTime := time.Date(now.Year(), now.Month(), now.Day(), resetHour, 0, 0, 0, time.UTC)
	
	// If current time is past today's reset, use tomorrow's
	if now.After(resetTime) || now.Equal(resetTime) {
		resetTime = resetTime.AddDate(0, 0, 1)
	}
	
	return resetTime
}

// GetCurrentPeriodStart calculates when the current daily limit period started
func GetCurrentPeriodStart(resetHour int) time.Time {
	now := time.Now().UTC()
	periodStart := time.Date(now.Year(), now.Month(), now.Day(), resetHour, 0, 0, 0, time.UTC)
	
	// If current time is before today's reset, use yesterday's reset time
	if now.Before(periodStart) {
		periodStart = periodStart.AddDate(0, 0, -1)
	}
	
	return periodStart
}