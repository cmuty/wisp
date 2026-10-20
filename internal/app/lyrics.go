package app

import (
	tea "charm.land/bubbletea/v2"
	"context"
	"strings"
	"time"
	"wisp/internal/library"
	"wisp/internal/lyrics"
)

type lyricResult struct {
	Sequence int
	ID       string
	Song     lyrics.Song
	Err      error
}

func (m *Model) toggleLyrics() tea.Cmd {
	m.lyricsOpen = !m.lyricsOpen
	if !m.lyricsOpen {
		return nil
	}
	m.lyricsFollow = true
	m.lyricsOffset = 0
	if m.current.ID == "" {
		m.lyricsMessage = "Play a track first."
		return nil
	}
	if m.lyricsID == m.current.ID && m.lyricsMessage == "" {
		return nil
	}
	return m.fetchLyrics()
}
func (m *Model) fetchLyrics() tea.Cmd {
	if m.lyricsCancel != nil {
		m.lyricsCancel()
	}
	m.lyricsSequence++
	sequence := m.lyricsSequence
	m.lyricsID = m.current.ID
	m.lyricsSong = lyrics.Song{}
	m.lyricsMessage = "Finding your words…"
	track := m.State.Tracks[m.current.ID]
	if m.snapshot.Duration > 0 {
		track.Duration = m.snapshot.Duration
	}
	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
	m.lyricsCancel = cancel
	return func() tea.Msg {
		defer cancel()
		s, e := m.lyricsClient.Find(ctx, track)
		return lyricResult{Sequence: sequence, ID: track.ID, Song: s, Err: e}
	}
}
func (m *Model) lyricRows() []string {
	if len(m.lyricsSong.Lines) > 0 {
		rows := make([]string, len(m.lyricsSong.Lines))
		for i, l := range m.lyricsSong.Lines {
			rows[i] = l.Text
		}
		return rows
	}
	if m.lyricsSong.Plain != "" {
		if m.lyricsSong.Plain == "Instrumental — no lyrics." {
			return []string{m.tr(m.lyricsSong.Plain)}
		}
		return strings.Split(m.lyricsSong.Plain, "\n")
	}
	return []string{m.tr(m.lyricsMessage)}
}
func (m *Model) lyricsBody(t Theme, width, height int) []string {
	if m.current.ID == "" {
		return []string{style(t.Text).Bold(true).Render(m.tr("Lyrics")), "", style(t.Muted).Render(m.tr("Play a track first."))}
	}
	label := m.tr("Lyrics")
	if m.current.ID != "" {
		label += " · " + m.current.Artist + " — " + m.current.Title
	}
	rows := []string{style(t.Text).Bold(true).Render(library.Clean(label)), ""}
	items := m.lyricRows()
	limit := max(1, height-2)
	active := lyrics.Active(m.lyricsSong.Lines, m.snapshot.Position+m.lyricsTiming)
	if m.lyricsFollow {
		m.lyricsOffset = max(0, min(max(0, len(items)-limit), active-limit/2))
	}
	start := min(m.lyricsOffset, max(0, len(items)-1))
	for i := start; i < min(len(items), start+limit); i++ {
		c := t.Muted
		p := "  "
		if len(m.lyricsSong.Lines) > 0 && i == active {
			c = t.Accent
			p = "› "
		}
		text := p + library.Clean(items[i])
		rendered := style(c).Render(text)
		if len(m.lyricsSong.Lines) > 0 && i == active {
			rendered = gradientText(t, text)
		}
		rows = append(rows, rendered)
	}
	if len(m.lyricsSong.Lines) == 0 && m.lyricsSong.Plain != "" {
		rows = rows[:min(len(rows), max(2, height-1))]
		rows = append(rows, style(t.Muted).Render(m.tr("No timing for this recording.")))
	}
	return rows
}
