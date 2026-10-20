package app

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"fmt"
	"github.com/charmbracelet/x/ansi"
	"sort"
	"strings"
	"wisp/internal/source"
)

type browserState struct {
	IDs                 []string
	Title, Message      string
	Warnings            []string
	Index, Offset, Page int
}

var tabs = []string{"Library", "Playlists", "Likes", "Recent"}

func (m *Model) switchTab(n int) {
	if m.tab == 0 {
		m.libraryView = browserState{append([]string(nil), m.results...), m.title, m.message, m.warnings, m.index, m.offset, m.pageOffset}
	}
	m.tab = (n + len(tabs)) % len(tabs)
	m.playlistOpen = false
	m.lyricsOpen = false
	if m.tab == 0 {
		b := m.libraryView
		m.results = b.IDs
		m.title = b.Title
		m.message = b.Message
		m.warnings = b.Warnings
		m.index = b.Index
		m.offset = b.Offset
		m.pageOffset = b.Page
		return
	}
	m.show(tabs[m.tab], nil)
	switch m.tab {
	case 1:
		m.playlistNames = nil
		for name := range m.State.Playlists {
			m.playlistNames = append(m.playlistNames, name)
		}
		sort.Strings(m.playlistNames)
		m.message = "/import a link or /playlists new \"name\""
	case 2:
		m.show("Your likes", m.State.Likes)
	case 3:
		m.show("Recently played", m.State.Recent)
	}
}
func (m *Model) itemCount() int {
	if m.tab == 1 && !m.playlistOpen {
		return len(m.playlistNames)
	}
	return len(m.results)
}
func (m *Model) move(delta int) {
	if m.stage == "language" {
		m.choice = (m.choice + delta%2 + 2) % 2
		return
	}
	if m.lyricsOpen {
		m.lyricsFollow = false
		m.lyricsOffset = max(0, min(max(0, len(m.lyricRows())-1), m.lyricsOffset+delta))
		return
	}
	if m.stage == "settings" {
		m.choice = (m.choice + delta%6 + 6) % 6
		return
	}
	if m.stage == "theme" || m.stage == "sources" {
		count := len(Themes)
		if m.stage == "sources" {
			count = len(source.Names) + 1
		}
		m.choice = (m.choice + delta%count + count) % count
		if m.stage == "theme" {
			m.State.Profile.Theme = Themes[m.choice].ID
			m.applyInputTheme()
		}
		return
	}
	if m.itemCount() > 0 {
		m.index = max(0, min(m.itemCount()-1, m.index+delta))
	} else {
		m.pageOffset = max(0, m.pageOffset+delta)
	}
}
func (m *Model) activate() tea.Cmd {
	if m.lyricsOpen && m.current.ID != "" {
		return m.control("cycle", "pause")
	}
	if m.tab == 1 && !m.playlistOpen {
		if m.index < len(m.playlistNames) {
			name := m.playlistNames[m.index]
			m.show(name, m.State.Playlists[name])
			m.playlistOpen = true
		}
		return nil
	}
	id, _ := m.selectedID()
	if m.current.ID != "" && (id == m.current.ID || id == "") && m.Engine.Ready() {
		return m.control("cycle", "pause")
	}
	if id != "" {
		return m.playFromResults(m.index)
	}
	return nil
}
func (m *Model) likeSelected() {
	if m.lyricsOpen && m.current.ID != "" {
		m.toggleLike(m.current.ID)
		return
	}
	id, _ := m.selectedID()
	if id == "" {
		id = m.current.ID
	}
	if id == "" {
		return
	}
	m.toggleLike(id)
}
func (m *Model) toggleLike(id string) {
	found := -1
	for i, k := range m.State.Likes {
		if k == id {
			found = i
			break
		}
	}
	if found >= 0 {
		m.State.Likes = append(m.State.Likes[:found], m.State.Likes[found+1:]...)
		m.message = "Removed from likes."
	} else {
		m.State.Likes = append(m.State.Likes, id)
		m.message = "Saved to likes."
	}
	if m.tab == 2 {
		m.results = append([]string(nil), m.State.Likes...)
		m.index = min(m.index, max(0, len(m.results)-1))
	}
	m.persist()
}
func (m *Model) recordRecent(id string) {
	ids := []string{id}
	for _, k := range m.State.Recent {
		if k != id && len(ids) < 200 {
			ids = append(ids, k)
		}
	}
	m.State.Recent = ids
	m.persist()
	if m.tab == 3 {
		m.results = append([]string(nil), ids...)
	}
}
func (m *Model) navigation(t Theme) string {
	rows := []string{}
	for i, name := range tabs {
		name = m.tr(name)
		c := t.Muted
		if i == m.tab {
			c = t.Accent
			name = "[" + name + "]"
		}
		rendered := style(c).Render(name)
		if i == m.tab {
			rendered = gradientText(t, name)
		}
		rows = append(rows, rendered)
	}
	return strings.Join(rows, "   ")
}
func (m *Model) playlistBody(t Theme, height int) []string {
	rows := []string{style(t.Text).Bold(true).Render(m.tr("Your playlists")), ""}
	limit := max(1, height-4)
	if m.index < m.offset {
		m.offset = m.index
	}
	if m.index >= m.offset+limit {
		m.offset = m.index - limit + 1
	}
	for i := m.offset; i < min(len(m.playlistNames), m.offset+limit); i++ {
		name := m.playlistNames[i]
		prefix := "  "
		c := t.Muted
		if i == m.index {
			prefix = "› "
			c = t.Accent
		}
		rows = append(rows, style(c).Render(fmt.Sprintf(m.tr("%s%s   %d tracks"), prefix, name, len(m.State.Playlists[name]))))
	}
	if len(m.playlistNames) == 0 {
		rows = append(rows, style(t.Muted).Render(m.tr(m.message)))
	} else {
		rows = append(rows, "", style(t.Muted).Render(m.tr("Enter opens playlist · Esc returns from tracks")))
	}
	return rows
}
func (m *Model) settingsBody(t Theme) []string {
	rows := []string{style(t.Text).Bold(true).Render(m.tr("Make room for your sound.")), ""}
	entries := []string{m.tr("Theme") + "   " + m.tr(FindTheme(m.State.Profile.Theme).Name)}
	for i, n := range source.Names {
		mark := "[ ]"
		if m.State.Profile.Sources[n] {
			mark = "[x]"
		}
		entries = append(entries, mark+" "+source.Labels[i])
	}
	languageName := "English"
	if m.State.Profile.Language == "ru" {
		languageName = "Русский"
	}
	entries = append(entries, m.tr("Language")+"   "+languageName, m.tr("Done"))
	for i, s := range entries {
		p := "  "
		c := t.Muted
		if i == m.choice {
			p = "› "
			c = t.Accent
		}
		rows = append(rows, style(c).Render(p+s))
	}
	return append(rows, "", style(t.Muted).Render(m.tr("Spotify: playlist metadata; audio matched on YouTube.")), style(t.Muted).Render(m.tr("Local music is always enabled.")))
}
func (m *Model) settingsKey(key string) {
	switch key {
	case "up":
		m.move(-1)
	case "down":
		m.move(1)
	case "enter", "space":
		switch m.choice {
		case 0:
			m.settingsReturn = true
			m.beginTheme()
		case 4:
			m.beginLanguage()
		case 5:
			m.stage = "ready"
		default:
			n := source.Names[m.choice-1]
			m.State.Profile.Sources[n] = !m.State.Profile.Sources[n]
			m.persist()
		}
	}
}

