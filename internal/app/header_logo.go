package app

import (
	"github.com/charmbracelet/x/ansi"
	"strings"
)

const headerLogoWidth = 15

func headerLogoRows(t Theme, plain bool, height int) []string {
	v := refinedLogos[0]
	if t.ID == "afterglow" {
		v = refinedLogos[2]
	} else if t.ID != "aurora" {
		v.Colors = []string{t.Accent, t.Second}
	}
	if plain {
		rows := []string{`\          /`, ` \   /\   / `, `  \ /  \ /  `, `   V    V   `, `            `}
		if height < 5 {
			rows = []string{`\   /\   /  `, ` \ /  \ /   `, `  V    V    `}
		}
		for i := range rows {
			rows[i] = style(logoColor(v.Colors, float64(i)/float64(max(1, len(rows)-1)), 1)).Render(rows[i])
		}
		for i := range rows {
			rows[i] = pad(rows[i], headerLogoWidth)
		}
		for len(rows) < height {
			rows = append(rows, strings.Repeat(" ", headerLogoWidth))
		}
		return rows
	}
	return refinedLogoRowsSized(v, headerLogoWidth, height)
}
func headerLogo(t Theme, plain bool) string { return headerLogoRows(t, plain, 5)[2] }
func (m *Model) headerRows(t Theme, width int) []string {
	height := 6
	if m.height < 26 {
		height = 3
	}
	logo := headerLogoRows(t, m.Plain, height)
	rows := append([]string{""}, logo...)
	username := style(t.Text).Render("@" + m.State.Profile.Username)
	rows[1] = pad(rows[1], headerLogoWidth) + "  " + ansi.Truncate(username, max(1, width-headerLogoWidth-2), "…")
	// Navigation retains its original screen row while identity sits above it.
	center := m.headerLine(t, width, false)
	rows[2] = pad(rows[2], headerLogoWidth) + ansi.Cut(center, headerLogoWidth, width)
	return append(rows, "")
}
