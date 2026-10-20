//go:build windows

package player

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/Microsoft/go-winio"
)

func socketName() string {
	return fmt.Sprintf(`\\.\pipe\wisp-%d-%d`, os.Getpid(), time.Now().UnixNano())
}
func dial(ctx context.Context, name string) (net.Conn, error) {
	return winio.DialPipeContext(ctx, name)
}
func hideWindow(c *exec.Cmd) { c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true} }
func cleanupSocket(string)   {}
