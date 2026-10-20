package app

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"wisp/internal/library"
	"wisp/internal/source"
)

func TestVolumeKeySteps(t *testing.T) {
	for _, test := range []struct {
		key  tea.KeyPressMsg
		want int
	}{
		{tea.KeyPressMsg{Code: '+'}, 5}, {tea.KeyPressMsg{Code: '-'}, -5},
		{tea.KeyPressMsg{Code: '+', Mod: tea.ModShift}, 1}, {tea.KeyPressMsg{Code: '-', Mod: tea.ModShift}, -1},
		{tea.KeyPressMsg{Code: '_', Text: "_"}, -1}, {tea.KeyPressMsg{Code: '=', Mod: tea.ModShift, ShiftedCode: '+'}, 1},
		{tea.KeyPressMsg{Code: tea.KeyKpPlus}, 5}, {tea.KeyPressMsg{Code: tea.KeyKpMinus, Mod: tea.ModShift}, -1},
		{tea.KeyPressMsg{Code: '-', Mod: tea.ModCtrl}, 0},
	} {
		if got := volumeDelta(test.key); got != test.want {
			t.Errorf("%#v: got %d, want %d", test.key, got, test.want)
		}
	}
}
func TestVolumeKeysPreserveInputAndPendingTasks(t *testing.T) {
	m := completionFixture(t)
	defer m.Engine.Close()
	m.operation = 42
	m.busy = true
	_, cmd := m.Update(tea.KeyPressMsg{Code: '+', Text: "+"})
	if cmd == nil || m.volume != 75 || m.input.Value() != "" || m.operation != 42 {
		t.Fatal("volume key interfered with the active task")
	}
	_, next := m.Update(tea.KeyPressMsg{Code: '+', Text: "+"})
	if next != nil || m.volume != 80 {
		t.Fatal("repeat was not coalesced")
	}
	_, next = m.Update(cmd())
	if next == nil {
		t.Fatal("latest volume was lost")
	}
	_, _ = m.Update(next())
	if m.Engine.Snapshot().Volume != 80 {
		t.Fatal("queued volume not applied")
	}
	m.input.SetValue("artist")
	m.input.CursorEnd()
	_, _ = m.Update(tea.KeyPressMsg{Code: '-', Text: "-"})
	if m.input.Value() != "artist-" || m.volume != 80 {
		t.Fatal("shortcut stole a character from text input")
	}
	cmd = m.setVolume(1000)
	_, _ = m.Update(cmd())
	if m.volume != 100 {
		t.Fatal("upper volume bound")
	}
	cmd = m.setVolume(-1000)
	_, _ = m.Update(cmd())
	if m.volume != 0 {
		t.Fatal("lower volume bound")
	}
}
func TestWindowTitleAndQuietPlayback(t *testing.T) {
	m := completionFixture(t)
	m.State.Put([]library.Track{{ID: "local:test", Source: "local", Title: "Song", Artist: "Artist"}})
	m.results = []string{"local:test"}
	_, _ = m.Update(taskResult{ID: m.operation, Kind: "play", Track: m.State.Tracks["local:test"]})
	if m.message != "" {
		t.Fatal("duplicate playback message remains")
	}
	view := m.View()
	if view.WindowTitle != "wisp" || !strings.Contains(view.Content, "70%") {
		t.Fatal("title or visible volume missing")
	}
	_, _ = m.Update(tea.WindowSizeMsg{Width: 30, Height: 10})
	if m.View().WindowTitle != "wisp" {
		t.Fatal("small frame lost title")
	}
	m.stage = "welcome"
	m.setup = true
	_, _ = m.Update(tea.WindowSizeMsg{Width: 86, Height: 30})
	if m.View().WindowTitle != "wisp" {
		t.Fatal("onboarding lost title")
	}
}
func TestPublicSpotifyDoesNotSpamSearch(t *testing.T) {
	t.Setenv("WISP_SPOTIFY_TOKEN", "")
	c := source.New()
	c.PublicWeb = true
	r := c.Search(t.Context(), "song", map[string]bool{"spotify": true}, nil)
	if len(r.Warnings) != 0 || len(r.ImportOnly) != 1 {
		t.Fatal("routine Spotify capability shown as a warning")
	}
}
