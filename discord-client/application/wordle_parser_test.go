package application

import (
	"context"
	"reflect"
	"testing"
)

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