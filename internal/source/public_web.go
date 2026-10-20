package source

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"wisp/internal/library"
)

const publicMaxBytes = 16 << 20
const publicUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/130.0 Safari/537.36"

var scScripts = regexp.MustCompile(`src="(https://a-v2\.sndcdn\.com/assets/[^" ]+\.js)"`)
var scID = regexp.MustCompile(`client_id\s*[:=]\s*"(\w{32})"`)
var nextData = regexp.MustCompile(`(?s)<script[^>]*id="__NEXT_DATA__"[^>]*>(.*?)</script>`)

func (c *Client) publicBytes(ctx context.Context, raw string) ([]byte, error) {
	return c.publicRequest(ctx, http.MethodGet, raw, nil)
}
func (c *Client) publicRequest(ctx context.Context, method, raw string, body []byte) ([]byte, error) {
	u, err := url.Parse(raw)
	if err != nil || u.User != nil {
		return nil, errors.New("invalid public URL")
	}
	allowed := false
	for _, base := range []string{c.WebSC, c.WebSCAPI, c.WebSP, c.WebYT, "https://a-v2.sndcdn.com"} {
		b, e := url.Parse(base)
		if e == nil && u.Scheme == b.Scheme && u.Host == b.Host {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil, errors.New("unexpected public endpoint host")
	}
	req, err := http.NewRequestWithContext(ctx, method, u.String(), bytes.NewReader(body))
	if err != nil {
		return nil, errors.New("invalid public request")
	}
	req.Header.Set("User-Agent", publicUserAgent)
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", c.WebYT)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, errors.New("public website request failed")
	}
	defer resp.Body.Close()
	if resp.Header.Get("x-amzn-waf-action") == "challenge" {
		return nil, errors.New("website requests browser verification; retry later or use another enabled source")
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("public website returned HTTP %d; this content may be unavailable or restricted", resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, publicMaxBytes+1))
	if err != nil {
		return nil, errors.New("could not read public response")
	}
	if len(b) > publicMaxBytes {
		return nil, errors.New("public response exceeds 16 MiB")
	}
	return b, nil
}
func (c *Client) publicJSON(ctx context.Context, raw string, dst any) error {
	b, err := c.publicBytes(ctx, raw)
	if err != nil {
		return err
	}
	if json.Unmarshal(b, dst) != nil {
		return errors.New("public website response changed; update wisp")
	}
	return nil
}

// The ID belongs to the public website client, not to a user's account.
func (c *Client) publicSCID(ctx context.Context) (string, error) {
	c.scMu.Lock()
	defer c.scMu.Unlock()
	if c.scClientID != "" && time.Now().Before(c.scExpires) {
		return c.scClientID, nil
	}
	html, err := c.publicBytes(ctx, c.WebSC)
	if err != nil {
		return "", err
	}
	scripts := scScripts.FindAllSubmatch(html, -1)
	for i, scanned := len(scripts)-1, 0; i >= 0 && scanned < 10; i, scanned = i-1, scanned+1 {
		b, e := c.publicBytes(ctx, string(scripts[i][1]))
		if e != nil {
			continue
		}
		if match := scID.FindSubmatch(b); len(match) == 2 {
			c.scClientID = string(match[1])
			c.scExpires = time.Now().Add(time.Hour)
			return c.scClientID, nil
		}
	}
	return "", errors.New("SoundCloud public client changed; update wisp or use /connect soundcloud")
}
func (c *Client) scGet(ctx context.Context, route string, params url.Values, dst any) error {
	id, err := c.publicSCID(ctx)
	if err != nil {
		return err
	}
	if params == nil {
		params = url.Values{}
	}
	params.Set("client_id", id)
	return c.publicJSON(ctx, c.WebSCAPI+route+"?"+params.Encode(), dst)
}
func (c *Client) searchPublicSC(ctx context.Context, q string) ([]library.Track, error) {
	var r struct {
		Collection []scTrack `json:"collection"`
	}
	if err := c.scGet(ctx, "/search/tracks", url.Values{"q": {q}, "limit": {"15"}}, &r); err != nil {
		return nil, err
	}
	out := []library.Track{}
	for _, t := range r.Collection {
		out = append(out, t.track())
	}
	return out, nil
}

