package player

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

type Snapshot struct {
	Position, Duration float64
	Volume             float64
	Paused, Active     bool
	End                int
	Error              string
}
type response struct {
	Data  json.RawMessage `json:"data"`
	Error string          `json:"error"`
}
type Engine struct {
	life    sync.Mutex
	closed  bool
	mu      sync.Mutex
	write   sync.Mutex
	conn    net.Conn
	proc    *exec.Cmd
	done    chan struct{}
	socket  string
	seq     int
	pending map[int]chan response
	state   Snapshot
	volume  int
}

const DefaultVolume = 70
const MaxVolume = 100

func New() *Engine {
	return &Engine{pending: map[int]chan response{}, volume: DefaultVolume, state: Snapshot{Volume: DefaultVolume}}
}
func (e *Engine) Snapshot() Snapshot { e.mu.Lock(); defer e.mu.Unlock(); return e.state }
func (e *Engine) Ready() bool        { e.mu.Lock(); defer e.mu.Unlock(); return e.conn != nil }

func (e *Engine) Start() error {
	e.life.Lock()
	defer e.life.Unlock()
	if e.closed {
		return errors.New("audio engine has shut down")
	}
	if e.Ready() {
		return nil
	}
	exe := os.Getenv("WISP_MPV")
	if exe == "" {
		exe = "mpv"
		if self, err := os.Executable(); err == nil {
			for _, relative := range []string{"mpv.exe", filepath.Join("mpv", "mpv.exe")} {
				p := filepath.Join(filepath.Dir(self), relative)
				if _, err = os.Stat(p); err == nil {
					exe = p
					break
				}
			}
		}
	}
	resolved, err := exec.LookPath(exe)
	if err != nil {
		return errors.New("mpv not found: install mpv or set WISP_MPV to mpv.exe")
	}
	socket := socketName()
	e.mu.Lock()
	volume := e.volume
	e.mu.Unlock()
	cmd := exec.Command(resolved, "--no-config", "--idle=yes", "--no-video", "--no-terminal", "--audio-display=no", "--ytdl=no", fmt.Sprintf("--volume=%d", volume), "--input-ipc-server="+socket)
	if output := os.Getenv("WISP_AUDIO_OUTPUT"); output != "" {
		cmd.Args = append(cmd.Args, "--ao="+output)
	}
	hideWindow(cmd)
	if err = cmd.Start(); err != nil {
		return fmt.Errorf("start mpv: %w", err)
	}
	done := make(chan struct{})
	go func() { _ = cmd.Wait(); close(done) }()
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	var conn net.Conn
	for ctx.Err() == nil {
		conn, err = dial(ctx, socket)
		if err == nil {
			break
		}
		select {
		case <-done:
			cleanupSocket(socket)
			return errors.New("mpv exited before opening IPC")
		case <-time.After(40 * time.Millisecond):
		}
	}
	if conn == nil {
		_ = cmd.Process.Kill()
		<-done
		cleanupSocket(socket)
		return errors.New("mpv IPC did not become ready within 4 seconds")
	}
	e.mu.Lock()
	e.conn = conn
	e.proc = cmd
	e.done = done
	e.socket = socket
	e.state = Snapshot{Volume: float64(volume)}
	e.mu.Unlock()
	go e.read(conn)
	for i, p := range []string{"time-pos", "duration", "pause", "idle-active", "volume"} {
		if err = e.Command("observe_property", i+1, p); err != nil {
			e.closeUnlocked()
			return err
		}
	}
	return nil
}

