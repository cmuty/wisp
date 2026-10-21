package source

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"wisp/internal/library"
)

var Names = []string{"soundcloud", "spotify", "youtube"}
var Labels = []string{"SoundCloud", "Spotify", "YouTube Music"}

const maxPages = 1000

type Client struct {
	HTTP                   *http.Client
	SC, SP, YT             string
	tokens                 sync.Map
	PublicWeb              bool
	WebSC, WebSP, WebSCAPI string
	scMu                   sync.Mutex
	scClientID             string
	scExpires              time.Time
	WebYT                  string
	ytMu                   sync.Mutex
	ytVersion              string
	ytExpires              time.Time
}

func New() *Client {
	return &Client{HTTP: &http.Client{Timeout: 15 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}, SC: "https://api.soundcloud.com", SP: "https://api.spotify.com/v1", YT: "https://www.googleapis.com/youtube/v3", PublicWeb: os.Getenv("WISP_OFFICIAL_API_ONLY") != "1", WebSC: "https://soundcloud.com", WebSP: "https://open.spotify.com", WebSCAPI: "https://api-v2.soundcloud.com", WebYT: "https://music.youtube.com"}
}
func Credential(name string) string {
	switch name {
	case "soundcloud":
		return os.Getenv("WISP_SOUNDCLOUD_TOKEN")
	case "spotify":
		return os.Getenv("WISP_SPOTIFY_TOKEN")
	case "youtube":
		return os.Getenv("WISP_YOUTUBE_KEY")
	}
	return ""
}
func Label(name string) string {
	for i, n := range Names {
		if n == name {
			return Labels[i]
		}
	}
	return name
}
func (c *Client) Credential(name string) string {
	if v, ok := c.tokens.Load(name); ok {
		return v.(string)
	}
	return strings.TrimSpace(Credential(name))
}
func (c *Client) SetCredential(name, value string) error {
	valid := false
	for _, n := range Names {
		if n == name {
			valid = true
			break
		}
	}
	if !valid {
		return errors.New("choose soundcloud, spotify or youtube")
	}
	value = strings.TrimSpace(value)
	if len(value) == 0 || len(value) > 8192 {
		return errors.New("enter a token or API key of 1–8192 characters")
	}
	for _, r := range value {
		if r <= 32 || r >= 127 {
			return errors.New("paste only the token or API key, without headers or whitespace")
		}
	}
	c.tokens.Store(name, value)
	return nil
}
func (c *Client) Status(name string) string {
	if c.PublicWeb && c.Credential(name) == "" {
		if name == "spotify" {
			return "public links / audio via YouTube"
		}
		return "public web / no token needed"
	}
	if c.Credential(name) == "" {
		return "not connected /connect " + name
	}
	return "credential supplied; not verified"
}

func (c *Client) get(ctx context.Context, raw, src string, dst any) error {
	u, err := url.Parse(raw)
	if err != nil {
		return errors.New("invalid API URL")
	}
	base := map[string]string{"soundcloud": c.SC, "spotify": c.SP, "youtube": c.YT}[src]
	b, err := url.Parse(base)
	if err != nil {
		return err
	}
	// Pagination URLs cannot send provider credentials to a different origin.
	if u.Scheme != b.Scheme || u.Host != b.Host || !strings.HasPrefix(u.Path, b.Path) {
		return errors.New("rejected cross-origin pagination URL")
	}
	token := c.Credential(src)
	if token == "" {
		return fmt.Errorf("%s is not connected. Use /connect %s", Label(src), src)
	}
	if src == "youtube" {
		q := u.Query()
		q.Set("key", token)
		u.RawQuery = q.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return errors.New("could not create API request")
	}
	req.Header.Set("Accept", "application/json")
	if src == "spotify" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if src == "soundcloud" {
		req.Header.Set("Authorization", "OAuth "+token)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("%s: network request failed", src)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		switch resp.StatusCode {
		case 401:
			return fmt.Errorf("%s: token expired or invalid; supply a new token", src)
		case 403:
			return fmt.Errorf("%s: access denied; check scopes, app access, quota and playlist ownership", src)
		case 429:
			return fmt.Errorf("%s: rate limited (Retry-After: %s)", src, library.Clean(resp.Header.Get("Retry-After")))
		}
		return fmt.Errorf("%s: HTTP %d", src, resp.StatusCode)
	}
	bdy, err := io.ReadAll(io.LimitReader(resp.Body, (8<<20)+1))
	if err != nil {
		return errors.New("could not read API response")
	}
	if len(bdy) > 8<<20 {
		return errors.New("API response exceeds 8 MiB")
	}
	if err = json.Unmarshal(bdy, dst); err != nil {
		return fmt.Errorf("%s: unexpected API response", src)
	}
	return nil
}

