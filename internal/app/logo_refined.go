package app

import (
	"fmt"
	"math"
	"strings"
)

type refinedLogo struct {
	Name, Caption string
	Colors        []string
	Path          []butterflyPoint
	Kind          int
}

var refinedLogos = []refinedLogo{
	{"01 · Aurora", "Бирюза → небесный → ирис", []string{"#63F6DF", "#78DEFF", "#888CF4", "#CA9BFA"}, []butterflyPoint{{3, 1}, {12, 12}, {21, 5}, {30, 12}, {39, 1}}, 0},
	{"02 · Crystal", "Ледяные грани и фиолетовая глубина", []string{"#B9F5FF", "#77CEFF", "#8C91F4", "#B9A0FA"}, []butterflyPoint{{3, 1}, {12, 12}, {21, 3.5}, {30, 12}, {39, 1}}, 1},
	{"03 · Afterglow", "Персиковый свет → розовый → лаванда", []string{"#FFCF97", "#F7A9B7", "#D594E7", "#9394EF"}, []butterflyPoint{{3, 1}, {12, 12}, {21, 4.5}, {30, 12}, {39, 1}}, 2},
	{"04 · Wisp", "Холодное ядро, растворяющийся край", []string{"#80EDEB", "#8DD8FF", "#AAAEF5", "#D2ABF4"}, []butterflyPoint{{3, 1}, {12, 12}, {21, 5}, {30, 12}, {39, 1}}, 3},
}

func logoColor(stops []string, u, brightness float64) string {
	u = max(0.0, min(.99999, u))
	scaled := u * float64(len(stops)-1)
	i := int(scaled)
	f := scaled - float64(i)
	a, b := rgb(stops[i]), rgb(stops[min(len(stops)-1, i+1)])
	return fmt.Sprintf("#%02x%02x%02x", int(min(255.0, (float64(a[0])*(1-f)+float64(b[0])*f)*brightness)), int(min(255.0, (float64(a[1])*(1-f)+float64(b[1])*f)*brightness)), int(min(255.0, (float64(a[2])*(1-f)+float64(b[2])*f)*brightness)))
}
func refinedLogoRows(v refinedLogo) []string { return refinedLogoRowsSized(v, 43, 15) }
func refinedLogoRowsSized(v refinedLogo, width, height int) []string {
	rows := make([]string, height)
	for rowIndex := 0; rowIndex < height; rowIndex++ {
		y := float64(rowIndex) * 14 / float64(max(1, height-1))
		var row strings.Builder
		for col := 0; col < width; col++ {
			x := float64(col) * 42 / float64(max(1, width-1))
			distance := math.Inf(1)
			side := 0.0
			for i := 1; i < len(v.Path); i++ {
				a, b := v.Path[i-1], v.Path[i]
				d := butterflySegment(float64(x)*.5, float64(y), a.X*.5, a.Y, b.X*.5, b.Y)
				if d < distance {
					distance = d
					side = ((b.X-a.X)*.5*(float64(y)-a.Y) - (b.Y-a.Y)*(float64(x)-a.X)*.5) / math.Hypot((b.X-a.X)*.5, b.Y-a.Y)
				}
			}
			glyph := " "
			light := 1.0
			u := .58*float64(y)/14 + .42*float64(x)/42
			switch v.Kind {
			case 0:
				if distance < .92 {
					glyph = "█"
					light = 1.06 - .12*distance
				} else if distance < 1.25 {
					glyph = "▓"
					light = .83
				} else if distance < 1.5 {
					glyph = "░"
					light = .50
				}
			case 1:
				if distance < 1.16 {
					glyph = "█"
					if side < -.35 {
						light = 1.2
					} else if side > .35 {
						light = .58
					} else {
						light = .92
					}
				}
			case 2:
				if distance < 1.0 {
					glyph = "█"
					light = 1.02
				} else if distance < 1.4 {
					glyph = "▒"
					light = .58
				}
				u = .60*float64(y)/14 + .40*float64(x)/42
			case 3:
				if distance < .77 {
					glyph = "█"
					light = 1.14
				} else if distance < 1.10 {
					glyph = "▓"
					light = .86
				} else if distance < 1.55 {
					glyph = "░"
					light = .46
				}
				if y > 11 && distance > .78 && distance < 1.55 {
					glyph = "░"
					light = .32
				}
			}
			if (v.Kind == 0 || v.Kind == 2) && glyph != "█" {
				glyph = " "
			}
			if glyph == " " {
				row.WriteByte(' ')
			} else {
				row.WriteString(style(logoColor(v.Colors, u, light)).Render(glyph))
			}
		}
		rows[rowIndex] = row.String()
	}
	return rows
}
func previewRefinedLogos() string {
	var output []string
	for band := 0; band < 2; band++ {
		a, b := refinedLogos[band*2], refinedLogos[band*2+1]
		output = append(output, pad(style("#E4EAF1").Bold(true).Render(a.Name), 49)+style("#E4EAF1").Bold(true).Render(b.Name), "")
		left, right := refinedLogoRows(a), refinedLogoRows(b)
		for i := range left {
			output = append(output, pad(left[i], 49)+right[i])
		}
		output = append(output, "", pad(style("#929CAF").Render(a.Caption), 49)+style("#929CAF").Render(b.Caption), "", "", "")
	}
	return strings.Join(output, "\n") + "\n"
}
