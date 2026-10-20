package app

import (
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"strings"
	"testing"
	"wisp/internal/library"
)

func TestNewThemesPersistAndFitOnboarding(t *testing.T) {
	for _, id := range []string{"aurora", "afterglow"} {
		m := completionFixture(t)
		m.State.Profile.Theme = id
		m.beginTheme()
		if Themes[m.choice].ID != id {
			t.Fatal("selection mismatch")
		}
		press(m, tea.KeyEnter)
		saved, err := library.Load(m.Path)
		if err != nil || saved.Profile.Theme != id {
			t.Fatal("theme not saved", id, err)
		}
		logo := LogoView(FindTheme(id), false)
		if len(strings.Split(logo, "\n")) != 15 {
			t.Fatal("wrong logo")
		}
		for _, size := range [][2]int{{44, 20}, {86, 30}, {120, 40}} {
			for _, stage := range []string{"language", "welcome", "username", "theme", "sources"} {
				m.stage = stage
				m.setup = true
				m.selected = m.State.Profile.Sources
				_, _ = m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
				view := m.View()
				rows := strings.Split(view.Content, "\n")
				if len(rows) > size[1] {
					t.Fatal("too tall", id, stage)
				}
				for _, row := range rows {
					if ansi.StringWidth(row) > size[0] {
						t.Fatal("too wide", id, stage)
					}
				}
				if stage == "theme" && !strings.Contains(ansi.Strip(view.Content), "Afterglow") {
					t.Fatal("theme choices hidden", size)
				}
				if stage == "username" && (view.Cursor == nil || view.Cursor.Y >= size[1]-1) {
					t.Fatal("input hidden", size)
				}
			}
		}
	}
	if LogoView(FindTheme("aurora"), false) == LogoView(FindTheme("afterglow"), false) {
		t.Fatal("logos identical")
	}
}
