package library

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

const MaxTracks = 10000

type Track struct {
	ID          string  `json:"id"`
	Source      string  `json:"source"`
	Title       string  `json:"title"`
	Artist      string  `json:"artist"`
	URL         string  `json:"url,omitempty"`
	Path        string  `json:"path,omitempty"`
	Duration    float64 `json:"duration,omitempty"`
	ISRC        string  `json:"isrc,omitempty"`
	Unavailable bool    `json:"unavailable,omitempty"`
}

type Profile struct {
	Language      string          `json:"language,omitempty"`
	Username      string          `json:"username"`
	Theme         string          `json:"theme"`
	Sources       map[string]bool `json:"sources"`
	Complete      bool            `json:"complete"`
	HideButterfly bool            `json:"hide_butterfly,omitempty"`
}

type State struct {
	Version   int                 `json:"version"`
	Profile   Profile             `json:"profile"`
	Tracks    map[string]Track    `json:"tracks"`
	Queue     []string            `json:"queue"`
	Likes     []string            `json:"likes"`
	Recent    []string            `json:"recent,omitempty"`
	Playlists map[string][]string `json:"playlists"`
	Bindings  map[string]string   `json:"bindings"`
}

func New() State {
	return State{Version: 1, Profile: Profile{Theme: "mist", Sources: map[string]bool{"soundcloud": true, "spotify": true, "youtube": true}}, Tracks: map[string]Track{}, Playlists: map[string][]string{}, Bindings: map[string]string{}}
}

func DefaultPath() string {
	if p := os.Getenv("WISP_HOME"); p != "" {
		return filepath.Join(p, "library.json")
	}
	p, err := os.UserConfigDir()
	if err != nil {
		p = "."
	}
	return filepath.Join(p, "wisp", "library.json")
}

func Load(path string) (State, error) {
	s := New()
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return s, err
	}
	if len(b) > 32<<20 {
		return s, errors.New("library exceeds 32 MiB")
	}
	if err = json.Unmarshal(b, &s); err != nil {
		return s, fmt.Errorf("library is invalid; keep the original file and repair it: %w", err)
	}
	if s.Version != 1 {
		return s, fmt.Errorf("unsupported library version %d", s.Version)
	}
	if s.Tracks == nil {
		s.Tracks = map[string]Track{}
	}
	if s.Playlists == nil {
		s.Playlists = map[string][]string{}
	}
	if s.Bindings == nil {
		s.Bindings = map[string]string{}
	}
	if s.Profile.Sources == nil {
		s.Profile.Sources = map[string]bool{}
	}
	return s, nil
}

func Save(path string, s State) error {
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	if len(b) > 32<<20 {
		return errors.New("library exceeds 32 MiB; remove unused playlists before saving")
	}
	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".wisp-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if err = f.Chmod(0600); err != nil {
		f.Close()
		return err
	}
	if _, err = f.Write(b); err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(f.Name(), path)
}

func Clean(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
}
func (s *State) Put(ts []Track) []string {
	ids := make([]string, 0, len(ts))
	for _, t := range ts {
		t.Title = Clean(t.Title)
		t.Artist = Clean(t.Artist)
		s.Tracks[t.ID] = t
		ids = append(ids, t.ID)
	}
	return ids
}
func (s State) Resolve(id string) Track {
	if target, ok := s.Bindings[id]; ok {
		if t, found := s.Tracks[target]; found {
			return t
		}
	}
	return s.Tracks[id]
}
func (s State) LocalSearch(q string) []Track {
	words := strings.Fields(normalize(q))
	ts := []Track{}
	for _, t := range s.Tracks {
		candidate := normalize(t.Artist + " " + t.Title)
		match := true
		for _, word := range words {
			if !strings.Contains(candidate, word) {
				match = false
				break
			}
		}
		if t.Source == "local" && match {
			ts = append(ts, t)
		}
	}
	sort.Slice(ts, func(i, j int) bool { return ts[i].Title < ts[j].Title })
	return ts
}
func normalize(s string) string {
	return strings.ToLower(strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			return r
		}
		return ' '
	}, s))
}
func Local(path string) (Track, error) {
	p, err := filepath.Abs(path)
	if err != nil {
		return Track{}, err
	}
	info, err := os.Stat(p)
	if err != nil {
		return Track{}, err
	}
	if !info.Mode().IsRegular() {
		return Track{}, errors.New("expected an audio file")
	}
	switch strings.ToLower(filepath.Ext(p)) {
	case ".mp3", ".flac", ".wav", ".ogg", ".opus", ".m4a", ".aac", ".aiff", ".alac":
	default:
		return Track{}, errors.New("unsupported audio extension")
	}
	h := sha256.Sum256([]byte(strings.ToLower(p)))
	name := strings.TrimSuffix(filepath.Base(p), filepath.Ext(p))
	return Track{ID: "local:" + hex.EncodeToString(h[:12]), Source: "local", Title: name, Artist: "Local library", Path: p}, nil
}
func Scan(path string) ([]Track, error) {
	ts := []Track{}
	err := filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		t, e := Local(p)
		if e != nil {
			return nil
		}
		if len(ts) >= MaxTracks {
			return errors.New("import exceeds 10,000 tracks; split the folder")
		}
		ts = append(ts, t)
		return nil
	})
	if err == nil && len(ts) == 0 {
		err = errors.New("no supported audio files found")
	}
	return ts, err
}
