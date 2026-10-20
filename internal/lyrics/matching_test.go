package lyrics

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"wisp/internal/library"
)

func TestArtistCreditMatching(t *testing.T) {
	for _, test := range []struct {
		candidate, requested string
		want                 bool
	}{
		{"tryavoid, zombiqe", "tryavoid", true},
		{"TRYAVOID, zombiqe", "  Tryavoid  ", true},
		{"tryavoid", "tryavoid, zombiqe", true},
		{"zombiqe & tryavoid", "tryavoid feat. zombiqe", true},
		{"Артист, Другой артист", "артист", true},
		{"tryavoid, zombiqe", "try", false},
		{"nottryavoid", "tryavoid", false},
		{"tryavoid, Someone", "tryavoid, zombiqe", false},
		{"Someone, Else", "tryavoid", false},
		{"", "tryavoid", false},
		{"tryavoid", "", false},
	} {
		if got := artistCreditMatches(test.candidate, test.requested); got != test.want {
			t.Errorf("%q / %q = %v; want %v", test.candidate, test.requested, got, test.want)
		}
	}
}

func TestCollaborationLyricsLookupKeepsRecordingChecks(t *testing.T) {
	for _, test := range []struct {
		name, title, artist string
		duration            float64
		wantMatch           bool
	}{
		{"collaboration", "#нашласелоха", "tryavoid, zombiqe", 125, true},
		{"wrong duration", "#нашласелоха", "tryavoid, zombiqe", 160, false},
		{"wrong artist", "#нашласелоха", "Someone else", 125, false},
		{"wrong version", "#нашласелоха (remix)", "tryavoid, zombiqe", 125, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("artist_name") != "tryavoid" || r.URL.Query().Get("track_name") != "#нашласелоха" {
					t.Error("request metadata changed", r.URL)
				}
				if r.URL.Path == "/api/get" {
					http.NotFound(w, r)
					return
				}
				fmt.Fprintf(w, `[{"trackName":%q,"artistName":%q,"duration":%v,"syncedLyrics":"[00:03.285]Invented collaboration fixture\n[00:09.630]Invented second fixture"}]`, test.title, test.artist, test.duration)
			}))
			defer server.Close()
			client := &Client{HTTP: server.Client(), Base: server.URL}
			song, err := client.Find(context.Background(), library.Track{Source: "soundcloud", Artist: "tryavoid", Title: "#нашласелоха", Duration: 125.381})
			if test.wantMatch {
				if err != nil || len(song.Lines) != 2 || song.Lines[0].Time != 3.285 {
					t.Fatal("collaboration lyrics missing", song, err)
				}
			} else if err == nil {
				t.Fatal("wrong recording accepted")
			}
		})
	}
}

func TestCleanedUploadMatchesCollaboration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/get" {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("track_name") != "Fixture" {
			fmt.Fprint(w, `[]`)
			return
		}
		fmt.Fprint(w, `[{"trackName":"Fixture","artistName":"Artist, Collaborator","duration":125,"syncedLyrics":"[00:01]Invented cleaned fixture"}]`)
	}))
	defer server.Close()
	client := &Client{HTTP: server.Client(), Base: server.URL}
	song, err := client.Find(context.Background(), library.Track{Source: "soundcloud", Artist: "Artist", Title: "Fixture (uploader123)", Duration: 125})
	if err != nil || len(song.Lines) != 1 {
		t.Fatal("cleaned title discarded collaboration", song, err)
	}
}
