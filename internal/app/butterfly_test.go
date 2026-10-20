package app

import (
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"strings"
	"testing"
	"wisp/internal/library"
	"wisp/internal/lyrics"
)

func TestButterflyFlightStaysAbovePlayer(t *testing.T) {
	for _, width := range []int{40, 82, 116} {
		for _, top := range []int{3, 8, 16} {
			border := 24
			seenFlight, seenWings := false, map[int]bool{}
			for frame := 0; frame < butterflyCycleFrames*3; frame++ {
				p := butterflyPosition(frame, width, top, border, false)
				if p.X < 0 || p.X+butterflyWidth > width || p.Y < top || p.Y+butterflyHeight > border {
					t.Fatal("out of bounds", width, top, frame, p)
				}
				if !p.Perched {
					seenFlight = true
				}
				seenWings[p.Wing] = true
			}
			if !seenFlight || len(seenWings) != 4 {
				t.Fatal("missing animation")
			}
			for _, frame := range []int{0, 32, 90, 175, 500} {
				p := butterflyPosition(frame, width, top, border, true)
				if p != butterflyPosition(0, width, top, border, true) || !p.Perched {
					t.Fatal("reduced motion moved")
				}
			}
		}
	}
}
func TestButterflyDoesNotCoverTextOrInput(t *testing.T) {
	m := completionFixture(t)
	m.current = library.Track{ID: "fixture", Artist: "Artist", Title: "Song"}
	m.lyricsOpen = true
	m.lyricsFollow = true
	m.snapshot.Active = true
	m.lyricsSong = lyrics.Song{Credit: "LRCLIB", Lines: []lyrics.Line{{Time: 0, Text: "An invented lyric line"}}}
	m.input.SetValue("typed search")
	for _, size := range [][2]int{{44, 20}, {86, 30}, {120, 40}} {
		_, _ = m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		for frame := 0; frame < butterflyCycleFrames; frame += 7 {
			m.butterflyFrame = frame
			v := m.View()
			rows := strings.Split(ansi.Strip(v.Content), "\n")
			if len(rows) != size[1] || v.Cursor == nil || v.Cursor.Y != size[1]-2 {
				t.Fatal("layout moved")
			}
			if !strings.Contains(rows[size[1]-2], "typed search") || !strings.Contains(ansi.Strip(v.Content), "An invented lyric line") {
				t.Fatal("covered content")
			}
			for _, row := range rows {
				if ansi.StringWidth(row) > size[0] {
					t.Fatal("overflow")
				}
			}
			if strings.Contains(v.Content, "LRCLIB") || strings.Contains(v.Content, "synced") {
				t.Fatal("provider footer visible")
			}
		}
	}
}
func TestButterflyUsesSideSpaceAndTurns(t *testing.T) {
	lines := make([]string, 24)
	for i := 2; i < 20; i++ {
		lines[i] = "Artist — a track [soundcloud]"
	}
	area, ok := butterflyFreeArea(lines, 100, 22)
	if !ok || area.Top > 3 || area.Left < 30 {
		t.Fatal("flight confined below list", area)
	}
	minX, maxX, minY, maxY := 1000, 0, 1000, 0
	turnsX, turnsY, lastDX, lastDY := 0, 0, 0, 0
	previous := butterflyPosition(butterflyRestFrames, area.Right-area.Left, area.Top, area.Bottom, false)
	for frame := butterflyRestFrames + 1; frame < butterflyCycleFrames; frame++ {
		p := butterflyPosition(frame, area.Right-area.Left, area.Top, area.Bottom, false)
		minX = min(minX, p.X)
		maxX = max(maxX, p.X)
		minY = min(minY, p.Y)
		maxY = max(maxY, p.Y)
		dx, dy := p.X-previous.X, p.Y-previous.Y
		if dx != 0 {
			if lastDX != 0 && dx*lastDX < 0 {
				turnsX++
			}
			lastDX = dx
		}
		if dy != 0 {
			if lastDY != 0 && dy*lastDY < 0 {
				turnsY++
			}
			lastDY = dy
		}
		previous = p
	}
	if maxX-minX < 15 || maxY-minY < 8 || turnsX < 2 || turnsY < 2 {
		t.Fatal("route has no loops", maxX-minX, maxY-minY, turnsX, turnsY)
	}
	for _, width := range []int{butterflyWidth, butterflyWidth + 1, 40} {
		for _, frame := range []int{0, 59, 60, 120, 599} {
			p := butterflyPosition(frame, width, 2, 20, false)
			if p.X < 0 || p.X+butterflyWidth > width {
				t.Fatal("narrow area overflow", p, width)
			}
		}
	}
}

func TestButterflyFacesTravelDirection(t *testing.T) {
	for _, v := range []struct {
		dx, dy float64
		want   int
	}{{0, -1, 0}, {2, -1, 1}, {2, 0, 2}, {2, 1, 3}, {0, 1, 4}, {-2, 1, 5}, {-2, 0, 6}, {-2, -1, 7}} {
		if got := butterflyHeading(v.dx, v.dy, 1); got != v.want {
			t.Fatal("wrong facing", v, got)
		}
	}
	headings := map[int]bool{}
	sprites := map[string]bool{}
	for frame := butterflyRestFrames; frame < butterflyCycleFrames; frame++ {
		p := butterflyPosition(frame, 80, 2, 26, false)
		headings[p.Heading] = true
		rows := butterflySprite(p, FindTheme("mist"))
		for _, row := range rows {
			if ansi.StringWidth(row) != butterflyWidth {
				t.Fatal("rotation clipped")
			}
		}
	}
	if len(headings) < 6 {
		t.Fatal("body did not turn along flight", headings)
	}
	for heading := 0; heading < 8; heading++ {
		rows := butterflySprite(butterflyPose{Heading: heading}, FindTheme("mist"))
		raw := ansi.Strip(strings.Join(rows[:], "\n"))
		if sprites[raw] {
			t.Fatal("directions look identical", heading)
		}
		sprites[raw] = true
	}
	if butterflyPosition(butterflyCycleFrames-1, 80, 2, 26, false).Heading != 0 {
		t.Fatal("landing not upright")
	}
}

func TestButterflyKeepsBroadWingsWhileFolding(t *testing.T) {
	for heading := 0; heading < 8; heading++ {
		for wing := 0; wing < 4; wing++ {
			pixels := butterflyPixels(butterflyPose{Heading: heading, Wing: wing})
			counts := [4]int{}
			for _, row := range pixels {
				for _, kind := range row {
					counts[kind]++
				}
			}
			if counts[1] < 10 || counts[2] < 10 || counts[1]+counts[2] <= counts[3] {
				t.Fatal("fold became a stick", heading, wing, counts)
			}
			for _, row := range butterflySpriteStyle(butterflyPose{Heading: heading, Wing: wing}, FindTheme("mist"), true) {
				if ansi.StringWidth(row) != butterflyWidth {
					t.Fatal("ASCII fallback overflow")
				}
			}
		}
	}
}
