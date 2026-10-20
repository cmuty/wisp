package app

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"wisp/internal/library"
	"wisp/internal/lyrics"
	"wisp/internal/player"
	"wisp/internal/source"
)

// Preview never persists a profile or starts the audio backend.
func Preview(stage, theme string, plain bool) string { return PreviewFrame(stage, theme, plain, -1) }
func PreviewFrame(stage, theme string, plain bool, frame int) string {
	return PreviewLanguage(stage, theme, plain, frame, "en")
}
func PreviewLanguage(stage, theme string, plain bool, frame int, language string) string {
	if stage == "logos-refined" {
		return previewRefinedLogos()
	}
	if stage == "logos" {
		return previewLogoProposals(theme, language)
	}
	if stage == "butterfly-directions" {
		return previewButterflyDirections(theme)
	}
	s := library.New()
	s.Profile.Language = language
	s.Profile.Username = "listener"
	s.Profile.Complete = true
	s.Profile.Theme = FindTheme(theme).ID
	m := New(s, "", player.New())
	m.width = 86
	m.height = 30
	m.Plain = plain
	m.ReducedMotion = true
	switch stage {
	case "language", "welcome", "username", "theme", "sources":
		m.stage = stage
		m.setup = true
		m.selected = s.Profile.Sources
		if stage == "theme" {
			m.beginTheme()
		}
	case "help":
		m.dispatch("/help")
	case "settings":
		m.setup = false
		m.stage = "settings"
		m.choice = 2
	case "playlists":
		m.setup = false
		m.State.Playlists["Night walks"] = []string{"fixture"}
		m.State.Playlists["Quiet mornings"] = []string{"fixture", "fixture"}
		m.switchTab(1)
	case "butterfly-search":
		m.setup = false
		var tracks []library.Track
		titles := []string{"Midnight echoes", "Window seat", "Soft signal", "Under the quiet sky", "Paper lanterns", "A little room for sound", "Blue hour", "Drifting home", "After the rain", "Somewhere, slowly", "First light", "No rush"}
		for i, title := range titles {
			tracks = append(tracks, library.Track{ID: fmt.Sprint(i), Source: "soundcloud", Artist: "wisp", Title: title})
		}
		m.show("Search: midnight", m.State.Put(tracks))
		m.index = 5
		m.current = tracks[5]
		m.message = ""
		m.snapshot = player.Snapshot{Position: 35, Duration: 180, Active: true}
	case "lyrics", "butterfly-flight":
		m.setup = false
		m.current = library.Track{ID: "fixture", Artist: "wisp", Title: "A little room for sound"}
		m.lyricsOpen = true
		m.lyricsFollow = true
		if stage == "butterfly-flight" {
			m.ReducedMotion = false
			m.butterflyFrame = butterflyRestFrames + butterflyFlightFrames/2
		}
		m.snapshot = player.Snapshot{Position: 35, Duration: 180, Active: true}
		m.lyricsSong = lyrics.Song{Credit: "Demo · invented lines", Lines: []lyrics.Line{{Time: 0, Text: "A room beyond the noise"}, {Time: 20, Text: "A little space to breathe"}, {Time: 32, Text: "Let every echo find its way"}, {Time: 40, Text: "Like mist between the trees"}, {Time: 50, Text: "The quiet stays with me"}}}
	case "ready":
		m.stage = "ready"
	case "disconnected":
		m.stage = "ready"
		m.setup = false
		m.title = "Search: москва слезам не верит"
		m.message = searchNotice(0, source.Result{Missing: []string{"soundcloud", "spotify"}})
		m.warnings = []string{"SoundCloud: not connected. /connect soundcloud", "Spotify: not connected. /connect spotify"}
	case "credential":
		m.setup = false
		m.beginCredential("soundcloud")
	case "completion":
		m.stage = "ready"
		m.setup = false
		m.input.SetValue("/p")
		m.input.CursorEnd()
	default:
		m.stage = "welcome"
	}
	if stage == "ready" {
		m.title = "Your listening space"
		m.message = "A quiet queue, ready for your music. /add a folder to begin."
	}
	if stage == "username" {
		m.message = ""
		m.input.Placeholder = m.tr("Your name")
	}
	if frame >= 0 {
		m.ReducedMotion = false
		m.butterflyFrame = frame
	}
	m.applyLanguage()
	_, _ = m.Update(tea.WindowSizeMsg{Width: 86, Height: 30})
	return strings.TrimRight(m.View().Content, "\n") + "\n"
}

func previewButterflyDirections(theme string) string {
	t := FindTheme(theme)
	labels := []string{"Up", "Up-right", "Right", "Down-right", "Down", "Down-left", "Left", "Up-left"}
	lines := []string{style(t.Accent).Render("wisp · flight directions"), ""}
	for band := 0; band < 2; band++ {
		line := ""
		for i := 0; i < 4; i++ {
			line += pad(style(t.Muted).Render(labels[band*4+i]), 20)
		}
		lines = append(lines, line, "")
		var sprites [4][butterflyHeight]string
		for i := 0; i < 4; i++ {
			sprites[i] = butterflySprite(butterflyPose{Heading: band*4 + i}, t)
		}
		for row := 0; row < butterflyHeight; row++ {
			line = ""
			for i := 0; i < 4; i++ {
				line += pad(sprites[i][row], 20)
			}
			lines = append(lines, line)
		}
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n") + "\n"
}
