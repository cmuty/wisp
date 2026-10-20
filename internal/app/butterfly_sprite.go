package app

import (
	"math"
	"strings"
	"sync"
)

func butterflyHeading(dx, dy, blend float64) int {
	if math.Hypot(dx, dy) < 1e-9 {
		return 0
	}
	angle := math.Atan2(dy, dx*.5) + math.Pi/2
	angle = math.Atan2(math.Sin(angle), math.Cos(angle)) * blend
	return (int(math.Round(angle/(math.Pi/4))) + 8) % 8
}

const butterflyDotsX = butterflyWidth * 2
const butterflyDotsY = butterflyHeight * 4

type butterflyRaster [butterflyDotsY][butterflyDotsX]uint8

var butterflyRasters sync.Map

func wingCurve(points []butterflyPoint, a, b, c, d butterflyPoint) []butterflyPoint {
	for i := 0; i <= 24; i++ {
		u := float64(i) / 24
		v := 1 - u
		points = append(points, butterflyPoint{v*v*v*a.X + 3*v*v*u*b.X + 3*v*u*u*c.X + u*u*u*d.X, v*v*v*a.Y + 3*v*v*u*b.Y + 3*v*u*u*c.Y + u*u*u*d.Y})
	}
	return points
}

var butterflyUpperWing = func() []butterflyPoint {
	p := wingCurve(nil, butterflyPoint{.7, 0}, butterflyPoint{3, -6}, butterflyPoint{6, -12}, butterflyPoint{10, -10})
	return wingCurve(p, butterflyPoint{10, -10}, butterflyPoint{15, -7}, butterflyPoint{13, 1}, butterflyPoint{.7, 0})
}()
var butterflyLowerWing = func() []butterflyPoint {
	p := wingCurve(nil, butterflyPoint{.7, 0}, butterflyPoint{5, -1}, butterflyPoint{10, 2}, butterflyPoint{7, 6})
	return wingCurve(p, butterflyPoint{7, 6}, butterflyPoint{3, 9}, butterflyPoint{1, 5}, butterflyPoint{.7, 0})
}()

func butterflyInside(x, y float64, polygon []butterflyPoint) bool {
	inside := false
	j := len(polygon) - 1
	for i, p := range polygon {
		q := polygon[j]
		if (p.Y > y) != (q.Y > y) && x < (q.X-p.X)*(y-p.Y)/(q.Y-p.Y)+p.X {
			inside = !inside
		}
		j = i
	}
	return inside
}
func butterflySegment(x, y, ax, ay, bx, by float64) float64 {
	dx, dy := bx-ax, by-ay
	u := max(0.0, min(1.0, ((x-ax)*dx+(y-ay)*dy)/max(.001, dx*dx+dy*dy)))
	return math.Hypot(x-ax-u*dx, y-ay-u*dy)
}
func butterflyDot(x, y float64, pose butterflyPose) uint8 {
	// Body, head and curved antennae stay short relative to the four broad wing lobes.
	if butterflySegment(x, y, 0, -2.8, 0, 5.0) < .60 || math.Hypot(x, y+4) < .95 {
		return 3
	}
	ax := math.Abs(x)
	if butterflySegment(ax, y, .4, -4.5, 2.4, -6.8) < .36 || butterflySegment(ax, y, 2.4, -6.8, 3.2, -9) < .36 || math.Hypot(ax-3.2, y+9) < .55 {
		return 3
	}
	if pose.Perched && butterflySegment(ax, y, .3, 4, 1.5, 10) < .26 {
		return 3
	}
	fold := []float64{1, .88, .72, .53}[pose.Wing]
	wx := ax / fold
	if !butterflyInside(wx, y, butterflyUpperWing) && !butterflyInside(wx, y, butterflyLowerWing) {
		return 0
	}
	// Eye-spots and short veins are negative space cut into the filled wing silhouette.
	if math.Pow((wx-7.7)/1.35, 2)+math.Pow((y+5.1)/1.15, 2) < 1 || math.Pow((wx-5.8)/1.15, 2)+math.Pow((y-4.6)/1.0, 2) < 1 {
		return 0
	}
	if wx > 2.5 && (butterflySegment(wx, y, 1, 0, 10, -3) < .27 || butterflySegment(wx, y, 1, 0, 7, 7) < .24) {
		return 0
	}
	if x < 0 {
		return 1
	}
	return 2
}
func butterflyPixels(pose butterflyPose) butterflyRaster {
	key := [3]int{pose.Heading, pose.Wing, 0}
	if pose.Perched {
		key[2] = 1
	}
	if cached, ok := butterflyRasters.Load(key); ok {
		return cached.(butterflyRaster)
	}
	var pixels butterflyRaster
	angle := float64(pose.Heading) * math.Pi / 4
	cs, sn := math.Cos(angle), math.Sin(angle)
	cy := float64(butterflyDotsY-1) / 2
	if pose.Perched {
		cy += 1
	}
	for y := 0; y < butterflyDotsY; y++ {
		for x := 0; x < butterflyDotsX; x++ {
			px, py := (float64(x)-float64(butterflyDotsX-1)/2)/.87, (float64(y)-cy)/.87
			pixels[y][x] = butterflyDot(px*cs+py*sn, -px*sn+py*cs, pose)
		}
	}
	if pose.Perched {
		bottom := -1
		for y, row := range pixels {
			for _, kind := range row {
				if kind != 0 {
					bottom = y
					break
				}
			}
		}
		if bottom >= 0 && bottom < butterflyDotsY-1 {
			shift := butterflyDotsY - 1 - bottom
			var aligned butterflyRaster
			for y := 0; y+shift < butterflyDotsY; y++ {
				aligned[y+shift] = pixels[y]
			}
			pixels = aligned
		}
	}
	butterflyRasters.Store(key, pixels)
	return pixels
}
func butterflySprite(pose butterflyPose, t Theme) [butterflyHeight]string {
	return butterflySpriteStyle(pose, t, false)
}
func butterflySpriteStyle(pose butterflyPose, t Theme, plain bool) [butterflyHeight]string {
	pixels := butterflyPixels(pose)
	var rows [butterflyHeight]string
	bits := [4][2]uint{{0, 3}, {1, 4}, {2, 5}, {6, 7}}
	for y := 0; y < butterflyHeight; y++ {
		var row strings.Builder
		for x := 0; x < butterflyWidth; x++ {
			mask := 0
			counts := [4]int{}
			for dy := 0; dy < 4; dy++ {
				for dx := 0; dx < 2; dx++ {
					kind := pixels[y*4+dy][x*2+dx]
					if kind > 0 {
						mask |= 1 << bits[dy][dx]
						counts[kind]++
					}
				}
			}
			if mask == 0 {
				row.WriteByte(' ')
				continue
			}
			color := logoColor(themeStops(t), .65*float64(x)/float64(butterflyWidth-1)+.35*float64(y)/float64(butterflyHeight-1), 1)
			if counts[3] > 0 {
				color = t.Text
			}
			glyph := string(rune(0x2800 + mask))
			if plain {
				glyph = string([]byte(" .:+o*#%@")[min(8, counts[1]+counts[2]+counts[3])])
			}
			row.WriteString(style(color).Render(glyph))
		}
		rows[y] = row.String()
	}
	return rows
}
