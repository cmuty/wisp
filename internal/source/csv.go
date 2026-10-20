package source

import (
	"bytes"
	"crypto/sha256"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"wisp/internal/library"
)

func ReadCSV(path string) (Import, error) {
	f, err := os.Open(path)
	if err != nil {
		return Import{}, err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, publicMaxBytes+1))
	if err != nil {
		return Import{}, err
	}
	if len(b) > publicMaxBytes {
		return Import{}, errors.New("CSV exceeds 16 MiB")
	}
	b = bytes.TrimPrefix(b, []byte{0xef, 0xbb, 0xbf})
	reader := csv.NewReader(bytes.NewReader(b))
	first, _, _ := bytes.Cut(b, []byte{'\n'})
	if bytes.Count(first, []byte{';'}) > bytes.Count(first, []byte{','}) {
		reader.Comma = ';'
	}
	header, err := reader.Read()
	if err != nil {
		return Import{}, err
	}
	columns := map[string]int{}
	for i, name := range header {
		columns[strings.Join(words(name), "")] = i
	}
	column := func(names ...string) int {
		for _, name := range names {
			if i, ok := columns[name]; ok {
				return i
			}
		}
		return -1
	}
	titleCol := column("trackname", "title", "name")
	artistCol := column("artistnames", "artistname", "artists", "artist")
	uriCol := column("spotifyuri", "trackuri", "uri", "trackurl", "url")
	durationCol := column("durationms", "durationmilliseconds")
	if titleCol < 0 {
		return Import{}, errors.New("CSV needs a Track Name or Title column")
	}
	get := func(row []string, i int) string {
		if i < 0 || i >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[i])
	}
	out := Import{Name: "Imported CSV", Notice: "CSV metadata imported; Spotify entries match public YouTube recordings during playback."}
	for {
		row, e := reader.Read()
		if e == io.EOF {
			break
		}
		if e != nil {
			return Import{}, fmt.Errorf("invalid CSV row %d: %w", len(out.Tracks)+2, e)
		}
		title, artist := get(row, titleCol), get(row, artistCol)
		if title == "" {
			return Import{}, errors.New("CSV contains an empty track title")
		}
		raw := get(row, uriCol)
		id := ""
		if strings.HasPrefix(raw, "spotify:track:") {
			id = strings.TrimPrefix(raw, "spotify:track:")
		} else if i := strings.Index(raw, "open.spotify.com/track/"); i >= 0 {
			id = strings.Split(strings.Split(raw[i+len("open.spotify.com/track/"):], "?")[0], "/")[0]
		}
		if id == "" || !identifier.MatchString(id) {
			h := sha256.Sum256([]byte(artist + "\n" + title))
			id = fmt.Sprintf("csv-%x", h[:12])
			raw = ""
		} else {
			raw = "https://open.spotify.com/track/" + id
		}
		duration, _ := strconv.ParseFloat(get(row, durationCol), 64)
		out.Tracks = append(out.Tracks, library.Track{ID: "spotify:" + id, Source: "spotify", Title: title, Artist: artist, URL: raw, Duration: duration / 1000})
		if len(out.Tracks) > library.MaxTracks {
			return Import{}, errors.New("CSV exceeds 10,000 tracks; nothing imported")
		}
	}
	return out, nil
}
