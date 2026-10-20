package app

import (
	"fmt"
	"math"
	"strings"
	"wisp/internal/i18n"
)

type logoProposal struct {
	Name, Description string
	Rows              []string
}

func logoProposals() []logoProposal {
	points := [][]butterflyPoint{
		{{2, 1}, {10, 9}, {18, 3}, {26, 9}, {34, 1}},
		{{2, 1}, {10, 9}, {18, 4}, {26, 9}, {34, 1}},
		{{2, 1}, {10, 9}, {18, 2}, {26, 9}, {34, 1}},
	}
	names := []string{"01 · Prism", "02 · Ribbon", "03 · Signal"}
	descriptions := []string{"Layered pixels with a soft luminous edge.", "A hollow double stroke, drawn like a ribbon.", "A W with the rhythm of an audio spectrum."}
	proposals := make([]logoProposal, 3)
	for variant, path := range points {
		rows := make([]string, 11)
		for y := 0; y < 11; y++ {
			var row strings.Builder
			for x := 0; x < 37; x++ {
				distance := math.Inf(1)
				for i := 1; i < len(path); i++ {
					distance = math.Min(distance, butterflySegment(float64(x)*.5, float64(y), path[i-1].X*.5, path[i-1].Y, path[i].X*.5, path[i].Y))
				}
				glyph := " "
				switch variant {
				case 0:
					if distance < .58 {
						glyph = "█"
					} else if distance < .95 {
						glyph = "▓"
					} else if distance < 1.25 {
						glyph = "░"
					}
				case 1:
					if distance > .60 && distance < 1.25 {
						glyph = "█"
					} else if distance <= .60 && y == 9 {
						glyph = "▄"
					}
				case 2:
					if distance < 1.1 {
						if x%4 == 0 {
							glyph = "▒"
						} else if distance < .65 {
							glyph = "█"
						} else {
							glyph = "▅"
						}
					}
				}
				row.WriteString(glyph)
			}
			rows[y] = row.String()
		}
		proposals[variant] = logoProposal{names[variant], descriptions[variant], rows}
	}
	return proposals
}
func proposalGradient(t Theme, rows []string) []string {
	a, b := rgb(t.Accent), rgb(t.Second)
	out := make([]string, len(rows))
	for i, row := range rows {
		u := float64(i) / float64(max(1, len(rows)-1))
		c := fmt.Sprintf("#%02x%02x%02x", int(float64(a[0])*(1-u)+float64(b[0])*u), int(float64(a[1])*(1-u)+float64(b[1])*u), int(float64(a[2])*(1-u)+float64(b[2])*u))
		out[i] = style(c).Render(row)
	}
	return out
}
func previewLogoProposals(theme, language string) string {
	t := FindTheme(theme)
	proposals := logoProposals()
	columns := make([][]string, len(proposals))
	for i, proposal := range proposals {
		rows := []string{style(t.Text).Bold(true).Render(proposal.Name), ""}
		rows = append(rows, proposalGradient(t, proposal.Rows)...)
		rows = append(rows, "", style(t.Muted).Render(i18n.Text(language, proposal.Description)))
		columns[i] = rows
	}
	var rows []string
	for y := 0; y < len(columns[0]); y++ {
		row := ""
		for i := range columns {
			row += pad(columns[i][y], 43)
		}
		rows = append(rows, row)
	}
	return strings.Join(rows, "\n") + "\n"
}
