//go:build windows

package mediakeys

import (
	"fmt"
	"runtime"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	wmQuit = 0x0012
)

var user32 = windows.NewLazySystemDLL("user32.dll")
var setHook = user32.NewProc("SetWindowsHookExW")
var unhook = user32.NewProc("UnhookWindowsHookEx")
var callNext = user32.NewProc("CallNextHookEx")
var getMessage = user32.NewProc("GetMessageW")
var peekMessage = user32.NewProc("PeekMessageW")
var postThreadMessage = user32.NewProc("PostThreadMessageW")

type message struct {
	Window         uintptr
	ID             uint32
	WParam, LParam uintptr
	Time           uint32
	X, Y           int32
	Private        uint32
}

type keyboardEvent struct {
	VirtualKey, ScanCode, Flags, Time uint32
	ExtraInfo                         uintptr
}

type Listener struct {
	Events chan Action
	thread uint32
	done   chan struct{}
	once   sync.Once
}

// Start owns only the four media transport keys; ordinary keyboard input is untouched.
func Start() (*Listener, error) {
	l := &Listener{Events: make(chan Action, 16), done: make(chan struct{})}
	ready := make(chan error, 1)
	go l.run(ready)
	if err := <-ready; err != nil {
		<-l.done
		return nil, err
	}
	return l, nil
}

func (l *Listener) run(ready chan<- error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer close(l.done)
	defer close(l.Events)
	l.thread = windows.GetCurrentThreadId()
	var msg message
	// Create the thread message queue before Close can post WM_QUIT.
	peekMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0, 0)
	var down [4]bool
	callback := syscall.NewCallback(func(code int32, kind uintptr, data *keyboardEvent) uintptr {
		if code == 0 {
			vk := data.VirtualKey
			action := Action(0)
			switch vk {
			case 0xB3:
				action = TogglePause
			case 0xB0:
				action = Next
			case 0xB1:
				action = Previous
			case 0xB2:
				action = Stop
			}
			if action != 0 {
				if kind == 0x0100 || kind == 0x0104 {
					if !down[action-1] {
						down[action-1] = true
						select {
						case l.Events <- action:
						default:
						}
					}
				} else if kind == 0x0101 || kind == 0x0105 {
					down[action-1] = false
				}
				return 1
			}
		}
		r, _, _ := callNext.Call(0, uintptr(code), kind, uintptr(unsafe.Pointer(data)))
		return r
	})
	module, _, _ := windows.NewLazySystemDLL("kernel32.dll").NewProc("GetModuleHandleW").Call(0)
	hook, _, err := setHook.Call(13, callback, module, 0)
	if hook == 0 {
		ready <- fmt.Errorf("install media-key handler: %w", err)
		return
	}
	defer unhook.Call(hook)
	ready <- nil
	for {
		ok, _, _ := getMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(ok) <= 0 {
			return
		}
	}
}

func (l *Listener) Close() {
	l.once.Do(func() { postThreadMessage.Call(uintptr(l.thread), wmQuit, 0, 0); <-l.done })
}
