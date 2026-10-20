package app

import (
	"github.com/charmbracelet/x/ansi"
	"github.com/rivo/uniseg"
	"strings"
)

func themeStops(t Theme) []string {
	switch t.ID {
	case "aurora":
		return refinedLogos[0].Colors
	case "afterglow":
		return refinedLogos[2].Colors
	}
	return []string{t.Accent, t.Second}
}
func gradientText(t Theme, text string) string {
	width := ansi.StringWidth(text)
	position := 0
	var out strings.Builder
	graphemes := uniseg.NewGraphemes(text)
	for graphemes.Next() {
		glyph := graphemes.Str()
		out.WriteString(style(logoColor(themeStops(t), float64(position)/float64(max(1, width-1)), 1)).Render(glyph))
		position += ansi.StringWidth(glyph)
	}
	return out.String()
}

// Every segment uses its absolute position on the full scale. Advancing playback
// reveals the next color instead of rescaling the whole palette into a short fill.
func progressSegment(t Theme, glyph string, start, count, total int, brightness float64) string {
	var out strings.Builder
	for i := 0; i < count; i++ {
		out.WriteString(style(logoColor(themeStops(t), float64(start+i)/float64(max(1, total-1)), brightness)).Render(glyph))
	}
	return out.String()
}
