package source

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"wisp/internal/library"
)

type extractorTrack struct {
	ID           string            `json:"id"`
	Title        string            `json:"title"`
	Artist       string            `json:"artist"`
	Uploader     string            `json:"uploader"`
	Channel      string            `json:"channel"`
	URL          string            `json:"url"`
	Webpage      string            `json:"webpage_url"`
	Duration     float64           `json:"duration"`
	DRM          bool              `json:"has_drm"`
	Availability string            `json:"availability"`
	Entries      []*extractorTrack `json:"entries"`
}

func (t extractorTrack) track() library.Track {
	artist := t.Artist
	if artist == "" {
		artist = t.Uploader
	}
	if artist == "" {
		artist = t.Channel
	}
	raw := "https://www.youtube.com/watch?v=" + t.ID
	return library.Track{ID: "youtube:" + t.ID, Source: "youtube", Title: library.Clean(t.Title), Artist: library.Clean(artist), URL: raw, Duration: t.Duration, Unavailable: t.DRM || t.Availability == "private" || t.Availability == "premium_only"}
}
func extractorPath() (string, error) {
	if p := os.Getenv("WISP_YTDLP"); p != "" {
		return exec.LookPath(p)
	}
	if self, err := os.Executable(); err == nil {
		for _, relative := range []string{"yt-dlp.exe", filepath.Join("tools", "yt-dlp.exe")} {
			p := filepath.Join(filepath.Dir(self), relative)
			if _, err = os.Stat(p); err == nil {
				return p, nil
			}
		}
	}
	if p, err := exec.LookPath("yt-dlp"); err == nil {
		return p, nil
	}
	return "", errors.New("yt-dlp is missing. Run scripts/setup-audio.ps1, install wisp, or set WISP_YTDLP")
}

