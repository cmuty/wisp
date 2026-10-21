package app

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"wisp/internal/command"
	"wisp/internal/library"
	"wisp/internal/source"
)

var help = []string{
	"/language en|ru       Choose English or Russian",
	"Tab / Shift+Tab: Library · Playlists · Likes · Recent",
	"Enter: play / pause selected; Shift+Enter: like selected",
	"Mouse wheel / up/down: browse; F2 or /lyrics: lyrics",
	"Left/right: seek ±5s; Shift+left/right: ±1s (empty input)",
	"Playing a result queues the tracks following it in the list.",
	"/settings             Theme and enabled search sources",
	"/recent               Recently played tracks",
	"/lyrics [offset 0.5]  Show lyrics; align a matched recording",
	"Plain text searches all enabled sources + your local library.",
	"/add \"C:\\Music\"       Add an audio file or folder to the queue",
	"/import URL [name]     Import a whole playlist as a local snapshot",
	"/import file.m3u       Import local paths from M3U / M3U8",
	"/search words          Search connected sources + local music",
	"/connect [source]      Supply a token/key in a hidden input",
	"/play [n]  /pause      Play result n or queue; toggle pause",
	"/next  /stop           Next queued track; stop playback",
	"/seek +15  /volume 70  Seek in seconds; volume 0–100",
	"+ / - volume ±5; Shift with + / - adjusts ±1 (empty input).",
	"/queue [add n|remove n|clear]   Inspect or edit your queue",
	"/like [n]  /likes      Toggle a like; list your liked tracks",
	"/playlists             List local playlists",
	"/playlists new|save|open|delete \"name\"",
	"/playlists add \"name\" n    Add result n to a local playlist",
	"/themes  /profile      Theme picker; sources and connection status",
	"/profile name \"name\"  Rename your local profile",
	"/open [n]              Open the original service page",
	"/match [n]  /bind n    Find a local/SoundCloud recording; confirm match",
	"/sleep 30m|off         Stop audio when the timer expires",
	"/quit                  Save and exit",
	"Type / for hints. Tab completes; ↑/↓ choose; Enter runs.",
	"Ctrl+P/N: history. PgUp/PgDn: scroll. Esc: hide hints/back.",
}

