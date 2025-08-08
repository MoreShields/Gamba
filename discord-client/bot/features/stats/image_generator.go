package stats

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	log "github.com/sirupsen/logrus"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gomono"

	"gambler/discord-client/domain/entities"
	"gambler/discord-client/domain/utils"
)

// TableColumn defines a column in the scoreboard table
type TableColumn struct {
	Header      string
	Width       int
	XPosition   int
	ColorRGB    [3]float64
	Highlighted bool // Whether this column should be highlighted for certain rows
}

// TableRow represents a single row of data
type TableRow struct {
	Rank        int
	IsTop3      bool
	Data        []string
	Highlighted []bool // Which columns are highlighted for this row
}

// TableStyle defines the visual style of the table
type TableStyle struct {
	Width              int
	Height             int
	Padding            int
	RowHeight          int
	BackgroundGradient bool
	HeaderBG           [4]float64            // RGBA
	HighlightColors    map[string][4]float64 // Named highlights like "volume", "donation"
}

// ScoreboardImageGenerator generates scoreboard images
type ScoreboardImageGenerator struct {
	style TableStyle
}

// NewScoreboardImageGenerator creates a new image generator with default style
func NewScoreboardImageGenerator() *ScoreboardImageGenerator {
	return &ScoreboardImageGenerator{
		style: TableStyle{
			Width:              380,
			Height:             320,
			Padding:            15,
			RowHeight:          26,
			BackgroundGradient: true,
			HeaderBG:           [4]float64{0.2, 0.2, 0.3, 0.3},
			HighlightColors: map[string][4]float64{
				"volume":   {0.9, 0.95, 1, 0.25},  // White-blue gradient
				"donation": {1, 0.3, 0.55, 0.35},  // Coral pink gradient
				"gold":     {1, 0.84, 0, 0.1},     // Gold for 1st place
				"silver":   {0.8, 0.8, 0.8, 0.08}, // Silver for 2nd place
				"bronze":   {0.8, 0.5, 0.2, 0.06}, // Bronze for 3rd place
			},
		},
	}
}

// GenerateBitsScoreboard generates the bits leaderboard image
func (g *ScoreboardImageGenerator) GenerateBitsScoreboard(users []*entities.ScoreboardEntry) ([]byte, error) {
	// Define columns for bits scoreboard
	columns := []TableColumn{
		{Header: "#", Width: 20, XPosition: g.style.Padding, ColorRGB: [3]float64{0.85, 0.85, 0.9}},
		{Header: "User", Width: 120, XPosition: g.style.Padding + 20, ColorRGB: [3]float64{1.0, 1.0, 1.0}},
		{Header: "Balance", Width: 80, XPosition: g.style.Padding + 140, ColorRGB: [3]float64{0.85, 1.0, 0.85}},
		{Header: "Volume", Width: 80, XPosition: g.style.Padding + 220, ColorRGB: [3]float64{0.85, 0.85, 1.0}, Highlighted: true},
		{Header: "Donated", Width: 80, XPosition: g.style.Padding + 300, ColorRGB: [3]float64{1.0, 0.9, 1.0}, Highlighted: true},
	}

	// Find max volume and donations
	var maxVolume, maxDonated int64
	for _, user := range users {
		if user.TotalVolume > maxVolume {
			maxVolume = user.TotalVolume
		}
		if user.TotalDonations > maxDonated {
			maxDonated = user.TotalDonations
		}
	}

	// Convert users to table rows
	rows := make([]TableRow, len(users))
	for i, user := range users {
		username := user.Username
		if len(username) > 15 {
			username = username[:14] + "…"
		}

		rows[i] = TableRow{
			Rank:   user.Rank,
			IsTop3: i < 3,
			Data: []string{
				fmt.Sprintf("%d", user.Rank),
				username,
				utils.FormatShortNotation(user.TotalBalance),
				utils.FormatShortNotation(user.TotalVolume),
				utils.FormatShortNotation(user.TotalDonations),
			},
			Highlighted: []bool{
				false, // rank
				false, // username
				false, // balance
				user.TotalVolume == maxVolume && maxVolume > 0,
				user.TotalDonations == maxDonated && maxDonated > 0,
			},
		}
	}

	return g.generateTable(columns, rows)
}

