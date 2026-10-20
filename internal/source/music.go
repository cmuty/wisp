package source

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"wisp/internal/library"
)

var musicVersion = regexp.MustCompile(`"INNERTUBE_CLIENT_VERSION"\s*:\s*"([^"]+)"`)

const musicSongsFilter = "EgWKAQIIAWoKEAkQBRAKEAMQBA%3D%3D"

func (c *Client) musicClientVersion(ctx context.Context) (string, error) {
	c.ytMu.Lock()
	defer c.ytMu.Unlock()
	if c.ytVersion != "" && time.Now().Before(c.ytExpires) {
		return c.ytVersion, nil
	}
	b, err := c.publicBytes(ctx, c.WebYT)
	if err != nil {
		return "", err
	}
	m := musicVersion.FindSubmatch(b)
	if len(m) != 2 {
		return "", errors.New("YouTube Music public page changed; update wisp")
	}
	c.ytVersion = string(m[1])
	c.ytExpires = time.Now().Add(time.Hour)
	return c.ytVersion, nil
}
func (c *Client) searchMusic(ctx context.Context, q string) ([]library.Track, error) {
	version, err := c.musicClientVersion(ctx)
	if err != nil {
		return nil, err
	}
	payload, _ := json.Marshal(map[string]any{"context": map[string]any{"client": map[string]string{"clientName": "WEB_REMIX", "clientVersion": version, "hl": "en"}}, "query": q, "params": musicSongsFilter})
	b, err := c.publicRequest(ctx, http.MethodPost, c.WebYT+"/youtubei/v1/search?prettyPrint=false", payload)
	if err != nil {
		return nil, err
	}
	return parseMusicSearch(b)
}
func parseMusicSearch(b []byte) ([]library.Track, error) {
	var root any
	if json.Unmarshal(b, &root) != nil {
		return nil, errors.New("invalid YouTube Music search response")
	}
	out := []library.Track{}
	seen := map[string]bool{}
	nodes := 0
	var walk func(any, int)
	walk = func(v any, depth int) {
		nodes++
		if depth > 64 || nodes > 100000 || len(out) >= 20 {
			return
		}
		switch x := v.(type) {
		case map[string]any:
			if raw, ok := x["musicResponsiveListItemRenderer"]; ok {
				if item, ok := raw.(map[string]any); ok {
					if t, valid := musicItem(item); valid && !seen[t.ID] {
						seen[t.ID] = true
						out = append(out, t)
					}
				}
				return
			}
			for _, child := range x {
				walk(child, depth+1)
			}
		case []any:
			for _, child := range x {
				walk(child, depth+1)
			}
		}
	}
	walk(root, 0)
	return out, nil
}
func nested(v any, path ...string) any {
	for _, key := range path {
		m, ok := v.(map[string]any)
		if !ok {
			return nil
		}
		v = m[key]
	}
	return v
}
func stringAt(v any, path ...string) string { s, _ := nested(v, path...).(string); return s }
func musicItem(item map[string]any) (library.Track, bool) {
	columns, _ := item["flexColumns"].([]any)
	if len(columns) == 0 {
		return library.Track{}, false
	}
	runsAt := func(index int) []any {
		if index >= len(columns) {
			return nil
		}
		r, _ := nested(columns[index], "musicResponsiveListItemFlexColumnRenderer", "text", "runs").([]any)
		return r
	}
	first := runsAt(0)
	title := ""
	id := stringAt(item, "playlistItemData", "videoId")
	for _, run := range first {
		title += stringAt(run, "text")
		if id == "" {
			id = stringAt(run, "navigationEndpoint", "watchEndpoint", "videoId")
		}
	}
	if id == "" {
		id = stringAt(item, "overlay", "musicItemThumbnailOverlayRenderer", "content", "musicPlayButtonRenderer", "playNavigationEndpoint", "watchEndpoint", "videoId")
	}
	if title == "" || !identifier.MatchString(id) {
		return library.Track{}, false
	}
	meta := ""
	duration := 0.0
	for _, run := range runsAt(1) {
		text := stringAt(run, "text")
		meta += text
		if seconds, ok := parseClock(text); ok {
			duration = seconds
		}
	}
	artist := strings.TrimSpace(strings.Split(meta, " • ")[0])
	return library.Track{ID: "youtube:" + id, Source: "youtube", Title: library.Clean(title), Artist: library.Clean(artist), Duration: duration, URL: "https://www.youtube.com/watch?v=" + id}, true
}
func parseClock(s string) (float64, bool) {
	parts := strings.Split(strings.TrimSpace(s), ":")
	if len(parts) < 2 || len(parts) > 3 {
		return 0, false
	}
	seconds := 0
	for _, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil || n < 0 || n > 59 {
			return 0, false
		}
		seconds = seconds*60 + n
	}
	return float64(seconds), true
}
