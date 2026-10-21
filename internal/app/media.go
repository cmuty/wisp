package app

import (
	tea "charm.land/bubbletea/v2"
	"context"
	"wisp/internal/mediakeys"
)

type MediaKeysUnavailable struct{ Err error }

func (m *Model) handleMediaKey(action mediakeys.Action) tea.Cmd {
	if m.stage != "ready" {
		return nil
	}
	if m.busy {
		if len(m.mediaPending) < 8 {
			m.mediaPending = append(m.mediaPending, action)
		}
		return nil
	}
	switch action {
	case mediakeys.TogglePause:
		if m.current.ID != "" && m.Engine.Ready() {
			return m.control("cycle", "pause")
		}
		if len(m.State.Queue) > 0 {
			return m.playNext()
		}
	case mediakeys.Next:
		return m.playNext()
	case mediakeys.Previous:
		return m.playPrevious()
	case mediakeys.Stop:
		if !m.Engine.Ready() {
			return nil
		}
		return m.task("stop", func(context.Context) taskResult { return taskResult{Err: m.Engine.Command("stop")} })
	}
	return nil
}

func (m *Model) playPrevious() tea.Cmd {
	if len(m.playHistory) == 0 {
		return nil
	}
	id := m.playHistory[len(m.playHistory)-1]
	next := append([]string{}, m.State.Queue...)
	if m.current.ID != "" {
		next = append([]string{m.current.ID}, next...)
	}
	cmd := m.play(id)
	if cmd == nil {
		return nil
	}
	return func() tea.Msg {
		r := cmd().(taskResult)
		r.Previous, r.ReplaceQueue, r.NextQueue = true, true, next
		return r
	}
}
