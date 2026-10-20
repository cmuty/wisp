package app

import tea "charm.land/bubbletea/v2"

type seekResult struct {
	TrackID string
	Err     error
}

func seekDelta(k tea.KeyPressMsg) int {
	if k.Mod & ^tea.ModShift != 0 {
		return 0
	}
	direction := 0
	switch k.Code {
	case tea.KeyLeft:
		direction = -1
	case tea.KeyRight:
		direction = 1
	}
	step := 5
	if k.Mod&tea.ModShift != 0 {
		step = 1
	}
	return direction * step
}
func (m *Model) seekRelative(delta int) tea.Cmd {
	if m.current.ID == "" || !m.Engine.Ready() || (m.busy && (m.taskKind == "play" || m.taskKind == "stop")) {
		return nil
	}
	m.pendingSeek = max(-86400, min(86400, m.pendingSeek+delta))
	if m.seekBusy {
		return nil
	}
	return m.sendSeek()
}
func (m *Model) sendSeek() tea.Cmd {
	delta := m.pendingSeek
	m.pendingSeek = 0
	m.seekBusy = true
	id := m.current.ID
	return func() tea.Msg { return seekResult{TrackID: id, Err: m.Engine.Command("seek", delta, "relative+exact")} }
}

// Copy the visible continuation so later searches cannot change the playing queue.
// The replacement is committed only after the selected recording opens successfully.
func (m *Model) playFromResults(index int) tea.Cmd {
	if index < 0 || index >= len(m.results) {
		return nil
	}
	next := append([]string{}, m.results[index+1:]...)
	cmd := m.play(m.results[index])
	if cmd == nil {
		return nil
	}
	return func() tea.Msg { r := cmd().(taskResult); r.ReplaceQueue = true; r.NextQueue = next; return r }
}
