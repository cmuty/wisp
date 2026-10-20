//go:build !windows

package source

import "os/exec"

func hideExtractor(*exec.Cmd) {}
