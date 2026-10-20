package app

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

type Theme struct{ ID, Name, Description, Accent, Second, Text, Muted, Border string }

var Themes = []Theme{
	{"mist", "Mist", "Glacial blue, fading into lilac", "#87DCEF", "#B5A3E8", "#E4EAF1", "#929CAF", "#414A60"},
	{"ember", "Ember", "Peach light through evening smoke", "#F3B68C", "#D58EAF", "#EEE5DF", "#B09D99", "#5D4C53"},
	{"haze", "Haze", "Silver fog, one ice-blue signal", "#BDE8F2", "#B9C0CE", "#E1E5EA", "#9DA5B2", "#454C58"},
	{"aurora", "Aurora", "Turquoise light, sky blue and iris", "#76E8E5", "#B69BFA", "#E4F0F5", "#8C9DB4", "#455873"},
	{"afterglow", "Afterglow", "Peach light, rose and lavender", "#FFCA9F", "#DCA2E6", "#F4E8EE", "#B5A0B3", "#664E66"},
}
var Logo = []string{
	"██                  ██",
	" ██                ██ ",
	"  ██      ██      ██  ",
	"   ██    ████    ██   ",
	"    ██  ██  ██  ██    ",
	"     ████    ████     ",
	"      ██      ██      ",
}

func FindTheme(id string) Theme {
	for _, t := range Themes {
		if t.ID == id {
			return t
		}
	}
	return Themes[0]
}
func style(color string) lipgloss.Style { return lipgloss.NewStyle().Foreground(lipgloss.Color(color)) }
func LogoView(t Theme, plain bool) string {
	if plain {
		return "  \\    /\\    /\n   \\  /  \\  /\n    \\/    \\/\n"
	}
	rows := make([]string, len(Logo))
	for i, row := range Logo {
		f := float64(i) / float64(len(Logo)-1)
		c := logoColor(themeStops(t), f, 1)
		rows[i] = style(c).Render(row)
	}
	return strings.Join(rows, "\n")
}
func rgb(s string) [3]int {
	var r, g, b int
	_, _ = fmt.Sscanf(s, "#%02x%02x%02x", &r, &g, &b)
	return [3]int{r, g, b}
}

func (m *Model) applyInputTheme() {
	t := FindTheme(m.State.Profile.Theme)
	s := m.input.Styles()
	s.Focused.Prompt = style(t.Accent)
	s.Focused.Text = style(t.Text)
	s.Focused.Placeholder = style(t.Muted)
	s.Blurred = s.Focused
	s.Cursor.Color = lipgloss.Color(t.Accent)
	m.input.SetStyles(s)
}
