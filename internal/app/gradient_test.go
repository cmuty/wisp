package app

import (
	"github.com/charmbracelet/x/ansi"
	"regexp"
	"strings"
	"testing"
)

func TestLogosHaveNoShadedHalo(t *testing.T) {
	for _, id := range []string{"mist", "ember", "haze", "aurora", "afterglow"} {
		for _, row := range headerLogoRows(FindTheme(id), false, 6) {
			if strings.ContainsAny(ansi.Strip(row), "░▒▓") {
				t.Fatal("halo remains", id)
			}
		}
	}
	for _, id := range []string{"aurora", "afterglow"} {
		if strings.ContainsAny(ansi.Strip(LogoView(FindTheme(id), false)), "░▒▓") {
			t.Fatal("startup halo remains")
		}
	}
}
func TestProgressColorsUseAbsoluteScale(t *testing.T) {
	for _, id := range []string{"aurora", "afterglow", "mist"} {
		theme := FindTheme(id)
		short := progressSegment(theme, "━", 0, 4, 60, 1)
		long := progressSegment(theme, "━", 0, 25, 60, 1)
		if !strings.HasPrefix(long, short) {
			t.Fatal("gradient rescaled as playback advanced")
		}
		if ansi.StringWidth(long) != 25 {
			t.Fatal("progress width changed")
		}
		if progressSegment(theme, "━", 4, 1, 60, 1) != progressSegment(theme, "━", 0, 5, 60, 1)[len(short):] {
			t.Fatal("split mismatch")
		}
	}
}
func TestGradientPreservesTextAndUsesSeveralColors(t *testing.T) {
	text := "› 12 · музыка é 🎵"
	theme := FindTheme("afterglow")
	rendered := gradientText(theme, text)
	if ansi.Strip(rendered) != text || ansi.StringWidth(rendered) != ansi.StringWidth(text) {
		t.Fatal("text corrupted")
	}
	colors := regexp.MustCompile(`\x1b\[38;2;[0-9;]+m`).FindAllString(rendered, -1)
	seen := map[string]bool{}
	for _, color := range colors {
		seen[color] = true
	}
	if len(seen) < 5 {
		t.Fatal("not a gradient", len(seen))
	}
}