func (e *Engine) Command(args ...any) error { _, err := e.request(args); return err }
func (e *Engine) SetVolume(value int) error {
	if value < 0 || value > MaxVolume {
		return errors.New("volume must be 0–100")
	}
	e.life.Lock()
	defer e.life.Unlock()
	if e.closed {
		return errors.New("audio engine has shut down")
	}
	if e.Ready() {
		if err := e.Command("set_property", "volume", value); err != nil {
			return err
		}
	}
	e.mu.Lock()
	e.volume = value
	e.state.Volume = float64(value)
	e.mu.Unlock()
	return nil
}
func (e *Engine) request(args []any) (json.RawMessage, error) {
	e.mu.Lock()
	conn := e.conn
	if conn == nil {
		e.mu.Unlock()
		return nil, errors.New("nothing is playing")
	}
	e.seq++
	id := e.seq
	ch := make(chan response, 1)
	e.pending[id] = ch
	e.mu.Unlock()
	defer func() { e.mu.Lock(); delete(e.pending, id); e.mu.Unlock() }()
	b, _ := json.Marshal(map[string]any{"command": args, "request_id": id})
	b = append(b, '\n')
	e.write.Lock()
	_ = conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	_, err := conn.Write(b)
	e.write.Unlock()
	if err != nil {
		return nil, errors.New("could not write to mpv IPC")
	}
	select {
	case r := <-ch:
		if r.Error != "success" {
			return nil, fmt.Errorf("mpv: %s", r.Error)
		}
		return r.Data, nil
	case <-time.After(3 * time.Second):
		return nil, errors.New("mpv command timed out")
	}
}
func (e *Engine) Load(path string) error {
	if err := e.Start(); err != nil {
		return err
	}
	if err := e.Command("set_property", "pause", false); err != nil {
		return err
	}
	return e.Command("loadfile", path, "replace")
}
func (e *Engine) read(conn net.Conn) {
	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 4096), 1<<20)
	for scanner.Scan() {
		var m struct {
			Event     string          `json:"event"`
			Name      string          `json:"name"`
			Data      json.RawMessage `json:"data"`
			Request   int             `json:"request_id"`
			Error     string          `json:"error"`
			Reason    string          `json:"reason"`
			FileError string          `json:"file_error"`
		}
		if json.Unmarshal(scanner.Bytes(), &m) != nil {
			continue
		}
		e.mu.Lock()
		if m.Request != 0 {
			if ch, ok := e.pending[m.Request]; ok {
				select {
				case ch <- response{Data: m.Data, Error: m.Error}:
				default:
				}
			}
		}
		if m.Event == "property-change" {
			switch m.Name {
			case "time-pos":
				_ = json.Unmarshal(m.Data, &e.state.Position)
			case "duration":
				_ = json.Unmarshal(m.Data, &e.state.Duration)
			case "pause":
				_ = json.Unmarshal(m.Data, &e.state.Paused)
			case "volume":
				_ = json.Unmarshal(m.Data, &e.state.Volume)
			case "idle-active":
				var idle bool
				if json.Unmarshal(m.Data, &idle) == nil {
					e.state.Active = !idle
				}
			}
		}
		if m.Event == "start-file" {
			e.state.Position = 0
			e.state.Duration = 0
			e.state.Error = ""
		}
		if m.Event == "end-file" {
			if m.Reason == "eof" {
				e.state.End++
			}
			if m.Reason == "error" {
				e.state.Error = "mpv could not play this track: " + m.FileError
			}
		}
		e.mu.Unlock()
	}
	e.mu.Lock()
	if e.conn == conn {
		e.conn = nil
		e.state.Active = false
		for _, ch := range e.pending {
			select {
			case ch <- response{Error: "IPC connection closed"}:
			default:
			}
		}
	}
	e.mu.Unlock()
	_ = conn.Close()
}
func (e *Engine) Close() {
	e.life.Lock()
	defer e.life.Unlock()
	e.closed = true
	e.closeUnlocked()
}
func (e *Engine) closeUnlocked() {
	if e.Ready() {
		_ = e.Command("quit")
	}
	e.mu.Lock()
	conn, proc, done, socket := e.conn, e.proc, e.done, e.socket
	e.conn = nil
	e.proc = nil
	e.done = nil
	e.mu.Unlock()
	if conn != nil {
		_ = conn.Close()
	}
	if proc != nil && done != nil {
		select {
		case <-done:
		case <-time.After(time.Second):
			_ = proc.Process.Kill()
			<-done
		}
	}
	cleanupSocket(socket)
}