type Result struct {
	Tracks               []library.Track
	Warnings             []string
	Attempted, Succeeded int
	Missing              []string
	ImportOnly           []string
}

func (c *Client) Search(ctx context.Context, q string, enabled map[string]bool, local []library.Track) Result {
	type part struct {
		tracks             []library.Track
		err                error
		missing, attempted bool
	}
	parts := make([]part, len(Names))
	var wg sync.WaitGroup
	for i, n := range Names {
		if !enabled[n] {
			continue
		}
		if c.PublicWeb && n == "spotify" && c.Credential(n) == "" {
			continue
		}
		if c.Credential(n) == "" && !c.PublicWeb {
			parts[i].missing = true
			continue
		}
		parts[i].attempted = true
		wg.Add(1)
		go func(i int, n string) { defer wg.Done(); parts[i].tracks, parts[i].err = c.search(ctx, n, q) }(i, n)
	}
	wg.Wait()
	r := Result{Tracks: local}
	if c.PublicWeb && enabled["spotify"] && c.Credential("spotify") == "" {
		r.ImportOnly = append(r.ImportOnly, "spotify")
	}
	for i, p := range parts {
		if p.missing {
			r.Missing = append(r.Missing, Names[i])
			r.Warnings = append(r.Warnings, Label(Names[i])+": not connected. /connect "+Names[i])
			continue
		}
		if p.attempted {
			r.Attempted++
			if p.err == nil {
				r.Succeeded++
			}
		}
		if p.err != nil {
			r.Warnings = append(r.Warnings, Label(Names[i])+": "+p.err.Error())
		}
		for _, t := range p.tracks {
			t.Title = library.Clean(t.Title)
			t.Artist = library.Clean(t.Artist)
			r.Tracks = append(r.Tracks, t)
		}
	}
	return r
}
func (c *Client) search(ctx context.Context, src, q string) ([]library.Track, error) {
	if c.PublicWeb && c.Credential(src) == "" {
		switch src {
		case "soundcloud":
			return c.searchPublicSC(ctx, q)
		case "youtube":
			return c.searchPublicYT(ctx, q)
		}
	}
	switch src {
	case "spotify":
		var r struct {
			Tracks struct {
				Items []spTrack `json:"items"`
			} `json:"tracks"`
		}
		err := c.get(ctx, c.SP+"/search?type=track&limit=10&q="+url.QueryEscape(q), src, &r)
		ts := []library.Track{}
		for _, t := range r.Tracks.Items {
			ts = append(ts, t.track())
		}
		return ts, err
	case "soundcloud":
		var r struct {
			Collection []scTrack `json:"collection"`
		}
		err := c.get(ctx, c.SC+"/tracks?linked_partitioning=true&limit=15&q="+url.QueryEscape(q), src, &r)
		ts := []library.Track{}
		for _, t := range r.Collection {
			ts = append(ts, t.track())
		}
		return ts, err
	case "youtube":
		var r ytPage
		err := c.get(ctx, c.YT+"/search?part=snippet&type=video&videoCategoryId=10&maxResults=15&q="+url.QueryEscape(q), src, &r)
		return r.tracks(), err
	}
	return nil, errors.New("unknown source")
}

