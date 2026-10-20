//go:build windows

package terminal

import (
	"golang.org/x/sys/windows"
	"unsafe"
)

var kernel = windows.NewLazySystemDLL("kernel32.dll")
var getTitle = kernel.NewProc("GetConsoleTitleW")
var setTitle = kernel.NewProc("SetConsoleTitleW")

// SetTitle also works in the legacy Windows console, independent of OSC support.
func SetTitle(title string) func() {
	var buffer [4096]uint16
	n, _, _ := getTitle.Call(uintptr(unsafe.Pointer(&buffer[0])), uintptr(len(buffer)))
	old := windows.UTF16ToString(buffer[:min(n, uintptr(len(buffer)))])
	apply := func(s string) {
		p, err := windows.UTF16PtrFromString(s)
		if err == nil {
			_, _, _ = setTitle.Call(uintptr(unsafe.Pointer(p)))
		}
	}
	apply(title)
	return func() { apply(old) }
}
