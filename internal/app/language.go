package app

import (
	tea "charm.land/bubbletea/v2"
	"wisp/internal/i18n"
)

var languages = []string{"en", "ru"}

func (m *Model) tr(text string) string { return i18n.Text(m.State.Profile.Language, text) }
func (m *Model) applyLanguage() {
	m.input.Placeholder = m.tr("Search music or type /help")
	if m.stage == "username" {
		m.input.Placeholder = m.tr("Your name")
	}
}
func (m *Model) beginLanguage() {
	m.languageReturn = "ready"
	if m.stage == "settings" {
		m.languageReturn = "settings"
	}
	if m.setup {
		m.languageReturn = "welcome"
	}
	m.stage = "language"
	m.choice = 0
	if m.State.Profile.Language == "ru" {
		m.choice = 1
	}
}
func (m *Model) languageKey(key string) tea.Cmd {
	switch key {
	case "up", "down", "left", "right":
		m.choice = 1 - m.choice
	case "enter":
		m.State.Profile.Language = languages[m.choice]
		m.stage = m.languageReturn
		if m.stage == "" {
			m.stage = "welcome"
		}
		m.choice = 0
		m.applyLanguage()
		m.persist()
	}
	return nil
}
func (m *Model) languageBody(t Theme) []string {
	rows := []string{style(t.Text).Bold(true).Render("Choose your language / Выбери язык"), ""}
	for i, name := range []string{"English", "Русский"} {
		prefix := "  "
		c := t.Muted
		if i == m.choice {
			prefix = "› "
			c = t.Accent
		}
		rows = append(rows, style(c).Render(prefix+name))
	}
	return append(rows, "", style(t.Muted).Render("↑ ↓ · Enter — select / выбрать"))
}

func (m *Model) displayTitle() string {
	if m.tab == 1 && m.playlistOpen {
		return m.title
	}
	return m.tr(m.title)
}
