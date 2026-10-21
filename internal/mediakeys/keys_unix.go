//go:build !windows

package mediakeys

import "fmt"

type Listener struct{ Events chan Action }

func Start() (*Listener, error) { return nil, fmt.Errorf("global media keys require Windows") }
func (l *Listener) Close()      {}
