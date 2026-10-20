//go:build !windows

package player

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func socketName() string {
	return filepath.Join(os.TempDir(), fmt.Sprintf("wisp-%d-%d.sock", os.Getpid(), time.Now().UnixNano()))
}
func dial(ctx context.Context, name string) (net.Conn, error) {
	return (&net.Dialer{}).DialContext(ctx, "unix", name)
}
func hideWindow(*exec.Cmd) {}
func cleanupSocket(name string) {
	if name != "" {
		_ = os.Remove(name)
	}
}
