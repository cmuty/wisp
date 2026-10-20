package player

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestIPCRequestsAndEvents(t *testing.T) {
	a, b := net.Pipe()
	defer b.Close()
	e := New()
	e.conn = a
	go e.read(a)
	go func() {
		s := bufio.NewScanner(b)
		for s.Scan() {
			var r struct {
				Request int   `json:"request_id"`
				Command []any `json:"command"`
			}
			_ = json.Unmarshal(s.Bytes(), &r)
			fmt.Fprintf(b, "{\"request_id\":%d,\"error\":\"success\"}\n", r.Request)
			fmt.Fprint(b, "{\"event\":\"property-change\",\"name\":\"time-pos\",\"data\":42.5}\n")
		}
	}()
	if err := e.Command("set_property", "pause", true); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) && e.Snapshot().Position != 42.5 {
		time.Sleep(time.Millisecond)
	}
	if e.Snapshot().Position != 42.5 {
		t.Fatal("property events not applied")
	}
	e.Close()
	if e.Ready() {
		t.Fatal("IPC remained open")
	}
	if err := e.Start(); err == nil {
		t.Fatal("closed engine restarted")
	}
}

// Opt-in integration uses silence and a null audio device, so it never makes sound.
func TestRealMPV(t *testing.T) {
	if os.Getenv("WISP_TEST_MPV") == "" {
		t.Skip("set WISP_TEST_MPV to test a real mpv executable")
	}
	t.Setenv("WISP_MPV", os.Getenv("WISP_TEST_MPV"))
	t.Setenv("WISP_AUDIO_OUTPUT", "null")
	path := filepath.Join(t.TempDir(), "silence.wav")
	writeSilence(t, path, 4)
	e := New()
	defer e.Close()
	if err := e.Load(path); err != nil {
		t.Fatal(err)
	}
	if err := e.SetVolume(31); err != nil {
		t.Fatal(err)
	}
	actual, err := e.request([]any{"get_property", "volume"})
	if err != nil {
		t.Fatal(err)
	}
	var value float64
	if json.Unmarshal(actual, &value) != nil || value != 31 {
		t.Fatalf("actual mpv volume: %s (%v)", actual, err)
	}
	waitFor(t, func() bool { s := e.Snapshot(); return s.Active && s.Duration > 3.9 && s.Position > 0.1 })
	if err := e.Command("set_property", "pause", true); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool { return e.Snapshot().Paused })
	if err := e.Command("seek", 3.7, "absolute"); err != nil {
		t.Fatal(err)
	}
	if err := e.Command("set_property", "pause", false); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool { return e.Snapshot().End == 1 })
	if e.Snapshot().Error != "" {
		t.Fatal(e.Snapshot().Error)
	}
	e.Close()
	if e.Ready() {
		t.Fatal("mpv did not close")
	}
}
func waitFor(t *testing.T, fn func() bool) {
	t.Helper()
	deadline := time.Now().Add(6 * time.Second)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("mpv did not reach the expected playback state")
}
func writeSilence(t *testing.T, path string, seconds int) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	dataSize := uint32(8000 * 2 * seconds)
	_, _ = f.WriteString("RIFF")
	_ = binary.Write(f, binary.LittleEndian, dataSize+36)
	_, _ = f.WriteString("WAVEfmt ")
	for _, v := range []any{uint32(16), uint16(1), uint16(1), uint32(8000), uint32(16000), uint16(2), uint16(16)} {
		_ = binary.Write(f, binary.LittleEndian, v)
	}
	_, _ = f.WriteString("data")
	_ = binary.Write(f, binary.LittleEndian, dataSize)
	_, _ = f.Write(make([]byte, dataSize))
}