// GenerateGameScoreboard generates a game leaderboard image (works for both LoL and TFT)
func (g *ScoreboardImageGenerator) GenerateGameScoreboard(users []*entities.LOLLeaderboardEntry, usernames map[int64]string) ([]byte, error) {
	// Define columns for game scoreboard - P/L first, then Win%
	columns := []TableColumn{
		{Header: "#", Width: 20, XPosition: g.style.Padding, ColorRGB: [3]float64{0.85, 0.85, 0.9}},
		{Header: "User", Width: 120, XPosition: g.style.Padding + 20, ColorRGB: [3]float64{1.0, 1.0, 1.0}},
		{Header: "P/L", Width: 80, XPosition: g.style.Padding + 140, ColorRGB: [3]float64{1.0, 1.0, 1.0}}, // White default, will be colored per row
		{Header: "Win%", Width: 100, XPosition: g.style.Padding + 230, ColorRGB: [3]float64{0.85, 0.85, 1.0}},
	}

	// Convert users to table rows
	rows := make([]TableRow, len(users))
	for i, user := range users {
		username := usernames[user.DiscordID]
		if username == "" {
			username = fmt.Sprintf("User%d", user.DiscordID)
		}
		if len(username) > 15 {
			username = username[:14] + "…"
		}

		// Format win percentage with record
		var winPctStr string
		if user.TotalPredictions > 0 {
			winPctStr = fmt.Sprintf("%.1f%% (%d/%d)", user.AccuracyPercentage, user.CorrectPredictions, user.TotalPredictions)
		} else {
			winPctStr = "0.0% (0/0)"
		}

		// Format P/L with + or - sign
		var profitLossStr string
		if user.ProfitLoss >= 0 {
			profitLossStr = "+" + utils.FormatShortNotation(user.ProfitLoss)
		} else {
			profitLossStr = utils.FormatShortNotation(user.ProfitLoss) // Already has minus sign
		}

		rows[i] = TableRow{
			Rank:   user.Rank,
			IsTop3: i < 3,
			Data: []string{
				fmt.Sprintf("%d", user.Rank),
				username,
				profitLossStr, // P/L first
				winPctStr,     // Win% second
			},
			Highlighted: []bool{false, false, false, false},
		}
	}

	return g.generateTable(columns, rows)
}


