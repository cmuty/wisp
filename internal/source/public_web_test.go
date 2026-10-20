package source

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"wisp/internal/library"
	"wisp/internal/player"
)

func TestPublicSoundCloudSearchAndStream(t *testing.T) {
	t.Setenv("WISP_SOUNDCLOUD_TOKEN", "")
	var base string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			t.Error("public mode sent account authorization")
		}
		if r.URL.Query().Get("client_id") != "public-fixture" {
			t.Error("public client ID missing")
		}
		switch r.URL.Path {
		case "/search/tracks":
			fmt.Fprint(w, `{"collection":[{"id":42,"title":"Track","duration":120000,"user":{"username":"Artist"}}]}`)
		case "/tracks/42":
			fmt.Fprintf(w, `{"id":42,"policy":"ALLOW","streamable":true,"media":{"transcodings":[{"url":%q,"format":{"protocol":"hls"}}]}}`, base+"/media/42")
		case "/media/42":
			fmt.Fprint(w, `{"url":"https://cdn.example.test/audio.m3u8?expires=123"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	base = server.URL
	c := New()
	c.WebSCAPI = base
	c.scClientID = "public-fixture"
	c.scExpires = time.Now().Add(time.Hour)
	tracks, err := c.searchPublicSC(context.Background(), "track")
	if err != nil || len(tracks) != 1 {
		t.Fatalf("%v %v", tracks, err)
	}
	stream, err := c.streamPublicSC(context.Background(), tracks[0])
	if err != nil || !strings.HasPrefix(stream, "https://cdn.example.test/") {
		t.Fatalf("stream resolution failed: %v", err)
	}
}
func TestPublicSoundCloudRejectsRestrictedTrack(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"id":42,"policy":"SNIP","streamable":true}`)
	}))
	defer s.Close()
	c := New()
	c.WebSCAPI = s.URL
	c.scClientID = "fixture"
	c.scExpires = time.Now().Add(time.Hour)
	if _, err := c.streamPublicSC(context.Background(), library.Track{ID: "soundcloud:42"}); err == nil {
		t.Fatal("preview-only track treated as a full stream")
	}
}
func TestSpotifyEmbedSnapshotPreservesDuplicates(t *testing.T) {
	b := []byte(`<script id="__NEXT_DATA__" type="application/json">{"props":{"pageProps":{"state":{"data":{"entity":{"title":"Night","trackList":[{"uri":"spotify:track:one","title":"Song","subtitle":"Artist","duration":120000},{"uri":"spotify:track:one","title":"Song","subtitle":"Artist","duration":120000}]}}}}}}</script>`)
	r, err := parseSpotifyEmbed(b, "playlist")
	if err != nil || len(r.Tracks) != 2 || r.Tracks[0].ID != r.Tracks[1].ID || r.Notice == "" || !strings.Contains(r.Name, "snapshot") {
		t.Fatalf("snapshot semantics missing: %#v %v", r, err)
	}
}
func TestMusicSearchParserUsesSongsAndArtists(t *testing.T) {
	b := []byte(`{"contents":[{"musicResponsiveListItemRenderer":{"flexColumns":[{"musicResponsiveListItemFlexColumnRenderer":{"text":{"runs":[{"text":"Москва слезам не верит","navigationEndpoint":{"watchEndpoint":{"videoId":"valid_ID-42"}}}]}}},{"musicResponsiveListItemFlexColumnRenderer":{"text":{"runs":[{"text":"Artist"},{"text":" • "},{"text":"Album"},{"text":" • "},{"text":"3:25"}]}}}]}}]}`)
	ts, err := parseMusicSearch(b)
	if err != nil || len(ts) != 1 || ts[0].Artist != "Artist" || ts[0].Duration != 205 || ts[0].ID != "youtube:valid_ID-42" {
		t.Fatalf("bad music result: %#v %v", ts, err)
	}
}
func TestPublicRequestsRejectOtherOrigins(t *testing.T) {
	c := New()
	if _, err := c.publicBytes(context.Background(), "https://other.example.test/steal"); err == nil {
		t.Fatal("cross-origin endpoint accepted")
	}
}

func TestPublicBrowserChallengeIsReported(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("x-amzn-waf-action", "challenge")
		w.WriteHeader(202)
	}))
	defer s.Close()
	c := New()
	c.WebSC = s.URL
	if _, err := c.publicBytes(context.Background(), s.URL); err == nil || !strings.Contains(err.Error(), "browser verification") {
		t.Fatalf("challenge hidden: %v", err)
	}
}
func TestPublicSpotifyIsImportOnly(t *testing.T) {
	t.Setenv("WISP_SPOTIFY_TOKEN", "")
	c := New()
	r := c.Search(context.Background(), "song", map[string]bool{"spotify": true}, nil)
	if len(r.ImportOnly) != 1 || r.Attempted != 0 || len(r.Missing) != 0 {
		t.Fatalf("Spotify import treated as missing account: %#v", r)
	}
}
func TestCSVCompleteMetadata(t *testing.T) {
	path := filepath.Join(t.TempDir(), "playlist.csv")
	_ = os.WriteFile(path, []byte("\xef\xbb\xbfTrack Name,Artist Name(s),Spotify URI,Duration (ms)\nSong,Artist,spotify:track:abc123,200000\nSong,Artist,spotify:track:abc123,200000\n"), 0600)
	r, err := ReadCSV(path)
	if err != nil || len(r.Tracks) != 2 || r.Tracks[0].Duration != 200 || r.Tracks[0].ID != r.Tracks[1].ID {
		t.Fatalf("CSV incomplete: %#v %v", r, err)
	}
}
func TestMatchingRejectsWrongVersionAndDuration(t *testing.T) {
	a := library.Track{Title: "Quiet Song", Artist: "Artist", Duration: 200}
	if matchScore(a, library.Track{Title: "Artist Quiet Song Official Audio", Duration: 201}) < 0.9 {
		t.Fatal("good match rejected")
	}
	for _, b := range []library.Track{{Title: "Artist Quiet Song live", Duration: 200}, {Title: "Artist Quiet Song", Duration: 400}, {Title: "Unrelated music", Duration: 200}} {
		if matchScore(a, b) > 0 {
			t.Fatal("bad recording matched")
		}
	}
}

