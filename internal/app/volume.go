package app

import (
	tea "charm.land/bubbletea/v2"
	"wisp/internal/player"
)

type volumeResult struct {
	Value int
	Err   error
}

// Printable variants keep legacy consoles usable when they omit Shift metadata.
func volumeDelta(key tea.KeyPressMsg) int {
	if key.Mod & ^tea.ModShift != 0 {
		return 0
	}
	code := key.Code
	if key.Text == "_" {
		code = '_'
	}
	direction := 0
	switch code {
	case '+', '=', tea.KeyKpPlus, tea.KeyKpEqual:
		direction = 1
	case '-', '−', '_', tea.KeyKpMinus:
		direction = -1
	}
	if direction == 0 {
		return 0
	}
	step := 5
	if key.Mod&tea.ModShift != 0 || code == '_' {
		step = 1
	}
	return direction * step
}
func (m *Model) setVolume(value int) tea.Cmd {
	m.volume = max(0, min(player.MaxVolume, value))
	if m.volumeBusy {
		return nil
	}
	return m.sendVolume()
}

// Coalesce repeated key presses while IPC is pending; other player tasks keep running.
func (m *Model) sendVolume() tea.Cmd {
	m.volumeBusy = true
	value := m.volume
	return func() tea.Msg { return volumeResult{Value: value, Err: m.Engine.SetVolume(value)} }
}