// generateTable creates the actual image
func (g *ScoreboardImageGenerator) generateTable(columns []TableColumn, rows []TableRow) ([]byte, error) {
	start := time.Now()
	defer func() {
		log.WithField("duration_ms", time.Since(start).Milliseconds()).
			WithField("row_count", len(rows)).
			Debug("Scoreboard image generation completed")
	}()
	// Calculate dynamic height based on number of rows
	// Header (25px) + header padding (30px) + rows + bottom padding (15px)
	height := 25 + 30 + (len(rows) * g.style.RowHeight) + 15

	// Add extra space for LoL footer text if this is the LoL scoreboard
	hasLoLFooter := false
	for _, col := range columns {
		if col.Header == "P/L" {
			hasLoLFooter = true
			height += 25 // Extra space for footer text
			break
		}
	}

	if height < g.style.Height {
		height = g.style.Height // Minimum height
	}

	dc := gg.NewContext(g.style.Width, height)

	// Enable better text rendering
	dc.SetFillRule(gg.FillRuleWinding)

	// Draw gradient background with subtle texture
	if g.style.BackgroundGradient {
		for i := 0; i < height; i++ {
			t := float64(i) / float64(height)
			baseR := 0.02 + t*0.03
			baseG := 0.02 + t*0.05
			baseB := 0.05 + t*0.1

			// Add subtle noise for texture
			for x := 0; x < g.style.Width; x++ {
				// Simple pseudo-random noise based on position
				noise := (float64((x*i)%7) - 3.5) / 255.0
				dc.SetRGB(baseR+noise, baseG+noise, baseB+noise)
				dc.SetPixel(x, i)
			}
		}
	}

	// Load fonts
	face, err := loadFont(gomono.TTF, 11)
	if err != nil {
		return nil, fmt.Errorf("failed to load font: %w", err)
	}
	dc.SetFontFace(face)

	// Draw headers
	y := float64(25)

	// Header background
	dc.SetRGBA(0.3, 0.3, 0.4, 0.4)
	dc.DrawRectangle(0, y-15, float64(g.style.Width), 20)
	dc.Fill()

	// Header text - bright white
	dc.SetRGB(1.0, 1.0, 1.0)
	for _, col := range columns {
		drawSharpText(dc, col.Header, float64(col.XPosition), y)
	}

	// Header underline
	dc.SetRGBA(0.6, 0.6, 0.7, 0.7)
	dc.SetLineWidth(1)
	dc.DrawLine(0, y+8, float64(g.style.Width), y+8)
	dc.Stroke()

	// Draw data rows
	y += 30
	for i, row := range rows {
		// Determine row highlight
		highlightType := ""
		// Check each column for highlights
		for j := 0; j < len(columns) && j < len(row.Highlighted); j++ {
			if row.Highlighted[j] && columns[j].Highlighted {
				// Determine highlight type based on column header
				if columns[j].Header == "Volume" {
					highlightType = "volume"
					break
				} else if columns[j].Header == "Donated" {
					highlightType = "donation"
					break
				}
			}
		}
		if highlightType == "" && row.IsTop3 {
			switch i {
			case 0:
				highlightType = "gold"
			case 1:
				highlightType = "silver"
			case 2:
				highlightType = "bronze"
			}
		}

		// Draw row highlight
		if highlightType != "" {
			if highlightType == "volume" || highlightType == "donation" {
				// Gradient highlight
				color := g.style.HighlightColors[highlightType]
				for j := 0; j < g.style.RowHeight; j++ {
					alpha := color[3] - (color[3] * 0.7 * float64(j) / float64(g.style.RowHeight))
					dc.SetRGBA(color[0], color[1], color[2], alpha)
					dc.DrawLine(0, y-15+float64(j), float64(g.style.Width), y-15+float64(j))
					dc.Stroke()
				}
			} else {
				// Solid highlight for top 3
				color := g.style.HighlightColors[highlightType]
				dc.SetRGBA(color[0], color[1], color[2], color[3])
				dc.DrawRectangle(0, y-15, float64(g.style.Width), float64(g.style.RowHeight))
				dc.Fill()
			}
		} else {
			// Default subtle highlight
			dc.SetRGBA(0.5, 0.5, 0.6, 0.02)
			dc.DrawRectangle(0, y-15, float64(g.style.Width), float64(g.style.RowHeight))
			dc.Fill()
		}

		// Draw rank indicator
		if row.IsTop3 {
			// Colored circle for top 3
			var red, green, blue float64
			switch i {
			case 0:
				red, green, blue = 1, 0.84, 0 // Gold
			case 1:
				red, green, blue = 0.75, 0.75, 0.75 // Silver
			case 2:
				red, green, blue = 0.8, 0.5, 0.2 // Bronze
			}
			dc.SetRGB(red, green, blue)
			dc.DrawCircle(float64(g.style.Padding+3), y-4, 5)
			dc.Fill()

			// Rank number in circle
			dc.SetRGB(0, 0, 0)
			rankFace, _ := loadFont(gobold.TTF, 9)
			dc.SetFontFace(rankFace)
			dc.DrawStringAnchored(fmt.Sprintf("%d", row.Rank), float64(g.style.Padding+3), y-5, 0.5, 0.4)
			dc.SetFontFace(face)
		} else {
			// Regular rank number
			dc.SetRGB(columns[0].ColorRGB[0], columns[0].ColorRGB[1], columns[0].ColorRGB[2])
			drawSharpText(dc, row.Data[0], float64(columns[0].XPosition), y)
		}

		// Draw other columns
		for j := 1; j < len(columns) && j < len(row.Data); j++ {
			col := columns[j]

			// Special handling for P/L column - color based on positive/negative
			if col.Header == "P/L" && j < len(row.Data) {
				if strings.HasPrefix(row.Data[j], "+") {
					dc.SetRGB(0.4, 1.0, 0.4) // Bright green for profit
				} else if strings.HasPrefix(row.Data[j], "-") {
					dc.SetRGB(1.0, 0.4, 0.4) // Bright red for loss
				} else {
					dc.SetRGB(0.8, 0.8, 0.8) // Gray for zero
				}
			} else if j < len(row.Highlighted) && row.Highlighted[j] {
				// Special color for highlighted cells
				if highlightType == "volume" {
					dc.SetRGB(0.95, 0.95, 1.0) // Bright blue-white
				} else if highlightType == "donation" {
					dc.SetRGB(1.0, 0.9, 0.98) // Bright pink-white
				} else {
					dc.SetRGB(col.ColorRGB[0], col.ColorRGB[1], col.ColorRGB[2])
				}
			} else {
				dc.SetRGB(col.ColorRGB[0], col.ColorRGB[1], col.ColorRGB[2])
			}

			// Draw special icons for highest values based on column header
			if j < len(row.Highlighted) && row.Highlighted[j] && col.Highlighted {
				if col.Header == "Volume" {
					drawDiceIcon(dc, float64(col.XPosition-20), y-8)
					// Restore color after drawing icon
					if highlightType == "volume" {
						dc.SetRGB(0.9, 0.9, 1.0) // Bright blue-white
					} else {
						dc.SetRGB(col.ColorRGB[0], col.ColorRGB[1], col.ColorRGB[2])
					}
				} else if col.Header == "Donated" {
					drawGiftIcon(dc, float64(col.XPosition-20), y-8)
					// Restore color after drawing icon
					if highlightType == "donation" {
						dc.SetRGB(1.0, 0.85, 0.95) // Bright pink-white
					} else {
						dc.SetRGB(col.ColorRGB[0], col.ColorRGB[1], col.ColorRGB[2])
					}
				}
			}

			drawSharpText(dc, row.Data[j], float64(col.XPosition), y)
		}

		y += float64(g.style.RowHeight)
	}

	// Add footer text for LoL scoreboard
	if hasLoLFooter {
		y += 15                                   // Add some spacing
		dc.SetRGB(0.7, 0.7, 0.7)                  // Gray text
		footerFace, _ := loadFont(gomono.TTF, 10) // Slightly smaller font
		dc.SetFontFace(footerFace)
		footerText := "Minimum 5 wagers to qualify"
		// Center the text
		w, _ := dc.MeasureString(footerText)
		x := (float64(g.style.Width) - w) / 2
		drawSharpText(dc, footerText, x, y)
	}

	// Convert to PNG bytes
	var buf bytes.Buffer
	if err := dc.EncodePNG(&buf); err != nil {
		return nil, fmt.Errorf("failed to encode PNG: %w", err)
	}

	return buf.Bytes(), nil
}

