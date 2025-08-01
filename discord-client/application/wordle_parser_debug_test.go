package application

import (
	"context"
	"strings"
	"testing"
)

func TestParseWordleResultsDebug(t *testing.T) {
	// The exact line from production logs
	line := "4/6: <@217883936964083713> <@133008606202167296> @Piplup"
	
	// Find all @ positions that are not part of <@userid> pattern
	atPositions := []int{}
	for i := 0; i < len(line); i++ {
		if line[i] == '@' && (i == 0 || line[i-1] != '<') {
			atPositions = append(atPositions, i)
		}
	}
	
	t.Logf("Line: %q", line)
	t.Logf("Found %d @ positions: %v", len(atPositions), atPositions)
	
	// Extract nickname from each @ position
	for _, pos := range atPositions {
		// Find the end of the nickname
		end := pos + 1
		for end < len(line) {
			ch := line[end]
			if ch == '@' || ch == '<' || ch == '\n' {
				break
			}
			end++
		}
		
		nickname := strings.TrimSpace(line[pos+1:end])
		t.Logf("Position %d: extracted nickname %q (raw: %q)", pos, nickname, line[pos+1:end])
	}
	
	// Expected: should find @Piplup at position 49
	if len(atPositions) != 1 {
		t.Errorf("Expected 1 @ position, got %d", len(atPositions))
	}
	
	if len(atPositions) > 0 && atPositions[0] != 49 {
		t.Errorf("Expected @ at position 49, got %d", atPositions[0])
	}
}

func TestWordleParserEdgeCases(t *testing.T) {
	resolver := &mockUserResolver{
		nickToIDs: map[string][]int64{
			"Piplup": {1000001},
		},
	}
	
	tests := []struct {
		name        string
		content     string
		wantUserIDs []string
		wantNicks   []string
	}{
		{
			name:        "nickname at exact end of line",
			content:     "4/6: @Piplup",
			wantUserIDs: []string{"1000001"},
			wantNicks:   []string{"Piplup"},
		},
		{
			name:        "nickname at end of line with newline",
			content:     "4/6: @Piplup\n",
			wantUserIDs: []string{"1000001"},
			wantNicks:   []string{"Piplup"},
		},
		{
			name:        "nickname at end of line after user mentions",
			content:     "4/6: <@217883936964083713> <@133008606202167296> @Piplup",
			wantUserIDs: []string{"217883936964083713", "133008606202167296", "1000001"},
			wantNicks:   []string{"Piplup"},
		},
		{
			name:        "nickname at end of line after user mentions with newline",
			content:     "4/6: <@217883936964083713> <@133008606202167296> @Piplup\n",
			wantUserIDs: []string{"217883936964083713", "133008606202167296", "1000001"},
			wantNicks:   []string{"Piplup"},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := parseWordleResults(context.Background(), tt.content, 123456, resolver)
			if err != nil {
				t.Fatalf("parseWordleResults() error = %v", err)
			}
			
			// Extract user IDs
			var gotUserIDs []string
			for _, r := range results {
				gotUserIDs = append(gotUserIDs, r.UserID)
			}
			
			// Check we got all expected user IDs
			if len(gotUserIDs) != len(tt.wantUserIDs) {
				t.Errorf("Got %d results, want %d", len(gotUserIDs), len(tt.wantUserIDs))
				t.Errorf("Got user IDs: %v", gotUserIDs)
				t.Errorf("Want user IDs: %v", tt.wantUserIDs)
			}
			
			for i, wantID := range tt.wantUserIDs {
				if i >= len(gotUserIDs) {
					t.Errorf("Missing user ID at index %d: want %s", i, wantID)
				} else if gotUserIDs[i] != wantID {
					t.Errorf("User ID at index %d: got %s, want %s", i, gotUserIDs[i], wantID)
				}
			}
		})
	}
}

func TestWordleParserCaseSensitivity(t *testing.T) {
	// Test that nickname resolution is case-sensitive (as it should be to match Discord)
	resolver := &mockUserResolver{
		nickToIDs: map[string][]int64{
			"piplup": {1000001}, // lowercase in Discord
		},
	}
	
	// Wordle bot mentions with capital P
	content := "4/6: @Piplup"
	
	results, err := parseWordleResults(context.Background(), content, 123456, resolver)
	if err != nil {
		t.Fatalf("parseWordleResults() error = %v", err)
	}
	
	// Should find no results because "Piplup" != "piplup"
	if len(results) != 0 {
		t.Errorf("Expected 0 results for case mismatch, got %d", len(results))
		for _, r := range results {
			t.Logf("  Found: UserID=%s", r.UserID)
		}
	}
}