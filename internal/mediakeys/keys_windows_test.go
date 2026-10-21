//go:build windows

package mediakeys

import (
	"os"
	"testing"
	"time"
)

func TestWindowsMediaKeysAndCleanup(t *testing.T) {
	if os.Getenv("WISP_TEST_MEDIA_KEYS") != "1" {
		t.Skip("set WISP_TEST_MEDIA_KEYS=1 for Windows media-key integration")
	}
	l, err := Start()
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	emit := user32.NewProc("keybd_event")
	for _, item := range []struct {
		vk     uintptr
		action Action
	}{{0xB3, TogglePause}, {0xB0, Next}, {0xB1, Previous}, {0xB2, Stop}} {
		emit.Call(item.vk, 0, 0, 0)
		emit.Call(item.vk, 0, 2, 0)
		select {
		case action := <-l.Events:
			if action != item.action {
				t.Fatalf("got %v, want %v", action, item.action)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("Windows media event did not arrive", item.action)
		}
	}
	l.Close()
	l2, err := Start()
	if err != nil {
		t.Fatal("media keys were not released on shutdown", err)
	}
	l2.Close()
}
