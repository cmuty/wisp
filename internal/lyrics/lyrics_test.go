package lyrics

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"wisp/internal/library"
)

func TestParseAndSeek(t *testing.T) {
	rows := Parse("[offset:-500]\n[00:03.00]third\n[00:01.00][00:02.00]echo\n[ar:fixture]\n[00:99]invalid")
	if len(rows) != 3 || rows[0].Time != 0.5 || rows[2].Time != 2.5 {
		t.Fatalf("%+v", rows)
	}
	for _, v := range []struct {
		pos  float64
		want int
	}{{0, -1}, {0.5, 0}, {2, 1}, {99, 2}, {0, -1}} {
		if got := Active(rows, v.pos); got != v.want {
			t.Fatalf("position %v: %d", v.pos, got)
		}
	}
}
func TestLookupAndNoWrongRecording(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("artist_name") != "Fixture artist" || r.URL.Query().Get("track_name") != "Fixture song" {
			t.Error("bad metadata", r.URL)
		}
		if r.URL.Path == "/api/get" {
			http.NotFound(w, r)
			return
		}
		fmt.Fprint(w, `[{"trackName":"Fixture song","artistName":"Fixture artist","duration":999,"syncedLyrics":"[00:01]Wrong version"},{"trackName":"Fixture song","artistName":"Fixture artist","duration":120,"syncedLyrics":"[00:01]Invented fixture line"}]`)
	}))
	defer server.Close()
	c := &Client{HTTP: server.Client(), Base: server.URL}
	s, e := c.Find(context.Background(), library.Track{Source: "soundcloud", Artist: "uploader", Title: "Fixture artist - Fixture song", Duration: 120})
	if e != nil || len(s.Lines) != 1 || s.Lines[0].Text != "Invented fixture line" {
		t.Fatal(s, e)
	}
}
func TestLocalLRCAndRateLimit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sample.wav")
	if e := os.WriteFile(filepath.Join(filepath.Dir(path), "sample.lrc"), []byte("[00:00]Local fixture"), 0600); e != nil {
		t.Fatal(e)
	}
	s, e := New().Find(context.Background(), library.Track{Path: path})
	if e != nil || s.Credit != "Local LRC" {
		t.Fatal(s, e)
	}
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++; w.WriteHeader(429) }))
	defer server.Close()
	_, e = (&Client{HTTP: server.Client(), Base: server.URL}).Find(context.Background(), library.Track{Title: "fixture"})
	if e == nil || calls != 1 {
		t.Fatal("must not retry rate limit", calls, e)
	}
}