func (m *Model) header(t Theme, width int) string { return m.headerLine(t, width, true) }
func (m *Model) headerLine(t Theme, width int, showUser bool) string {
	nav := m.navigation(t)
	if lipgloss.Width(nav)+20 > width {
		names := []string{"Lib", "Lists", "Likes", "Recent"}
		if m.State.Profile.Language == "ru" {
			names = []string{"Муз.", "Листы", "Лайки", "История"}
		}
		parts := []string{}
		for i, n := range names {
			n = m.tr(n)
			c := t.Muted
			if i == m.tab {
				c = t.Accent
				n = "[" + n + "]"
			}
			rendered := style(c).Render(n)
			if i == m.tab {
				rendered = gradientText(t, n)
			}
			parts = append(parts, rendered)
		}
		nav = strings.Join(parts, "  ")
	}
	navWidth := lipgloss.Width(nav)
	left := headerLogo(t, m.Plain) + "  " + style(t.Text).Render("@"+m.State.Profile.Username)
	if width < headerLogoWidth+navWidth+12 {
		if !showUser {
			return strings.Repeat(" ", min(width, headerLogoWidth))
		}
		return ansi.Truncate(left, width, "…")
	}
	left = ansi.Truncate(left, max(headerLogoWidth+3, width-navWidth-3), "…")
	x := max(lipgloss.Width(left)+3, (width-navWidth)/2)
	if !showUser {
		left = strings.Repeat(" ", lipgloss.Width(left))
	}
	row := left + strings.Repeat(" ", max(0, x-lipgloss.Width(left))) + nav
	right := style(t.Muted).Render(m.tr("a little room for sound"))
	if width-lipgloss.Width(row) >= lipgloss.Width(right)+3 {
		row += strings.Repeat(" ", width-lipgloss.Width(row)-lipgloss.Width(right)) + right
	}
	return row
}
