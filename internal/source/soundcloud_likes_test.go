package source

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestSoundCloudLikesPagination(t *testing.T) {
	for _, broken := range []bool{false, true} {
		var base string
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("client_id") != "fixture" {
				t.Error("missing public client")
			}
			switch r.URL.Path {
			case "/resolve":
				if r.URL.Query().Get("url") != "https://soundcloud.com/listener" {
					t.Error("likes URL was not reduced to profile")
				}
				fmt.Fprint(w, `{"kind":"user","id":123,"username":"Listener"}`)
			case "/users/123/track_likes":
				if r.URL.Query().Get("cursor") == "next" {
					if broken {
						http.Error(w, "unavailable", 503)
						return
					}
					fmt.Fprint(w, `{"collection":[{"track":{"id":1,"title":"First"}},{"track":{"id":2,"title":"Second"}}],"next_href":null}`)
				} else {
					fmt.Fprintf(w, `{"collection":[{"track":{"id":1,"title":"First"}}],"next_href":%q}`, base+"/users/123/track_likes?cursor=next")
				}
			default:
				http.NotFound(w, r)
			}
		}))
		base = s.URL
		c := New()
		c.WebSCAPI = base
		c.scClientID = "fixture"
		c.scExpires = time.Now().Add(time.Hour)
		c.PublicWeb = true
		r, err := c.Import(context.Background(), "https://soundcloud.com/listener/likes?utm_source=clipboard")
		s.Close()
		if broken {
			if err == nil || len(r.Tracks) != 0 {
				t.Fatal("partial likes were imported")
			}
			continue
		}
		if err != nil || !r.Likes || len(r.Tracks) != 2 || r.Tracks[1].ID != "soundcloud:2" {
			t.Fatalf("unexpected likes: %+v %v", r, err)
		}
	}
	if IsSoundCloudLikes("https://soundcloud.com.evil.test/listener/likes") || IsSoundCloudLikes("https://soundcloud.com/listener/sets/likes") {
		t.Fatal("incorrect likes classification")
	}
}

func TestLiveSoundCloudLikes(t *testing.T) {
	if os.Getenv("WISP_TEST_LIKES") != "1" {
		t.Skip("set WISP_TEST_LIKES=1 for public likes integration")
	}
	c := New()
	c.PublicWeb = true
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	r, err := c.Import(ctx, "https://soundcloud.com/diadem-630788865/likes?utm_source=clipboard&utm_medium=text&utm_campaign=social_sharing")
	if err != nil || !r.Likes || len(r.Tracks) == 0 {
		t.Fatalf("likes import failed: count=%d error=%v", len(r.Tracks), err)
	}
	t.Logf("public likes imported: %d tracks", len(r.Tracks))
}
