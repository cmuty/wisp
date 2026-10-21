package app

import (
	"strings"
	"testing"
	"wisp/internal/mediakeys"
)

func TestMediaTransportAndPreviousQueue(t *testing.T) {
	m := playbackFixture(t)
	_, _ = m.Update(m.playFromResults(0)())
	m.move(2)
	m.input.SetValue("unfinished search")
	apply := func(action mediakeys.Action) {
		t.Helper()
		_, cmd := m.Update(action)
		if cmd == nil {
			t.Fatal("media action ignored", action)
		}
		result := cmd()
		if r, ok := result.(taskResult); ok && r.Err != nil {
			t.Fatal(r.Err)
		}
		_, _ = m.Update(result)
		if m.input.Value() != "unfinished search" {
			t.Fatal("media key modified input")
		}
	}
	apply(mediakeys.TogglePause)
	if m.current.ID != "a" || m.index != 2 {
		t.Fatal("pause followed list selection")
	}
	apply(mediakeys.Next)
	if m.current.ID != "b" || strings.Join(m.State.Queue, ",") != "c" {
		t.Fatal("next broke queue")
	}
	apply(mediakeys.Previous)
	if m.current.ID != "a" || strings.Join(m.State.Queue, ",") != "b,c" || len(m.playHistory) != 0 {
		t.Fatal("previous broke continuation", m.State.Queue, m.playHistory)
	}
	apply(mediakeys.Next)
	if m.current.ID != "b" {
		t.Fatal("next did not return to recording after previous")
	}
	apply(mediakeys.Stop)
	if m.current.ID != "" {
		t.Fatal("media stop did not stop")
	}
}

func TestFailedPreviousPreservesHistoryAndQueue(t *testing.T) {
	m := completionFixture(t)
	m.playHistory = []string{"missing"}
	m.State.Queue = []string{"future"}
	if cmd := m.playPrevious(); cmd != nil {
		t.Fatal("missing track should not dispatch")
	}
	if len(m.playHistory) != 1 || m.State.Queue[0] != "future" {
		t.Fatal("failed previous discarded state")
	}
	m.busy = true
	m.taskKind = "search"
	for i := 0; i < 20; i++ {
		m.Update(mediakeys.Next)
	}
	if len(m.mediaPending) != 8 || m.taskKind != "search" {
		t.Fatal("media event cancelled search or unbounded buffering")
	}
}
