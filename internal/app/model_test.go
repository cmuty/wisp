package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"wisp/internal/library"
	"wisp/internal/player"
	"wisp/internal/source"
)

func press(m *Model, code rune) { _, _ = m.Update(tea.KeyPressMsg{Code: code}) }
func fixture(t *testing.T) *Model {
	t.Helper()
	return New(library.New(), filepath.Join(t.TempDir(), "library.json"), player.New())
}

func TestLivePublicSearchToPlayback(t *testing.T) {
	if os.Getenv("WISP_TEST_PUBLIC") != "1" || os.Getenv("WISP_TEST_MPV") == "" {
		t.Skip("set WISP_TEST_PUBLIC=1 and WISP_TEST_MPV for the public player flow")
	}
	t.Setenv("WISP_SOUNDCLOUD_TOKEN", "")
	t.Setenv("WISP_SPOTIFY_TOKEN", "")
	t.Setenv("WISP_YOUTUBE_KEY", "")
	t.Setenv("WISP_MPV", os.Getenv("WISP_TEST_MPV"))
	t.Setenv("WISP_AUDIO_OUTPUT", "null")
	m := fixture(t)
	m.stage = "ready"
	m.setup = false
	defer m.Engine.Close()
	m.Sources.PublicWeb = true
	cmd := m.dispatch("москва слезам не верит")
	if cmd == nil {
		t.Fatal("search was not dispatched")
	}
	_, _ = m.Update(cmd())
	if len(m.results) == 0 || strings.Contains(m.message, "not connected") {
		t.Fatal("public search did not reach the player UI")
	}
	t.Logf("first source: %s; source notices: %v", m.State.Tracks[m.results[0]].Source, m.warnings)
	cmd = m.dispatch("/play 1")
	if cmd == nil {
		t.Fatal("play was not dispatched")
	}
	_, _ = m.Update(cmd())
	if m.current.ID == "" {
		t.Fatal(m.message)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	for ctx.Err() == nil {
		if m.Engine.Snapshot().Position > 0.05 {
			t.Logf("aggregated %d results; selected track is decoding without account credentials", len(m.results))
			return
		}
		time.Sleep(40 * time.Millisecond)
	}
	t.Fatal("player did not decode audio")
}

func TestCredentialInputDoesNotLeak(t *testing.T) {
	m := fixture(t)
	m.stage = "ready"
	m.setup = false
	m.input.SetValue("/connect spotify")
	press(m, tea.KeyEnter)
	if m.stage != "credential" {
		t.Fatal("credential form did not open")
	}
	secret := "fixture-credential-do-not-store"
	m.input.SetValue(secret)
	if strings.Contains(m.View().Content, secret) {
		t.Fatal("secret is visible")
	}
	press(m, tea.KeyEnter)
	if m.stage != "ready" || m.Sources.Credential("spotify") != secret {
		t.Fatal("session credential was not applied")
	}
	if strings.Contains(strings.Join(m.history, " "), secret) {
		t.Fatal("secret in history")
	}
	data, err := os.ReadFile(m.Path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), secret) {
		t.Fatal("secret saved to disk")
	}
	m.input.SetValue("/connect spotify should-not-be-in-history")
	press(m, tea.KeyEnter)
	if strings.Contains(strings.Join(m.history, " "), "should-not-be-in-history") {
		t.Fatal("inline credential retained in history")
	}
}
func TestSearchStatesAreDistinct(t *testing.T) {
	missing := searchNotice(0, source.Result{Missing: []string{"spotify"}})
	failed := searchNotice(0, source.Result{Attempted: 1})
	empty := searchNotice(0, source.Result{Attempted: 1, Succeeded: 1})
	if !strings.Contains(missing, "not connected") || !strings.Contains(failed, "failed") || !strings.Contains(empty, "No matching") {
		t.Fatal(missing, failed, empty)
	}
	if strings.Contains(missing, "0 results") || strings.Contains(failed, "0 results") {
		t.Fatal("failure presented as an empty search")
	}
}
func TestCredentialCancelAndResize(t *testing.T) {
	m := fixture(t)
	m.setup = false
	m.stage = "ready"
	m.beginCredential("youtube")
	m.input.SetValue(strings.Repeat("x", 512))
	_, _ = m.Update(tea.WindowSizeMsg{Width: 44, Height: 20})
	v := m.View()
	if v.Cursor == nil || v.Cursor.X >= 44 {
		t.Fatal("credential cursor outside terminal")
	}
	for _, row := range strings.Split(v.Content, "\n") {
		if ansi.StringWidth(row) > 44 {
			t.Fatal("credential frame is too wide")
		}
	}
	press(m, tea.KeyEsc)
	if m.stage != "ready" || m.input.Value() != "" {
		t.Fatal("cancel did not clear secret")
	}
}
func TestOnboarding(t *testing.T) {
	m := fixture(t)
	press(m, tea.KeyEnter)
	press(m, tea.KeyEnter)
	if m.stage != "username" {
		t.Fatal(m.stage)
	}
	m.input.SetValue("echo")
	press(m, tea.KeyEnter)
	press(m, tea.KeyDown)
	press(m, tea.KeyEnter)
	if m.stage != "sources" || m.State.Profile.Theme != "ember" {
		t.Fatal("theme not saved")
	}
	was := m.selected["soundcloud"]
	press(m, tea.KeySpace)
	if was == m.selected["soundcloud"] {
		t.Fatal("space did not toggle")
	}
	for range 3 {
		press(m, tea.KeyDown)
	}
	press(m, tea.KeyEnter)
	for range 12 {
		_, _ = m.Update(pulse(time.Now()))
	}
	saved, err := library.Load(m.Path)
	if err != nil || !saved.Profile.Complete || saved.Profile.Username != "echo" || m.stage != "ready" {
		t.Fatalf("%#v %s %v", saved, m.stage, err)
	}
}
func TestFrameFitsAndInputStaysAtBottom(t *testing.T) {
	for _, size := range [][2]int{{44, 20}, {86, 30}, {120, 40}} {
		for _, stage := range []string{"welcome", "username", "theme", "sources", "ready"} {
			m := fixture(t)
			m.stage = stage
			m.selected = m.State.Profile.Sources
			_, _ = m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
			v := m.View()
			rows := strings.Split(v.Content, "\n")
			if len(rows) > size[1] {
				t.Errorf("%s %v: %d rows", stage, size, len(rows))
			}
			for _, row := range rows {
				if ansi.StringWidth(row) > size[0] {
					t.Errorf("%s %v: row width %d", stage, size, ansi.StringWidth(row))
				}
			}
			if stage == "ready" && (v.Cursor == nil || v.Cursor.Y != size[1]-2) {
				t.Error("input not pinned to bottom")
			}
		}
	}
}
func TestFailedOrCancelledImportIsNotCommitted(t *testing.T) {
	m := fixture(t)
	m.stage = "ready"
	m.operation = 4
	m.busy = true
	_, _ = m.Update(taskResult{ID: 3, Kind: "import", Name: "stale", Tracks: []library.Track{{ID: "x"}}})
	if len(m.State.Playlists) != 0 {
		t.Fatal("stale task committed")
	}
	_, _ = m.Update(taskResult{ID: 4, Kind: "import", Name: "bad", Err: os.ErrPermission, Tracks: []library.Track{{ID: "x"}}})
	if len(m.State.Playlists) != 0 || len(m.State.Tracks) != 0 {
		t.Fatal("failed import committed partial data")
	}
}
func TestM3UAndQueue(t *testing.T) {
	m := fixture(t)
	m.stage = "ready"
	p := filepath.Join(t.TempDir(), "one.wav")
	_ = os.WriteFile(p, []byte("fixture"), 0600)
	ts, err := library.Scan(p)
	if err != nil {
		t.Fatal(err)
	}
	m.show("local", m.State.Put(ts))
	m.dispatch("/queue add 1")
	m.dispatch(`/playlists save "Night"`)
	m.dispatch("/like 1")
	if len(m.State.Queue) != 1 || len(m.State.Playlists["Night"]) != 1 || len(m.State.Likes) != 1 {
		t.Fatal("library commands failed")
	}
	m3u := filepath.Join(filepath.Dir(p), "test.m3u")
	_ = os.WriteFile(m3u, []byte("#EXTM3U\none.wav\none.wav\n"), 0600)
	r, err := readM3U(m3u)
	if err != nil || len(r.Tracks) != 2 {
		t.Fatalf("M3U lost duplicates: %#v %v", r, err)
	}
}
