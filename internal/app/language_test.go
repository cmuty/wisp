package app

import (
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"strings"
	"testing"
	"wisp/internal/library"
	"wisp/internal/player"
)

func TestFirstRunLanguageBeforeWelcome(t *testing.T) {
	m := fixture(t)
	if m.stage != "language" {
		t.Fatal("language is not first")
	}
	view := ansi.Strip(m.View().Content)
	if !strings.Contains(view, "English") || !strings.Contains(view, "Русский") {
		t.Fatal(view)
	}
	press(m, tea.KeyDown)
	press(m, tea.KeyEnter)
	if m.stage != "welcome" || m.State.Profile.Language != "ru" {
		t.Fatal("Russian not selected")
	}
	if !strings.Contains(m.View().Content, "Добро пожаловать") {
		t.Fatal("welcome not translated")
	}
	saved, e := library.Load(m.Path)
	if e != nil || saved.Profile.Language != "ru" {
		t.Fatal("language not persisted", e)
	}
	resumed := New(saved, m.Path, player.New())
	if resumed.stage != "welcome" {
		t.Fatal("asked for language again")
	}
	press(m, tea.KeyEnter)
	if m.stage != "username" || m.input.Placeholder != "Твоё имя" {
		t.Fatal("username not translated")
	}
}
func TestLanguageSwitchPreservesLibraryAndUserText(t *testing.T) {
	m := completionFixture(t)
	m.State.Profile.Complete = true
	m.State.Profile.Username = "Welcome to wisp."
	m.State.Playlists["Your likes"] = []string{"a"}
	m.State.Put([]library.Track{{ID: "a", Artist: "Search music", Title: "Nothing playing"}})
	m.dispatch("/language ru")
	if m.input.Placeholder == "Search music or type /help" {
		t.Fatal("input not translated")
	}
	m.dispatch(`/playlists open "Your likes"`)
	view := ansi.Strip(m.View().Content)
	if !strings.Contains(view, "Your likes") || !strings.Contains(view, "Search music — Nothing playing") {
		t.Fatal("translated user data", view)
	}
	m.dispatch("/settings")
	m.choice = 4
	press(m, tea.KeyEnter)
	if m.stage != "language" {
		t.Fatal("settings language missing")
	}
	press(m, tea.KeyUp)
	press(m, tea.KeyEnter)
	if m.stage != "settings" || m.State.Profile.Language != "en" {
		t.Fatal("switch back failed")
	}
	if m.State.Profile.Username != "Welcome to wisp." || len(m.State.Playlists) != 1 {
		t.Fatal("library changed")
	}
}
func TestRussianHelpAndHintsComplete(t *testing.T) {
	m := completionFixture(t)
	m.State.Profile.Language = "ru"
	for _, line := range help {
		if m.tr(line) == line {
			t.Errorf("untranslated help %q", line)
		}
	}
	for _, group := range [][]commandHint{commandHints, argumentHints} {
		for _, hint := range group {
			if hint.Description != "English" && hint.Description != "Русский" && m.tr(hint.Description) == hint.Description {
				t.Errorf("untranslated hint %q", hint.Description)
			}
		}
	}
	for _, stage := range []string{"language", "welcome", "username", "theme", "sources", "syncing", "ready", "settings"} {
		for _, size := range [][2]int{{44, 20}, {86, 30}, {120, 40}} {
			m.stage = stage
			m.setup = stage != "ready" && stage != "settings"
			m.selected = m.State.Profile.Sources
			_, _ = m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
			view := m.View()
			rows := strings.Split(view.Content, "\n")
			if len(rows) > size[1] {
				t.Fatal("height overflow", stage, size)
			}
			for _, row := range rows {
				if ansi.StringWidth(row) > size[0] {
					t.Fatal("width overflow", stage, size)
				}
			}
		}
	}
}
func TestExistingProfileKeepsOnboardingComplete(t *testing.T) {
	s := library.New()
	s.Profile.Complete = true
	m := New(s, "", player.New())
	if m.stage != "ready" || m.State.Profile.Language != "en" {
		t.Fatal("legacy profile reset")
	}
}
