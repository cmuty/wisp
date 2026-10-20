package app

import (
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"math"
	"strings"
	"time"
)

const (
	butterflyWidth         = 13
	butterflyHeight        = 6
	butterflyRestFrames    = 60
	butterflyFlightFrames  = 540
	butterflyCycleFrames   = butterflyRestFrames + butterflyFlightFrames
	butterflyFrameInterval = time.Second / 30
)

type butterflyPulse time.Time

func (m *Model) butterflyTick() tea.Cmd {
	if m.ReducedMotion {
		return nil
	}
	delay := butterflyFrameInterval
	if m.stage != "ready" {
		delay = 250 * time.Millisecond
	}
	return tea.Tick(delay, func(t time.Time) tea.Msg { return butterflyPulse(t) })
}

type butterflyPose struct {
	X, Y, Wing, Bank, Heading int
	Perched                   bool
}
type butterflyArea struct{ Left, Top, Right, Bottom int }
type butterflyPoint struct{ X, Y float64 }

func butterflyPerch(cycle, width int) float64 {
	return float64(max(0, width-butterflyWidth)) * (0.5 + 0.30*math.Sin(float64(cycle)*1.7+0.8))
}
func butterflyCurve(a, b, c, d, t float64) float64 {
	return .5 * ((2 * b) + (-a+c)*t + (2*a-5*b+4*c-d)*t*t + (-a+3*b-3*c+d)*t*t*t)
}
func butterflyRoute(frame, width, top, border int) butterflyPoint {
	cycle, phase := frame/butterflyCycleFrames, frame%butterflyCycleFrames
	bottom := float64(max(top, border-butterflyHeight))
	spanX := float64(max(0, width-butterflyWidth))
	spanY := max(0.0, bottom-float64(top))
	p := func(x, y float64) butterflyPoint {
		if cycle%2 != 0 {
			x = 1 - x
		}
		return butterflyPoint{x * spanX, float64(top) + y*spanY}
	}
	points := []butterflyPoint{{butterflyPerch(cycle, width), bottom}, p(.12, .56), p(.64, .06), p(.91, .42), p(.52, .76), p(.13, .19), p(.67, .31), {butterflyPerch(cycle+1, width), bottom}}
	u := float64(max(0, phase-butterflyRestFrames)) / float64(butterflyFlightFrames-1)
	// Ease takeoff and landing while the spline itself keeps curved, continuous turns.
	u = .5 - .5*math.Cos(math.Pi*u)
	scaled := u * float64(len(points)-1)
	i := min(len(points)-2, int(scaled))
	t := scaled - float64(i)
	a, b, c, d := points[max(0, i-1)], points[i], points[i+1], points[min(len(points)-1, i+2)]
	return butterflyPoint{butterflyCurve(a.X, b.X, c.X, d.X, t), butterflyCurve(a.Y, b.Y, c.Y, d.Y, t)}
}
func butterflyPosition(frame, width, top, border int, reduced bool) butterflyPose {
	if reduced {
		frame = 0
	}
	frame = max(0, frame)
	cycle, phase := frame/butterflyCycleFrames, frame%butterflyCycleFrames
	pose := butterflyPose{X: int(math.Round(butterflyPerch(cycle, width))), Y: max(top, border-butterflyHeight), Perched: phase < butterflyRestFrames}
	if pose.Perched {
		if phase >= 36 {
			pose.Wing = []int{0, 1, 2, 1}[(phase-36)/6]
		}
		return pose
	}
	position := butterflyRoute(frame, width, top, border)
	pose.X = int(math.Round(position.X))
	pose.Y = int(math.Round(position.Y))
	pose.X = max(0, min(width-butterflyWidth, pose.X))
	pose.Y = max(top, min(border-butterflyHeight, pose.Y))
	pose.Wing = []int{0, 1, 2, 3, 2, 1}[(frame/2)%6]
	next := butterflyRoute(min(frame+2, cycle*butterflyCycleFrames+butterflyCycleFrames-1), width, top, border)
	if next.X-position.X > .12 {
		pose.Bank = 1
	} else if next.X-position.X < -.12 {
		pose.Bank = -1
	}
	previous := butterflyRoute(max(cycle*butterflyCycleFrames+butterflyRestFrames, frame-4), width, top, border)
	ahead := butterflyRoute(min(frame+4, cycle*butterflyCycleFrames+butterflyCycleFrames-1), width, top, border)
	// Lift off from an upright resting pose and turn upright again before touching the rail.
	blend := math.Min(1, math.Min(float64(phase-butterflyRestFrames)/20, float64(butterflyCycleFrames-1-phase)/36))
	blend = blend * blend * (3 - 2*blend)
	pose.Heading = butterflyHeading(ahead.X-previous.X, ahead.Y-previous.Y, blend)
	return pose
}

// A suffix rectangle reaches the rail, so every flight can land without covering text.
// The right side of a full track list remains available even when the bottom is crowded.
func butterflyFreeArea(lines []string, width, border int) (butterflyArea, bool) {
	best := butterflyArea{}
	bestScore := 0
	left := 1
	for y := border - 1; y >= 2; y-- {
		occupied := ansi.StringWidth(strings.TrimRight(ansi.Strip(lines[y]), " "))
		if occupied > 0 {
			left = max(left, occupied+2)
		}
		available, height := width-1-left, border-y
		if available < butterflyWidth || height < butterflyHeight {
			continue
		}
		// Both dimensions matter: an open side corridor beats a narrow horizontal strip.
		score := (available - butterflyWidth + 4) * (height - butterflyHeight + 3)
		if score > bestScore {
			bestScore = score
			best = butterflyArea{left, y, width - 1, border}
		}
	}
	return best, bestScore > 0
}
func (m *Model) drawButterfly(lines []string, t Theme, width, border int) {
	if m.stage != "ready" || border >= len(lines) || border < butterflyHeight+2 {
		return
	}
	area, ok := butterflyFreeArea(lines, width, border)
	if !ok {
		return
	}
	pose := butterflyPosition(m.butterflyFrame, area.Right-area.Left, area.Top, area.Bottom, m.ReducedMotion)
	pose.X += area.Left
	if pose.Perched {
		foot := "┴"
		if m.Plain {
			foot = "+"
		}
		x := pose.X + butterflyWidth/2
		lines[border] = ansi.Cut(lines[border], 0, x) + style(t.Accent).Render(foot) + ansi.Cut(lines[border], x+1, width)
	}
	for row, sprite := range butterflySpriteStyle(pose, t, m.Plain) {
		i := pose.Y + row
		line := lines[i]
		if missing := pose.X - ansi.StringWidth(line); missing > 0 {
			line += strings.Repeat(" ", missing)
		}
		lines[i] = ansi.Cut(line, 0, pose.X) + sprite + ansi.Cut(line, pose.X+butterflyWidth, width)
	}
}
