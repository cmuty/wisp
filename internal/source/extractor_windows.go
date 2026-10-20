//go:build windows

package source

import (
	"os/exec"
	"syscall"
)

func hideExtractor(cmd *exec.Cmd) { cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true} }