type publicSCTrack struct {
	scTrack
	Policy     string `json:"policy"`
	Streamable *bool  `json:"streamable"`
	Media      struct {
		Transcodings []struct {
			URL     string `json:"url"`
			Snipped bool   `json:"snipped"`
			Format  struct {
				Protocol string `json:"protocol"`
				Mime     string `json:"mime_type"`
			} `json:"format"`
		} `json:"transcodings"`
	} `json:"media"`
}

func (c *Client) streamPublicSC(ctx context.Context, t library.Track) (string, error) {
	id := strings.TrimPrefix(t.ID, "soundcloud:")
	if i := strings.LastIndex(id, ":"); i >= 0 {
		id = id[i+1:]
	}
	if !identifier.MatchString(id) {
		return "", errors.New("invalid SoundCloud track identifier")
	}
	var track publicSCTrack
	if err := c.scGet(ctx, "/tracks/"+url.PathEscape(id), nil, &track); err != nil {
		return "", err
	}
	if track.Policy == "BLOCK" || track.Policy == "SNIP" || track.Access == "blocked" || (track.Streamable != nil && !*track.Streamable) {
		return "", errors.New("SoundCloud does not expose a full public stream for this track")
	}
	client, err := c.publicSCID(ctx)
	if err != nil {
		return "", err
	}
	for _, protocol := range []string{"progressive", "hls"} {
		for _, media := range track.Media.Transcodings {
			if media.Snipped || media.Format.Protocol != protocol {
				continue
			}
			u, e := url.Parse(media.URL)
			if e != nil {
				continue
			}
			q := u.Query()
			q.Set("client_id", client)
			u.RawQuery = q.Encode()
			var stream struct {
				URL string `json:"url"`
			}
			if c.publicJSON(ctx, u.String(), &stream) != nil {
				continue
			}
			if validStreamURL(stream.URL) {
				return stream.URL, nil
			}
		}
	}
	return "", errors.New("SoundCloud has no usable public audio stream for this track")
}
func validStreamURL(raw string) bool {
	u, e := url.Parse(raw)
	return e == nil && u.Scheme == "https" && u.Host != "" && u.User == nil
}

func (c *Client) importPublicSC(ctx context.Context, raw string) (Import, error) {
	var p struct {
		Kind       string      `json:"kind"`
		ID         json.Number `json:"id"`
		Title      string      `json:"title"`
		TrackCount int         `json:"track_count"`
		Tracks     []scTrack   `json:"tracks"`
	}
	if err := c.scGet(ctx, "/resolve", url.Values{"url": {raw}}, &p); err != nil {
		return Import{}, err
	}
	if p.Kind == "track" {
		var t scTrack
		if err := c.scGet(ctx, "/tracks/"+p.ID.String(), nil, &t); err != nil {
			return Import{}, err
		}
		return Import{Name: t.Title, Tracks: []library.Track{t.track()}}, nil
	}
	if len(p.Tracks) > library.MaxTracks {
		return Import{}, errors.New("playlist exceeds 10,000 tracks")
	}
	if p.TrackCount > len(p.Tracks) {
		return Import{}, errors.New("SoundCloud returned only part of the playlist; nothing imported")
	}
	tracks := make([]library.Track, len(p.Tracks))
	missing := []string{}
	for i, t := range p.Tracks {
		tracks[i] = t.track()
		if t.Title == "" && t.ID.String() != "" {
			missing = append(missing, t.ID.String())
		}
	}
	resolved := map[string]library.Track{}
	for start := 0; start < len(missing); start += 50 {
		var batch []scTrack
		if err := c.scGet(ctx, "/tracks", url.Values{"ids": {strings.Join(missing[start:min(start+50, len(missing))], ",")}}, &batch); err != nil {
			return Import{}, err
		}
		for _, t := range batch {
			resolved[t.ID.String()] = t.track()
		}
	}
	for i, t := range p.Tracks {
		if t.Title != "" {
			continue
		}
		if found, ok := resolved[t.ID.String()]; ok {
			tracks[i] = found
		} else {
			tracks[i].Title = "Unavailable playlist item"
			tracks[i].Unavailable = true
		}
	}
	name := library.Clean(p.Title)
	if name == "" {
		return Import{}, errors.New("not a public SoundCloud playlist")
	}
	return Import{Name: name, Tracks: tracks}, nil
}

