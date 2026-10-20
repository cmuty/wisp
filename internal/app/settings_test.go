package app

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"wisp/internal/library"
	"wisp/internal/player"
)

func TestButterflySettingPersistsAndInvalidatesAnimation(t *testing.T) {
	m := completionFixture(t)
	m.ReducedMotion = false
	m.dispatch("/settings")
	m.choice = 5
	oldGeneration := m.butterflyGeneration
	press(m, tea.KeyEnter)
	if !m.State.Profile.HideButterfly || m.butterflyTick() != nil {
		t.Fatal("disabled butterfly still schedules animation")
	}
	saved, err := library.Load(m.Path)
	if err != nil || !saved.Profile.HideButterfly {
		t.Fatal("butterfly preference not saved", err)
	}
	reloaded := New(saved, m.Path, player.New())
	if !reloaded.State.Profile.HideButterfly || reloaded.butterflyTick() != nil {
		t.Fatal("butterfly preference lost on restart")
	}
	m.stage = "ready"
	for _, glyph := range ansi.Strip(m.View().Content) {
		if glyph >= 0x2800 && glyph <= 0x28ff {
			t.Fatal("hidden butterfly still rendered")
		}
	}
	m.stage = "settings"
	_, command := m.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	if m.State.Profile.HideButterfly || command == nil {
		t.Fatal("butterfly animation did not restart")
	}
	m.stage = "ready"
	frame := m.butterflyFrame
	_, command = m.Update(butterflyPulse{Generation: oldGeneration})
	if command != nil || m.butterflyFrame != frame {
		t.Fatal("stale animation chain restarted")
	}
}

func TestSourceSelectionHasNoTechnicalSubtitles(t *testing.T) {
	for _, language := range []string{"en", "ru"} {
		view := ansi.Strip(PreviewLanguage("sources", "aurora", false, -1, language))
		for _, label := range []string{"SoundCloud", "Spotify", "YouTube Music"} {
			if !strings.Contains(view, label) {
				t.Fatal("source label missing", label)
			}
		}
		for _, text := range []string{"token", "токен", "audio", "аудио", "metadata"} {
			if strings.Contains(view, text) {
				t.Fatal("technical subtitle remains", text)
			}
		}
	}
}

func TestSettingsKeepsSelectedRowVisibleInSmallTerminal(t *testing.T) {
	m := completionFixture(t)
	m.dispatch("/settings")
	_, _ = m.Update(tea.WindowSizeMsg{Width: 44, Height: 20})
	m.choice = 5
	if !strings.Contains(ansi.Strip(m.View().Content), "› [x] Butterfly") {
		t.Fatal("butterfly setting is clipped")
	}
	press(m, tea.KeyDown)
	if !strings.Contains(ansi.Strip(m.View().Content), "› Done") {
		t.Fatal("Done is clipped")
	}
	press(m, tea.KeyEnter)
	if m.stage != "ready" {
		t.Fatal("could not leave settings")
	}
}
