package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"wisp/internal/command"
	"wisp/internal/i18n"
	"wisp/internal/library"
	"wisp/internal/lyrics"
	"wisp/internal/mediakeys"
	"wisp/internal/player"
	"wisp/internal/source"
)

type pulse time.Time
type taskResult struct {
	Previous           bool
	ReplaceQueue       bool
	NextQueue          []string
	ID                 int
	Kind, Name, Notice string
	Tracks             []library.Track
	Warnings           []string
	Track              library.Track
	Err                error
	Report             source.Result
}
type Model struct {
	playHistory                  []string
	mediaPending                 []mediakeys.Action
	languageReturn               string
	butterflyFrame               int
	butterflyGeneration          uint64
	lyricsClient                 *lyrics.Client
	seekBusy                     bool
	pendingSeek                  int
	lyricsSequence               int
	lyricsTiming                 float64
	tab                          int
	libraryView                  browserState
	playlistNames                []string
	playlistOpen, settingsReturn bool
	lyricsOpen, lyricsFollow     bool
	lyricsOffset                 int
	lyricsID, lyricsMessage      string
	lyricsSong                   lyrics.Song
	lyricsCancel                 context.CancelFunc
	State                        library.State
	Path                         string
	Engine                       *player.Engine
	Sources                      *source.Client
	input                        textinput.Model
	width, height                int
	stage                        string
	setup                        bool
	choice, frame                int
	selected                     map[string]bool
	oldTheme                     string
	results                      []string
	index, offset                int
	selectionPending             bool
	pageOffset                   int
	title, message               string
	saveError                    string
	warnings                     []string
	current                      library.Track
	queued                       string
	snapshot                     player.Snapshot
	seenEnd                      int
	sleep                        time.Time
	busy                         bool
	taskKind                     string
	operation                    int
	cancel                       context.CancelFunc
	match                        string
	history                      []string
	historyIndex                 int
	historyDraft                 string
	credentialSource             string
	lastSearch                   string
	completionText               string
	completionIndex              int
	completionHidden             bool
	volume                       int
	volumeBusy                   bool
	Plain, ReducedMotion         bool
}

