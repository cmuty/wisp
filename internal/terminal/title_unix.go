//go:build !windows

package terminal

// The TUI View supplies the OSC window title on other platforms.
func SetTitle(string) func() { return func() {} }
