package lyrics

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"wisp/internal/library"
)

func Test503RetryRecovers(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls < 3 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(503)
			return
		}
		fmt.Fprint(w, `{"syncedLyrics":"[00:00]Invented recovery fixture"}`)
	}))
	defer server.Close()
	c := &Client{HTTP: server.Client(), Base: server.URL}
	s, e := c.Find(context.Background(), library.Track{Title: "fixture", Artist: "fixture", Duration: 120})
	if e != nil || calls != 3 || len(s.Lines) != 1 {
		t.Fatal("did not recover", e, calls)
	}
}
func Test503BoundedAndRetryAfter(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(503)
	}))
	defer server.Close()
	c := &Client{HTTP: server.Client(), Base: server.URL}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, e := c.Find(ctx, library.Track{Title: "fixture"})
	if !statusIs(e, 503) || calls != 1 || !strings.Contains(Message(e), "F5") {
		t.Fatal(e, calls)
	}
	ctx2, cancel2 := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel2()
	_, e = c.Find(ctx2, library.Track{Title: "fixture"})
	if calls != 1 {
		t.Fatal("retry ignored server cooldown", e, calls)
	}
}
func TestCacheSurvivesRestartAndServiceOutage(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		fmt.Fprint(w, `{"syncedLyrics":"[00:01]Invented cached fixture"}`)
	}))
	defer server.Close()
	dir := filepath.Join(t.TempDir(), "cache")
	track := library.Track{ID: "a", Title: "fixture", Duration: 120}
	c := &Client{HTTP: server.Client(), Base: server.URL, CacheDir: dir}
	first, e := c.Find(context.Background(), track)
	if e != nil {
		t.Fatal(e)
	}
	server.Close()
	second, e := (&Client{HTTP: server.Client(), Base: server.URL, CacheDir: dir}).Find(context.Background(), track)
	if e != nil || calls != 1 || len(second.Lines) != len(first.Lines) || !strings.Contains(second.Credit, "cached") {
		t.Fatal("cache unavailable", e, calls, second.Credit)
	}
}
func TestCleanUploadCreditKeepsRecordingVersions(t *testing.T) {
	if cleanUploadTitle("набор положительных качеств(22kageyoshi)") != "набор положительных качеств" {
		t.Fatal("upload credit kept")
	}
	for _, s := range []string{"Song (live)", "Song (remix)", "Song (acoustic)"} {
		if cleanUploadTitle(s) != s {
			t.Fatal("recording version removed", s)
		}
	}
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path == "/api/get" {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("track_name") == "Fixture (uploader123)" {
			fmt.Fprint(w, "[]")
			return
		}
		fmt.Fprint(w, `[{"trackName":"Fixture","artistName":"Artist","duration":120,"syncedLyrics":"[00:00]Invented normalized fixture"}]`)
	}))
	defer server.Close()
	s, e := (&Client{HTTP: server.Client(), Base: server.URL}).Find(context.Background(), library.Track{Source: "soundcloud", Artist: "uploader", Title: "Artist - Fixture (uploader123)", Duration: 120})
	if e != nil || calls != 4 || len(s.Lines) != 1 {
		t.Fatal("normalized lookup failed", e, calls)
	}
}
