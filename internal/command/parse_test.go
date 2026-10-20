package command

import (
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	cases := []struct {
		line, name string
		args       []string
		bad        bool
	}{
		{`/add "C:\My Music\track.flac"`, "add", []string{`C:\My Music\track.flac`}, false},
		{`/playlists new "Night ""drive"""`, "playlists", []string{"new", `Night "drive"`}, false},
		{`/search 'quiet piano'`, "search", []string{"quiet piano"}, false},
		{`a song with spaces`, "search", []string{"a song with spaces"}, false},
		{`https://music.youtube.com/playlist?list=PL123`, "import", []string{"https://music.youtube.com/playlist?list=PL123"}, false},
		{`/add "broken`, "", nil, true},
	}
	for _, c := range cases {
		t.Run(c.line, func(t *testing.T) {
			r, err := Parse(c.line)
			if (err != nil) != c.bad {
				t.Fatalf("error: %v", err)
			}
			if err == nil && (r.Name != c.name || !reflect.DeepEqual(r.Args, c.args)) {
				t.Fatalf("got %#v", r)
			}
		})
	}
}