var identifier = regexp.MustCompile(`^[A-Za-z0-9_-]{1,160}$`)

func PlaylistURL(raw string) (string, string, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.User != nil || u.Port() != "" {
		return "", "", errors.New("use a full https playlist URL")
	}
	host := strings.ToLower(u.Hostname())
	p := strings.Split(strings.Trim(u.Path, "/"), "/")
	switch host {
	case "open.spotify.com":
		for i, s := range p {
			if s == "playlist" && i+1 < len(p) && identifier.MatchString(p[i+1]) {
				return "spotify", p[i+1], nil
			}
			if s == "track" && i+1 < len(p) && identifier.MatchString(p[i+1]) {
				return "spotify", "track:" + p[i+1], nil
			}
		}
	case "youtube.com", "www.youtube.com", "music.youtube.com", "youtu.be":
		id := u.Query().Get("list")
		if identifier.MatchString(id) {
			return "youtube", id, nil
		}
		video := u.Query().Get("v")
		if host == "youtu.be" && len(p) == 1 {
			video = p[0]
		}
		if identifier.MatchString(video) {
			return "youtube", "video:" + video, nil
		}
	case "soundcloud.com", "www.soundcloud.com":
		if len(p) == 3 && p[1] == "sets" {
			return "soundcloud", u.String(), nil
		}
		if len(p) == 2 && identifier.MatchString(p[0]) && identifier.MatchString(p[1]) {
			return "soundcloud", u.String(), nil
		}
	}
	return "", "", errors.New("expected a SoundCloud /sets/, Spotify /playlist/, or YouTube ?list= URL; expand shortened links first")
}

type Import struct {
	Likes  bool
	Name   string
	Tracks []library.Track
	Notice string
}

