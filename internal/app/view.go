package app

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"wisp/internal/library"
	"wisp/internal/source"
)

func (m *Model) View() tea.View {
	t := FindTheme(m.State.Profile.Theme)
	w, h := m.width, m.height
	if w < 44 || h < 20 {
		v := tea.NewView(m.tr("wisp needs a terminal of at least 44 × 20.\nResize this window. Ctrl+C exits."))
		v.WindowTitle = "wisp"
		v.AltScreen = true
		v.MouseMode = tea.MouseModeCellMotion
		return v
	}
	width := w - 4
	var lines []string
	if m.stage != "ready" && m.setup {
		lines = m.wizard(t, width)
		inputRow := -1
		if m.stage == "username" {
			inputRow = len(lines)
			lines = append(lines, m.input.View())
		}
		for len(lines) < h-2 {
			lines = append(lines, "")
		}
		lines = lines[:min(h-2, len(lines))]
		lines = append([]string{""}, lines...)
		for i := range lines {
			lines[i] = "  " + ansi.Truncate(lines[i], width, "")
		}
		v := tea.NewView(strings.Join(lines, "\n"))
		v.WindowTitle = "wisp"
		v.AltScreen = true
		v.MouseMode = tea.MouseModeCellMotion
		if inputRow >= 0 {
			c := m.input.Cursor()
			if c != nil {
				c.X += 2
				c.Y = inputRow + 1
				v.Cursor = c
			}
		}
		return v
	}
	lines = append(lines, m.headerRows(t, width)...)
	headerHeight := len(lines)
	completionRows := m.completionRows(t, width)
	bodyHeight := h - 7 - headerHeight - len(completionRows)
	contentHeight := bodyHeight
	if m.stage == "ready" {
		contentHeight -= 2
	}
	body := m.body(t, width, contentHeight)
	if m.tab == 1 && !m.playlistOpen {
		body = m.playlistBody(t, contentHeight)
	}
	if m.lyricsOpen {
		body = m.lyricsBody(t, width, contentHeight)
	}
	if m.stage == "language" {
		body = m.languageBody(t)
	}
	if m.stage == "settings" {
		body = m.settingsBody(t)
	}
	if m.stage == "theme" {
		body = []string{style(t.Text).Bold(true).Render(m.tr("Choose the color of your quiet.")), ""}
		for i, theme := range Themes {
			prefix := "  "
			color := t.Muted
			if i == m.choice {
				prefix = "› "
				color = t.Accent
			}
			body = append(body, style(color).Render(prefix+m.tr(theme.Name)+"   "+m.tr(theme.Description)))
		}
	}
	if m.stage == "sources" {
		body = []string{style(t.Text).Bold(true).Render(m.tr("Search sources")), ""}
		for i, n := range source.Names {
			prefix := "  "
			check := "[ ]"
			color := t.Muted
			if m.selected[n] {
				check = "[x]"
			}
			if i == m.choice {
				prefix = "› "
				color = t.Accent
			}
			body = append(body, style(color).Render(prefix+check+" "+source.Labels[i]+"   "+m.tr(m.Sources.Status(n))))
		}
		body = append(body, "")
		color := t.Muted
		if m.choice == len(source.Names) {
			color = t.Accent
		}
		body = append(body, style(color).Render(m.tr("› Continue")), "", style(t.Muted).Render(m.tr("Local files are always available.")))
	}
	if m.stage == "credential" {
		body = []string{style(t.Text).Bold(true).Render(m.tr("Connect " + source.Label(m.credentialSource))), "", style(t.Muted).Render(m.tr("Paste a credential from your registered API app.")), style(t.Muted).Render(m.tr("This is not your account password.")), "", style(t.Muted).Render(m.tr("Input is hidden and never added to command history.")), style(t.Muted).Render(m.tr("Kept in memory for this session; not saved to disk.")), "", style(t.Second).Render(m.tr(m.message))}
	}
	body = body[:min(len(body), contentHeight)]
	for len(body) < bodyHeight {
		body = append(body, "")
	}
	lines = append(lines, body[:bodyHeight]...)
	border := style(t.Border)
	tl, tr, bl, br, hor, vert := "╭", "╮", "╰", "╯", "─", "│"
	if m.Plain {
		tl = "+"
		tr = "+"
		bl = "+"
		br = "+"
		hor = "-"
		vert = "|"
	}
	inner := width - 4
	name := m.tr("Nothing playing")
	state := "idle"
	if m.current.ID != "" {
		name = m.current.Artist + " — " + m.current.Title
		state = "playing"
		if m.snapshot.Paused {
			state = "paused"
		}
		if !m.snapshot.Active {
			state = "loading"
		}
	}
	now := gradientText(t, m.tr(state)) + "  " + style(t.Text).Render(library.Clean(name))
	lines = append(lines, progressSegment(t, tl, 0, 1, width, .5)+progressSegment(t, hor, 1, width-2, width, .5)+progressSegment(t, tr, width-1, 1, width, .5), border.Render(vert)+" "+pad(now, inner)+" "+border.Render(vert))
	barWidth := max(4, inner-24)
	position, duration := m.snapshot.Position, m.snapshot.Duration
	if m.current.ID == "" {
		position = 0
		duration = 0
	}
	ratio := 0.0
	if duration > 0 {
		ratio = position / duration
	}
	ratio = max(0.0, min(1.0, ratio))
	filled := int(float64(barWidth) * ratio)
	solid, empty := "━", "━"
	if m.Plain {
		solid = "="
		empty = "-"
	}
	bar := progressSegment(t, solid, 0, filled, barWidth, 1) + progressSegment(t, empty, filled, barWidth-filled, barWidth, .27) + style(t.Muted).Render("  "+clock(position)+" / "+clock(duration)+fmt.Sprintf("  %d%%", m.volume))
	lines = append(lines, border.Render(vert)+" "+pad(bar, inner)+" "+border.Render(vert), progressSegment(t, bl, 0, 1, width, .5)+progressSegment(t, hor, 1, width-2, width, .5)+progressSegment(t, br, width-1, 1, width, .5), "")
	lines = append(lines, completionRows...)
	lines = append(lines, m.input.View())
	hint := "Enter play/pause · ←→ seek · Tab tabs · F2 lyrics"
	if len(m.results) == 0 {
		hint = "Search music   /import a link   /help"
	}
	if m.lyricsOpen {
		hint = "F2 back · F5 retry · Home follow · wheel browse"
	}
	if m.busy {
		hint = m.spinner() + " working   Esc cancels"
		if m.taskKind == "play" || m.taskKind == "control" || m.taskKind == "stop" {
			hint = m.spinner() + " opening audio"
		}
	}
	if !m.sleep.IsZero() {
		hint = m.tr(hint)
		hint += "   sleep " + time.Until(m.sleep).Round(time.Second).String()
	}
	if m.match != "" {
		hint = "Choose the right recording, then /bind n"
	}
	if m.stage == "theme" || m.stage == "sources" || m.stage == "settings" {
		hint = "↑↓ move   Enter apply / toggle   Esc back"
	}
	if m.stage == "language" {
		hint = "↑ ↓ · Enter — select / выбрать"
	}
	if m.stage == "credential" {
		hint = "Enter use credential   Esc cancel"
	}
	if len(completionRows) > 0 {
		hint = "Tab complete   ↑↓ choose   Enter run   Esc hide"
	}
	if m.saveError != "" {
		hint = m.saveError
	}
	lines = append(lines, style(t.Muted).Render(m.tr(hint)))
	m.drawButterfly(lines, t, width, bodyHeight+headerHeight)
	for i := range lines {
		lines[i] = "  " + ansi.Truncate(lines[i], width, "")
	}
	v := tea.NewView(strings.Join(lines, "\n"))
	v.WindowTitle = "wisp"
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	c := m.input.Cursor()
	if c != nil && (m.stage == "ready" || m.stage == "credential") {
		c.X += 2
		c.Y = h - 2
		v.Cursor = c
	}
	return v
}
func (m *Model) wizard(t Theme, width int) []string {
	lines := []string{"", LogoViewSize(t, m.Plain, width, max(5, m.height-14)), ""} // Split the logo before calculating cursor rows.
	lines = strings.Split(strings.Join(lines, "\n"), "\n")
	switch m.stage {
	case "language":
		lines = append(lines, m.languageBody(t)...)
	case "welcome":
		lines = append(lines, style(t.Text).Bold(true).Render(m.tr("Welcome to wisp.")), style(t.Muted).Render(m.tr("A little room for sound.")), "", style(t.Muted).Render(m.tr("No wisp account needed. Your profile stays on this device.")), "", style(t.Accent).Render(m.tr("Press Enter to find your frequency.")))
	case "username":
		lines = append(lines, style(t.Text).Bold(true).Render(m.tr("What name should echo here?")), style(t.Muted).Render(m.tr("A name for your listening space. You can change it later.")), "", style(t.Muted).Render(m.tr(m.message)), "")
	case "theme":
		lines = append(lines, style(t.Text).Bold(true).Render(m.tr("Choose the color of your quiet.")), style(t.Muted).Render(m.tr("↑ ↓ to preview · Enter to keep")), "")
		for i, v := range Themes {
			prefix := "  "
			if i == m.choice {
				prefix = "› "
			}
			row := prefix + m.tr(v.Name) + "   " + m.tr(v.Description)
			if i == m.choice {
				lines = append(lines, style(t.Accent).Render(row))
			} else {
				lines = append(lines, style(t.Muted).Render(row))
			}
		}
	case "sources":
		// Use a smaller heading to keep the connection details visible at 20 rows.
		lines = []string{"", style(t.Accent).Bold(true).Render("wisp / " + m.State.Profile.Username), "", style(t.Text).Bold(true).Render(m.tr("Where should we look for your next track?")), style(t.Muted).Render(m.tr("↑↓ · Space toggle · Enter on Continue")), ""}
		for i, n := range source.Names {
			check := "[ ]"
			if m.selected[n] {
				check = "[x]"
			}
			prefix := "  "
			if m.choice == i {
				prefix = "› "
			}
			color := t.Muted
			if m.choice == i {
				color = t.Accent
			}
			lines = append(lines, style(color).Render(prefix+check+" "+source.Labels[i]), style(t.Muted).Render("      "+m.tr(m.Sources.Status(n))), "")
		}
		color := t.Muted
		if m.choice == len(source.Names) {
			color = t.Accent
		}
		lines = append(lines, style(color).Render(m.tr("› Continue")), "", style(t.Muted).Render(m.tr("Local files are always available. Selection does not sign you in.")))
	case "syncing":
		phase := "preparing your library…"
		if m.frame > 4 {
			phase = "checking source settings…"
		}
		if m.frame > 8 {
			phase = "saving your listening space…"
		}
		lines = append(lines, style(t.Accent).Render(m.spinner()+" "+m.tr(phase)), "", style(t.Muted).Render(m.tr("Your files stay yours. Your queue starts here.")))
	}
	for i := range lines {
		lines[i] = ansi.Truncate(lines[i], width, "")
	}
	return lines
}
func (m *Model) body(t Theme, width, height int) []string {
	lines := []string{style(t.Text).Bold(true).Render(m.displayTitle()), ""}
	if len(m.results) == 0 {
		rows := strings.Split(ansi.Wrap(m.tr(m.message), width, ""), "\n")
		m.pageOffset = min(m.pageOffset, max(0, len(rows)-(height-2)))
		for _, row := range rows[m.pageOffset:] {
			lines = append(lines, style(t.Muted).Render(row))
		}
	} else {
		limit := max(1, height-5)
		if m.index < m.offset {
			m.offset = m.index
		}
		if m.index >= m.offset+limit {
			m.offset = m.index - limit + 1
		}
		for i := m.offset; i < min(len(m.results), m.offset+limit); i++ {
			track := m.State.Tracks[m.results[i]]
			prefix := "  "
			color := t.Text
			if i == m.index {
				prefix = "› "
				color = t.Accent
			}
			label := track.Source
			if track.Unavailable {
				label += m.tr(" / unavailable")
			} else if target := m.State.Resolve(track.ID); target.ID != track.ID {
				label += " → " + target.Source
			} else if track.Source == "spotify" && m.Sources.PublicWeb {
				label += " → YouTube"
			} else if (track.Source == "spotify" || track.Source == "youtube") && !m.Sources.PublicWeb {
				label += m.tr(" / metadata")
			}
			suffix := style(t.Muted).Render("  [" + label + "]")
			text := fmt.Sprintf("%s%3d  %s — %s", prefix, i+1, track.Artist, track.Title)
			available := max(8, width-lipgloss.Width(suffix))
			rendered := style(color).Render(ansi.Truncate(text, available, "…"))
			if i == m.index {
				rendered = gradientText(t, ansi.Truncate(text, available, "…"))
			}
			lines = append(lines, rendered+suffix)
		}
		lines = append(lines, "", style(t.Muted).Render(ansi.Truncate(m.tr(m.message), width, "…")))
	}
	for _, warning := range m.warnings {
		if len(lines) >= height {
			break
		}
		lines = append(lines, style(t.Second).Render(ansi.Truncate(m.tr(warning), width, "…")))
	}
	return lines
}
func (m *Model) spinner() string {
	if m.ReducedMotion {
		return "·"
	}
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	if m.Plain {
		return []string{"|", "/", "-", "\\"}[m.frame%4]
	}
	return frames[m.frame%len(frames)]
}
func pad(s string, width int) string {
	s = ansi.Truncate(s, width, "")
	return s + strings.Repeat(" ", max(0, width-lipgloss.Width(s)))
}
func clock(n float64) string {
	if n < 0 {
		n = 0
	}
	seconds := int(n)
	return fmt.Sprintf("%02d:%02d", seconds/60, seconds%60)
}
