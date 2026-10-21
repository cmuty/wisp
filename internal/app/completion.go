package app

import (
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"
)

type commandHint struct {
	Command, Syntax, Description, Insert string
}

// Syntax is explanatory; only Insert enters the input, so placeholders never run.
var commandHints = []commandHint{
	{"/language", "/language [en|ru]", "Change interface language", "/language "},
	{"/settings", "/settings", "Theme and search sources", "/settings"},
	{"/recent", "/recent", "Recently played tracks", "/recent"},
	{"/lyrics", "/lyrics", "Toggle lyrics (F2)", "/lyrics"},
	{"/help", "/help", "Show all commands", "/help"},
	{"/add", "/add <path>", "Add an audio file or folder", "/add "},
	{"/search", "/search <words>", "Search music", "/search "},
	{"/import", "/import <URL or file>", "Import a playlist", "/import "},
	{"/play", "/play [n]", "Play a result or your queue", "/play "},
	{"/pause", "/pause", "Pause or resume playback", "/pause"},
	{"/next", "/next", "Play the next queued track", "/next"},
	{"/prev", "/prev", "Play the previous track", "/prev"},
	{"/stop", "/stop", "Stop playback", "/stop"},
	{"/seek", "/seek <seconds>", "Seek; +15 or -10 is relative", "/seek "},
	{"/volume", "/volume <0-100>", "Set playback volume", "/volume "},
	{"/queue", "/queue [action]", "Show or edit your queue", "/queue "},
	{"/like", "/like [n]", "Toggle a liked track", "/like "},
	{"/likes", "/likes", "Show your liked tracks", "/likes"},
	{"/playlists", "/playlists [action]", "Manage local playlists", "/playlists "},
	{"/themes", "/themes", "Preview and choose a theme", "/themes"},
	{"/profile", "/profile [action]", "Profile and search sources", "/profile "},
	{"/sources", "/sources", "Choose search sources", "/sources"},
	{"/connect", "/connect [source]", "Optional API credentials", "/connect "},
	{"/open", "/open [n]", "Open the original service page", "/open "},
	{"/match", "/match [n]", "Find another recording", "/match "},
	{"/bind", "/bind <n>", "Confirm a recording match", "/bind "},
	{"/sleep", "/sleep <30m or off>", "Set or cancel the sleep timer", "/sleep "},
	{"/quit", "/quit", "Save and exit wisp", "/quit"},
	{"/exit", "/exit", "Save and exit wisp", "/exit"},
}

var argumentHints = []commandHint{
	{"/language en", "/language en", "English", "/language en"},
	{"/language ru", "/language ru", "Русский", "/language ru"},
	{"/connect soundcloud", "/connect soundcloud", "Enter a SoundCloud token", "/connect soundcloud"},
	{"/connect spotify", "/connect spotify", "Enter a Spotify token", "/connect spotify"},
	{"/connect youtube", "/connect youtube", "Enter a YouTube API key", "/connect youtube"},
	{"/queue add", "/queue add <n>", "Queue a result", "/queue add "},
	{"/queue remove", "/queue remove <n>", "Remove a queue position", "/queue remove "},
	{"/queue clear", "/queue clear", "Clear the queue", "/queue clear"},
	{"/playlists new", "/playlists new <name>", "Create a playlist", "/playlists new "},
	{"/playlists save", "/playlists save <name>", "Save the queue as a playlist", "/playlists save "},
	{"/playlists open", "/playlists open <name>", "Show a local playlist", "/playlists open "},
	{"/playlists add", "/playlists add <name> <n>", "Add a result to a playlist", "/playlists add "},
	{"/playlists delete", "/playlists delete <name>", "Delete a local playlist", "/playlists delete "},
	{"/profile name", "/profile name <name>", "Change your display name", "/profile name "},
	{"/sleep off", "/sleep off", "Cancel the sleep timer", "/sleep off"},
}

func hintsFor(line string) []commandHint {
	if !strings.HasPrefix(line, "/") {
		return nil
	}
	lower := strings.ToLower(line)
	var matches []commandHint
	if !strings.ContainsAny(lower, " \t") {
		for _, h := range commandHints {
			if h.Command == lower {
				return []commandHint{h}
			}
			if strings.HasPrefix(h.Command, lower) {
				matches = append(matches, h)
			}
		}
		return matches
	}
	for _, h := range argumentHints {
		if h.Command == lower {
			return []commandHint{h}
		}
		if strings.HasPrefix(h.Command, lower) {
			matches = append(matches, h)
		}
	}
	if len(matches) > 0 {
		return matches
	}
	// Keep a usage hint while entering arguments. Tab must preserve those values.
	best := commandHint{}
	for _, group := range [][]commandHint{commandHints, argumentHints} {
		for _, h := range group {
			if strings.HasPrefix(lower, h.Command+" ") && len(h.Command) > len(best.Command) {
				best = h
			}
		}
	}
	if best.Command != "" {
		best.Insert = line
		return []commandHint{best}
	}
	return nil
}

func (m *Model) completionItems() []commandHint {
	if m.stage != "ready" || m.setup || m.busy || m.input.Position() != utf8.RuneCountInString(m.input.Value()) {
		return nil
	}
	if m.completionHidden && m.completionText == m.input.Value() {
		return nil
	}
	return hintsFor(m.input.Value())
}

func (m *Model) refreshCompletion() {
	if m.completionText != m.input.Value() {
		m.completionText = m.input.Value()
		m.completionIndex = 0
		m.completionHidden = false
	}
}

func (m *Model) selectedCompletion(items []commandHint) int {
	if m.completionText != m.input.Value() {
		return 0
	}
	return max(0, min(m.completionIndex, len(items)-1))
}

func (m *Model) completionRows(t Theme, width int) []string {
	items := m.completionItems()
	if len(items) == 0 {
		return nil
	}
	selected := m.selectedCompletion(items)
	start := max(0, selected-3)
	rows := []string{}
	for i := start; i < min(len(items), start+4); i++ {
		prefix, color := "  ", t.Muted
		if i == selected {
			prefix, color = "› ", t.Accent
		}
		nameWidth := min(30, max(14, width/2))
		name := ansi.Truncate(m.tr(items[i].Syntax), nameWidth, "…")
		row := style(color).Render(prefix+pad(name, nameWidth)) + style(t.Muted).Render("  "+m.tr(items[i].Description))
		rows = append(rows, ansi.Truncate(row, width, "…"))
	}
	return rows
}