func (c *Client) Import(ctx context.Context, raw string) (Import, error) {
	src, id, err := PlaylistURL(raw)
	if err != nil {
		return Import{}, err
	}
	result := Import{}
	if src == "soundcloud" && IsSoundCloudLikes(raw) {
		if !c.PublicWeb {
			return result, errors.New("public SoundCloud likes require public sources mode")
		}
		return c.importPublicSCLikes(ctx, id)
	}
	if src == "spotify" && strings.HasPrefix(id, "track:") && (!c.PublicWeb || c.Credential(src) != "") {
		var t spTrack
		if err = c.get(ctx, c.SP+"/tracks/"+strings.TrimPrefix(id, "track:"), src, &t); err != nil {
			return result, err
		}
		return Import{Name: t.Name, Tracks: []library.Track{t.track()}}, nil
	}
	if c.PublicWeb && c.Credential(src) == "" {
		switch src {
		case "soundcloud":
			return c.importPublicSC(ctx, id)
		case "spotify":
			return c.importPublicSpotify(ctx, id)
		case "youtube":
			return c.importPublicYT(ctx, raw)
		}
	}
	switch src {
	case "spotify":
		var meta struct {
			Name string `json:"name"`
		}
		if err = c.get(ctx, c.SP+"/playlists/"+id, src, &meta); err != nil {
			return result, err
		}
		result.Name = meta.Name
		next := c.SP + "/playlists/" + id + "/items?limit=50"
		seen := map[string]bool{}
		for page := 0; next != ""; page++ {
			if page >= maxPages || seen[next] {
				return result, errors.New("invalid playlist pagination")
			}
			seen[next] = true
			var r struct {
				Items []struct {
					Item   *spTrack `json:"item"`
					Legacy *spTrack `json:"track"`
				} `json:"items"`
				Next string `json:"next"`
			}
			if err = c.get(ctx, next, src, &r); err != nil {
				return result, err
			}
			for _, item := range r.Items {
				t := item.Item
				if t == nil {
					t = item.Legacy
				}
				if t == nil || t.ID == "" {
					result.Tracks = append(result.Tracks, library.Track{ID: fmt.Sprintf("spotify:unavailable:%s:%d", id, len(result.Tracks)), Source: src, Title: "Unavailable playlist item", Unavailable: true})
				} else {
					result.Tracks = append(result.Tracks, t.track())
				}
			}
			if len(result.Tracks) > library.MaxTracks {
				return result, errors.New("playlist exceeds 10,000 items")
			}
			next = r.Next
		}
	case "soundcloud":
		var meta struct {
			URN   string      `json:"urn"`
			ID    json.Number `json:"id"`
			Title string      `json:"title"`
		}
		if err = c.get(ctx, c.SC+"/resolve?url="+url.QueryEscape(id), src, &meta); err != nil {
			return result, err
		}
		key := meta.URN
		if key == "" {
			key = meta.ID.String()
		}
		if key == "" {
			return result, errors.New("SoundCloud returned no playlist identifier")
		}
		result.Name = meta.Title
		next := c.SC + "/playlists/" + url.PathEscape(key) + "/tracks?linked_partitioning=true&limit=200"
		seen := map[string]bool{}
		for page := 0; next != ""; page++ {
			if page >= maxPages || seen[next] {
				return result, errors.New("invalid playlist pagination")
			}
			seen[next] = true
			var r struct {
				Collection []scTrack `json:"collection"`
				Next       string    `json:"next_href"`
			}
			if err = c.get(ctx, next, src, &r); err != nil {
				return result, err
			}
			for _, t := range r.Collection {
				result.Tracks = append(result.Tracks, t.track())
			}
			if len(result.Tracks) > library.MaxTracks {
				return result, errors.New("playlist exceeds 10,000 items")
			}
			next = r.Next
		}
	case "youtube":
		var meta struct {
			Items []struct {
				Snippet struct {
					Title string `json:"title"`
				} `json:"snippet"`
			} `json:"items"`
		}
		if err = c.get(ctx, c.YT+"/playlists?part=snippet&id="+url.QueryEscape(id), src, &meta); err != nil {
			return result, err
		}
		if len(meta.Items) == 0 {
			return result, errors.New("playlist is private, unavailable, or not a public YouTube playlist")
		}
		result.Name = meta.Items[0].Snippet.Title
		pageToken := ""
		seen := map[string]bool{}
		for page := 0; ; page++ {
			if page >= maxPages || seen[pageToken] {
				return result, errors.New("invalid playlist pagination")
			}
			seen[pageToken] = true
			var r ytPage
			next := c.YT + "/playlistItems?part=snippet,contentDetails&maxResults=50&playlistId=" + url.QueryEscape(id) + "&pageToken=" + url.QueryEscape(pageToken)
			if err = c.get(ctx, next, src, &r); err != nil {
				return result, err
			}
			result.Tracks = append(result.Tracks, r.tracks()...)
			if len(result.Tracks) > library.MaxTracks {
				return result, errors.New("playlist exceeds 10,000 items")
			}
			if r.Next == "" {
				break
			}
			pageToken = r.Next
		}
	}
	result.Name = library.Clean(result.Name)
	if result.Name == "" {
		result.Name = src + " playlist"
	}
	return result, nil
}

type spTrack struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Type     string  `json:"type"`
	Duration float64 `json:"duration_ms"`
	Artists  []struct {
		Name string `json:"name"`
	} `json:"artists"`
	External struct {
		ISRC string `json:"isrc"`
	} `json:"external_ids"`
	URLs struct {
		Spotify string `json:"spotify"`
	} `json:"external_urls"`
}

