package app

import (
	tea "charm.land/bubbletea/v2"
	"encoding/binary"
	"github.com/charmbracelet/x/ansi"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"wisp/internal/library"
	"wisp/internal/lyrics"
)

func TestTabsPlaylistsLikesRecent(t *testing.T) {
	m := completionFixture(t)
	m.State.Put([]library.Track{{ID: "a", Title: "A"}, {ID: "b", Title: "B"}})
	m.show("Search fixture", []string{"a", "b"})
	m.index = 1
	m.State.Playlists["Night"] = []string{"a", "b"}
	press(m, tea.KeyTab)
	if m.tab != 1 || m.itemCount() != 1 {
		t.Fatal("playlist tab")
	}
	press(m, tea.KeyEnter)
	if !m.playlistOpen || len(m.results) != 2 {
		t.Fatal("open playlist")
	}
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Mod: tea.ModShift})
	if len(m.State.Likes) != 1 || m.State.Likes[0] != "a" || m.busy {
		t.Fatal("like must not play")
	}
	press(m, tea.KeyEscape)
	if m.playlistOpen {
		t.Fatal("back")
	}
	press(m, tea.KeyTab)
	if m.tab != 2 || len(m.results) != 1 {
		t.Fatal("likes")
	}
	m.recordRecent("a")
	m.recordRecent("b")
	m.recordRecent("a")
	press(m, tea.KeyTab)
	if strings.Join(m.results, ",") != "a,b" {
		t.Fatal(m.results)
	}
	saved, e := library.Load(m.Path)
	if e != nil || strings.Join(saved.Recent, ",") != "a,b" {
		t.Fatal("history persistence", e)
	}
	press(m, tea.KeyTab)
	if m.tab != 0 || m.title != "Search fixture" || m.index != 1 {
		t.Fatal("library state lost")
	}
}
func TestSettingsMouseAndFrame(t *testing.T) {
	m := completionFixture(t)
	m.dispatch("/settings")
	if m.stage != "settings" {
		t.Fatal("settings")
	}
	press(m, tea.KeyDown)
	was := m.State.Profile.Sources["soundcloud"]
	press(m, tea.KeyEnter)
	if was == m.State.Profile.Sources["soundcloud"] {
		t.Fatal("source toggle")
	}
	press(m, tea.KeyUp)
	press(m, tea.KeyEnter)
	if m.stage != "theme" {
		t.Fatal("theme")
	}
	press(m, tea.KeyEnter)
	if m.stage != "settings" {
		t.Fatal("theme return")
	}
	press(m, tea.KeyEscape)
	for i := 0; i < 30; i++ {
		m.results = append(m.results, "a")
	}
	_, _ = m.Update(tea.MouseWheelMsg{Button: tea.MouseWheelDown})
	if m.index != 1 {
		t.Fatal("wheel", m.index)
	}
	_, _ = m.Update(tea.MouseWheelMsg{Button: tea.MouseWheelUp})
	if m.index != 0 {
		t.Fatal("wheel up", m.index)
	}
	m.lyricsOpen = true
	m.lyricsSong = lyrics.Song{Lines: []lyrics.Line{{Time: 0, Text: "First fixture"}, {Time: 2, Text: "Second fixture"}}}
	m.snapshot.Position = 2.5
	for _, stage := range []string{"ready", "settings", "theme"} {
		m.stage = stage
		for _, size := range [][2]int{{44, 20}, {86, 30}, {120, 40}} {
			_, _ = m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
			v := m.View()
			if v.MouseMode == tea.MouseModeNone {
				t.Fatal("mouse off")
			}
			rows := strings.Split(v.Content, "\n")
			if len(rows) != size[1] {
				t.Fatal(stage, len(rows))
			}
			for _, row := range rows {
				if ansi.StringWidth(row) > size[0] {
					t.Fatal("overflow")
				}
			}
		}
	}
}
func TestLyricsStaleReplyAndSync(t *testing.T) {
	m := completionFixture(t)
	m.current = library.Track{ID: "b"}
	m.lyricsID = "b"
	m.lyricsOpen = true
	_, _ = m.Update(lyricResult{ID: "a", Song: lyrics.Song{Plain: "stale"}})
	if m.lyricsSong.Plain != "" {
		t.Fatal("stale lyrics")
	}
	_, _ = m.Update(lyricResult{ID: "b", Song: lyrics.Song{Lines: []lyrics.Line{{Time: 0, Text: "First fixture"}, {Time: 2, Text: "Second fixture"}}}})
	m.snapshot.Position = 2.5
	body := ansi.Strip(strings.Join(m.lyricsBody(FindTheme("mist"), 80, 15), "\n"))
	if !strings.Contains(body, "› Second fixture") {
		t.Fatal(body)
	}
	m.snapshot.Position = 0.5
	body = ansi.Strip(strings.Join(m.lyricsBody(FindTheme("mist"), 80, 15), "\n"))
	if !strings.Contains(body, "› First fixture") {
		t.Fatal("seek did not update", body)
	}
}
func TestEnterPausesRealPlayer(t *testing.T) {
	mpv := os.Getenv("WISP_TEST_MPV")
	if mpv == "" {
		t.Skip("set WISP_TEST_MPV for real audio backend")
	}
	t.Setenv("WISP_MPV", mpv)
	t.Setenv("WISP_AUDIO_OUTPUT", "null")
	m := completionFixture(t)
	defer m.Engine.Close()
	path := filepath.Join(t.TempDir(), "fixture.wav")
	data := make([]byte, 44+8000*2*8)
	copy(data, "RIFF")
	binary.LittleEndian.PutUint32(data[4:], uint32(len(data)-8))
	copy(data[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(data[16:], 16)
	binary.LittleEndian.PutUint16(data[20:], 1)
	binary.LittleEndian.PutUint16(data[22:], 1)
	binary.LittleEndian.PutUint32(data[24:], 8000)
	binary.LittleEndian.PutUint32(data[28:], 16000)
	binary.LittleEndian.PutUint16(data[32:], 2)
	binary.LittleEndian.PutUint16(data[34:], 16)
	copy(data[36:], "data")
	binary.LittleEndian.PutUint32(data[40:], uint32(len(data)-44))
	if e := os.WriteFile(path, data, 0600); e != nil {
		t.Fatal(e)
	}
	track, e := library.Local(path)
	if e != nil {
		t.Fatal(e)
	}
	m.results = m.State.Put([]library.Track{track})
	cmd := m.activate()
	_, _ = m.Update(cmd())
	if m.current.ID == "" {
		t.Fatal(m.message)
	}
	for _, paused := range []bool{true, false, true} {
		_, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
		if cmd == nil || m.taskKind != "control" {
			t.Fatal("Enter restarted track")
		}
		_, _ = m.Update(cmd())
		deadline := time.Now().Add(time.Second)
		for m.Engine.Snapshot().Paused != paused && time.Now().Before(deadline) {
			time.Sleep(10 * time.Millisecond)
		}
		if m.Engine.Snapshot().Paused != paused {
			t.Fatal("pause not applied")
		}
	}
	if len(m.State.Recent) != 1 {
		t.Fatal("pause duplicated recent")
	}
}

func TestPlaylistMutationsRefreshBrowser(t *testing.T) {
	m := completionFixture(t)
	m.switchTab(1)
	m.dispatch(`/playlists new "Night"`)
	if len(m.playlistNames) != 1 || m.playlistNames[0] != "Night" {
		t.Fatal("new playlist hidden")
	}
	m.dispatch(`/playlists delete "Night"`)
	if len(m.playlistNames) != 0 {
		t.Fatal("deleted playlist visible")
	}
	m.State.Put([]library.Track{{ID: "a", Title: "A"}})
	m.State.Likes = []string{"a"}
	m.switchTab(2)
	m.dispatch("/like 1")
	if len(m.results) != 0 {
		t.Fatal("unliked row visible")
	}
}

func TestLyricsControlsOverridePlaylistBrowser(t *testing.T) {
	m := completionFixture(t)
	m.State.Put([]library.Track{{ID: "playing", Title: "Playing"}})
	m.State.Playlists["Night"] = []string{"playing"}
	m.switchTab(1)
	m.current = m.State.Tracks["playing"]
	m.lyricsOpen = true
	cmd := m.activate()
	if m.playlistOpen || cmd == nil || m.taskKind != "control" {
		t.Fatal("lyrics Enter opened playlist")
	}
	m.likeSelected()
	if len(m.State.Likes) != 1 || m.State.Likes[0] != "playing" {
		t.Fatal("lyrics like missed current track")
	}
}
func TestLyricsRetryUsesIndependentLookup(t *testing.T) {
	m := completionFixture(t)
	m.current = library.Track{ID: "fixture"}
	m.State.Tracks["fixture"] = m.current
	m.lyricsOpen = true
	m.lyricsSequence = 4
	m.operation = 9
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyF5})
	if cmd == nil || m.lyricsSequence != 5 || m.operation != 9 || m.busy {
		t.Fatal("retry disturbed playback")
	}
	if m.lyricsCancel != nil {
		m.lyricsCancel()
	}
}