func TestLivePublicSources(t *testing.T) {
	if os.Getenv("WISP_TEST_PUBLIC") != "1" {
		t.Skip("set WISP_TEST_PUBLIC=1 for real public websites")
	}
	for _, key := range []string{"WISP_SOUNDCLOUD_TOKEN", "WISP_SPOTIFY_TOKEN", "WISP_YOUTUBE_KEY"} {
		t.Setenv(key, "")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	c := New()
	c.PublicWeb = true
	t.Run("SoundCloud", func(t *testing.T) {
		ts, err := c.searchPublicSC(ctx, "москва слезам не верит")
		if err != nil || len(ts) == 0 {
			t.Fatalf("public search failed: %v", err)
		}
		t.Logf("%d results without user token", len(ts))
		if _, err = c.streamPublicSC(ctx, ts[0]); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("YouTubeMusic", func(t *testing.T) {
		ts, err := c.searchPublicYT(ctx, "москва слезам не верит")
		if err != nil || len(ts) == 0 {
			t.Fatalf("public music search failed: %v", err)
		}
		t.Logf("%d song results without API key", len(ts))
		for _, track := range ts {
			if track.Duration > 1800 {
				t.Fatal("movie returned by song search")
			}
		}
	})
	t.Run("SpotifyEmbed", func(t *testing.T) {
		r, err := c.importPublicSpotify(ctx, "37i9dQZF1DXcBWIGoYBM5M")
		if err != nil || len(r.Tracks) == 0 {
			t.Fatalf("public import failed: %v", err)
		}
		t.Logf("%d public metadata entries; limit disclosed", len(r.Tracks))
	})
	t.Run("YouTubeStream", func(t *testing.T) {
		if _, err := extractorPath(); err != nil {
			t.Skip("yt-dlp not installed")
		}
		tracks, err := c.searchPublicYT(ctx, "москва слезам не верит")
		if err != nil || len(tracks) == 0 {
			t.Fatal("could not select public music recording")
		}
		if _, err := c.streamPublicYT(ctx, tracks[0]); err != nil {
			t.Fatal(err)
		}
		t.Log("resolved public test video without cookies or user token")
	})
	t.Run("SpotifyMatch", func(t *testing.T) {
		if _, err := extractorPath(); err != nil {
			t.Skip("yt-dlp not installed")
		}
		r, err := c.importPublicSpotify(ctx, "37i9dQZF1DXcBWIGoYBM5M")
		if err != nil || len(r.Tracks) == 0 {
			t.Fatal("could not import public metadata")
		}
		if _, err = c.streamMatchedSpotify(ctx, r.Tracks[0]); err != nil {
			t.Fatal(err)
		}
		t.Log("public Spotify metadata matched to an accessible YouTube recording")
	})
}

func TestLivePublicPlayback(t *testing.T) {
	if os.Getenv("WISP_TEST_PUBLIC") != "1" || os.Getenv("WISP_TEST_MPV") == "" {
		t.Skip("set WISP_TEST_PUBLIC=1 and WISP_TEST_MPV for null-output playback")
	}
	t.Setenv("WISP_MPV", os.Getenv("WISP_TEST_MPV"))
	t.Setenv("WISP_AUDIO_OUTPUT", "null")
	t.Setenv("WISP_SOUNDCLOUD_TOKEN", "")
	c := New()
	c.PublicWeb = true
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	for _, name := range []string{"SoundCloud", "YouTube"} {
		t.Run(name, func(t *testing.T) {
			var raw string
			var err error
			if name == "SoundCloud" {
				ts, e := c.searchPublicSC(ctx, "москва слезам не верит")
				if e != nil || len(ts) == 0 {
					t.Fatal("no public track")
				}
				raw, err = c.streamPublicSC(ctx, ts[0])
			} else {
				ts, e := c.searchPublicYT(ctx, "москва слезам не верит")
				if e != nil || len(ts) == 0 {
					t.Fatal("no public track")
				}
				raw, err = c.streamPublicYT(ctx, ts[0])
			}
			if err != nil {
				t.Fatal(err)
			}
			engine := player.New()
			defer engine.Close()
			if err = engine.Load(raw); err != nil {
				t.Fatal(err)
			}
			deadline := time.Now().Add(12 * time.Second)
			for time.Now().Before(deadline) {
				s := engine.Snapshot()
				if s.Error != "" {
					t.Fatal(s.Error)
				}
				if s.Active && s.Position > 0.05 {
					t.Log("decoded public audio via mpv null output")
					return
				}
				time.Sleep(40 * time.Millisecond)
			}
			t.Fatal("public stream did not begin decoding")
		})
	}
}
