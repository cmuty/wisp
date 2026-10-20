package app

import (
	"github.com/charmbracelet/x/ansi"
	"strings"
	"testing"
)

func TestHeaderLogoReplacesWordmark(t *testing.T) {
	m := completionFixture(t)
	m.State.Profile.Username = "emotions"
	for _, theme := range Themes {
		m.State.Profile.Theme = theme.ID
		raw := m.header(theme, 100)
		text := ansi.Strip(raw)
		if strings.Contains(text, "wisp") || strings.Contains(text, " / ") || !strings.Contains(text, "@emotions") {
			t.Fatal("old wordmark remains", text)
		}
		if ansi.StringWidth(headerLogo(theme, false)) != headerLogoWidth || strings.Contains(raw, "\n") {
			t.Fatal("logo changed header size")
		}
	}
	if headerLogo(FindTheme("aurora"), false) == headerLogo(FindTheme("afterglow"), false) {
		t.Fatal("theme colors identical")
	}
	m.Plain = true
	if !strings.Contains(ansi.Strip(m.header(FindTheme("mist"), 80)), "@emotions") {
		t.Fatal("ASCII fallback")
	}
}
