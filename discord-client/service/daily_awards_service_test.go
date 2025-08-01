package service

import (
	"testing"
)

func TestWordleDailyAwardStreak(t *testing.T) {
	award := WordleDailyAward{
		DiscordID:  123456789,
		GuessCount: 3,
		Reward:     35000, // 7000 base * 5 streak
		Streak:     5,
	}

	// Test GetStreak method
	if award.GetStreak() != 5 {
		t.Errorf("GetStreak() = %d, want %d", award.GetStreak(), 5)
	}

	// Test GetDetails method
	if award.GetDetails() != "3/6" {
		t.Errorf("GetDetails() = %s, want %s", award.GetDetails(), "3/6")
	}

	// Test GetReward method
	if award.GetReward() != 35000 {
		t.Errorf("GetReward() = %d, want %d", award.GetReward(), 35000)
	}
}