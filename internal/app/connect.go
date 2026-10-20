package app

import (
	"fmt"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"wisp/internal/source"
)

func (m *Model) beginCredential(name string) tea.Cmd {
	valid := false
	for _, n := range source.Names {
		if n == name {
			valid = true
			break
		}
	}
	if !valid {
		m.message = "Choose /connect soundcloud, /connect spotify or /connect youtube."
		return nil
	}
	m.credentialSource = name
	m.stage = "credential"
	m.message = ""
	m.input.SetValue("")
	m.input.CharLimit = 8192
	m.input.EchoMode = textinput.EchoPassword
	m.input.EchoCharacter = '*'
	m.input.Prompt = m.tr("token › ")
	m.input.Placeholder = m.tr("Paste OAuth access token")
	if name == "youtube" {
		m.input.Prompt = m.tr("key › ")
		m.input.Placeholder = m.tr("Paste YouTube Data API key")
	}
	m.input.SetWidth(max(8, m.width-12))
	return m.input.Focus()
}
func (m *Model) endCredential() {
	m.input.Reset()
	m.input.EchoMode = textinput.EchoNormal
	m.input.CharLimit = 2048
	m.input.Prompt = "› "
	m.input.Placeholder = m.tr("Search music or type /help")
	m.input.SetWidth(max(8, m.width-8))
	m.credentialSource = ""
	m.stage = "ready"
}

func searchNotice(count int, r source.Result) string {
	if count > 0 {
		return fmt.Sprintf("%d results. ↑/↓ select, Enter play, /open service player.", count)
	}
	if r.Succeeded > 0 {
		return "No matching tracks found. Try another title or artist."
	}
	if r.Attempted > 0 {
		return "Online search failed. Check the source errors below.\nLocal library: no matching tracks."
	}
	if len(r.Missing) > 0 {
		return "Online search is not connected.\nType /connect to set up a source.\nOr add local music: /add \"C:\\Music\"."
	}
	if len(r.ImportOnly) > 0 {
		return "Spotify public mode imports links, not catalog search.\nUse /import with a public Spotify playlist URL.\nEnable SoundCloud or YouTube in /profile to search."
	}
	return "No local matches. Add music with /add \"C:\\Music\".\nFor online search, choose sources in /profile and use /connect."
}
