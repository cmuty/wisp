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
)

func TestHeaderHasSectionsBesideUsername(t *testing.T) {
	m := completionFixture(t)
	m.State.Profile.Username = "emotions"
	for _, width := range []int{44, 86, 120} {
		_, _ = m.Update(tea.WindowSizeMsg{Width: width, Height: 30})
		rows := strings.Split(ansi.Strip(m.View().Content), "\n")
		if width > 60 && !strings.Contains(rows[2], "Recent") || strings.TrimSpace(rows[7]) != "" {
			t.Fatal("sections not in header", rows[:2])
		}
		if ansi.StringWidth(rows[0]) > width {
			t.Fatal("header overflow")
		}
	}
	m.State.Profile.Username = strings.Repeat("я", 32)
	if ansi.StringWidth(m.header(FindTheme("mist"), 40)) > 40 {
		t.Fatal("long username overflow")
	}
}
func TestSeekKeysPreserveTypingAndOperations(t *testing.T) {
	for _, v := range []struct {
		k    tea.KeyPressMsg
		want int
	}{{tea.KeyPressMsg{Code: tea.KeyLeft}, -5}, {tea.KeyPressMsg{Code: tea.KeyRight}, 5}, {tea.KeyPressMsg{Code: tea.KeyLeft, Mod: tea.ModShift}, -1}, {tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModShift}, 1}, {tea.KeyPressMsg{Code: tea.KeyLeft, Mod: tea.ModCtrl}, 0}} {
		if seekDelta(v.k) != v.want {
			t.Fatal(v)
		}
	}
	m := completionFixture(t)
	m.input.SetValue("query")
	m.input.CursorEnd()
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	if m.input.Position() != 4 || m.input.Value() != "query" || m.seekBusy {
		t.Fatal("arrow did not edit input", cmd)
	}
	m.operation = 8
	m.busy = true
	m.taskKind = "search"
	m.seekBusy = true
	m.pendingSeek = 6
	_, cmd = m.Update(seekResult{TrackID: m.current.ID})
	if cmd == nil || m.operation != 8 || !m.busy || m.pendingSeek != 0 {
		t.Fatal("seek lost or cancelled search")
	}
}
func TestFailedSelectionKeepsQueue(t *testing.T) {
	m := completionFixture(t)
	m.results = m.State.Put([]library.Track{{ID: "bad", Source: "local", Path: filepath.Join(t.TempDir(), "missing.wav")}, {ID: "next"}})
	m.State.Queue = []string{"old"}
	cmd := m.playFromResults(0)
	m.results = []string{"unrelated"}
	r := cmd().(taskResult)
	if !r.ReplaceQueue || strings.Join(r.NextQueue, ",") != "next" || r.Err == nil {
		t.Fatal("continuation not captured", r)
	}
	_, _ = m.Update(r)
	if strings.Join(m.State.Queue, ",") != "old" {
		t.Fatal("failed play overwrote queue")
	}
}

func TestAutomaticPlaybackFollowsCurrentWithoutStealingBrowsing(t *testing.T) {
	for _, browsing := range []bool{false, true} {
		m := completionFixture(t)
		m.results = m.State.Put([]library.Track{{ID: "a", Title: "First"}, {ID: "b", Title: "Second"}, {ID: "c", Title: "Third"}})
		m.current = m.State.Tracks["a"]
		if browsing {
			m.move(2)
		}
		_, _ = m.Update(taskResult{Kind: "play", Track: m.State.Tracks["b"]})
		want := 1
		if browsing {
			want = 2
		}
		if m.index != want || m.selectionPending != browsing {
			t.Fatalf("browsing=%v: cursor=%d pending=%v", browsing, m.index, m.selectionPending)
		}
		body := ansi.Strip(strings.Join(m.body(FindTheme("mist"), 100, 20), "\n"))
		if !strings.Contains(body, ">   2") || strings.Contains(body, ">   1") {
			t.Fatal("current recording is not marked independently of cursor", body)
		}
		m.snapshot.Paused = true
		body = ansi.Strip(strings.Join(m.body(FindTheme("mist"), 100, 20), "\n"))
		if !strings.Contains(body, "=   2") {
			t.Fatal("paused recording is not marked", body)
		}
	}
}

func TestEnterPausesAfterQueueMovesOutsideVisibleResults(t *testing.T) {
	m := playbackFixture(t)
	cmd := m.playFromResults(0)
	_, _ = m.Update(cmd())
	m.State.Queue = []string{"b"}
	cmd = m.playNext()
	m.results = []string{"a"}
	_, _ = m.Update(cmd())
	_, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil || m.taskKind != "control" || m.current.ID != "b" {
		t.Fatal("Enter restarted stale selection instead of pausing", m.taskKind, m.current.ID)
	}
	_, _ = m.Update(cmd())
	m.move(1)
	m.results = []string{"a", "b", "c"}
	m.move(2)
	_, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil || m.taskKind != "play" {
		t.Fatal("deliberate selection no longer starts a recording")
	}
}