type boundedBuffer struct {
	bytes.Buffer
	max      int
	overflow bool
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	if b.Len()+len(p) > b.max {
		b.overflow = true
		return 0, errors.New("extractor output exceeded its size limit")
	}
	return b.Buffer.Write(p)
}
func runExtractor(ctx context.Context, args ...string) (extractorTrack, error) {
	exe, err := extractorPath()
	if err != nil {
		return extractorTrack{}, err
	}
	base := []string{"--ignore-config", "--no-warnings", "--no-progress", "--no-colors", "--no-cache-dir", "--skip-download", "--socket-timeout", "15", "--retries", "1", "--extractor-retries", "1"}
	base = append(base, "--no-plugin-dirs")
	if node, err := exec.LookPath("node"); err == nil {
		base = append(base, "--js-runtimes", "node:"+node)
	} else if p := filepath.Join(filepath.Dir(exe), "deno.exe"); fileExists(p) {
		base = append(base, "--js-runtimes", "deno:"+p)
	}
	cmd := exec.CommandContext(ctx, exe, append(base, args...)...)
	hideExtractor(cmd)
	out := &boundedBuffer{max: publicMaxBytes}
	stderr := &boundedBuffer{max: 64 << 10}
	cmd.Stdout = out
	cmd.Stderr = stderr
	for _, v := range os.Environ() {
		key, _, _ := strings.Cut(v, "=")
		if key == "WISP_SOUNDCLOUD_TOKEN" || key == "WISP_SPOTIFY_TOKEN" || key == "WISP_YOUTUBE_KEY" {
			continue
		}
		cmd.Env = append(cmd.Env, v)
	}
	if err = cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return extractorTrack{}, ctx.Err()
		}
		if out.overflow || stderr.overflow {
			return extractorTrack{}, errors.New("extractor output exceeded its size limit")
		}
		lower := strings.ToLower(stderr.String())
		switch {
		case strings.Contains(lower, "this video is unavailable"):
			return extractorTrack{}, errors.New("this YouTube recording is unavailable; choose another search result")
		case strings.Contains(lower, "sign in") || strings.Contains(lower, "login") || strings.Contains(lower, "private"):
			return extractorTrack{}, errors.New("YouTube requires sign-in or restricts this video; no public stream available")
		case strings.Contains(lower, "403") || strings.Contains(lower, "429"):
			return extractorTrack{}, errors.New("YouTube rejected the request; retry later or update yt-dlp")
		default:
			return extractorTrack{}, errors.New("YouTube extraction failed; check connectivity and update yt-dlp")
		}
	}
	var r extractorTrack
	if json.Unmarshal(out.Bytes(), &r) != nil {
		return r, errors.New("unexpected yt-dlp JSON response")
	}
	return r, nil
}
func fileExists(p string) bool { info, err := os.Stat(p); return err == nil && info.Mode().IsRegular() }
func (c *Client) searchPublicYT(ctx context.Context, q string) ([]library.Track, error) {
	return c.searchMusic(ctx, q)
}
func (c *Client) importPublicYT(ctx context.Context, raw string) (Import, error) {
	r, err := runExtractor(ctx, "--flat-playlist", "--dump-single-json", "--playlist-end", fmt.Sprint(library.MaxTracks+1), "--", raw)
	if err != nil {
		return Import{}, err
	}
	if len(r.Entries) > library.MaxTracks {
		return Import{}, errors.New("playlist exceeds 10,000 tracks; nothing imported")
	}
	out := Import{Name: library.Clean(r.Title)}
	if r.Entries == nil && identifier.MatchString(r.ID) {
		return Import{Name: library.Clean(r.Title), Tracks: []library.Track{r.track()}}, nil
	}
	for i, entry := range r.Entries {
		if entry == nil || !identifier.MatchString(entry.ID) {
			out.Tracks = append(out.Tracks, library.Track{ID: fmt.Sprintf("youtube:unavailable:%s:%d", r.ID, i), Source: "youtube", Title: "Unavailable playlist item", Unavailable: true})
			continue
		}
		out.Tracks = append(out.Tracks, entry.track())
	}
	if out.Name == "" {
		out.Name = "YouTube playlist"
	}
	return out, nil
}
func (c *Client) streamPublicYT(ctx context.Context, t library.Track) (string, error) {
	id := strings.TrimPrefix(t.ID, "youtube:")
	if !identifier.MatchString(id) {
		return "", errors.New("invalid YouTube video identifier")
	}
	r, err := runExtractor(ctx, "--no-playlist", "--dump-single-json", "--format", "bestaudio/best", "--", "https://www.youtube.com/watch?v="+id)
	if err != nil {
		return "", err
	}
	if r.DRM {
		return "", errors.New("DRM-protected content is unavailable")
	}
	if !validStreamURL(r.URL) {
		return "", errors.New("no direct public audio stream available")
	}
	return r.URL, nil
}
func (c *Client) streamMatchedSpotify(ctx context.Context, t library.Track) (string, error) {
	candidates, err := c.searchPublicYT(ctx, t.Artist+" "+t.Title+" audio")
	if err != nil {
		return "", err
	}
	best := -1
	score := 0.0
	for i, candidate := range candidates {
		value := matchScore(t, candidate)
		if value > score {
			score = value
			best = i
		}
	}
	if best < 0 || score < 0.64 {
		return "", errors.New("no confident public YouTube match; use /match to choose a recording manually")
	}
	return c.streamPublicYT(ctx, candidates[best])
}
func words(s string) []string {
	return strings.Fields(strings.ToLower(strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			return r
		}
		return ' '
	}, s)))
}
func overlap(query []string, candidate string) float64 {
	if len(query) == 0 {
		return 1
	}
	found := 0
	for _, word := range query {
		if strings.Contains(candidate, word) {
			found++
		}
	}
	return float64(found) / float64(len(query))
}
func matchScore(original, candidate library.Track) float64 {
	if candidate.Unavailable {
		return 0
	}
	text := strings.Join(words(candidate.Artist+" "+candidate.Title), " ")
	requested := strings.Join(words(original.Title), " ")
	for _, version := range []string{"live", "remix", "cover", "karaoke", "slowed", "sped"} {
		if strings.Contains(" "+text+" ", " "+version+" ") && !strings.Contains(" "+requested+" ", " "+version+" ") {
			return 0
		}
	}
	if original.Duration > 0 && candidate.Duration > 0 {
		difference := original.Duration - candidate.Duration
		if difference < 0 {
			difference = -difference
		}
		if difference > max(10.0, original.Duration*0.06) {
			return 0
		}
	}
	title := overlap(words(matchTitle(original.Title)), text)
	artist := overlap(words(original.Artist), text)
	if title < 0.65 || artist < 0.4 {
		return 0
	}
	return title*0.75 + artist*0.25
}

var featuredArtist = regexp.MustCompile(`(?i)\s*[\(\[]\s*(feat|ft|with)\b.*?[\)\]]`)

func matchTitle(s string) string { return featuredArtist.ReplaceAllString(s, "") }