// drawDiceIcon draws a 3D dice icon
func drawDiceIcon(dc *gg.Context, x, y float64) {
	diceSize := 12.0

	// 3D effect - right side
	dc.SetRGB(0.7, 0.7, 0.7)
	dc.MoveTo(x+diceSize, y)
	dc.LineTo(x+diceSize+3, y-3)
	dc.LineTo(x+diceSize+3, y+diceSize-3)
	dc.LineTo(x+diceSize, y+diceSize)
	dc.ClosePath()
	dc.Fill()

	// 3D effect - top side
	dc.SetRGB(0.85, 0.85, 0.85)
	dc.MoveTo(x, y)
	dc.LineTo(x+3, y-3)
	dc.LineTo(x+diceSize+3, y-3)
	dc.LineTo(x+diceSize, y)
	dc.ClosePath()
	dc.Fill()

	// Main dice face
	dc.SetRGB(0.95, 0.95, 0.95)
	dc.DrawRoundedRectangle(x, y, diceSize, diceSize, 1.5)
	dc.Fill()

	// Dice border
	dc.SetRGB(0.3, 0.3, 0.3)
	dc.SetLineWidth(0.5)
	dc.DrawRoundedRectangle(x, y, diceSize, diceSize, 1.5)
	dc.Stroke()

	// Dice dots (5 pattern)
	dc.SetRGB(0.1, 0.1, 0.1)
	dotSize := 1.2
	dc.DrawCircle(x+3, y+3, dotSize)
	dc.DrawCircle(x+diceSize-3, y+3, dotSize)
	dc.DrawCircle(x+diceSize/2, y+diceSize/2, dotSize)
	dc.DrawCircle(x+3, y+diceSize-3, dotSize)
	dc.DrawCircle(x+diceSize-3, y+diceSize-3, dotSize)
	dc.Fill()
}

