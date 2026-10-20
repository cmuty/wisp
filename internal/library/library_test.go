package library

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPersistenceAndCorruption(t *testing.T) {
	p := filepath.Join(t.TempDir(), "nested", "library.json")
	s := New()
	s.Profile.Username = "echo"
	s.Profile.Complete = true
	s.Put([]Track{{ID: "local:1", Title: "x", Source: "local"}})
	s.Queue = []string{"local:1", "local:1"}
	if err := Save(p, s); err != nil {
		t.Fatal(err)
	}
	s.Profile.Theme = "ember"
	if err := Save(p, s); err != nil {
		t.Fatal(err)
	}
	r, err := Load(p)
	if err != nil || len(r.Queue) != 2 || r.Profile.Theme != "ember" {
		t.Fatalf("roundtrip: %#v, %v", r, err)
	}
	if err := os.WriteFile(p, []byte("corrupt"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(p); err == nil {
		t.Fatal("must report corruption, not replace the library")
	}
	b, _ := os.ReadFile(p)
	if string(b) != "corrupt" {
		t.Fatal("modified corrupt file")
	}
}
func TestLocalScanAndBinding(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"one.flac", "two.wav", "cover.jpg"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("fixture"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	ts, err := Scan(dir)
	if err != nil || len(ts) != 2 {
		t.Fatalf("scan: %d %v", len(ts), err)
	}
	s := New()
	s.Put(ts)
	s.Put([]Track{{ID: "spotify:1", Source: "spotify", Title: "original"}})
	s.Bindings["spotify:1"] = ts[0].ID
	if s.Resolve("spotify:1").Source != "local" || s.Tracks["spotify:1"].Title != "original" {
		t.Fatal("binding replaced the source identity")
	}
}
