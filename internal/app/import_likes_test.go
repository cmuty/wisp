package app

import (
	"testing"
	"wisp/internal/library"
)

func TestImportedLikesMergeAndOpenLikes(t *testing.T) {
	m := completionFixture(t)
	m.State.Likes = m.State.Put([]library.Track{{ID: "old", Title: "Existing"}})
	tracks := []library.Track{{ID: "old", Title: "Existing"}, {ID: "new", Title: "Imported"}}
	for i := 0; i < 2; i++ {
		m.Update(taskResult{Kind: "import", ImportLikes: true, Tracks: tracks})
		if m.tab != 2 || len(m.State.Likes) != 2 || len(m.State.Playlists) != 0 || len(m.results) != 2 {
			t.Fatal("likes import did not merge into Likes", m.State.Likes, m.tab)
		}
	}
	s, err := library.Load(m.Path)
	if err != nil || len(s.Likes) != 2 {
		t.Fatal("likes not persisted", err)
	}
	m.index = 1
	if cmd := m.activate(); cmd == nil || m.taskKind != "play" {
		t.Fatal("imported liked track cannot be activated")
	}
}
