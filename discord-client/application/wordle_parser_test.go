package application

import (
	"context"
	"reflect"
	"testing"
)

// mockUserResolver implements UserResolver for testing
type mockUserResolver struct {
	nickToIDs map[string][]int64
}

func (m *mockUserResolver) ResolveUsersByNick(ctx context.Context, guildID int64, nickname string) ([]int64, error) {
	if ids, ok := m.nickToIDs[nickname]; ok {
		return ids, nil
	}
	return nil, nil
}

func TestParseWordleResults(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []WordleResult
		wantErr bool
	}{
		{
			name: "sample wordle message",
			content: "**Your group is on a 70 day streak!** 🔥 Here are yesterday's results:\n" +
				"👑 3/6: <@133008606202167296> <@135678825894903808>\n" +
				"4/6: <@232153861941362688> <@160868610301100032> <@233264723175407627> <@141402190152597504> @Piplup\n" +
				"5/6: @ChancellorLoaf <@217883936964083713>",
			want: []WordleResult{
				{UserID: "133008606202167296", GuessCount: 3, MaxGuesses: 6},
				{UserID: "135678825894903808", GuessCount: 3, MaxGuesses: 6},
				{UserID: "232153861941362688", GuessCount: 4, MaxGuesses: 6},
				{UserID: "160868610301100032", GuessCount: 4, MaxGuesses: 6},
				{UserID: "233264723175407627", GuessCount: 4, MaxGuesses: 6},
				{UserID: "141402190152597504", GuessCount: 4, MaxGuesses: 6},
				{UserID: "217883936964083713", GuessCount: 5, MaxGuesses: 6},
			},
			wantErr: false,
		},
		{
			name: "single line single user",
			content: "3/6: <@123456789>",
			want: []WordleResult{
				{UserID: "123456789", GuessCount: 3, MaxGuesses: 6},
			},
			wantErr: false,
		},
		{
			name: "multiple users same score",
			content: "2/6: <@111111111> <@222222222> <@333333333>",
			want: []WordleResult{
				{UserID: "111111111", GuessCount: 2, MaxGuesses: 6},
				{UserID: "222222222", GuessCount: 2, MaxGuesses: 6},
				{UserID: "333333333", GuessCount: 2, MaxGuesses: 6},
			},
			wantErr: false,
		},
		{
			name: "no score pattern",
			content: "Just some text with <@123456789> mentioned",
			want: []WordleResult{},
			wantErr: false,
		},
		{
			name: "score without users",
			content: "3/6: @SomeUser @AnotherUser",
			want: []WordleResult{},
			wantErr: false,
		},
		{
			name: "mixed valid and invalid lines",
			content: "Some intro text\n3/6: <@123456789>\nJust text\n4/6: @NotAUser\n5/6: <@987654321>",
			want: []WordleResult{
				{UserID: "123456789", GuessCount: 3, MaxGuesses: 6},
				{UserID: "987654321", GuessCount: 5, MaxGuesses: 6},
			},
			wantErr: false,
		},
		{
			name: "perfect score",
			content: "1/6: <@123456789>",
			want: []WordleResult{
				{UserID: "123456789", GuessCount: 1, MaxGuesses: 6},
			},
			wantErr: false,
		},
		{
			name: "max guesses",
			content: "6/6: <@123456789>",
			want: []WordleResult{
				{UserID: "123456789", GuessCount: 6, MaxGuesses: 6},
			},
			wantErr: false,
		},
		{
			name: "empty content",
			content: "",
			want: []WordleResult{},
			wantErr: false,
		},
		{
			name: "crown emoji before score",
			content: "👑 3/6: <@123456789>",
			want: []WordleResult{
				{UserID: "123456789", GuessCount: 3, MaxGuesses: 6},
			},
			wantErr: false,
		},
		{
			name: "problematic message with nickname at newline and spaces in nickname",
			content: "**Your group is on a 74 day streak!** 🔥 Here are yesterday's results:\n" +
				"👑 3/6: <@233264723175407627>\n" +
				"4/6: <@217883936964083713> <@133008606202167296> @Piplup\n" +
				"5/6: @Shid @ChancellorLoaf @Captain Rowdy <@135678825894903808> <@141402190152597504>",
			want: []WordleResult{
				{UserID: "233264723175407627", GuessCount: 3, MaxGuesses: 6},
				{UserID: "217883936964083713", GuessCount: 4, MaxGuesses: 6},
				{UserID: "133008606202167296", GuessCount: 4, MaxGuesses: 6},
				// Note: @Piplup should be parsed but since we don't have a resolver, it won't appear
				{UserID: "135678825894903808", GuessCount: 5, MaxGuesses: 6},
				{UserID: "141402190152597504", GuessCount: 5, MaxGuesses: 6},
				// Note: @Shid, @ChancellorLoaf, @Captain Rowdy should be parsed but since we don't have a resolver, they won't appear
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseWordleResults(context.Background(), tt.content, 123456, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseWordleResults() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				// Handle case where we expect empty results
				if len(tt.want) == 0 && len(got) == 0 {
					return
				}
				t.Errorf("parseWordleResults() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseWordleResultsWithNicknameResolution(t *testing.T) {
	resolver := &mockUserResolver{
		nickToIDs: map[string][]int64{
			"Piplup":         {1000001},
			"Shid":           {1000002},
			"ChancellorLoaf": {1000003},
			"Captain Rowdy":  {1000004},
		},
	}

	tests := []struct {
		name    string
		content string
		want    []WordleResult
	}{
		{
			name: "nicknames with spaces and at end of line",
			content: "**Your group is on a 74 day streak!** 🔥 Here are yesterday's results:\n" +
				"👑 3/6: <@233264723175407627>\n" +
				"4/6: <@217883936964083713> <@133008606202167296> @Piplup\n" +
				"5/6: @Shid @ChancellorLoaf @Captain Rowdy <@135678825894903808> <@141402190152597504>",
			want: []WordleResult{
				{UserID: "233264723175407627", GuessCount: 3, MaxGuesses: 6},
				{UserID: "217883936964083713", GuessCount: 4, MaxGuesses: 6},
				{UserID: "133008606202167296", GuessCount: 4, MaxGuesses: 6},
				{UserID: "1000001", GuessCount: 4, MaxGuesses: 6}, // Piplup
				{UserID: "135678825894903808", GuessCount: 5, MaxGuesses: 6},
				{UserID: "141402190152597504", GuessCount: 5, MaxGuesses: 6},
				{UserID: "1000002", GuessCount: 5, MaxGuesses: 6}, // Shid
				{UserID: "1000003", GuessCount: 5, MaxGuesses: 6}, // ChancellorLoaf
				{UserID: "1000004", GuessCount: 5, MaxGuesses: 6}, // Captain Rowdy
			},
		},
		{
			name: "mixed nicknames and mentions",
			content: "3/6: @TestUser <@123456789> @AnotherUser\n" +
				"4/6: <@987654321> @Multi Word Name",
			want: []WordleResult{
				{UserID: "123456789", GuessCount: 3, MaxGuesses: 6},
				{UserID: "987654321", GuessCount: 4, MaxGuesses: 6},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseWordleResults(context.Background(), tt.content, 123456, resolver)
			if err != nil {
				t.Fatalf("parseWordleResults() error = %v", err)
			}

			// Create a map to check results easier
			resultMap := make(map[string]int)
			for _, r := range got {
				resultMap[r.UserID] = r.GuessCount
			}

			// Check we got all expected results
			for _, expected := range tt.want {
				if actualGuesses, ok := resultMap[expected.UserID]; !ok {
					t.Errorf("Missing result for user %s", expected.UserID)
				} else if actualGuesses != expected.GuessCount {
					t.Errorf("User %s: got %d guesses, want %d", expected.UserID, actualGuesses, expected.GuessCount)
				}
			}

			// Check we didn't get extra results
			if len(got) != len(tt.want) {
				t.Errorf("Got %d results, want %d", len(got), len(tt.want))
			}
		})
	}
}