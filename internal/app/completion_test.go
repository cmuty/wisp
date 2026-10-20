package app

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func completionFixture(t *testing.T) *Model {
	m := fixture(t)
	m.stage = "ready"
	m.setup = false
	return m
}

func typeCompletionText(m *Model, value string) {
	for _, r := range value {
		_, _ = m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
}

func TestCommandHintsAppearBeforeSubmit(t *testing.T) {
	m := completionFixture(t)
	typeCompletionText(m, "/he")
	view := ansi.Strip(m.View().Content)
	if !strings.Contains(view, "/help") || !strings.Contains(view, "Show all commands") {
		t.Fatalf("no live help hint:\n%s", view)
	}
	if len(m.history) != 0 || m.title != "Your listening space" {
		t.Fatal("typing submitted the command")
	}
	press(m, tea.KeyTab)
	if m.input.Value() != "/help" {
		t.Fatalf("Tab inserted %q", m.input.Value())
	}
	if !strings.Contains(ansi.Strip(m.View().Content), "Show all commands") {
		t.Fatal("exact /help lost its live description")
	}
	press(m, tea.KeyEnter)
	if m.title != "A little help" || m.input.Value() != "" {
		t.Fatal("Enter did not execute the completed command")
	}
}

func TestCommandHintSelectionAndArguments(t *testing.T) {
	m := completionFixture(t)
	typeCompletionText(m, "/p")
	items := m.completionItems()
	if len(items) < 2 {
		t.Fatal("expected several /p commands")
	}
	press(m, tea.KeyDown)
	chosen := items[1]
	press(m, tea.KeyTab)
	if m.input.Value() != chosen.Insert {
		t.Fatalf("did not insert selected suggestion: %q", m.input.Value())
	}
	m.input.SetValue("/connect sp")
	m.input.CursorEnd()
	press(m, tea.KeyTab)
	if m.input.Value() != "/connect spotify" {
		t.Fatal(m.input.Value())
	}
	m.input.SetValue("/queue a")
	m.input.CursorEnd()
	press(m, tea.KeyTab)
	if m.input.Value() != "/queue add " {
		t.Fatal(m.input.Value())
	}
	typeCompletionText(m, "12")
	press(m, tea.KeyTab)
	if m.input.Value() != "/queue add 12" {
		t.Fatal("Tab replaced a user argument")
	}
}

func TestCommandHintsEscapeAndExclusions(t *testing.T) {
	m := completionFixture(t)
	typeCompletionText(m, "/he")
	press(m, tea.KeyEsc)
	if m.input.Value() != "/he" || len(m.completionItems()) != 0 {
		t.Fatal("first Esc should hide hints and preserve input")
	}
	typeCompletionText(m, "l")
	if len(m.completionItems()) == 0 {
		t.Fatal("editing did not reopen hints")
	}
	for _, stage := range []string{"credential", "username", "welcome", "theme", "sources", "syncing"} {
		m.stage = stage
		m.input.SetValue("/help")
		m.input.CursorEnd()
		if len(m.completionItems()) != 0 {
			t.Fatalf("completion active in %s", stage)
		}
	}
	m.stage = "ready"
	m.input.SetValue("a normal search")
	m.input.CursorEnd()
	if len(m.completionItems()) != 0 {
		t.Fatal("search text triggered command hints")
	}
	m.input.SetValue("/help")
	m.input.SetCursor(2)
	if len(m.completionItems()) != 0 {
		t.Fatal("completion would replace text while editing in the middle")
	}
}

func TestCommandHintsStayInsideTerminal(t *testing.T) {
	for _, size := range [][2]int{{44, 20}, {86, 30}, {120, 40}} {
		m := completionFixture(t)
		typeCompletionText(m, "/")
		_, _ = m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		for range 9 {
			press(m, tea.KeyDown)
		}
		v := m.View()
		rows := strings.Split(v.Content, "\n")
		if len(rows) != size[1] {
			t.Fatalf("%v: frame has %d rows", size, len(rows))
		}
		if v.Cursor == nil || v.Cursor.Y != size[1]-2 {
			t.Fatalf("%v: input moved from bottom", size)
		}
		for _, row := range rows {
			if ansi.StringWidth(row) > size[0] {
				t.Fatal("hint overflowed terminal width")
			}
		}
		if len(m.completionRows(FindTheme(m.State.Profile.Theme), size[0]-4)) > 4 {
			t.Fatal("too many hint rows")
		}
	}
}