// drawGiftIcon draws a 3D gift box icon
func drawGiftIcon(dc *gg.Context, x, y float64) {
	giftSize := 12.0

	// 3D effect - right side
	dc.SetRGB(0.7, 0.3, 0.6)
	dc.MoveTo(x+giftSize, y+3)
	dc.LineTo(x+giftSize+2, y+1)
	dc.LineTo(x+giftSize+2, y+giftSize-1)
	dc.LineTo(x+giftSize, y+giftSize+1)
	dc.ClosePath()
	dc.Fill()

	// 3D effect - top (lid)
	dc.SetRGB(0.8, 0.4, 0.7)
	dc.MoveTo(x, y+3)
	dc.LineTo(x+2, y+1)
	dc.LineTo(x+giftSize+2, y+1)
	dc.LineTo(x+giftSize, y+3)
	dc.ClosePath()
	dc.Fill()

	// Main gift box
	dc.SetRGB(0.9, 0.5, 0.8)
	dc.DrawRoundedRectangle(x, y+3, giftSize, giftSize-2, 1)
	dc.Fill()

	// Gift ribbon vertical
	dc.SetRGB(1, 0.8, 0.9)
	dc.DrawRectangle(x+giftSize/2-1.5, y, 3, giftSize+1)
	dc.Fill()

	// Gift ribbon horizontal
	dc.DrawRectangle(x, y+4, giftSize, 3)
	dc.Fill()

	// Ribbon bow
	dc.SetRGB(1, 0.7, 0.85)
	dc.DrawCircle(x+giftSize/2-2, y, 2)
	dc.DrawCircle(x+giftSize/2+2, y, 2)
	dc.Fill()

	// Center knot
	dc.SetRGB(0.9, 0.6, 0.75)
	dc.DrawCircle(x+giftSize/2, y, 1.5)
	dc.Fill()
}

// drawSharpText draws text with enhanced sharpness using multiple rendering passes
func drawSharpText(dc *gg.Context, text string, x, y float64) {
	// Draw a very subtle shadow for depth (improves perceived sharpness)
	dc.Push()
	dc.SetRGBA(0, 0, 0, 0.5)
	dc.DrawString(text, x+0.5, y+0.5)
	dc.Pop()

	// Draw the main text
	dc.DrawString(text, x, y)
}

// loadFont loads a font from byte data
func loadFont(fontData []byte, size float64) (font.Face, error) {
	f, err := truetype.Parse(fontData)
	if err != nil {
		return nil, err
	}
	face := truetype.NewFace(f, &truetype.Options{
		Size:       size,
		DPI:        72,
		Hinting:    font.HintingFull,
		SubPixelsX: 4,
		SubPixelsY: 4,
	})
	return face, nil
}