func (t spTrack) track() library.Track {
	a := []string{}
	for _, v := range t.Artists {
		a = append(a, v.Name)
	}
	return library.Track{ID: "spotify:" + t.ID, Source: "spotify", Title: t.Name, Artist: strings.Join(a, ", "), Duration: t.Duration / 1000, URL: t.URLs.Spotify, ISRC: t.External.ISRC, Unavailable: t.Type != "" && t.Type != "track"}
}

type scTrack struct {
	ID       json.Number `json:"id"`
	URN      string      `json:"urn"`
	Title    string      `json:"title"`
	Duration float64     `json:"duration"`
	URL      string      `json:"permalink_url"`
	Access   string      `json:"access"`
	User     struct {
		Name string `json:"username"`
	} `json:"user"`
}

func (t scTrack) track() library.Track {
	id := t.URN
	if id == "" {
		id = t.ID.String()
	}
	return library.Track{ID: "soundcloud:" + id, Source: "soundcloud", Title: t.Title, Artist: t.User.Name, Duration: t.Duration / 1000, URL: t.URL, Unavailable: t.Access == "blocked"}
}

type ytPage struct {
	Next  string `json:"nextPageToken"`
	Items []struct {
		ID      json.RawMessage `json:"id"`
		Snippet struct {
			Title    string `json:"title"`
			Channel  string `json:"channelTitle"`
			Owner    string `json:"videoOwnerChannelTitle"`
			Resource struct {
				Video string `json:"videoId"`
			} `json:"resourceId"`
		} `json:"snippet"`
		Details struct {
			Video string `json:"videoId"`
		} `json:"contentDetails"`
	} `json:"items"`
}

func (p ytPage) tracks() []library.Track {
	ts := []library.Track{}
	for _, i := range p.Items {
		id := i.Details.Video
		if id == "" {
			id = i.Snippet.Resource.Video
		}
		if id == "" {
			var searchID struct {
				Video string `json:"videoId"`
			}
			_ = json.Unmarshal(i.ID, &searchID)
			id = searchID.Video
		}
		if id == "" {
			continue
		}
		artist := i.Snippet.Owner
		if artist == "" {
			artist = i.Snippet.Channel
		}
		ts = append(ts, library.Track{ID: "youtube:" + id, Source: "youtube", Title: i.Snippet.Title, Artist: artist, URL: "https://www.youtube.com/watch?v=" + id, Unavailable: i.Snippet.Title == "Private video" || i.Snippet.Title == "Deleted video"})
	}
	return ts
}

func (c *Client) Stream(ctx context.Context, t library.Track) (string, error) {
	if t.Unavailable {
		return "", errors.New("this track is unavailable")
	}
	if t.Source == "local" {
		if _, err := os.Stat(t.Path); err != nil {
			return "", err
		}
		return t.Path, nil
	}
	if c.PublicWeb {
		switch t.Source {
		case "soundcloud":
			if c.Credential("soundcloud") == "" {
				return c.streamPublicSC(ctx, t)
			}
		case "youtube":
			return c.streamPublicYT(ctx, t)
		case "spotify":
			return c.streamMatchedSpotify(ctx, t)
		}
	}
	if t.Source != "soundcloud" {
		return "", fmt.Errorf("%s is metadata only: /open to listen in its player, or /match then /bind a local or SoundCloud recording", t.Source)
	}
	id := strings.TrimPrefix(t.ID, "soundcloud:")
	var r struct {
		HLS    string `json:"hls_aac_160_url"`
		HTTP   string `json:"http_mp3_128_url"`
		Legacy string `json:"http_mp3_128_url_legacy"`
	}
	if err := c.get(ctx, c.SC+"/tracks/"+url.PathEscape(id)+"/streams", "soundcloud", &r); err != nil {
		return "", err
	}
	raw := r.HLS
	if raw == "" {
		raw = r.HTTP
	}
	if raw == "" {
		raw = r.Legacy
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return "", errors.New("SoundCloud did not supply an authorized stream for this track")
	}
	return raw, nil
}