func New(s library.State, path string, e *player.Engine) *Model {
	if s.Profile.Complete {
		s.Profile.Language = i18n.Normalize(s.Profile.Language)
	}
	in := textinput.New()
	in.Prompt = "› "
	in.Placeholder = "Search music or type /help"
	in.CharLimit = 2048
	in.SetWidth(72)
	in.SetVirtualCursor(false)
	in.Focus()
	m := &Model{State: s, Path: path, Engine: e, Sources: source.New(), input: in, width: 86, height: 30, stage: "ready", title: "Your listening space", message: "Search a song, paste a playlist link, or /add local music.", Plain: os.Getenv("WISP_ASCII") == "1", ReducedMotion: os.Getenv("WISP_REDUCED_MOTION") == "1"}
	if !s.Profile.Complete {
		m.stage = "welcome"
		m.setup = true
		if s.Profile.Language != "en" && s.Profile.Language != "ru" {
			m.stage = "language"
			m.languageReturn = "welcome"
		}
	}
	m.lyricsClient = lyrics.New()
	m.lyricsClient.CacheDir = filepath.Join(filepath.Dir(path), "lyrics-cache")
	m.applyInputTheme()
	m.applyLanguage()
	m.volume = int(e.Snapshot().Volume)
	return m
}
func tick() tea.Cmd {
	return tea.Tick(120*time.Millisecond, func(t time.Time) tea.Msg { return pulse(t) })
}
func (m *Model) Init() tea.Cmd { return tea.Batch(textinput.Blink, tick(), m.butterflyTick()) }
func (m *Model) persist() {
	if err := library.Save(m.Path, m.State); err != nil {
		m.saveError = "Could not save library: " + err.Error()
	} else {
		m.saveError = ""
	}
}
func (m *Model) task(kind string, fn func(context.Context) taskResult) tea.Cmd {
	m.busy = true
	m.taskKind = kind
	m.operation++
	id := m.operation
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	m.cancel = cancel
	return func() tea.Msg { defer cancel(); r := fn(ctx); r.ID = id; r.Kind = kind; return r }
}
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case MediaKeysUnavailable:
		m.message = "Media keys unavailable. Another app may be using them; Space/F4 still pauses."
		return m, nil
	case mediakeys.Action:
		return m, m.handleMediaKey(v)
	case butterflyPulse:
		if v.Generation != m.butterflyGeneration {
			return m, nil
		}
		if m.stage == "ready" {
			m.butterflyFrame++
		}
		return m, m.butterflyTick()
	case lyricResult:
		if v.ID != m.current.ID || v.ID != m.lyricsID || v.Sequence != m.lyricsSequence {
			return m, nil
		}
		m.lyricsSong = v.Song
		m.lyricsMessage = ""
		if v.Err != nil {
			m.lyricsMessage = lyrics.Message(v.Err)
		}
		return m, nil
	case tea.MouseWheelMsg:
		delta := 1
		if v.Button == tea.MouseWheelUp {
			delta = -1
		} else if v.Button != tea.MouseWheelDown {
			return m, nil
		}
		m.move(delta)
		return m, nil
	case tea.WindowSizeMsg:
		m.width = v.Width
		m.height = v.Height
		m.input.SetWidth(max(8, m.width-8))
		if m.stage == "credential" {
			m.input.SetWidth(max(8, m.width-14))
		}
	case pulse:
		m.frame++
		m.snapshot = m.Engine.Snapshot()
		if !m.volumeBusy {
			m.volume = int(m.snapshot.Volume)
		}
		if m.stage == "syncing" && m.frame >= 12 {
			m.State.Profile.Complete = true
			m.stage = "ready"
			m.setup = false
			m.persist()
			m.message = "Your space is ready. Search music, paste a link, or /add a folder."
		}
		if !m.sleep.IsZero() && time.Now().After(m.sleep) && !m.busy {
			m.sleep = time.Time{}
			m.message = "Sleep timer ended."
			return m, tea.Batch(tick(), m.task("stop", func(context.Context) taskResult {
				if !m.Engine.Ready() {
					return taskResult{}
				}
				return taskResult{Err: m.Engine.Command("stop")}
			}))
		}
		if m.snapshot.Error != "" {
			m.message = library.Clean(m.snapshot.Error)
		}
		if m.snapshot.End != m.seenEnd && !m.busy {
			m.seenEnd = m.snapshot.End
			if m.stage == "ready" {
				if len(m.State.Queue) > 0 {
					return m, tea.Batch(tick(), m.playNext())
				}
				m.current = library.Track{}
			}
		}
		if !m.busy && len(m.mediaPending) > 0 {
			action := m.mediaPending[0]
			m.mediaPending = m.mediaPending[1:]
			return m, tea.Batch(tick(), m.handleMediaKey(action))
		}
		return m, tick()
	case seekResult:
		m.seekBusy = false
		if v.Err != nil {
			m.pendingSeek = 0
			m.message = v.Err.Error()
			return m, nil
		}
		if v.TrackID != m.current.ID {
			m.pendingSeek = 0
			return m, nil
		}
		m.snapshot = m.Engine.Snapshot()
		if m.pendingSeek != 0 {
			return m, m.sendSeek()
		}
		return m, nil
	case volumeResult:
		m.volumeBusy = false
		if v.Err != nil {
			m.volume = int(m.Engine.Snapshot().Volume)
			m.message = v.Err.Error()
			return m, nil
		}
		if v.Value != m.volume {
			return m, m.sendVolume()
		}
		return m, nil
	case taskResult:
		if v.ID != m.operation {
			return m, nil
		}
		m.busy = false
		m.cancel = nil
		if v.Err != nil {
			m.queued = ""
			m.message = v.Err.Error()
			return m, nil
		}
		if v.Notice != "" {
			m.message = v.Notice
		}
		switch v.Kind {
		case "search":
			m.selectionPending = true
			m.tab = 0
			m.playlistOpen = false
			m.lyricsOpen = false
			m.results = m.State.Put(v.Tracks)
			m.title = v.Name
			m.warnings = v.Warnings
			m.index = 0
			m.offset = 0
			m.message = searchNotice(len(v.Tracks), v.Report)
			m.persist()
		case "add":
			m.selectionPending = true
			m.tab = 0
			m.playlistOpen = false
			ids := m.State.Put(v.Tracks)
			m.results = ids
			m.title = "Added to your library"
			m.State.Queue = append(m.State.Queue, ids...)
			m.index = 0
			m.offset = 0
			m.message = fmt.Sprintf("Added %d tracks to your queue. /play to begin.", len(ids))
			m.persist()
		case "import":
			m.selectionPending = true
			if m.tab == 0 {
				m.libraryView = browserState{append([]string(nil), m.results...), m.title, m.message, m.warnings, m.index, m.offset, m.pageOffset}
			}
			m.tab = 1
			m.playlistOpen = true
			m.warnings = nil
			if v.Notice != "" {
				m.warnings = append(m.warnings, v.Notice)
			}
			ids := m.State.Put(v.Tracks)
			name := v.Name
			for i := 2; ; i++ {
				if _, ok := m.State.Playlists[name]; !ok {
					break
				}
				name = fmt.Sprintf("%s (%d)", v.Name, i)
			}
			m.State.Playlists[name] = ids
			m.results = ids
			m.title = name
			m.index = 0
			m.offset = 0
			m.message = fmt.Sprintf("Imported %d items. Enter plays the selected track; /open opens its source.", len(ids))
			m.persist()
		case "play":
			if v.Previous {
				if len(m.playHistory) > 0 {
					m.playHistory = m.playHistory[:len(m.playHistory)-1]
				}
			} else if m.current.ID != "" && m.current.ID != v.Track.ID {
				m.playHistory = append(m.playHistory, m.current.ID)
				if len(m.playHistory) > 200 {
					m.playHistory = m.playHistory[len(m.playHistory)-200:]
				}
			}
			selected, _ := m.selectedID()
			follow := v.ReplaceQueue || !m.selectionPending || selected == m.current.ID || selected == v.Track.ID
			if v.ReplaceQueue {
				m.State.Queue = append([]string{}, v.NextQueue...)
			}
			if m.queued != "" && len(m.State.Queue) > 0 && m.State.Queue[0] == m.queued {
				m.State.Queue = m.State.Queue[1:]
				m.persist()
			}
			m.queued = ""
			m.current = v.Track
			m.lyricsTiming = 0
			m.recordRecent(v.Track.ID)
			if follow {
				m.selectionPending = false
				selected = v.Track.ID
			}
			if m.tab != 1 || m.playlistOpen {
				for i, id := range m.results {
					if id == selected {
						m.index = i
						break
					}
				}
			}
			m.message = ""
			m.snapshot = m.Engine.Snapshot()
			if m.lyricsOpen {
				return m, m.fetchLyrics()
			}
		case "stop":
			m.current = library.Track{}
		}
		return m, nil
	case tea.KeyPressMsg:
		key := v.String()
		if key == "ctrl+c" {
			if m.lyricsCancel != nil {
				m.lyricsCancel()
			}
			if m.cancel != nil {
				m.cancel()
			}
			return m, tea.Quit
		}
		if key == "esc" {
			if m.stage == "language" {
				if !m.setup {
					m.stage = "settings"
					m.choice = 4
				}
				return m, nil
			}
			if m.lyricsOpen {
				m.lyricsOpen = false
				return m, nil
			}
			if m.stage == "settings" {
				m.stage = "ready"
				return m, nil
			}
			if m.stage == "theme" && m.settingsReturn {
				m.State.Profile.Theme = m.oldTheme
				m.applyInputTheme()
				m.settingsReturn = false
				m.stage = "settings"
				m.choice = 0
				return m, nil
			}
			if m.stage == "ready" && m.tab == 1 && m.playlistOpen {
				m.switchTab(1)
				return m, nil
			}
			if m.stage == "credential" {
				m.endCredential()
				m.message = "Connection setup cancelled."
				return m, nil
			}
			if m.busy && (m.taskKind == "play" || m.taskKind == "control" || m.taskKind == "stop") {
				m.message = "Finishing the audio command…"
				return m, nil
			}
			if m.busy && m.cancel != nil {
				m.cancel()
				m.operation++
				m.busy = false
				m.message = "Cancelled."
				return m, nil
			}
			if !m.setup && (m.stage == "theme" || m.stage == "sources") {
				if m.stage == "theme" {
					m.State.Profile.Theme = m.oldTheme
					m.applyInputTheme()
				}
				m.stage = "ready"
				return m, nil
			}
			if len(m.completionItems()) > 0 {
				m.completionText = m.input.Value()
				m.completionHidden = true
				return m, nil
			}
			m.input.SetValue("")
			return m, nil
		}
		if m.stage == "language" {
			return m, m.languageKey(key)
		}
		if m.stage == "settings" {
			return m, m.settingsKey(key)
		}
		if m.stage == "welcome" {
			if key == "enter" {
				m.stage = "username"
				m.message = ""
				m.input.Placeholder = m.tr("Your name")
			}
			return m, nil
		}
		if m.stage == "credential" && key == "enter" {
			name := m.credentialSource
			if err := m.Sources.SetCredential(name, m.input.Value()); err != nil {
				m.message = err.Error()
				return m, nil
			}
			m.endCredential()
			m.State.Profile.Sources[name] = true
			m.persist()
			m.title = "Source settings"
			m.results = nil
			m.warnings = nil
			m.message = source.Label(name) + " credential supplied for this session. Search to verify access."
			if m.lastSearch != "" {
				return m, m.search(m.lastSearch, m.State.Profile.Sources)
			}
			return m, nil
		}
		if m.stage == "theme" || m.stage == "sources" {
			count := len(Themes)
			if m.stage == "sources" {
				count = len(source.Names) + 1
			}
			if key == "up" {
				m.choice = (m.choice + count - 1) % count
			}
			if key == "down" {
				m.choice = (m.choice + 1) % count
			}
			if m.stage == "theme" {
				m.State.Profile.Theme = Themes[m.choice].ID
				m.applyInputTheme()
				if key == "enter" {
					if m.setup {
						m.beginSources()
					} else {
						m.stage = "ready"
						m.persist()
						m.message = "Theme updated."
						if m.settingsReturn {
							m.settingsReturn = false
							m.stage = "settings"
							m.choice = 0
						}
					}
				}
			} else if key == "enter" || key == "space" {
				if m.choice < len(source.Names) {
					n := source.Names[m.choice]
					m.selected[n] = !m.selected[n]
				} else if key == "enter" {
					m.State.Profile.Sources = m.selected
					if m.setup {
						m.stage = "syncing"
						m.frame = 0
					} else {
						m.stage = "ready"
						m.persist()
						m.message = "Search sources updated."
					}
				}
			}
			return m, nil
		}
		if m.stage == "syncing" {
			return m, nil
		}
		if m.stage == "username" && key == "enter" {
			name := strings.TrimSpace(library.Clean(m.input.Value()))
			if utf8.RuneCountInString(name) < 1 || utf8.RuneCountInString(name) > 32 {
				m.message = "Use 1–32 characters."
				return m, nil
			}
			m.State.Profile.Username = name
			m.input.SetValue("")
			m.input.Placeholder = m.tr("Search music or type /help")
			m.beginTheme()
			return m, nil
		}
		if m.stage == "ready" {
			if key == "f4" || (v.Code == tea.KeySpace && v.Mod == 0 && m.input.Value() == "") {
				if !m.busy && m.current.ID != "" && m.Engine.Ready() {
					return m, m.control("cycle", "pause")
				}
				return m, nil
			}
			if key == "f5" && m.lyricsOpen && m.current.ID != "" {
				return m, m.fetchLyrics()
			}
			if key == "f2" {
				return m, m.toggleLyrics()
			}
			if m.lyricsOpen && key == "home" {
				m.lyricsFollow = true
				return m, nil
			}
			if v.Code == tea.KeyEnter && v.Mod&tea.ModShift != 0 && m.input.Value() == "" {
				m.likeSelected()
				return m, nil
			}
			if m.input.Value() == "" {
				if delta := seekDelta(v); delta != 0 {
					return m, m.seekRelative(delta)
				}
				if delta := volumeDelta(v); delta != 0 {
					return m, m.setVolume(m.volume + delta)
				}
			}
			m.refreshCompletion()
			if items := m.completionItems(); len(items) > 0 {
				switch key {
				case "up", "down", "shift+tab":
					delta := 1
					if key != "down" {
						delta = -1
					}
					m.completionIndex = (m.selectedCompletion(items) + delta + len(items)) % len(items)
					return m, nil
				case "tab":
					m.input.SetValue(items[m.selectedCompletion(items)].Insert)
					m.input.CursorEnd()
					m.refreshCompletion()
					return m, nil
				}
			}
			if key == "tab" || key == "shift+tab" {
				delta := 1
				if key == "shift+tab" {
					delta = -1
				}
				m.switchTab(m.tab + delta)
				return m, nil
			}
			if key == "pgdown" || key == "pgup" {
				delta := max(1, m.height-14)
				if key == "pgup" {
					delta = -delta
				}
				m.move(delta)
				return m, nil
			}
			if key == "ctrl+p" && len(m.history) > 0 {
				if m.historyIndex == len(m.history) {
					m.historyDraft = m.input.Value()
				}
				m.historyIndex = max(0, m.historyIndex-1)
				m.input.SetValue(m.history[m.historyIndex])
				m.input.CursorEnd()
				return m, nil
			}
			if key == "ctrl+n" && len(m.history) > 0 {
				m.historyIndex = min(len(m.history), m.historyIndex+1)
				if m.historyIndex == len(m.history) {
					m.input.SetValue(m.historyDraft)
				} else {
					m.input.SetValue(m.history[m.historyIndex])
				}
				m.input.CursorEnd()
				return m, nil
			}
			if m.input.Value() == "" && (key == "up" || key == "down") {
				delta := 1
				if key == "up" {
					delta = -1
				}
				m.move(delta)
				return m, nil
			}
			if key == "enter" {
				if m.busy {
					m.message = "Working… Esc cancels the current request."
					return m, nil
				}
				line := strings.TrimSpace(m.input.Value())
				m.input.SetValue("")
				if line == "" {
					return m, m.activate()
				}
				parsed, _ := command.Parse(line)
				if parsed.Name != "connect" {
					m.history = append(m.history, line)
				}
				if len(m.history) > 100 {
					m.history = m.history[len(m.history)-100:]
				}
				m.historyIndex = len(m.history)
				return m, m.dispatch(line)
			}
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.refreshCompletion()
	return m, cmd
}
func (m *Model) beginTheme() {
	m.oldTheme = m.State.Profile.Theme
	m.stage = "theme"
	for i, t := range Themes {
		if t.ID == m.State.Profile.Theme {
			m.choice = i
		}
	}
}
func (m *Model) beginSources() {
	m.stage = "sources"
	m.choice = 0
	m.selected = map[string]bool{}
	for k, v := range m.State.Profile.Sources {
		m.selected[k] = v
	}
}
func (m *Model) play(id string) tea.Cmd {
	m.pendingSeek = 0
	m.queued = ""
	t, ok := m.State.Tracks[id]
	if !ok {
		m.message = "Track no longer exists."
		return nil
	}
	resolved := m.State.Resolve(id)
	m.message = "Opening " + t.Title + "…"
	return m.task("play", func(ctx context.Context) taskResult {
		raw, err := m.Sources.Stream(ctx, resolved)
		if err == nil {
			err = ctx.Err()
		}
		if err == nil {
			err = m.Engine.Load(raw)
		}
		if resolved.ID != t.ID {
			t.Title += " [via " + resolved.Source + "]"
		}
		return taskResult{Track: t, Err: err}
	})
}
func (m *Model) playNext() tea.Cmd {
	if len(m.State.Queue) == 0 {
		m.message = "Your queue is empty. /queue add 1"
		return nil
	}
	id := m.State.Queue[0]
	cmd := m.play(id)
	m.queued = id
	return cmd
}