func (m *Model) dispatch(line string) tea.Cmd {
	m.pageOffset = 0
	in, err := command.Parse(line)
	if err != nil {
		m.message = err.Error()
		return nil
	}
	a := in.Args
	switch in.Name {
	case "help", "connect", "queue", "match", "bind":
		if m.tab != 0 {
			m.switchTab(0)
		}
	}
	fail := func(s string) tea.Cmd { m.message = s; return nil }
	switch in.Name {
	case "language":
		if len(a) == 0 {
			m.beginLanguage()
			return nil
		}
		if len(a) != 1 || (a[0] != "en" && a[0] != "ru") {
			return fail("Use /language en or /language ru.")
		}
		m.State.Profile.Language = a[0]
		m.applyLanguage()
		m.persist()
		return fail("Language updated.")
	case "settings":
		m.stage = "settings"
		m.choice = 0
	case "recent":
		m.switchTab(3)
	case "lyrics":
		if len(a) > 0 {
			if len(a) != 2 || a[0] != "offset" {
				return fail("Use /lyrics or /lyrics offset <seconds>.")
			}
			n, e := strconv.ParseFloat(a[1], 64)
			if e != nil || n != n || n < -120 || n > 120 {
				return fail("Lyrics offset must be within ±120 seconds.")
			}
			m.lyricsTiming = n
			m.lyricsOpen = true
			return nil
		}
		return m.toggleLyrics()
	case "help":
		m.results = nil
		m.title = "A little help"
		m.message = strings.Join(help, "\n")
		m.warnings = nil
	case "quit", "exit":
		return tea.Quit
	case "themes":
		m.beginTheme()
	case "connect":
		if len(a) == 0 {
			m.results = nil
			m.warnings = nil
			m.title = "Account credentials (optional)"
			m.message = "Public SoundCloud / YouTube Music search needs no account token.\nSpotify public links import metadata; audio matches YouTube.\n\n/connect soundcloud   Optional OAuth access token\n/connect spotify      Optional OAuth access token\n/connect youtube      Optional YouTube Data API key\n\nPaste credentials only in the hidden input; kept for this run.\nSetup guide: docs/CONNECTIONS.md"
			return nil
		}
		if len(a) != 1 {
			return fail("Use /connect source, then paste the credential in the hidden input.")
		}
		return m.beginCredential(strings.ToLower(a[0]))
	case "profile", "sources":
		if len(a) > 0 && a[0] == "name" {
			name := library.Clean(strings.TrimSpace(strings.Join(a[1:], " ")))
			if len([]rune(name)) < 1 || len([]rune(name)) > 32 {
				return fail("Use /profile name with 1–32 characters.")
			}
			m.State.Profile.Username = name
			m.persist()
			return fail("Your name is now " + name + ".")
		}
		m.beginSources()
	case "search":
		q := strings.TrimSpace(strings.Join(a, " "))
		if q == "" {
			return fail("Use /search words, or just type a search.")
		}
		m.match = ""
		return m.search(q, m.State.Profile.Sources)
	case "add":
		if len(a) != 1 {
			return fail("Use /add \"C:\\Music\" or /add \"C:\\Music\\song.flac\".")
		}
		path := a[0]
		if strings.HasPrefix(path, "https://") {
			return m.dispatch("/import \"" + strings.ReplaceAll(path, "\"", "\"\"") + "\"")
		}
		return m.task("add", func(context.Context) taskResult {
			ts, err := library.Scan(path)
			return taskResult{Tracks: ts, Err: err}
		})
	case "import":
		if len(a) < 1 || len(a) > 2 {
			return fail("Use /import URL [\"playlist name\"] or /import file.m3u.")
		}
		raw := a[0]
		custom := ""
		if len(a) == 2 {
			custom = a[1]
		}
		if strings.HasPrefix(raw, "https://") {
			src, _, err := source.PlaylistURL(raw)
			if err != nil {
				return fail(err.Error())
			}
			if !m.State.Profile.Sources[src] {
				return fail("Enable " + src + " in /profile before importing.")
			}
		}
		return m.task("import", func(ctx context.Context) taskResult {
			var r source.Import
			var err error
			if strings.HasPrefix(raw, "https://") {
				r, err = m.Sources.Import(ctx, raw)
			} else if strings.EqualFold(filepath.Ext(raw), ".csv") {
				r, err = source.ReadCSV(raw)
			} else {
				r, err = readM3U(raw)
			}
			if custom != "" {
				r.Name = library.Clean(custom)
			}
			return taskResult{Name: r.Name, Tracks: r.Tracks, Err: err, Notice: r.Notice}
		})
	case "play":
		if len(a) == 0 {
			if len(m.State.Queue) > 0 {
				return m.playNext()
			}
			if len(m.results) > 0 {
				return m.playFromResults(m.index)
			}
			return fail("Add a track or search first.")
		}
		_, err := m.resultID(a[0])
		if err != nil {
			return fail(err.Error())
		}
		n, _ := strconv.Atoi(a[0])
		return m.playFromResults(n - 1)
	case "pause":
		return m.control("cycle", "pause")
	case "stop":
		return m.task("stop", func(context.Context) taskResult {
			if !m.Engine.Ready() {
				return taskResult{}
			}
			return taskResult{Err: m.Engine.Command("stop")}
		})
	case "next":
		return m.playNext()
	case "seek":
		if len(a) != 1 {
			return fail("Use /seek +15, /seek -10 or /seek 60 (seconds).")
		}
		n, err := strconv.ParseFloat(a[0], 64)
		if err != nil || n < -86400 || n > 86400 || n != n {
			return fail("Seek must be a finite number within ±86400 seconds.")
		}
		mode := "absolute"
		if strings.HasPrefix(a[0], "+") || strings.HasPrefix(a[0], "-") {
			mode = "relative"
		}
		return m.control("seek", n, mode)
	case "volume":
		if len(a) != 1 {
			return fail("Use /volume 0–100.")
		}
		n, err := strconv.Atoi(a[0])
		if err != nil || n < 0 || n > 100 {
			return fail("Volume must be 0–100.")
		}
		return m.setVolume(n)
	case "queue":
		if len(a) > 0 {
			switch a[0] {
			case "clear":
				m.State.Queue = nil
			case "add":
				if len(a) != 2 {
					return fail("Use /queue add n.")
				}
				id, err := m.resultID(a[1])
				if err != nil {
					return fail(err.Error())
				}
				m.State.Queue = append(m.State.Queue, id)
			case "remove":
				if len(a) != 2 {
					return fail("Use /queue remove n (queue position).")
				}
				n, err := strconv.Atoi(a[1])
				if err != nil || n < 1 || n > len(m.State.Queue) {
					return fail("Queue position is out of range.")
				}
				m.State.Queue = append(m.State.Queue[:n-1], m.State.Queue[n:]...)
			default:
				return fail("Use /queue [add n|remove n|clear].")
			}
			m.persist()
		}
		m.show("Your queue", m.State.Queue)
	case "like":
		id := ""
		if len(a) > 0 {
			id, err = m.resultID(a[0])
		} else if m.current.ID != "" {
			id = m.current.ID
		} else {
			id, err = m.selectedID()
		}
		if err != nil || id == "" {
			return fail("Select a result, play a track, or use /like n.")
		}
		m.toggleLike(id)
	case "likes":
		m.switchTab(2)
	case "playlists":
		return m.playlists(a)
	case "sleep":
		if len(a) != 1 {
			return fail("Use /sleep 30m, /sleep 1h or /sleep off.")
		}
		if a[0] == "off" {
			m.sleep = time.Time{}
			return fail("Sleep timer off.")
		}
		d, err := time.ParseDuration(a[0])
		if err != nil || d <= 0 || d > 24*time.Hour {
			return fail("Use a duration between 1ns and 24h, e.g. /sleep 30m.")
		}
		m.sleep = time.Now().Add(d)
		m.message = "Audio will stop in " + d.String() + "."
	case "open":
		id, err := m.selectedID()
		if len(a) > 0 {
			id, err = m.resultID(a[0])
		}
		if err != nil {
			return fail(err.Error())
		}
		t := m.State.Tracks[id]
		u, err := url.Parse(t.URL)
		if err != nil || u.Scheme != "https" || u.User != nil {
			return fail("This item has no service page.")
		}
		allowed := map[string]bool{"soundcloud.com": true, "www.soundcloud.com": true, "open.spotify.com": true, "www.youtube.com": true}
		if !allowed[strings.ToLower(u.Host)] {
			return fail("Unrecognized service page host.")
		}
		return m.task("open", func(context.Context) taskResult {
			return taskResult{Err: openURL(u.String()), Notice: "Opened the original service player."}
		})
	case "match":
		id, err := m.selectedID()
		if len(a) > 0 {
			id, err = m.resultID(a[0])
		}
		if err != nil {
			return fail(err.Error())
		}
		t := m.State.Tracks[id]
		m.match = id
		enabled := map[string]bool{"soundcloud": m.State.Profile.Sources["soundcloud"]}
		return m.search(t.Artist+" "+t.Title, enabled)
	case "bind":
		if m.match == "" || len(a) != 1 {
			return fail("Use /match n first, then /bind candidate-number.")
		}
		id, err := m.resultID(a[0])
		if err != nil {
			return fail(err.Error())
		}
		t := m.State.Tracks[id]
		if id == m.match || (t.Source != "local" && t.Source != "soundcloud") || t.Unavailable {
			return fail("Choose an available local or SoundCloud recording.")
		}
		m.State.Bindings[m.match] = id
		m.persist()
		m.match = ""
		m.message = "Recording linked. The original playlist entry is preserved."
	default:
		return fail("Unknown command /" + in.Name + ". Type /help.")
	}
	return nil
}
func (m *Model) control(args ...any) tea.Cmd {
	return m.task("control", func(context.Context) taskResult { return taskResult{Err: m.Engine.Command(args...)} })
}
func (m *Model) search(q string, enabled map[string]bool) tea.Cmd {
	m.lastSearch = q
	local := m.State.LocalSearch(q)
	selection := map[string]bool{}
	for k, v := range enabled {
		selection[k] = v
	}
	m.message = "Searching…"
	return m.task("search", func(ctx context.Context) taskResult {
		r := m.Sources.Search(ctx, q, selection, local)
		return taskResult{Name: "Search: " + q, Tracks: r.Tracks, Warnings: r.Warnings, Report: r}
	})
}
func (m *Model) resultID(s string) (string, error) {
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 || n > len(m.results) {
		return "", errors.New("result number is out of range")
	}
	return m.results[n-1], nil
}
func (m *Model) selectedID() (string, error) {
	if m.index < 0 || m.index >= len(m.results) {
		return "", errors.New("search or open a playlist first")
	}
	return m.results[m.index], nil
}
func (m *Model) show(title string, ids []string) {
	m.selectionPending = true
	m.title = title
	m.results = append([]string{}, ids...)
	m.index = 0
	m.offset = 0
	m.message = fmt.Sprintf("%d tracks. /play n or /queue add n.", len(ids))
	m.warnings = nil
	m.match = ""
}
func (m *Model) playlists(a []string) tea.Cmd {
	fail := func(s string) tea.Cmd { m.message = s; return nil }
	if len(a) == 0 {
		m.switchTab(1)
		return nil
	}
	if len(a) < 2 {
		return fail("Use /playlists new|save|open|delete \"name\" or /playlists add \"name\" n.")
	}
	name := library.Clean(a[1])
	if len(name) == 0 || len([]rune(name)) > 80 {
		return fail("Playlist names must be 1–80 characters.")
	}
	ids, exists := m.State.Playlists[name]
	switch a[0] {
	case "new":
		if exists {
			return fail("A playlist with that name already exists.")
		}
		m.State.Playlists[name] = []string{}
	case "save":
		if exists {
			return fail("Name already exists. Choose a new playlist name.")
		}
		m.State.Playlists[name] = append([]string{}, m.State.Queue...)
	case "open":
		if !exists {
			return fail("Playlist not found.")
		}
		m.switchTab(1)
		m.show(name, ids)
		m.playlistOpen = true
		return nil
	case "delete":
		if !exists {
			return fail("Playlist not found.")
		}
		delete(m.State.Playlists, name)
	case "add":
		if !exists {
			return fail("Create the playlist first.")
		}
		if len(a) != 3 {
			return fail("Use /playlists add \"name\" n.")
		}
		id, err := m.resultID(a[2])
		if err != nil {
			return fail(err.Error())
		}
		m.State.Playlists[name] = append(ids, id)
	default:
		return fail("Unknown playlist action.")
	}
	m.persist()
	if m.tab == 1 && !m.playlistOpen && (a[0] == "new" || a[0] == "save" || a[0] == "delete") {
		m.switchTab(1)
	} else if m.tab == 1 && m.playlistOpen && m.title == name {
		m.show(name, m.State.Playlists[name])
	}
	return fail("Playlist updated: " + name)
}
func readM3U(path string) (source.Import, error) {
	r := source.Import{Name: strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))}
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".m3u" && ext != ".m3u8" {
		return r, errors.New("local playlist must be .m3u or .m3u8")
	}
	f, err := os.Open(path)
	if err != nil {
		return r, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 4096), 64<<10)
	for scanner.Scan() {
		s := strings.TrimSpace(strings.TrimPrefix(scanner.Text(), "\ufeff"))
		if s == "" || strings.HasPrefix(s, "#") {
			continue
		}
		if strings.Contains(s, "://") {
			return r, errors.New("M3U imports support local audio paths only")
		}
		if !filepath.IsAbs(s) {
			s = filepath.Join(filepath.Dir(path), s)
		}
		t, err := library.Local(s)
		if err != nil {
			return r, fmt.Errorf("playlist entry %d: %w", len(r.Tracks)+1, err)
		}
		r.Tracks = append(r.Tracks, t)
		if len(r.Tracks) > library.MaxTracks {
			return r, errors.New("playlist exceeds 10,000 tracks")
		}
	}
	return r, scanner.Err()
}
func openURL(u string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", u)
	case "darwin":
		cmd = exec.Command("open", u)
	default:
		cmd = exec.Command("xdg-open", u)
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}