func TestPauseShortcutsIgnoreSelectionAndPreserveInput(t *testing.T) {
	m := playbackFixture(t)
	_, _ = m.Update(m.playFromResults(0)())
	m.move(2)
	for _, key := range []tea.KeyPressMsg{{Code: tea.KeySpace, Text: " "}, {Code: tea.KeyF4}} {
		_, cmd := m.Update(key)
		if cmd == nil || m.taskKind != "control" || m.index != 2 || m.current.ID != "a" || m.input.Value() != "" {
			t.Fatal("pause shortcut used selection", m.taskKind, m.index, m.current.ID)
		}
		_, _ = m.Update(cmd())
	}
	m.input.SetValue("my")
	m.input.CursorEnd()
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	if m.input.Value() != "my " || m.busy {
		t.Fatal("space in search triggered pause", m.input.Value())
	}
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyF4})
	if cmd == nil || m.taskKind != "control" || m.input.Value() != "my " || m.index != 2 {
		t.Fatal("F4 changed input or selected recording")
	}
	_, _ = m.Update(cmd())
}
func playbackFixture(t *testing.T) *Model {
	t.Helper()
	mpv := os.Getenv("WISP_TEST_MPV")
	if mpv == "" {
		t.Skip("set WISP_TEST_MPV for playback integration")
	}
	t.Setenv("WISP_MPV", mpv)
	t.Setenv("WISP_AUDIO_OUTPUT", "null")
	m := completionFixture(t)
	t.Cleanup(m.Engine.Close)
	data := make([]byte, 44+8000*2*30)
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
	for _, id := range []string{"a", "b", "c"} {
		path := filepath.Join(t.TempDir(), id+".wav")
		if e := os.WriteFile(path, data, 0600); e != nil {
			t.Fatal(e)
		}
		track, e := library.Local(path)
		if e != nil {
			t.Fatal(e)
		}
		track.ID = id
		m.State.Put([]library.Track{track})
	}
	m.results = []string{"a", "b", "c"}
	return m
}
func TestRealSeekAndAutomaticQueue(t *testing.T) {
	m := playbackFixture(t)
	cmd := m.activate()
	_, _ = m.Update(cmd())
	if m.current.ID != "a" || strings.Join(m.State.Queue, ",") != "b,c" {
		t.Fatal("queue not populated", m.message, m.State.Queue)
	}
	wait := func(fn func() bool) {
		t.Helper()
		end := time.Now().Add(3 * time.Second)
		for !fn() && time.Now().Before(end) {
			time.Sleep(15 * time.Millisecond)
		}
		if !fn() {
			t.Fatal("player did not reach expected state", m.Engine.Snapshot())
		}
	}
	wait(func() bool { return m.Engine.Snapshot().Duration > 25 })
	if e := m.Engine.Command("set_property", "pause", true); e != nil {
		t.Fatal(e)
	}
	if e := m.Engine.Command("seek", 10, "absolute+exact"); e != nil {
		t.Fatal(e)
	}
	// Paused audio time-pos can trail the exact seek point by one output buffer.
	wait(func() bool { p := m.Engine.Snapshot().Position; return p > 9.5 && p < 10.5 })
	beforeSeek := m.Engine.Snapshot().Position
	_, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	if cmd == nil {
		t.Fatal("seek shortcut ignored")
	}
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyLeft, Mod: tea.ModShift})
	_, next := m.Update(cmd())
	if next == nil {
		t.Fatal("repeated seek dropped")
	}
	_, _ = m.Update(next())
	// The paused output buffer can lag again after each of the two seek commands.
	wait(func() bool { p := m.Engine.Snapshot().Position; return p > beforeSeek+3.25 && p < beforeSeek+4.75 })
	// Finish the file through mpv; the regular pulse must load the next queue entry.
	if e := m.Engine.Command("seek", 29.9, "absolute+exact"); e != nil {
		t.Fatal(e)
	}
	if e := m.Engine.Command("set_property", "pause", false); e != nil {
		t.Fatal(e)
	}
	wait(func() bool { return m.Engine.Snapshot().End > 0 })
	_, cmd = m.Update(pulse(time.Now()))
	if cmd == nil || !m.busy {
		t.Fatal("EOF did not continue queue")
	}
	batch, ok := cmd().(tea.BatchMsg)
	if !ok {
		t.Fatal("expected tick and playback batch")
	}
	for _, run := range batch {
		if run == nil {
			continue
		}
		msg := run()
		if r, ok := msg.(taskResult); ok {
			_, _ = m.Update(r)
		}
	}
	if m.current.ID != "b" || strings.Join(m.State.Queue, ",") != "c" {
		t.Fatal("wrong next track", m.current.ID, m.State.Queue, m.message)
	}
	if m.index != 1 {
		t.Fatal("cursor did not follow automatic queue transition", m.index)
	}
	_, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil || m.taskKind != "control" {
		t.Fatal("Enter restarted original recording after EOF")
	}
	_, _ = m.Update(cmd())
	wait(func() bool { return m.Engine.Snapshot().Paused })
	if m.current.ID != "b" || strings.Join(m.State.Queue, ",") != "c" {
		t.Fatal("pause changed current recording or queue")
	}
	saved, e := library.Load(m.Path)
	if e != nil || strings.Join(saved.Queue, ",") != "c" {
		t.Fatal("queue not persisted", e)
	}
}
