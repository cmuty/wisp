package source

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPlaylistURLs(t *testing.T) {
	for _, raw := range []string{"https://soundcloud.com/artist/sets/night", "https://open.spotify.com/playlist/abc123?si=foo", "https://music.youtube.com/playlist?list=PL123"} {
		if _, _, err := PlaylistURL(raw); err != nil {
			t.Error(raw, err)
		}
	}
	for _, raw := range []string{"https://soundcloud.com.evil.test/u/sets/x", "https://open.spotify.com:444/playlist/abc", "http://youtube.com/playlist?list=x", "https://youtube.com/playlist?list=../x", "https://user@open.spotify.com/playlist/abc"} {
		if _, _, err := PlaylistURL(raw); err == nil {
			t.Error("accepted", raw)
		}
	}
}
func TestSpotifyWholePlaylist(t *testing.T) {
	t.Setenv("WISP_SPOTIFY_TOKEN", "token")
	var base string
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer token" {
			t.Error("missing auth")
		}
		switch r.URL.Path {
		case "/playlists/abc":
			fmt.Fprint(w, `{"name":"Night"}`)
		case "/playlists/abc/items":
			calls++
			if r.URL.Query().Get("offset") == "2" {
				fmt.Fprint(w, `{"items":[{"item":null}],"next":null}`)
			} else {
				fmt.Fprintf(w, `{"items":[{"item":{"id":"one","name":"One","artists":[{"name":"Artist"}]}},{"item":{"id":"one","name":"One"}}],"next":%q}`, base+"/playlists/abc/items?offset=2")
			}
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	base = server.URL
	c := New()
	c.SP = base
	r, err := c.Import(context.Background(), "https://open.spotify.com/playlist/abc")
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 || len(r.Tracks) != 3 || r.Tracks[0].ID != r.Tracks[1].ID || !r.Tracks[2].Unavailable {
		t.Fatalf("lost order, duplicates or unavailable entry: %#v", r)
	}
}
func TestYouTubePagination(t *testing.T) {
	t.Setenv("WISP_YOUTUBE_KEY", "key")
	calls := 0
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("key") != "key" {
			t.Error("missing API key")
		}
		if r.URL.Path == "/playlists" {
			fmt.Fprint(w, `{"items":[{"snippet":{"title":"Library"}}]}`)
			return
		}
		calls++
		if r.URL.Query().Get("pageToken") == "next" {
			fmt.Fprint(w, `{"items":[{"snippet":{"title":"Two","resourceId":{"videoId":"two"}}}]}`)
		} else {
			fmt.Fprint(w, `{"nextPageToken":"next","items":[{"contentDetails":{"videoId":"one"},"snippet":{"title":"One"}}]}`)
		}
	}))
	defer s.Close()
	c := New()
	c.YT = s.URL
	r, err := c.Import(context.Background(), "https://music.youtube.com/playlist?list=PL123")
	if err != nil || calls != 2 || len(r.Tracks) != 2 || r.Tracks[1].ID != "youtube:two" {
		t.Fatalf("%#v %v", r, err)
	}
}
func TestSoundCloudResolveAndPagination(t *testing.T) {
	t.Setenv("WISP_SOUNDCLOUD_TOKEN", "token")
	var base string
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "OAuth token" {
			t.Error("missing OAuth")
		}
		switch r.URL.Path {
		case "/resolve":
			fmt.Fprint(w, `{"urn":"soundcloud:playlists:1","title":"Clouds"}`)
		case "/playlists/soundcloud:playlists:1/tracks":
			if r.URL.Query().Get("cursor") == "2" {
				fmt.Fprint(w, `{"collection":[{"id":2,"title":"Blocked","access":"blocked"}]}`)
			} else {
				fmt.Fprintf(w, `{"collection":[{"id":1,"title":"One","user":{"username":"Echo"}}],"next_href":%q}`, base+"/playlists/soundcloud:playlists:1/tracks?cursor=2")
			}
		default:
			http.NotFound(w, r)
		}
	}))
	defer s.Close()
	base = s.URL
	c := New()
	c.SC = base
	r, err := c.Import(context.Background(), "https://soundcloud.com/echo/sets/clouds")
	if err != nil || len(r.Tracks) != 2 || !r.Tracks[1].Unavailable {
		t.Fatalf("%#v %v", r, err)
	}
}
func TestPaginationCannotLeakCredentials(t *testing.T) {
	t.Setenv("WISP_SPOTIFY_TOKEN", "secret")
	hit := false
	evil := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hit = true }))
	defer evil.Close()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/items") {
			fmt.Fprintf(w, `{"items":[],"next":%q}`, evil.URL+"/steal")
		} else {
			fmt.Fprint(w, `{"name":"Test"}`)
		}
	}))
	defer s.Close()
	c := New()
	c.SP = s.URL
	_, err := c.Import(context.Background(), "https://open.spotify.com/playlist/abc")
	if err == nil || hit {
		t.Fatalf("cross-origin page was followed: %v", err)
	}
}
func TestPartialSearchAndNoSecretInError(t *testing.T) {
	t.Setenv("WISP_OFFICIAL_API_ONLY", "1")
	t.Setenv("WISP_YOUTUBE_KEY", "do-not-log")
	t.Setenv("WISP_SOUNDCLOUD_TOKEN", "")
	t.Setenv("WISP_SPOTIFY_TOKEN", "")
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "bad", 403) }))
	defer s.Close()
	c := New()
	c.YT = s.URL
	r := c.Search(context.Background(), "test", map[string]bool{"soundcloud": true, "spotify": true, "youtube": true}, nil)
	if len(r.Warnings) != 3 {
		t.Fatalf("warnings: %v", r.Warnings)
	}
	for _, s := range r.Warnings {
		if strings.Contains(s, "do-not-log") {
			t.Fatal("secret leaked")
		}
	}
}

func TestMissingCredentialsSkipNetwork(t *testing.T) {
	t.Setenv("WISP_OFFICIAL_API_ONLY", "1")
	for _, name := range []string{"WISP_SPOTIFY_TOKEN", "WISP_SOUNDCLOUD_TOKEN", "WISP_YOUTUBE_KEY"} {
		t.Setenv(name, "")
	}
	calls := 0
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		t.Error("unconfigured source made an HTTP request")
	}))
	defer s.Close()
	c := New()
	c.SP = s.URL
	c.SC = s.URL
	c.YT = s.URL
	r := c.Search(context.Background(), "song", map[string]bool{"spotify": true, "soundcloud": true, "youtube": true}, nil)
	if calls != 0 || r.Attempted != 0 || r.Succeeded != 0 || len(r.Missing) != 3 {
		t.Fatalf("unexpected search status: %#v", r)
	}
}
func TestSessionCredentialWorksWithoutEnvironment(t *testing.T) {
	t.Setenv("WISP_SPOTIFY_TOKEN", "")
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer session-fixture" {
			t.Error("session token missing")
		}
		fmt.Fprint(w, `{"tracks":{"items":[]}}`)
	}))
	defer s.Close()
	c := New()
	c.SP = s.URL
	if err := c.SetCredential("spotify", "session-fixture"); err != nil {
		t.Fatal(err)
	}
	r := c.Search(context.Background(), "missing song", map[string]bool{"spotify": true}, nil)
	if r.Attempted != 1 || r.Succeeded != 1 || len(r.Missing) != 0 || len(r.Tracks) != 0 {
		t.Fatalf("empty success not tracked: %#v", r)
	}
}
