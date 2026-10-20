package lyrics

import (
	"context"
	"errors"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"wisp/internal/library"
)

type Line struct {
	Time float64
	Text string
}
type Song struct {
	Lines         []Line
	Plain, Credit string
}
type Client struct {
	mu            sync.Mutex
	notBefore     time.Time
	blockedStatus int
	HTTP          *http.Client
	Base          string
	CacheDir      string
}

func New() *Client {
	return &Client{HTTP: &http.Client{Timeout: 8 * time.Second}, Base: "https://lrclib.net"}
}

var stamps = regexp.MustCompile(`\[(\d+):(\d{2}(?:\.\d+)?)\]`)
var offset = regexp.MustCompile(`\[offset:([+-]?\d+)\]`)

func Parse(raw string) []Line {
	shift := 0.0
	if v := offset.FindStringSubmatch(raw); len(v) > 1 {
		n, _ := strconv.Atoi(v[1])
		shift = float64(n) / 1000
	}
	var lines []Line
	for _, s := range strings.Split(raw, "\n") {
		matches := stamps.FindAllStringSubmatch(s, -1)
		text := library.Clean(strings.TrimSpace(stamps.ReplaceAllString(s, "")))
		for _, v := range matches {
			minutes, _ := strconv.Atoi(v[1])
			seconds, _ := strconv.ParseFloat(v[2], 64)
			if seconds >= 60 {
				continue
			}
			lines = append(lines, Line{float64(minutes*60) + seconds + shift, text})
		}
	}
	sort.SliceStable(lines, func(i, j int) bool { return lines[i].Time < lines[j].Time })
	return lines
}
func Active(lines []Line, position float64) int {
	return sort.Search(len(lines), func(i int) bool { return lines[i].Time > position }) - 1
}

type record struct {
	TrackName, ArtistName, PlainLyrics, SyncedLyrics string
	Duration                                         float64
	Instrumental                                     bool
}

func (c *Client) Find(ctx context.Context, t library.Track) (Song, error) {
	if t.Path != "" {
		path := strings.TrimSuffix(t.Path, filepath.Ext(t.Path)) + ".lrc"
		if info, e := os.Stat(path); e == nil && info.Size() <= 2<<20 {
			raw, e := os.ReadFile(path)
			if e == nil {
				lines := Parse(string(raw))
				if len(lines) > 0 {
					return Song{Lines: lines, Credit: "Local LRC"}, nil
				}
			}
		}
	}
	if song, ok := c.cached(t); ok {
		return song, nil
	}
	artist, title := t.Artist, t.Title
	// SoundCloud often stores the recording's artist in its title, rather than uploader metadata.
	if t.Source == "soundcloud" || t.Source == "local" {
		for _, sep := range []string{" — ", " - ", " – "} {
			if a, b, ok := strings.Cut(title, sep); ok {
				artist = strings.TrimSpace(a)
				title = strings.TrimSpace(b)
				break
			}
		}
	}
	title = strings.TrimSpace(strings.Split(title, " [via ")[0])
	q := url.Values{"track_name": {title}, "artist_name": {artist}}
	if t.Duration > 0 {
		q.Set("duration", strconv.Itoa(int(math.Round(t.Duration))))
	}
	var r record
	err := c.get(ctx, "/api/get?"+q.Encode(), &r)
	if statusIs(err, 404) && cleanUploadTitle(title) != title {
		q.Set("track_name", cleanUploadTitle(title))
		err = c.get(ctx, "/api/get?"+q.Encode(), &r)
		q.Set("track_name", title)
	}
	if err != nil {
		if !statusIs(err, 404) {
			return Song{}, err
		}
		if ctx.Err() != nil {
			return Song{}, ctx.Err()
		}
		var candidates []record
		q.Del("duration")
		if e := c.get(ctx, "/api/search?"+q.Encode(), &candidates); e != nil {
			return Song{}, e
		}
		best := math.Inf(1)
		found := false
		for _, v := range candidates {
			if !strings.EqualFold(strings.TrimSpace(v.TrackName), title) || !artistCreditMatches(v.ArtistName, artist) {
				continue
			}
			diff := math.Abs(v.Duration - t.Duration)
			if t.Duration > 0 && diff > 3 {
				continue
			}
			if diff < best && (v.Instrumental || v.PlainLyrics != "" || v.SyncedLyrics != "") {
				r = v
				best = diff
				found = true
			}
		}
		if !found {
			cleaned := cleanUploadTitle(title)
			if cleaned != title {
				q.Set("track_name", cleaned)
				var more []record
				if e := c.get(ctx, "/api/search?"+q.Encode(), &more); e != nil {
					return Song{}, e
				}
				for _, v := range more {
					if strings.EqualFold(strings.TrimSpace(v.TrackName), cleaned) && artistCreditMatches(v.ArtistName, artist) && (t.Duration <= 0 || math.Abs(v.Duration-t.Duration) <= 3) {
						r = v
						found = true
						break
					}
				}
			}
			if !found {
				return Song{}, errors.New("No matching lyrics for this recording. Press F5 to retry, or use a local .lrc file.")
			}
		}
	}
	if r.Instrumental {
		song := Song{Plain: "Instrumental — no lyrics.", Credit: "LRCLIB"}
		c.store(t, song)
		return song, nil
	}
	song := Song{Lines: Parse(r.SyncedLyrics), Plain: r.PlainLyrics, Credit: "LRCLIB"}
	if len(song.Lines) == 0 && song.Plain == "" {
		return song, errors.New("Lyrics are unavailable for this recording.")
	}
	c.store(t, song)
	return song, nil
}

var uploadSuffix = regexp.MustCompile(`\s*\(([A-Za-z0-9_.@-]+)\)\s*$`)

func cleanUploadTitle(title string) string {
	match := uploadSuffix.FindStringSubmatch(title)
	if len(match) == 0 {
		return title
	}
	switch strings.ToLower(match[1]) {
	case "live", "remix", "slowed", "spedup", "acoustic", "instrumental", "edit", "cover", "demo", "remastered":
		return title
	}
	return strings.TrimSpace(uploadSuffix.ReplaceAllString(title, ""))
}
