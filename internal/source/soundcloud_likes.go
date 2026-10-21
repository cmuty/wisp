package source

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"wisp/internal/library"
)

func IsSoundCloudLikes(raw string) bool {
	src, _, err := PlaylistURL(raw)
	if err != nil || src != "soundcloud" {
		return false
	}
	u, _ := url.Parse(raw)
	p := strings.Split(strings.Trim(u.Path, "/"), "/")
	return len(p) == 2 && p[1] == "likes"
}

func (c *Client) importPublicSCLikes(ctx context.Context, raw string) (Import, error) {
	u, _ := url.Parse(raw)
	u.Path = strings.TrimSuffix(strings.TrimRight(u.Path, "/"), "/likes")
	u.RawQuery, u.Fragment = "", ""
	var user struct {
		ID   json.Number `json:"id"`
		Kind string      `json:"kind"`
		Name string      `json:"username"`
	}
	if err := c.scGet(ctx, "/resolve", url.Values{"url": {u.String()}}, &user); err != nil {
		return Import{}, err
	}
	if user.Kind != "user" || user.ID.String() == "" || !identifier.MatchString(user.ID.String()) {
		return Import{}, errors.New("SoundCloud profile is unavailable")
	}
	result := Import{Likes: true, Name: library.Clean(user.Name) + " · Likes"}
	route := "/users/" + user.ID.String() + "/track_likes"
	params := url.Values{"limit": {"200"}, "linked_partitioning": {"1"}}
	seenPages, seenTracks := map[string]bool{}, map[string]bool{}
	for page := 0; page < 1000; page++ {
		key := route + "?" + params.Encode()
		if seenPages[key] {
			return Import{}, errors.New("SoundCloud likes pagination repeated; nothing imported")
		}
		seenPages[key] = true
		var data struct {
			Collection []struct {
				Track *scTrack `json:"track"`
			} `json:"collection"`
			Next string `json:"next_href"`
		}
		if err := c.scGet(ctx, route, params, &data); err != nil {
			return Import{}, err
		}
		for _, like := range data.Collection {
			if like.Track == nil || like.Track.ID.String() == "" {
				continue
			}
			t := like.Track.track()
			if seenTracks[t.ID] {
				continue
			}
			seenTracks[t.ID] = true
			if len(result.Tracks) >= library.MaxTracks {
				return Import{}, errors.New("SoundCloud likes exceed 10,000 tracks; nothing imported")
			}
			if t.Title == "" {
				t.Title = "Unavailable liked track"
				t.Unavailable = true
			}
			result.Tracks = append(result.Tracks, t)
		}
		if data.Next == "" {
			return result, nil
		}
		next, err := url.Parse(data.Next)
		base, _ := url.Parse(c.WebSCAPI)
		if err != nil || next.User != nil || next.Scheme != base.Scheme || next.Host != base.Host || next.Path != route {
			return Import{}, errors.New("unexpected SoundCloud likes pagination URL; nothing imported")
		}
		params = next.Query()
		params.Del("client_id")
	}
	return Import{}, errors.New("SoundCloud likes pagination limit reached; nothing imported")
}