func (c *Client) importPublicSpotify(ctx context.Context, id string) (Import, error) {
	resource := "playlist"
	if strings.HasPrefix(id, "track:") {
		resource = "track"
		id = strings.TrimPrefix(id, "track:")
	}
	b, err := c.publicBytes(ctx, c.WebSP+"/embed/"+resource+"/"+url.PathEscape(id))
	if err != nil {
		return Import{}, err
	}
	return parseSpotifyEmbed(b, id)
}
func parseSpotifyEmbed(b []byte, id string) (Import, error) {
	match := nextData.FindSubmatch(b)
	if len(match) != 2 {
		return Import{}, errors.New("Spotify public page changed or playlist is unavailable")
	}
	var r struct {
		Props struct {
			Page struct {
				State struct {
					Data struct {
						Entity struct {
							URI      string  `json:"uri"`
							Duration float64 `json:"duration"`
							Artists  []struct {
								Name string `json:"name"`
							} `json:"artists"`
							Title  string `json:"title"`
							Name   string `json:"name"`
							Tracks []struct {
								URI      string  `json:"uri"`
								Title    string  `json:"title"`
								Subtitle string  `json:"subtitle"`
								Duration float64 `json:"duration"`
							} `json:"trackList"`
						} `json:"entity"`
					} `json:"data"`
				} `json:"state"`
			} `json:"pageProps"`
		} `json:"props"`
	}
	if json.Unmarshal(match[1], &r) != nil {
		return Import{}, errors.New("could not parse public Spotify metadata")
	}
	e := r.Props.Page.State.Data.Entity
	name := e.Title
	if name == "" {
		name = e.Name
	}
	if name == "" {
		return Import{}, errors.New("Spotify playlist is not public")
	}
	out := Import{Name: library.Clean(name) + " [public snapshot]", Notice: "Spotify embed snapshot: up to 100 entries; import CSV for a complete larger playlist. Audio matches YouTube."}
	if strings.HasPrefix(e.URI, "spotify:track:") {
		artist := []string{}
		for _, a := range e.Artists {
			artist = append(artist, a.Name)
		}
		trackID := strings.TrimPrefix(e.URI, "spotify:track:")
		if !identifier.MatchString(trackID) {
			return Import{}, errors.New("invalid public Spotify track ID")
		}
		return Import{Name: library.Clean(name), Notice: "Spotify metadata; audio is matched to a public YouTube recording.", Tracks: []library.Track{{ID: "spotify:" + trackID, Source: "spotify", Title: library.Clean(name), Artist: library.Clean(strings.Join(artist, ", ")), Duration: e.Duration / 1000, URL: "https://open.spotify.com/track/" + trackID}}}, nil
	}
	for i, t := range e.Tracks {
		trackID := strings.TrimPrefix(t.URI, "spotify:track:")
		if trackID == t.URI || !identifier.MatchString(trackID) {
			out.Tracks = append(out.Tracks, library.Track{ID: fmt.Sprintf("spotify:unavailable:%s:%d", id, i), Source: "spotify", Title: "Unavailable playlist item", Unavailable: true})
			continue
		}
		out.Tracks = append(out.Tracks, library.Track{ID: "spotify:" + trackID, Source: "spotify", Title: library.Clean(t.Title), Artist: library.Clean(t.Subtitle), Duration: t.Duration / 1000, URL: "https://open.spotify.com/track/" + trackID})
	}
	return out, nil
}
