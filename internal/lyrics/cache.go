package lyrics

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"wisp/internal/library"
)

const maxCacheEntries = 128

func (c *Client) cachePath(t library.Track) string {
	if c.CacheDir == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%.0f", t.ID, t.Source, t.Artist, t.Title, t.Path, t.Duration)))
	return filepath.Join(c.CacheDir, hex.EncodeToString(sum[:])+".json")
}
func (c *Client) cached(t library.Track) (Song, bool) {
	path := c.cachePath(t)
	if path == "" {
		return Song{}, false
	}
	f, e := os.Open(path)
	if e != nil {
		return Song{}, false
	}
	defer f.Close()
	info, e := f.Stat()
	if e != nil || info.Size() > 2<<20 {
		return Song{}, false
	}
	var s Song
	if json.NewDecoder(f).Decode(&s) != nil || len(s.Lines) == 0 && s.Plain == "" {
		return Song{}, false
	}
	s.Credit += " · cached"
	return s, true
}
func (c *Client) store(t library.Track, s Song) {
	path := c.cachePath(t)
	if path == "" {
		return
	}
	raw, e := json.Marshal(s)
	if e != nil || len(raw) > 2<<20 {
		return
	}
	if os.MkdirAll(c.CacheDir, 0700) != nil {
		return
	}
	f, e := os.CreateTemp(c.CacheDir, ".lyrics-*")
	if e != nil {
		return
	}
	name := f.Name()
	defer os.Remove(name)
	if _, e = f.Write(raw); e != nil {
		f.Close()
		return
	}
	if f.Close() != nil {
		return
	}
	if os.Rename(name, path) != nil {
		return
	}
	// Only this client's hashed JSON cache entries are eligible for eviction.
	entries, e := os.ReadDir(c.CacheDir)
	if e != nil {
		return
	}
	type item struct {
		path string
		time int64
	}
	var files []item
	for _, v := range entries {
		if v.IsDir() || len(v.Name()) != 69 || !strings.HasSuffix(v.Name(), ".json") {
			continue
		}
		if _, e := hex.DecodeString(strings.TrimSuffix(v.Name(), ".json")); e != nil {
			continue
		}
		info, e := v.Info()
		if e == nil {
			files = append(files, item{filepath.Join(c.CacheDir, v.Name()), info.ModTime().UnixNano()})
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].time > files[j].time })
	for i := maxCacheEntries; i < len(files); i++ {
		os.Remove(files[i].path)
	}
}
